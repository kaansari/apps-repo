package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/agent"
	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/platform"
)

type Server struct {
	Agent    *agent.Agent
	Platform *platform.Client
}

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type ChatResponse struct {
	Reply   string   `json:"reply"`
	Actions []string `json:"actions,omitempty"`
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /agent/chat", s.chat)
	return withCORS(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := r.Header.Get("Authorization")
	session, err := s.Platform.ValidateSession(ctx, auth)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json request"})
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	reply, actions, err := s.Agent.Chat(ctx, session, req.SessionID, req.Message)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ChatResponse{Reply: reply, Actions: actions})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
