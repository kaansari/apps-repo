package server

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		s.logger.Log(r.Context(), logLevel(status), "http.request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", rec.bytes,
			"remote_addr", clientAddr(r),
			"user_agent", r.UserAgent(),
		)
	})
}

func (s *Server) logActivity(r *http.Request, event string, status int, request any, response any) {
	attrs := []any{
		"event", event,
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"remote_addr", clientAddr(r),
	}

	if isDev(s.cfg.Env) {
		if request != nil {
			attrs = append(attrs, "request", sanitizeValue(request))
		}
		if response != nil {
			attrs = append(attrs, "response", sanitizeValue(response))
		}
	}

	s.logger.Log(r.Context(), logLevel(status), "app.activity", attrs...)
}

func clientAddr(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isDev(env string) bool {
	env = strings.ToLower(strings.TrimSpace(env))
	return env == "" || env == "dev" || env == "development" || env == "local"
}

func sanitizeValue(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return "<unserializable>"
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "<unserializable>"
	}
	return sanitizeDecoded(decoded)
}

func sanitizeDecoded(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveKey(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			typed[key] = sanitizeDecoded(child)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = sanitizeDecoded(child)
		}
		return typed
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(key)
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret")
}

func logLevel(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	if status >= 400 {
		return slog.LevelWarn
	}
	return slog.LevelInfo
}
