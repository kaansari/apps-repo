package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const systemPrompt = `You are Ceerat ChatGPT Client, a helpful assistant for CEERAT Construction.

Be concise, warm, and practical. Help users discuss construction services, customer follow-up, orders, scheduling, and general business questions.

Never ask for passwords, API keys, tokens, or full security credentials.`

type Config struct {
	Port         string
	OpenAIAPIKey string
	OpenAIModel  string
	Env          string
}

func loadConfig() Config {
	return Config{
		Port:         env("CEERAT_CHATGPT_CLIENT_PORT", env("PORT", "3010")),
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:  env("OPENAI_MODEL", "gpt-4.1-mini"),
		Env:          env("CEERAT_ENV", env("APP_ENV", "development")),
	}
}

type Server struct {
	cfg       Config
	openai    *OpenAIClient
	static    http.Handler
	logger    *slog.Logger
	mu        sync.Mutex
	histories map[string][]message
}

func NewServer(cfg Config) *Server {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "ceerat-chatgpt-client", "env", cfg.Env)
	return &Server{
		cfg:       cfg,
		openai:    NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel),
		static:    http.FileServer(http.Dir(webRoot())),
		logger:    logger,
		histories: map[string][]message{},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	mux.HandleFunc("POST /get-prompt-result", s.promptResult)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		s.static.ServeHTTP(w, r)
	})
	return s.requestLogger(mux)
}

func (s *Server) promptResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt   string `json:"prompt"`
		ThreadID string `json:"threadId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request.", http.StatusBadRequest)
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		http.Error(w, "Prompt is required.", http.StatusBadRequest)
		return
	}
	if s.cfg.OpenAIAPIKey == "" {
		http.Error(w, "OPENAI_API_KEY is not set.", http.StatusBadGateway)
		return
	}

	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		threadID = newThreadID()
	}
	w.Header().Set("Thread-ID", threadID)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	messages := s.messagesFor(threadID, req.Prompt)
	var reply strings.Builder
	err := s.openai.StreamChat(r.Context(), messages, func(delta string) error {
		if delta == "" {
			return nil
		}
		reply.WriteString(delta)
		if _, err := io.WriteString(w, delta); err != nil {
			return err
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return nil
	})
	if err != nil {
		s.logger.Error("chat.failed", "thread_id", threadID, "error", err)
		if reply.Len() == 0 {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	s.saveTurn(threadID, req.Prompt, reply.String())
}

func (s *Server) messagesFor(threadID, prompt string) []message {
	s.mu.Lock()
	defer s.mu.Unlock()

	history := append([]message(nil), s.histories[threadID]...)
	messages := make([]message, 0, len(history)+2)
	messages = append(messages, message{Role: "system", Content: systemPrompt})
	messages = append(messages, history...)
	messages = append(messages, message{Role: "user", Content: prompt})
	return messages
}

func (s *Server) saveTurn(threadID, prompt, reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	history := append(s.histories[threadID], message{Role: "user", Content: prompt}, message{Role: "assistant", Content: reply})
	if len(history) > 20 {
		history = history[len(history)-20:]
	}
	s.histories[threadID] = history
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Info("http.request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type OpenAIClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewOpenAIClient(apiKey, model string) *OpenAIClient {
	return &OpenAIClient{apiKey: apiKey, model: model, http: http.DefaultClient}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *OpenAIClient) StreamChat(ctx context.Context, messages []message, onDelta func(string) error) error {
	body, err := json.Marshal(map[string]any{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai request failed: %s", strings.TrimSpace(string(data)))
	}
	return readChatStream(resp.Body, onDelta)
}

func readChatStream(r io.Reader, onDelta func(string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return err
		}
		if chunk.Error != nil {
			return errors.New(chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			if err := onDelta(choice.Delta.Content); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newThreadID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("thread-%d", time.Now().UnixNano())
	}
	return "thread-" + hex.EncodeToString(b[:])
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func webRoot() string {
	candidates := []string{
		filepath.Join("ai", "ceerat-chatgpt-client", "web"),
		"web",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "web"
}

func main() {
	cfg := loadConfig()
	app := NewServer(cfg)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("ceerat-chatgpt-client listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}
