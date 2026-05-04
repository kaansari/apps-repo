package platform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	authpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/auth"
	customerpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/customer"
	orderpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/order"
	servicepb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn      *grpc.ClientConn
	auth      authpb.AuthClient
	customers customerpb.CustomerServiceClient
	services  servicepb.ServiceManagerClient
	orders    orderpb.OrderManagerClient
}

type Session struct {
	Token  string
	UserID string
}

func New(address string) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:      conn,
		auth:      authpb.NewAuthClient(conn),
		customers: customerpb.NewCustomerServiceClient(conn),
		services:  servicepb.NewServiceManagerClient(conn),
		orders:    orderpb.NewOrderManagerClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ValidateSession(ctx context.Context, bearerToken string) (*Session, error) {
	token := strings.TrimSpace(strings.TrimPrefix(bearerToken, "Bearer "))
	if token == "" {
		return nil, errors.New("missing bearer token")
	}
	resp, err := c.auth.ValidateToken(ctx, &authpb.Token{Token: token})
	if err != nil {
		return nil, err
	}
	if !resp.GetValid() {
		return nil, errors.New("invalid token")
	}
	userID, err := userIDFromJWT(token)
	if err != nil {
		return nil, err
	}
	return &Session{Token: token, UserID: userID}, nil
}

func userIDFromJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", errors.New("invalid jwt format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", err
	}
	if claims.User.ID == "" {
		return "", errors.New("jwt does not contain user.id")
	}
	return claims.User.ID, nil
}

func (c *Client) CreateCustomer(ctx context.Context, userID string, in *customerpb.Customer) (*customerpb.Customer, error) {
	if in.GetUserId() == "" {
		in.UserId = userID
	}
	resp, err := c.customers.CreateCustomer(ctx, &customerpb.CreateCustomerRequest{UserId: userID, Customer: in})
	if err != nil {
		return nil, err
	}
	if len(resp.GetErrors()) > 0 {
		return nil, errors.New(resp.GetErrors()[0].GetDescription())
	}
	return resp.GetCustomer(), nil
}

func (c *Client) ListCustomers(ctx context.Context, userID string) ([]*customerpb.Customer, error) {
	resp, err := c.customers.ListCustomers(ctx, &customerpb.ListCustomersRequest{UserId: userID, PageSize: 50})
	if err != nil {
		return nil, err
	}
	return resp.GetCustomers(), nil
}

func (c *Client) ListServices(ctx context.Context, category, serviceType string) ([]*servicepb.Service, error) {
	resp, err := c.services.ListServices(ctx, &servicepb.ListServicesRequest{Category: category, Type: serviceType, PageSize: 50})
	if err != nil {
		return nil, err
	}
	return resp.GetServices(), nil
}

func (c *Client) AssignServiceToCustomer(ctx context.Context, customerID, serviceID, status, orderedAt string) (*servicepb.CustomerService, error) {
	resp, err := c.services.AssignServiceToCustomer(ctx, &servicepb.AssignServiceToCustomerRequest{
		CustomerId: customerID,
		ServiceId:  serviceID,
		Status:     status,
		OrderedAt:  orderedAt,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetCustomerService(), nil
}

func (c *Client) CreateOrder(ctx context.Context, userID string, req *orderpb.CreateOrderRequest) (*orderpb.Order, error) {
	req.UserId = userID
	resp, err := c.orders.CreateOrder(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}

func (c *Client) ListOrders(ctx context.Context, userID, customerID, status string) ([]*orderpb.Order, error) {
	resp, err := c.orders.ListOrders(ctx, &orderpb.ListOrdersRequest{
		UserId:     userID,
		CustomerId: customerID,
		Status:     status,
		PageSize:   50,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetOrders(), nil
}

func (c *Client) GetOrder(ctx context.Context, userID, orderID string) (*orderpb.Order, error) {
	resp, err := c.orders.GetOrder(ctx, &orderpb.GetOrderRequest{UserId: userID, Id: orderID})
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}

func (c *Client) UpdateOrderStatus(ctx context.Context, userID, orderID, status string) (*orderpb.Order, error) {
	resp, err := c.orders.UpdateOrderStatus(ctx, &orderpb.UpdateOrderStatusRequest{UserId: userID, Id: orderID, Status: status})
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}

func (c *Client) AddServiceToOrder(ctx context.Context, userID, orderID string, service *orderpb.CreateOrderServiceInput) (*orderpb.Order, error) {
	resp, err := c.orders.AddServiceToOrder(ctx, &orderpb.AddServiceToOrderRequest{UserId: userID, OrderId: orderID, Service: service})
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}

func (c *Client) RemoveServiceFromOrder(ctx context.Context, userID, orderID, orderServiceID string) (*orderpb.Order, error) {
	resp, err := c.orders.RemoveServiceFromOrder(ctx, &orderpb.RemoveServiceFromOrderRequest{UserId: userID, OrderId: orderID, OrderServiceId: orderServiceID})
	if err != nil {
		return nil, err
	}
	return resp.GetOrder(), nil
}
