package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/apiclient"
	"github.com/kaansari/ceerat-platform/apps/ceerat-web-ui/internal/config"
)

const sessionCookie = "ceerat_session"

type Server struct {
	cfg           config.Config
	api           apiClient
	templates     *template.Template
	static        fs.FS
	chatGPTClient fs.FS
	logger        *slog.Logger
}

type pageData struct {
	Title    string
	User     apiclient.User
	AgentURL string
}

type apiClient interface {
	Register(ctx context.Context, name, company, email, password string) (apiclient.Session, error)
	Login(ctx context.Context, email, password string) (apiclient.Session, error)
	Session(ctx context.Context, token string) (apiclient.Session, error)
	UpdateProfileSession(ctx context.Context, user apiclient.User) (apiclient.Session, error)
	UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) (apiclient.User, error)
	Dashboard(ctx context.Context, userID string) (apiclient.Dashboard, error)
	CreateCustomer(ctx context.Context, userID string, customer apiclient.Customer) (apiclient.Customer, error)
	UpdateCustomer(ctx context.Context, userID string, customer apiclient.Customer) (apiclient.Customer, error)
	AssignServiceToCustomer(ctx context.Context, customerID, serviceID, status, orderedAt string) (apiclient.CustomerService, error)
	UpdateCustomerService(ctx context.Context, customerService apiclient.CustomerService) (apiclient.CustomerService, error)
	CreateOrder(ctx context.Context, userID string, input apiclient.CreateOrderInput) (apiclient.Order, error)
	ListOrders(ctx context.Context, userID, customerID, status string) ([]apiclient.Order, error)
	GetOrder(ctx context.Context, userID, id string) (apiclient.Order, error)
	UpdateOrderStatus(ctx context.Context, userID, id, status string) (apiclient.Order, error)
	AddServiceToOrder(ctx context.Context, userID, orderID string, input apiclient.OrderServiceInput) (apiclient.Order, error)
	RemoveServiceFromOrder(ctx context.Context, userID, orderID, orderServiceID string) (apiclient.Order, error)
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
	chatGPTClient := os.DirFS(filepath.Join(root, "web", "chatgpt-client"))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "ceerat-web-ui", "env", cfg.Env)
	return &Server{cfg: cfg, api: api, templates: tmpl, static: static, chatGPTClient: chatGPTClient, logger: logger}, nil
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
	mux.HandleFunc("GET /orders", s.ordersPage)
	mux.HandleFunc("GET /preferences", s.preferencesPage)
	mux.HandleFunc("GET /chatgpt-client", s.chatGPTClientPage)
	mux.HandleFunc("GET /chatgpt-client/", s.chatGPTClientAsset)

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/register", s.register)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("POST /api/profile", s.updateProfile)
	mux.HandleFunc("POST /api/logout", s.logout)
	mux.HandleFunc("POST /api/change-password", s.changePassword)
	mux.HandleFunc("GET /api/dashboard", s.dashboard)
	mux.HandleFunc("POST /api/customers", s.createCustomer)
	mux.HandleFunc("POST /api/customers/update", s.updateCustomer)
	mux.HandleFunc("POST /api/customer-services", s.assignServiceToCustomer)
	mux.HandleFunc("POST /api/customer-services/update", s.updateCustomerService)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders/{id}", s.getOrder)
	mux.HandleFunc("PATCH /api/orders/{id}/status", s.updateOrderStatus)
	mux.HandleFunc("POST /api/orders/{id}/services", s.addServiceToOrder)
	mux.HandleFunc("DELETE /api/orders/{id}/services/{orderServiceId}", s.removeServiceFromOrder)
	mux.HandleFunc("POST /api/agent/chat", s.agentChat)
	mux.HandleFunc("POST /api/chatgpt-client/get-prompt-result", s.chatGPTClientPrompt)

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
	s.render(w, "home.html", s.pageData("Ceerat", session.User))
}

func (s *Server) preferencesPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.render(w, "preferences.html", s.pageData("Preferences", session.User))
}

func (s *Server) ordersPage(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.render(w, "orders.html", s.pageData("Orders", session.User))
}

func (s *Server) pageData(title string, user apiclient.User) pageData {
	return pageData{
		Title:    title,
		User:     user,
		AgentURL: s.cfg.AgentBaseURL,
	}
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

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	updated, err := s.api.UpdateProfileSession(ctx, apiclient.User{
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

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	if _, err := s.api.UpdatePassword(ctx, session.User.ID, req.CurrentPassword, req.NewPassword); err != nil {
		s.logActivity(r, "password.update", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "password.update", http.StatusOK, req, map[string]bool{"ok": true})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	dashboard, err := s.api.Dashboard(ctx, session.User.ID)
	if err != nil {
		s.logActivity(r, "dashboard.load", http.StatusBadGateway, nil, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadGateway, cleanError(err))
		return
	}

	s.logActivity(r, "dashboard.load", http.StatusOK, nil, map[string]int{
		"customers":        len(dashboard.Customers),
		"services":         len(dashboard.Services),
		"customerServices": len(dashboard.CustomerServices),
	})
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) agentChat(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if strings.TrimSpace(string(body)) == "" {
		writeError(w, http.StatusBadRequest, "Message is required.")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(s.cfg.AgentBaseURL, "/")+"/agent/chat", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create agent request.")
		return
	}
	req.Header.Set("Authorization", "Bearer "+session.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logActivity(r, "agent.chat", http.StatusBadGateway, nil, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadGateway, "Agent service is unavailable.")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *Server) chatGPTClientPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/chatgpt-client/", http.StatusSeeOther)
}

func (s *Server) chatGPTClientIndex(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireSession(w, r); !ok {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFileFS(w, r, s.chatGPTClient, "index.html")
}

func (s *Server) chatGPTClientAsset(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/chatgpt-client/" {
		s.chatGPTClientIndex(w, r)
		return
	}
	http.StripPrefix("/chatgpt-client/", http.FileServer(http.FS(s.chatGPTClient))).ServeHTTP(w, r)
}

func (s *Server) chatGPTClientPrompt(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req struct {
		Prompt      string           `json:"prompt"`
		ThreadID    string           `json:"threadId"`
		Attachments []chatAttachment `json:"attachments"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.ThreadID = strings.TrimSpace(req.ThreadID)
	if req.Prompt == "" && len(req.Attachments) == 0 {
		http.Error(w, "Message is required.", http.StatusBadRequest)
		return
	}
	if req.ThreadID == "" {
		req.ThreadID = session.User.ID
	}

	body, err := json.Marshal(map[string]string{
		"message":    chatGPTClientAgentMessage(req.Prompt, req.Attachments),
		"session_id": req.ThreadID,
	})
	if err != nil {
		http.Error(w, "Could not create agent request.", http.StatusInternalServerError)
		return
	}

	agentResp, err := s.postAgentChat(r.Context(), session.Token, body)
	if err != nil {
		s.logActivity(r, "chatgpt-client.agent-chat", http.StatusBadGateway, nil, map[string]string{"error": cleanError(err)})
		http.Error(w, "Agent service is unavailable.", http.StatusBadGateway)
		return
	}
	defer agentResp.Body.Close()

	var payload struct {
		Reply string `json:"reply"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(agentResp.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid agent response.", http.StatusBadGateway)
		return
	}
	if agentResp.StatusCode < http.StatusOK || agentResp.StatusCode >= http.StatusMultipleChoices {
		if payload.Error == "" {
			payload.Error = "Agent request failed."
		}
		http.Error(w, payload.Error, agentResp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Thread-ID", req.ThreadID)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(payload.Reply))
}

type chatAttachment struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	PreviewURL string `json:"previewUrl"`
}

func chatGPTClientAgentMessage(prompt string, attachments []chatAttachment) string {
	prompt = strings.TrimSpace(prompt)
	if len(attachments) == 0 {
		return prompt
	}

	var b strings.Builder
	if prompt != "" {
		b.WriteString(prompt)
	} else {
		b.WriteString("The user uploaded files without additional text.")
	}
	b.WriteString("\n\nUploaded attachments for this message:\n")
	for i, attachment := range attachments {
		if i >= 8 {
			b.WriteString("- Additional attachments omitted.\n")
			break
		}
		name := strings.TrimSpace(attachment.Name)
		if name == "" {
			name = "unnamed file"
		}
		fileType := strings.TrimSpace(attachment.Type)
		if fileType == "" {
			fileType = "application/octet-stream"
		}
		b.WriteString("- ")
		b.WriteString(name)
		b.WriteString(" (")
		b.WriteString(fileType)
		if attachment.Size > 0 {
			b.WriteString(", ")
			b.WriteString(formatByteCount(attachment.Size))
		}
		b.WriteString(")")
		if attachment.PreviewURL != "" && strings.HasPrefix(fileType, "image/") {
			b.WriteString(" [image selected in the chat UI]")
		}
		b.WriteString("\n")
	}
	b.WriteString("\nUse this attachment context when answering. Do not claim files were permanently attached unless a platform tool has attached them.")
	return b.String()
}

func formatByteCount(size int64) string {
	const unit = int64(1024)
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}
	div, exp := unit, 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (s *Server) postAgentChat(ctx context.Context, token string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.cfg.AgentBaseURL, "/")+"/agent/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req customerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.FirstName) == "" && strings.TrimSpace(req.LastName) == "" {
		writeError(w, http.StatusBadRequest, "Customer name is required.")
		return
	}

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	customer, err := s.api.CreateCustomer(ctx, session.User.ID, req.toCustomer())
	if err != nil {
		s.logActivity(r, "customer.create", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "customer.create", http.StatusCreated, req, customer)
	writeJSON(w, http.StatusCreated, map[string]apiclient.Customer{"customer": customer})
}

func (s *Server) updateCustomer(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req customerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, http.StatusBadRequest, "Customer id is required.")
		return
	}
	if strings.TrimSpace(req.FirstName) == "" && strings.TrimSpace(req.LastName) == "" {
		writeError(w, http.StatusBadRequest, "Customer name is required.")
		return
	}

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	customer, err := s.api.UpdateCustomer(ctx, session.User.ID, req.toCustomer())
	if err != nil {
		s.logActivity(r, "customer.update", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "customer.update", http.StatusOK, req, customer)
	writeJSON(w, http.StatusOK, map[string]apiclient.Customer{"customer": customer})
}

func (s *Server) assignServiceToCustomer(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req struct {
		CustomerID string `json:"customerId"`
		ServiceID  string `json:"serviceId"`
		Status     string `json:"status"`
		OrderedAt  string `json:"orderedAt"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.CustomerID) == "" || strings.TrimSpace(req.ServiceID) == "" {
		writeError(w, http.StatusBadRequest, "Customer and service are required.")
		return
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "ordered"
	}

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	customerService, err := s.api.AssignServiceToCustomer(ctx, req.CustomerID, req.ServiceID, req.Status, req.OrderedAt)
	if err != nil {
		s.logActivity(r, "customer_service.assign", http.StatusBadRequest, req, map[string]string{"userId": session.User.ID, "error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "customer_service.assign", http.StatusCreated, req, customerService)
	writeJSON(w, http.StatusCreated, map[string]apiclient.CustomerService{"customerService": customerService})
}

func (s *Server) updateCustomerService(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}

	var req struct {
		ID         string `json:"id"`
		CustomerID string `json:"customerId"`
		ServiceID  string `json:"serviceId"`
		Status     string `json:"status"`
		OrderedAt  string `json:"orderedAt"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeError(w, http.StatusBadRequest, "Customer service id is required.")
		return
	}

	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	customerService, err := s.api.UpdateCustomerService(ctx, apiclient.CustomerService{
		ID:         req.ID,
		CustomerID: req.CustomerID,
		ServiceID:  req.ServiceID,
		Status:     req.Status,
		OrderedAt:  req.OrderedAt,
	})
	if err != nil {
		s.logActivity(r, "customer_service.update", http.StatusBadRequest, req, map[string]string{"userId": session.User.ID, "error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}

	s.logActivity(r, "customer_service.update", http.StatusOK, req, customerService)
	writeJSON(w, http.StatusOK, map[string]apiclient.CustomerService{"customerService": customerService})
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	orders, err := s.api.ListOrders(ctx, session.User.ID, r.URL.Query().Get("customerId"), r.URL.Query().Get("status"))
	if err != nil {
		s.logActivity(r, "order.list", http.StatusBadGateway, nil, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadGateway, cleanError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string][]apiclient.Order{"orders": orders})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	order, err := s.api.GetOrder(ctx, session.User.ID, r.PathValue("id"))
	if err != nil {
		s.logActivity(r, "order.get", http.StatusNotFound, nil, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusNotFound, cleanError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]apiclient.Order{"order": order})
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	var req apiclient.CreateOrderInput
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.CustomerID) == "" || len(req.Services) == 0 {
		writeError(w, http.StatusBadRequest, "Customer and at least one service are required.")
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	order, err := s.api.CreateOrder(ctx, session.User.ID, req)
	if err != nil {
		s.logActivity(r, "order.create", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	s.logActivity(r, "order.create", http.StatusCreated, req, order)
	writeJSON(w, http.StatusCreated, map[string]apiclient.Order{"order": order})
}

func (s *Server) updateOrderStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	order, err := s.api.UpdateOrderStatus(ctx, session.User.ID, r.PathValue("id"), req.Status)
	if err != nil {
		s.logActivity(r, "order.status.update", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]apiclient.Order{"order": order})
}

func (s *Server) addServiceToOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	var req apiclient.OrderServiceInput
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	order, err := s.api.AddServiceToOrder(ctx, session.User.ID, r.PathValue("id"), req)
	if err != nil {
		s.logActivity(r, "order.service.add", http.StatusBadRequest, req, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]apiclient.Order{"order": order})
}

func (s *Server) removeServiceFromOrder(w http.ResponseWriter, r *http.Request) {
	session, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	ctx := apiclient.ContextWithToken(r.Context(), session.Token)
	order, err := s.api.RemoveServiceFromOrder(ctx, session.User.ID, r.PathValue("id"), r.PathValue("orderServiceId"))
	if err != nil {
		s.logActivity(r, "order.service.remove", http.StatusBadRequest, nil, map[string]string{"error": cleanError(err)})
		writeError(w, http.StatusBadRequest, cleanError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]apiclient.Order{"order": order})
}

type customerRequest struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
}

func (req customerRequest) toCustomer() apiclient.Customer {
	return apiclient.Customer{
		ID:        strings.TrimSpace(req.ID),
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Email:     strings.TrimSpace(req.Email),
		Phone:     strings.TrimSpace(req.Phone),
		Address: apiclient.Address{
			Line1:      strings.TrimSpace(req.Line1),
			Line2:      strings.TrimSpace(req.Line2),
			City:       strings.TrimSpace(req.City),
			State:      strings.TrimSpace(req.State),
			Country:    strings.TrimSpace(req.Country),
			PostalCode: strings.TrimSpace(req.PostalCode),
		},
	}
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
