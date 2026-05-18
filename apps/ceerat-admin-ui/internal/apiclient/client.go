package apiclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	authpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn         *grpc.ClientConn
	auth         authpb.AuthClient
	adminBaseURL string
	http         *http.Client
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Session struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Permission struct {
	ID          string `json:"id"`
	RoleID      string `json:"role_id"`
	Role        string `json:"role"`
	GRPCMethod  string `json:"grpc_method"`
	Description string `json:"description"`
}

type AdminData struct {
	Users       []User       `json:"users"`
	Roles       []Role       `json:"roles"`
	Permissions []Permission `json:"permissions"`
	Methods     []string     `json:"methods"`
}

func New(apiBaseURL, adminBaseURL string) (*Client, error) {
	target, err := grpcTarget(apiBaseURL)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:         conn,
		auth:         authpb.NewAuthClient(conn),
		adminBaseURL: strings.TrimRight(adminBaseURL, "/"),
		http:         &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Login(ctx context.Context, email, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	token, err := c.auth.Auth(ctx, &authpb.User{Email: strings.TrimSpace(email), Password: password})
	if err != nil {
		return Session{}, err
	}
	if !token.GetValid() || token.GetToken() == "" {
		return Session{}, errors.New("invalid email or password")
	}
	session, err := c.Session(ctx, token.GetToken())
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (c *Client) Session(ctx context.Context, token string) (Session, error) {
	user, err := userFromToken(token)
	if err != nil {
		return Session{}, err
	}
	var payload struct {
		User User `json:"user"`
	}
	if err := c.admin(ctx, token, http.MethodGet, "/api/admin/me", nil, &payload); err != nil {
		return Session{}, err
	}
	if payload.User.ID != "" {
		user = payload.User
	}
	return Session{User: user, Token: token}, nil
}

func (c *Client) AdminData(ctx context.Context, token string) (AdminData, error) {
	var data AdminData
	if err := c.admin(ctx, token, http.MethodGet, "/api/admin/users", nil, &data); err != nil {
		return AdminData{}, err
	}
	if err := c.admin(ctx, token, http.MethodGet, "/api/admin/roles", nil, &data); err != nil {
		return AdminData{}, err
	}
	if err := c.admin(ctx, token, http.MethodGet, "/api/admin/role-permissions", nil, &data); err != nil {
		return AdminData{}, err
	}
	if err := c.admin(ctx, token, http.MethodGet, "/api/admin/rbac/methods", nil, &data); err != nil {
		return AdminData{}, err
	}
	return data, nil
}

func (c *Client) AdminRequest(ctx context.Context, token, method, path string, body any, out any) error {
	return c.admin(ctx, token, method, path, body, out)
}

func (c *Client) admin(ctx context.Context, token, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.adminBaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&problem)
		if problem.Error == "" {
			problem.Error = resp.Status
		}
		return errors.New(problem.Error)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func grpcTarget(rawBaseURL string) (string, error) {
	if rawBaseURL == "" {
		return "", errors.New("api base url is required")
	}
	if strings.Contains(rawBaseURL, "://") {
		parsed, err := url.Parse(rawBaseURL)
		if err != nil {
			return "", err
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("invalid api base url %q", rawBaseURL)
		}
		return parsed.Host, nil
	}
	return rawBaseURL, nil
}

func userFromToken(token string) (User, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return User{}, errors.New("invalid token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return User{}, err
	}
	var claims struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return User{}, err
	}
	if claims.User.ID == "" {
		return User{}, errors.New("invalid token claims")
	}
	return claims.User, nil
}
