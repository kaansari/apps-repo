package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := NewServer(Config{WebUIURL: "http://web.example"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"target":"http://web.example/chatgpt-client"`) {
		t.Fatalf("expected health payload to include redirect target, got %s", rec.Body.String())
	}
}

func TestRedirectsToWebUIChatClient(t *testing.T) {
	srv := NewServer(Config{WebUIURL: "http://web.example/"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "http://web.example/chatgpt-client" {
		t.Fatalf("expected redirect to web UI chat client, got %q", rec.Header().Get("Location"))
	}
}
