package apiclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	authpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	auth authpb.AuthClient
}

type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Company string `json:"company"`
	Email   string `json:"email"`
}

type Session struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

func New(rawBaseURL string) (*Client, error) {
	target, err := grpcTarget(rawBaseURL)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, auth: authpb.NewAuthClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Register(ctx context.Context, name, company, email, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.Create(ctx, &authpb.User{
		Name:     strings.TrimSpace(name),
		Company:  strings.TrimSpace(company),
		Email:    strings.TrimSpace(email),
		Password: password,
	})
	if err != nil {
		return Session{}, err
	}
	if res.GetToken() == nil || !res.GetToken().GetValid() || res.GetToken().GetToken() == "" {
		return Session{}, errors.New("registration did not return a valid token")
	}

	return Session{User: userFromProto(res.GetUser()), Token: res.GetToken().GetToken()}, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	token, err := c.auth.Auth(ctx, &authpb.User{
		Email:    strings.TrimSpace(email),
		Password: password,
	})
	if err != nil {
		return Session{}, err
	}
	if !token.GetValid() || token.GetToken() == "" {
		return Session{}, errors.New("invalid email or password")
	}

	user, err := userFromToken(token.GetToken())
	if err != nil {
		return Session{}, err
	}
	if user.ID != "" {
		if fresh, err := c.GetUser(ctx, user.ID); err == nil {
			user = fresh
		}
	}

	return Session{User: user, Token: token.GetToken()}, nil
}

func (c *Client) Session(ctx context.Context, token string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	valid, err := c.auth.ValidateToken(ctx, &authpb.Token{Token: token})
	if err != nil {
		return Session{}, err
	}
	if !valid.GetValid() {
		return Session{}, errors.New("session expired")
	}

	user, err := userFromToken(token)
	if err != nil {
		return Session{}, err
	}
	if user.ID != "" {
		if fresh, err := c.GetUser(ctx, user.ID); err == nil {
			user = fresh
		}
	}

	return Session{User: user, Token: token}, nil
}

func (c *Client) GetUser(ctx context.Context, id string) (User, error) {
	res, err := c.auth.Get(ctx, &authpb.User{Id: id})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
}

func (c *Client) UpdateProfile(ctx context.Context, user User) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdateProfile(ctx, &authpb.User{
		Id:      user.ID,
		Name:    strings.TrimSpace(user.Name),
		Company: strings.TrimSpace(user.Company),
		Email:   strings.TrimSpace(user.Email),
	})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
}

func (c *Client) UpdateProfileSession(ctx context.Context, user User) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdateProfile(ctx, &authpb.User{
		Id:      user.ID,
		Name:    strings.TrimSpace(user.Name),
		Company: strings.TrimSpace(user.Company),
		Email:   strings.TrimSpace(user.Email),
	})
	if err != nil {
		return Session{}, err
	}

	session := Session{User: userFromProto(res.GetUser())}
	if res.GetToken() != nil && res.GetToken().GetValid() {
		session.Token = res.GetToken().GetToken()
	}
	return session, nil
}

func (c *Client) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdatePassword(ctx, &authpb.PasswordUpdate{
		Id:              id,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
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

func userFromProto(in *authpb.User) User {
	if in == nil {
		return User{}
	}
	return User{
		ID:      in.GetId(),
		Name:    in.GetName(),
		Company: in.GetCompany(),
		Email:   in.GetEmail(),
	}
}

func userFromToken(token string) (User, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return User{}, errors.New("invalid token format")
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
	if claims.User.ID == "" && claims.User.Email == "" {
		return User{}, errors.New("token does not include a user")
	}
	return claims.User, nil
}
