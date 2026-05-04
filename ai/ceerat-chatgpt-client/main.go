package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port     string
	WebUIURL string
	Env      string
}

type Server struct {
	cfg Config
}

func LoadConfig() Config {
	return Config{
		Port:     env("CEERAT_CHATGPT_CLIENT_PORT", env("PORT", "3010")),
		WebUIURL: strings.TrimRight(env("CEERAT_WEB_UI_URL", "http://localhost:3000"), "/"),
		Env:      env("CEERAT_ENV", env("APP_ENV", "development")),
	}
}

func NewServer(cfg Config) *Server {
	if cfg.WebUIURL == "" {
		cfg.WebUIURL = "http://localhost:3000"
	}
	cfg.WebUIURL = strings.TrimRight(cfg.WebUIURL, "/")
	return &Server{cfg: cfg}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("/", s.redirectToWebUI)
	return mux
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"target": s.chatURL(),
	})
}

func (s *Server) redirectToWebUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.chatURL(), http.StatusSeeOther)
}

func (s *Server) chatURL() string {
	return s.cfg.WebUIURL + "/chatgpt-client"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	cfg := LoadConfig()
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           NewServer(cfg).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("ceerat-chatgpt-client redirects to %s/chatgpt-client", cfg.WebUIURL)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
