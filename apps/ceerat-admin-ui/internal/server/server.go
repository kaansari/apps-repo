package server

import (
	"context"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaansari/ceerat-platform/apps/ceerat-admin-ui/internal/apiclient"
	"github.com/kaansari/ceerat-platform/apps/ceerat-admin-ui/internal/config"
)

const sessionCookie = "ceerat_admin_session"

type apiClient interface {
	Login(ctx context.Context, email, password string) (apiclient.Session, error)
	Session(ctx context.Context, token string) (apiclient.Session, error)
	AdminData(ctx context.Context, token string) (apiclient.AdminData, error)
	AdminRequest(ctx context.Context, token, method, path string, body any, out any) error
}

type Server struct {
	cfg       config.Config
	api       apiClient
	templates *template.Template
	static    fs.FS
}

type pageData struct {
	Title string
	User  apiclient.User
}

func New(cfg config.Config, api apiClient) (*Server, error) {
	root := appRoot()
	templates, err := template.ParseGlob(filepath.Join(root, "web", "templates", "*.html"))
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		api:       api,
		templates: templates,
		static:    os.DirFS(filepath.Join(root, "web", "static")),
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(s.static))))
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("GET /", s.homePage)
	mux.HandleFunc("GET /admin/users", s.usersPage)
	mux.HandleFunc("GET /admin/rbac", s.rbacPage)
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("GET /api/admin/bootstrap", s.bootstrap)
	mux.HandleFunc("POST /api/admin/proxy", s.proxy)
	return securityHeaders(mux)
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentSession(r); ok {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	s.render(w, "login.html", pageData{Title: "Admin Login"})
}

func (s *Server) homePage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) usersPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	s.render(w, "users.html", pageData{Title: "Users", User: session.User})
}

func (s *Server) rbacPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	s.render(w, "rbac.html", pageData{Title: "RBAC", User: session.User})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	session, err := s.api.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if session.User.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin role is required.")
		return
	}
	setSessionCookie(w, session.Token)
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	data, err := s.api.AdminData(r.Context(), session.Token)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) proxy(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req struct {
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	path := strings.TrimSpace(req.Path)
	if method == "" || path == "" || !strings.HasPrefix(path, "/api/admin/") {
		writeError(w, http.StatusBadRequest, "invalid admin request")
		return
	}
	var body any
	if len(req.Body) > 0 && string(req.Body) != "null" {
		var decoded any
		if err := json.Unmarshal(req.Body, &decoded); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		body = decoded
	}
	var out any
	if err := s.api.AdminRequest(r.Context(), session.Token, method, path, body, &out); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (apiclient.Session, bool) {
	session, ok := s.currentSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return apiclient.Session{}, false
	}
	if session.User.Role != "admin" {
		clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
	return session, err == nil && session.User.Role == "admin"
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 72 * 60 * 60})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func appRoot() string {
	if root := os.Getenv("CEERAT_ADMIN_UI_ROOT"); root != "" {
		return root
	}
	if wd, err := os.Getwd(); err == nil {
		candidates := []string{
			wd,
			filepath.Join(wd, "..", "apps-repo", "apps", "ceerat-admin-ui"),
			filepath.Join(wd, "apps-repo", "apps", "ceerat-admin-ui"),
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(filepath.Join(candidate, "web", "templates")); err == nil {
				return candidate
			}
		}
		for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "web", "templates")); err == nil {
				return dir
			}
		}
	}
	return "."
}
