package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := NewServer(Config{OpenAIModel: "test"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("expected health payload, got %s", rec.Body.String())
	}
}

func TestPromptRequiresAPIKey(t *testing.T) {
	srv := NewServer(Config{OpenAIModel: "test"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/get-prompt-result", strings.NewReader(`{"prompt":"hello"}`))

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}
