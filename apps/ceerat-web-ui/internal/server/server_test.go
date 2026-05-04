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
	session              apiclient.Session
	profileInput         apiclient.User
	profileOutput        apiclient.Session
	customerInput        apiclient.Customer
	customerUserID       string
	customerServiceInput apiclient.CustomerService
	assignedCustomerID   string
	assignedServiceID    string
	orderInput           apiclient.CreateOrderInput
	orderUserID          string
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

func (api *fakeAPI) Dashboard(ctx context.Context, userID string) (apiclient.Dashboard, error) {
	return apiclient.Dashboard{
		Customers: []apiclient.Customer{{ID: "customer-1", FirstName: "Amina", UserID: userID}},
		Services:  []apiclient.ServiceItem{{ID: "service-1", Name: "Bathroom plumbing"}},
		CustomerServices: []apiclient.CustomerService{{
			ID:         "customer-service-1",
			CustomerID: "customer-1",
			ServiceID:  "service-1",
			Status:     "ordered",
		}},
	}, nil
}

func (api *fakeAPI) CreateCustomer(ctx context.Context, userID string, customer apiclient.Customer) (apiclient.Customer, error) {
	api.customerUserID = userID
	api.customerInput = customer
	customer.ID = "customer-1"
	customer.UserID = userID
	return customer, nil
}

func (api *fakeAPI) UpdateCustomer(ctx context.Context, userID string, customer apiclient.Customer) (apiclient.Customer, error) {
	api.customerUserID = userID
	api.customerInput = customer
	customer.UserID = userID
	return customer, nil
}

func (api *fakeAPI) AssignServiceToCustomer(ctx context.Context, customerID, serviceID, status, orderedAt string) (apiclient.CustomerService, error) {
	api.assignedCustomerID = customerID
	api.assignedServiceID = serviceID
	return apiclient.CustomerService{ID: "customer-service-1", CustomerID: customerID, ServiceID: serviceID, Status: status, OrderedAt: orderedAt}, nil
}

func (api *fakeAPI) UpdateCustomerService(ctx context.Context, customerService apiclient.CustomerService) (apiclient.CustomerService, error) {
	api.customerServiceInput = customerService
	return customerService, nil
}

func (api *fakeAPI) CreateOrder(ctx context.Context, userID string, input apiclient.CreateOrderInput) (apiclient.Order, error) {
	api.orderUserID = userID
	api.orderInput = input
	return apiclient.Order{ID: "order-1", UserID: userID, CustomerID: input.CustomerID, OrderNumber: "ORD-2026-000001", Status: input.Status}, nil
}

func (api *fakeAPI) ListOrders(ctx context.Context, userID, customerID, status string) ([]apiclient.Order, error) {
	return []apiclient.Order{{ID: "order-1", UserID: userID, CustomerID: "customer-1", OrderNumber: "ORD-2026-000001", Status: "draft"}}, nil
}

func (api *fakeAPI) GetOrder(ctx context.Context, userID, id string) (apiclient.Order, error) {
	return apiclient.Order{ID: id, UserID: userID, CustomerID: "customer-1", OrderNumber: "ORD-2026-000001", Status: "draft"}, nil
}

func (api *fakeAPI) UpdateOrderStatus(ctx context.Context, userID, id, status string) (apiclient.Order, error) {
	return apiclient.Order{ID: id, UserID: userID, Status: status}, nil
}

func (api *fakeAPI) AddServiceToOrder(ctx context.Context, userID, orderID string, input apiclient.OrderServiceInput) (apiclient.Order, error) {
	return apiclient.Order{ID: orderID, UserID: userID, Services: []apiclient.OrderService{{ID: "order-service-1", ServiceID: input.ServiceID}}}, nil
}

func (api *fakeAPI) RemoveServiceFromOrder(ctx context.Context, userID, orderID, orderServiceID string) (apiclient.Order, error) {
	return apiclient.Order{ID: orderID, UserID: userID}, nil
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

func TestDashboardRequiresSessionAndLoadsData(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"},
			Token: "token",
		},
	}

	srv, err := New(config.Config{Port: "3000"}, api)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"customers"`) || !strings.Contains(rec.Body.String(), `"customerServices"`) {
		t.Fatalf("expected dashboard payload, got %s", rec.Body.String())
	}
}

func TestChatGPTClientPromptUsesAgentService(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"},
			Token: "token",
		},
	}

	var agentBody map[string]string
	var agentAuth string
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&agentBody); err != nil {
			t.Fatalf("decode agent body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reply":"Here are your customers."}`))
	}))
	defer agent.Close()

	srv, err := New(config.Config{Port: "3000", AgentBaseURL: agent.URL}, api)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]string{
		"prompt":   "list my customers",
		"threadId": "thread-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chatgpt-client/get-prompt-result", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "Here are your customers." {
		t.Fatalf("expected plain text agent reply, got %q", rec.Body.String())
	}
	if rec.Header().Get("Thread-ID") != "thread-1" {
		t.Fatalf("expected thread id header, got %q", rec.Header().Get("Thread-ID"))
	}
	if agentAuth != "Bearer token" {
		t.Fatalf("expected session token to be forwarded, got %q", agentAuth)
	}
	if agentBody["message"] != "list my customers" || agentBody["session_id"] != "thread-1" {
		t.Fatalf("expected agent payload to preserve prompt and thread, got %#v", agentBody)
	}
}

func TestCreateCustomerUsesSessionUser(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"},
			Token: "token",
		},
	}

	srv, err := New(config.Config{Port: "3000"}, api)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]string{
		"firstName": "Amina",
		"lastName":  "Ansari",
		"email":     "amina@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/customers", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if api.customerUserID != "user-1" {
		t.Fatalf("expected session user id, got %q", api.customerUserID)
	}
	if api.customerInput.FirstName != "Amina" {
		t.Fatalf("expected customer payload to be forwarded, got %#v", api.customerInput)
	}
}

func TestAssignServiceToCustomerForwardsRelationship(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"},
			Token: "token",
		},
	}

	srv, err := New(config.Config{Port: "3000"}, api)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(map[string]string{
		"customerId": "customer-1",
		"serviceId":  "service-1",
		"status":     "scheduled",
		"orderedAt":  "2026-05-03",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/customer-services", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if api.assignedCustomerID != "customer-1" || api.assignedServiceID != "service-1" {
		t.Fatalf("expected relationship ids to be forwarded, got %#v", api)
	}
}

func TestCreateOrderUsesSessionUser(t *testing.T) {
	api := &fakeAPI{
		session: apiclient.Session{
			User:  apiclient.User{ID: "user-1", Name: "Jane", Email: "jane@example.com"},
			Token: "token",
		},
	}

	srv, err := New(config.Config{Port: "3000"}, api)
	if err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(apiclient.CreateOrderInput{
		CustomerID: "customer-1",
		Status:     "scheduled",
		Services:   []apiclient.OrderServiceInput{{ServiceID: "service-1", Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	rec := httptest.NewRecorder()

	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if api.orderUserID != "user-1" {
		t.Fatalf("expected session user id, got %q", api.orderUserID)
	}
	if api.orderInput.CustomerID != "customer-1" || len(api.orderInput.Services) != 1 {
		t.Fatalf("expected order payload to be forwarded, got %#v", api.orderInput)
	}
}
