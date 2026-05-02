package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/apiclient"
	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/config"
)

type fakeAPI struct {
	session       apiclient.Session
	profileInput  apiclient.User
	profileOutput apiclient.Session
}

func (api *fakeAPI) Register(ctx context.Context, name, company, email, password string) (apiclient.Session, error) {
	return apiclient.Session{}, nil
}

func (api *fakeAPI) Login(ctx context.Context, email, password string) (apiclient.Session, error) {
	return apiclient.Session{}, nil
}

func (api *fakeAPI) Session(ctx context.Context, token string) (apiclient.Session, error) {
	return api.session, nil
}

func (api *fakeAPI) UpdateProfileSession(ctx context.Context, user apiclient.User) (apiclient.Session, error) {
	api.profileInput = user
	return api.profileOutput, nil
}

func (api *fakeAPI) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) (apiclient.User, error) {
	return apiclient.User{}, nil
}

func TestUpdateProfileUsesSessionUserIDAndRefreshesCookie(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Company: "Old Co", Email: "jane@example.com"},
			Token: "old-token",
		},
		profileOutput: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Company: "New Co", Email: "jane@example.com"},
			Token: "new-token",
		},
	}

	srv, err := New(config.Config{Port: "3000"}, api)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]string{
		"name":    "Jane",
		"company": "New Co",
		"email":   "jane@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/profile", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "old-token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if api.profileInput.ID != "user-1" {
		t.Fatalf("expected session user id to be used, got %#v", api.profileInput)
	}
	if api.profileInput.Company != "New Co" {
		t.Fatalf("expected company update to be forwarded, got %#v", api.profileInput)
	}
	if !strings.Contains(rec.Body.String(), `"company":"New Co"`) {
		t.Fatalf("expected response to include updated company, got %s", rec.Body.String())
	}

	var refreshed bool
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookie && cookie.Value == "new-token" {
			refreshed = true
		}
	}
	if !refreshed {
		t.Fatalf("expected refreshed session cookie, got %#v", rec.Result().Cookies())
	}
}
