package server

import (
	"context"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/apiclient"
	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/config"
)

const sessionCookie = "ceerat_session"

type Server struct {
	cfg       config.Config
	api       apiClient
	templates *template.Template
	static    fs.FS
	logger    *slog.Logger
}

type pageData struct {
	Title string
	User  apiclient.User
}

type apiClient interface {
	Register(ctx context.Context, name, company, email, password string) (apiclient.Session, error)
	Login(ctx context.Context, email, password string) (apiclient.Session, error)
	Session(ctx context.Context, token string) (apiclient.Session, error)
	UpdateProfileSession(ctx context.Context, user apiclient.User) (apiclient.Session, error)
	UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) (apiclient.User, error)
}

func New(cfg config.Config, api apiClient) (*Server, error) {
	root := appRoot()
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"displayName": displayName,
		"initials":    initials,
		"orText":      orText,
	}).ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		return nil, err
	}

	static := os.DirFS(filepath.Join(root, "web", "static"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "ceerat-web-ui", "env", cfg.Env)
	return &Server{cfg: cfg, api: api, templates: tmpl, static: static, logger: logger}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.static))))
	mux.HandleFunc("GET /manifest.json", s.staticFile("manifest.json", "application/manifest+json"))
	mux.HandleFunc("GET /service-worker.js", s.staticFile("service-worker.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("GET /offline", s.offline)

	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("GET /register", s.registerPage)
	mux.HandleFunc("GET /", s.homePage)
	mux.HandleFunc("GET /preferences", s.preferencesPage)

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/register", s.register)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("POST /api/profile", s.updateProfile)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("POST /api/change-password", s.changePassword)

	return s.requestLogger(securityHeaders(mux))
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentSession(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "login.html", pageData{Title: "Login"})
}

func (s *Server) registerPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentSession(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "register.html", pageData{Title: "Register"})
}

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.render(w, "home.html", pageData{Title: "Ceerat", User: session.User})
}

func (s *Server) preferencesPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.render(w, "preferences.html", pageData{Title: "Preferences", User: session.User})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Email and password are required.")
		return
	}

	session, err := s.api.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		s.logActivity(r, "auth.login", http.StatusUnauthorized, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusUnauthorized, cleanError(err))
		return
	}
	s.setSessionCookie(w, r, session.Token)
	s.logActivity(r, "auth.login", http.StatusOK, req, session)
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Company  string `json:"company"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "Name, email, and password are required.")
		return
	}

	session, err := s.api.Register(r.Context(), req.Name, req.Company, req.Email, req.Password)
	if err != nil {
		s.logActivity(r, "auth.register", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	s.setSessionCookie(w, r, session.Token)
	s.logActivity(r, "auth.register", http.StatusCreated, req, session)
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.logActivity(r, "session.current", http.StatusOK, nil, session)
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	s.logActivity(r, "auth.logout", http.StatusOK, nil, map[string]bool{"ok": true})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req struct {
		Name    string `json:"name"`
		Company string `json:"company"`
		Email   string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		writeError(w, http.StatusBadRequest, "Name and email are required.")
		return
	}

	updated, err := s.api.UpdateProfileSession(r.Context(), apiclient.User{
		ID:      session.User.ID,
		Name:    req.Name,
		Company: req.Company,
		Email:   req.Email,
	})
	if err != nil {
		s.logActivity(r, "profile.update", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	if updated.Token != "" {
		s.setSessionCookie(w, r, updated.Token)
	}

	s.logActivity(r, "profile.update", http.StatusOK, req, updated)
	writeJSON(w, http.StatusOK, map[string]apiclient.User{"user": updated.User})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "Current password and new password are required.")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		writeError(w, http.StatusBadRequest, "New password and confirmation must match.")
		return
	}

	if _, err := s.api.UpdatePassword(r.Context(), session.User.ID, req.CurrentPassword, req.NewPassword); err != nil {
		s.logActivity(r, "password.update", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "password.update", http.StatusOK, req, map[string]bool{"ok": true})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) requireSession(w http.ResponseWriter, r *http.Request) (apiclient.Session, bool) {
	session, ok := s.currentSession(r)
	if !ok {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusUnauthorized, "Please log in again.")
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		return apiclient.Session{}, false
	}
	return session, true
}

func (s *Server) currentSession(r *http.Request) (apiclient.Session, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return apiclient.Session{}, false
	}
	session, err := s.api.Session(r.Context(), cookie.Value)
	if err != nil {
		return apiclient.Session{}, false
	}
	return session, true
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
		MaxAge:   int((72 * time.Hour).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
		MaxAge:   -1,
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) staticFile(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		content, err := fs.ReadFile(s.static, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(content)
	}
}

func (s *Server) offline(w http.ResponseWriter, r *http.Request) {
	s.render(w, "offline.html", pageData{Title: "Offline"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func cleanError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if i := strings.LastIndex(msg, "desc = "); i >= 0 {
		msg = msg[i+7:]
	}
	if msg == "" {
		return "Request failed."
	}
	return strings.TrimSpace(msg)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func appRoot() string {
	candidates := []string{".", "apps/ceerat-web-ui", "../.."}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "web", "templates", "login.html")); err == nil {
			return candidate
		}
	}
	return "."
}

func displayName(user apiclient.User) string {
	if strings.TrimSpace(user.Name) != "" {
		return user.Name
	}
	if strings.TrimSpace(user.Email) != "" {
		return user.Email
	}
	return "there"
}

func initials(user apiclient.User) string {
	source := strings.TrimSpace(user.Name)
	if source == "" {
		source = strings.TrimSpace(user.Email)
	}
	if source == "" {
		return "C"
	}

	parts := strings.Fields(strings.ReplaceAll(source, "@", " "))
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) == 0 {
			return "C"
		}
		return strings.ToUpper(string(runes[0]))
	}

	first := []rune(parts[0])
	second := []rune(parts[1])
	return strings.ToUpper(string(first[0]) + string(second[0]))
}

func orText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
