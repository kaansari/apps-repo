package apiclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	authpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/auth"
	customerpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/customer"
	servicepb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn      *grpc.ClientConn
	auth      authpb.AuthClient
	customers customerpb.CustomerServiceClient
	services  servicepb.ServiceManagerClient
}

type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Company string `json:"company"`
	Email   string `json:"email"`
}

type Session struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postalCode"`
}

type Customer struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Address   Address `json:"address"`
	UserID    string  `json:"userId"`
	User      User    `json:"user"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

type ServiceItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Price        float64 `json:"price"`
	Type         string  `json:"type"`
	ScheduleDate string  `json:"scheduleDate"`
	StartDate    string  `json:"startDate"`
	AgentName    string  `json:"agentName"`
	Description  string  `json:"description"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type CustomerService struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customerId"`
	ServiceID  string      `json:"serviceId"`
	Customer   Customer    `json:"customer"`
	Service    ServiceItem `json:"service"`
	Status     string      `json:"status"`
	OrderedAt  string      `json:"orderedAt"`
}

type Dashboard struct {
	Customers        []Customer        `json:"customers"`
	Services         []ServiceItem     `json:"services"`
	CustomerServices []CustomerService `json:"customerServices"`
}

func New(rawBaseURL string) (*Client, error) {
	target, err := grpcTarget(rawBaseURL)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:      conn,
		auth:      authpb.NewAuthClient(conn),
		customers: customerpb.NewCustomerServiceClient(conn),
		services:  servicepb.NewServiceManagerClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Register(ctx context.Context, name, company, email, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.Create(ctx, &authpb.User{
		Name:     strings.TrimSpace(name),
		Company:  strings.TrimSpace(company),
		Email:    strings.TrimSpace(email),
		Password: password,
	})
	if err != nil {
		return Session{}, err
	}
	if res.GetToken() == nil || !res.GetToken().GetValid() || res.GetToken().GetToken() == "" {
		return Session{}, errors.New("registration did not return a valid token")
	}

	return Session{User: userFromProto(res.GetUser()), Token: res.GetToken().GetToken()}, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	token, err := c.auth.Auth(ctx, &authpb.User{
		Email:    strings.TrimSpace(email),
		Password: password,
	})
	if err != nil {
		return Session{}, err
	}
	if !token.GetValid() || token.GetToken() == "" {
		return Session{}, errors.New("invalid email or password")
	}

	user, err := userFromToken(token.GetToken())
	if err != nil {
		return Session{}, err
	}
	if user.ID != "" {
		if fresh, err := c.GetUser(ctx, user.ID); err == nil {
			user = fresh
		}
	}

	return Session{User: user, Token: token.GetToken()}, nil
}

func (c *Client) Session(ctx context.Context, token string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	valid, err := c.auth.ValidateToken(ctx, &authpb.Token{Token: token})
	if err != nil {
		return Session{}, err
	}
	if !valid.GetValid() {
		return Session{}, errors.New("session expired")
	}

	user, err := userFromToken(token)
	if err != nil {
		return Session{}, err
	}
	if user.ID != "" {
		if fresh, err := c.GetUser(ctx, user.ID); err == nil {
			user = fresh
		}
	}

	return Session{User: user, Token: token}, nil
}

func (c *Client) GetUser(ctx context.Context, id string) (User, error) {
	res, err := c.auth.Get(ctx, &authpb.User{Id: id})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
}

func (c *Client) UpdateProfile(ctx context.Context, user User) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdateProfile(ctx, &authpb.User{
		Id:      user.ID,
		Name:    strings.TrimSpace(user.Name),
		Company: strings.TrimSpace(user.Company),
		Email:   strings.TrimSpace(user.Email),
	})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
}

func (c *Client) UpdateProfileSession(ctx context.Context, user User) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdateProfile(ctx, &authpb.User{
		Id:      user.ID,
		Name:    strings.TrimSpace(user.Name),
		Company: strings.TrimSpace(user.Company),
		Email:   strings.TrimSpace(user.Email),
	})
	if err != nil {
		return Session{}, err
	}

	session := Session{User: userFromProto(res.GetUser())}
	if res.GetToken() != nil && res.GetToken().GetValid() {
		session.Token = res.GetToken().GetToken()
	}
	return session, nil
}

func (c *Client) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.auth.UpdatePassword(ctx, &authpb.PasswordUpdate{
		Id:              id,
		CurrentPassword: currentPassword,
		NewPassword:     newPassword,
	})
	if err != nil {
		return User{}, err
	}
	return userFromProto(res.GetUser()), nil
}

func (c *Client) Dashboard(ctx context.Context, userID string) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	customers, err := c.ListCustomers(ctx, userID)
	if err != nil {
		return Dashboard{}, err
	}
	services, err := c.ListServices(ctx)
	if err != nil {
		return Dashboard{}, err
	}

	customerServices := make([]CustomerService, 0)
	for _, customer := range customers {
		assigned, err := c.ListCustomerServices(ctx, customer.ID)
		if err != nil {
			return Dashboard{}, err
		}
		customerServices = append(customerServices, assigned...)
	}

	return Dashboard{
		Customers:        customers,
		Services:         services,
		CustomerServices: customerServices,
	}, nil
}

func (c *Client) ListCustomers(ctx context.Context, userID string) ([]Customer, error) {
	res, err := c.customers.ListCustomers(ctx, &customerpb.ListCustomersRequest{UserId: userID, PageSize: 100})
	if err != nil {
		return nil, err
	}
	return customersFromProto(res.GetCustomers()), nil
}

func (c *Client) CreateCustomer(ctx context.Context, userID string, customer Customer) (Customer, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.customers.CreateCustomer(ctx, &customerpb.CreateCustomerRequest{
		UserId:   userID,
		Customer: customerToProto(customer),
	})
	if err != nil {
		return Customer{}, err
	}
	return customerFromProto(res.GetCustomer()), nil
}

func (c *Client) UpdateCustomer(ctx context.Context, userID string, customer Customer) (Customer, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	customer.UserID = userID
	res, err := c.customers.UpdateCustomer(ctx, &customerpb.UpdateCustomerRequest{Customer: customerToProto(customer)})
	if err != nil {
		return Customer{}, err
	}
	return customerFromProto(res.GetCustomer()), nil
}

func (c *Client) ListServices(ctx context.Context) ([]ServiceItem, error) {
	res, err := c.services.ListServices(ctx, &servicepb.ListServicesRequest{PageSize: 200})
	if err != nil {
		return nil, err
	}
	return servicesFromProto(res.GetServices()), nil
}

func (c *Client) AssignServiceToCustomer(ctx context.Context, customerID, serviceID, status, orderedAt string) (CustomerService, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.services.AssignServiceToCustomer(ctx, &servicepb.AssignServiceToCustomerRequest{
		CustomerId: strings.TrimSpace(customerID),
		ServiceId:  strings.TrimSpace(serviceID),
		Status:     strings.TrimSpace(status),
		OrderedAt:  strings.TrimSpace(orderedAt),
	})
	if err != nil {
		return CustomerService{}, err
	}
	return customerServiceFromProto(res.GetCustomerService()), nil
}

func (c *Client) ListCustomerServices(ctx context.Context, customerID string) ([]CustomerService, error) {
	res, err := c.services.ListCustomerServices(ctx, &servicepb.ListCustomerServicesRequest{
		CustomerId: customerID,
		PageSize:   100,
	})
	if err != nil {
		return nil, err
	}
	return customerServicesFromProto(res.GetCustomerServices()), nil
}

func (c *Client) UpdateCustomerService(ctx context.Context, customerService CustomerService) (CustomerService, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	res, err := c.services.UpdateCustomerService(ctx, &servicepb.UpdateCustomerServiceRequest{
		CustomerService: customerServiceToProto(customerService),
	})
	if err != nil {
		return CustomerService{}, err
	}
	return customerServiceFromProto(res.GetCustomerService()), nil
}

func grpcTarget(rawBaseURL string) (string, error) {
	if rawBaseURL == "" {
		return "", errors.New("api base url is required")
	}
	if strings.Contains(rawBaseURL, "://") {
		parsed, err := url.Parse(rawBaseURL)
		if err != nil {
			return "", err
		}
		if parsed.Host == "" {
			return "", fmt.Errorf("invalid api base url %q", rawBaseURL)
		}
		return parsed.Host, nil
	}
	return rawBaseURL, nil
}

func userFromProto(in *authpb.User) User {
	if in == nil {
		return User{}
	}
	return User{
		ID:      in.GetId(),
		Name:    in.GetName(),
		Company: in.GetCompany(),
		Email:   in.GetEmail(),
	}
}

func customerFromProto(in *customerpb.Customer) Customer {
	if in == nil {
		return Customer{}
	}
	return Customer{
		ID:        in.GetId(),
		FirstName: in.GetFirstName(),
		LastName:  in.GetLastName(),
		Email:     in.GetEmail(),
		Phone:     in.GetPhone(),
		Address: Address{
			Line1:      in.GetAddress().GetLine1(),
			Line2:      in.GetAddress().GetLine2(),
			City:       in.GetAddress().GetCity(),
			State:      in.GetAddress().GetState(),
			Country:    in.GetAddress().GetCountry(),
			PostalCode: in.GetAddress().GetPostalCode(),
		},
		UserID:    in.GetUserId(),
		User:      userFromProto(in.GetUser()),
		CreatedAt: in.GetCreatedAt(),
		UpdatedAt: in.GetUpdatedAt(),
	}
}

func customersFromProto(in []*customerpb.Customer) []Customer {
	out := make([]Customer, 0, len(in))
	for _, customer := range in {
		out = append(out, customerFromProto(customer))
	}
	return out
}

func customerToProto(in Customer) *customerpb.Customer {
	return &customerpb.Customer{
		Id:        in.ID,
		FirstName: strings.TrimSpace(in.FirstName),
		LastName:  strings.TrimSpace(in.LastName),
		Email:     strings.TrimSpace(in.Email),
		Phone:     strings.TrimSpace(in.Phone),
		Address: &customerpb.Address{
			Line1:      strings.TrimSpace(in.Address.Line1),
			Line2:      strings.TrimSpace(in.Address.Line2),
			City:       strings.TrimSpace(in.Address.City),
			State:      strings.TrimSpace(in.Address.State),
			Country:    strings.TrimSpace(in.Address.Country),
			PostalCode: strings.TrimSpace(in.Address.PostalCode),
		},
		UserId: in.UserID,
	}
}

func serviceFromProto(in *servicepb.Service) ServiceItem {
	if in == nil {
		return ServiceItem{}
	}
	return ServiceItem{
		ID:           in.GetId(),
		Name:         in.GetName(),
		Category:     in.GetCategory(),
		Price:        in.GetPrice(),
		Type:         in.GetType(),
		ScheduleDate: in.GetScheduleDate(),
		StartDate:    in.GetStartDate(),
		AgentName:    in.GetAgentName(),
		Description:  in.GetDescription(),
		CreatedAt:    in.GetCreatedAt(),
		UpdatedAt:    in.GetUpdatedAt(),
	}
}

func servicesFromProto(in []*servicepb.Service) []ServiceItem {
	out := make([]ServiceItem, 0, len(in))
	for _, service := range in {
		out = append(out, serviceFromProto(service))
	}
	return out
}

func customerServiceFromProto(in *servicepb.CustomerService) CustomerService {
	if in == nil {
		return CustomerService{}
	}
	return CustomerService{
		ID:         in.GetId(),
		CustomerID: in.GetCustomerId(),
		ServiceID:  in.GetServiceId(),
		Customer:   customerFromProto(in.GetCustomer()),
		Service:    serviceFromProto(in.GetService()),
		Status:     in.GetStatus(),
		OrderedAt:  in.GetOrderedAt(),
	}
}

func customerServicesFromProto(in []*servicepb.CustomerService) []CustomerService {
	out := make([]CustomerService, 0, len(in))
	for _, customerService := range in {
		out = append(out, customerServiceFromProto(customerService))
	}
	return out
}

func customerServiceToProto(in CustomerService) *servicepb.CustomerService {
	return &servicepb.CustomerService{
		Id:         in.ID,
		CustomerId: strings.TrimSpace(in.CustomerID),
		ServiceId:  strings.TrimSpace(in.ServiceID),
		Status:     strings.TrimSpace(in.Status),
		OrderedAt:  strings.TrimSpace(in.OrderedAt),
	}
}

func userFromToken(token string) (User, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return User{}, errors.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return User{}, err
	}

	var claims struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return User{}, err
	}
	if claims.User.ID == "" && claims.User.Email == "" {
		return User{}, errors.New("token does not include a user")
	}
	return claims.User, nil
}
