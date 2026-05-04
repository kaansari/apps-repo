package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kaansari/ceerat-platform/ai/ceerat-agent-service/internal/platform"
	customerpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/customer"
	orderpb "github.com/kaansari/ceerat-platform/packages/ceerat-contracts/proto/order"
)

type ToolRunner struct {
	Platform *platform.Client
}

func toolDefinitions() []tool {
	obj := func(props map[string]any, required []string) map[string]any {
		return map[string]any{"type": "object", "properties": props, "required": required}
	}
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

	return []tool{
		{Type: "function", Function: toolFunction{
			Name: "create_customer", Description: "Create a customer assigned to the authenticated user.",
			Parameters: obj(map[string]any{
				"first_name":    str("Customer first name"),
				"last_name":     str("Customer last name"),
				"email":         str("Customer email"),
				"phone":         str("Customer phone"),
				"address_line1": str("Street address line 1"),
				"address_line2": str("Street address line 2"),
				"city":          str("City"),
				"state":         str("State"),
				"country":       str("Country"),
				"postal_code":   str("Postal or ZIP code"),
			}, []string{"first_name", "last_name"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "list_customers", Description: "List customers for the authenticated user.",
			Parameters: obj(map[string]any{}, []string{}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "list_services", Description: "List available platform services, optionally filtered by category or type.",
			Parameters: obj(map[string]any{
				"category": str("Optional service category"),
				"type":     str("Optional service type"),
			}, []string{}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "assign_service_to_customer", Description: "Assign an existing service to an existing customer.",
			Parameters: obj(map[string]any{
				"customer_id": str("Existing customer id"),
				"service_id":  str("Existing service id"),
				"status":      str("Assignment status. Use ordered unless user specifies another status."),
				"ordered_at":  str("Due/order/schedule date in YYYY-MM-DD or RFC3339 format"),
			}, []string{"customer_id", "service_id"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "create_order", Description: "Create an order for an existing customer with one or more existing services.",
			Parameters: obj(map[string]any{
				"customer_id":   str("Existing customer id"),
				"status":        str("Order status: draft, scheduled, in_progress, completed, or cancelled"),
				"schedule_date": str("Order schedule date in YYYY-MM-DD or RFC3339 format"),
				"start_date":    str("Order start date in YYYY-MM-DD or RFC3339 format"),
				"due_date":      str("Order due date in YYYY-MM-DD or RFC3339 format"),
				"notes":         str("Order notes"),
				"services": map[string]any{
					"type": "array",
					"items": obj(map[string]any{
						"service_id":    str("Existing service id"),
						"quantity":      map[string]any{"type": "integer", "description": "Quantity, defaults to 1"},
						"agent_name":    str("Assigned agent name"),
						"schedule_date": str("Service schedule date"),
						"start_date":    str("Service start date"),
						"due_date":      str("Service due date"),
					}, []string{"service_id"}),
				},
			}, []string{"customer_id", "services"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "list_orders", Description: "List orders for the authenticated user, optionally filtered by customer or status.",
			Parameters: obj(map[string]any{
				"customer_id": str("Optional customer id"),
				"status":      str("Optional order status"),
			}, []string{}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "get_order", Description: "Get a single order by id.",
			Parameters: obj(map[string]any{"order_id": str("Existing order id")}, []string{"order_id"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "update_order_status", Description: "Update an order status.",
			Parameters: obj(map[string]any{
				"order_id": str("Existing order id"),
				"status":   str("New status: draft, scheduled, in_progress, completed, or cancelled"),
			}, []string{"order_id", "status"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "add_service_to_order", Description: "Add an existing service to an existing order.",
			Parameters: obj(map[string]any{
				"order_id":      str("Existing order id"),
				"service_id":    str("Existing service id"),
				"quantity":      map[string]any{"type": "integer", "description": "Quantity, defaults to 1"},
				"agent_name":    str("Assigned agent name"),
				"schedule_date": str("Service schedule date"),
				"start_date":    str("Service start date"),
				"due_date":      str("Service due date"),
			}, []string{"order_id", "service_id"}),
		}},
		{Type: "function", Function: toolFunction{
			Name: "remove_service_from_order", Description: "Remove a service line from an order.",
			Parameters: obj(map[string]any{
				"order_id":         str("Existing order id"),
				"order_service_id": str("Existing order service line id"),
			}, []string{"order_id", "order_service_id"}),
		}},
	}
}

func (r *ToolRunner) Run(ctx context.Context, session *platform.Session, call toolCall) (string, error) {
	var args map[string]any
	if call.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", err
		}
	}

	switch call.Function.Name {
	case "create_customer":
		if argString(args, "first_name") == "" || argString(args, "last_name") == "" {
			return "", errors.New("first_name and last_name are required")
		}
		cust, err := r.Platform.CreateCustomer(ctx, session.UserID, &customerpb.Customer{
			FirstName: argString(args, "first_name"),
			LastName:  argString(args, "last_name"),
			Email:     argString(args, "email"),
			Phone:     argString(args, "phone"),
			Address: &customerpb.Address{
				Line1:      argString(args, "address_line1"),
				Line2:      argString(args, "address_line2"),
				City:       argString(args, "city"),
				State:      argString(args, "state"),
				Country:    argString(args, "country"),
				PostalCode: argString(args, "postal_code"),
			},
		})
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"created_customer": cust})

	case "list_customers":
		customers, err := r.Platform.ListCustomers(ctx, session.UserID)
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"customers": customers})

	case "list_services":
		services, err := r.Platform.ListServices(ctx, argString(args, "category"), argString(args, "type"))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"services": services})

	case "assign_service_to_customer":
		status := argString(args, "status")
		if status == "" {
			status = "ordered"
		}
		orderedAt := argString(args, "ordered_at")
		if strings.EqualFold(orderedAt, "today") || orderedAt == "" {
			orderedAt = time.Now().Format("2006-01-02")
		}
		cs, err := r.Platform.AssignServiceToCustomer(ctx, argString(args, "customer_id"), argString(args, "service_id"), status, orderedAt)
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"customer_service": cs})
	case "create_order":
		req := &orderpb.CreateOrderRequest{
			CustomerId:   argString(args, "customer_id"),
			Status:       argString(args, "status"),
			ScheduleDate: argString(args, "schedule_date"),
			StartDate:    argString(args, "start_date"),
			DueDate:      argString(args, "due_date"),
			Notes:        argString(args, "notes"),
			Services:     orderServiceInputs(args["services"]),
		}
		order, err := r.Platform.CreateOrder(ctx, session.UserID, req)
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"order": order})
	case "list_orders":
		orders, err := r.Platform.ListOrders(ctx, session.UserID, argString(args, "customer_id"), argString(args, "status"))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"orders": orders})
	case "get_order":
		order, err := r.Platform.GetOrder(ctx, session.UserID, argString(args, "order_id"))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"order": order})
	case "update_order_status":
		order, err := r.Platform.UpdateOrderStatus(ctx, session.UserID, argString(args, "order_id"), argString(args, "status"))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"order": order})
	case "add_service_to_order":
		order, err := r.Platform.AddServiceToOrder(ctx, session.UserID, argString(args, "order_id"), orderServiceInput(args))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"order": order})
	case "remove_service_from_order":
		order, err := r.Platform.RemoveServiceFromOrder(ctx, session.UserID, argString(args, "order_id"), argString(args, "order_service_id"))
		if err != nil {
			return "", err
		}
		return jsonResult(map[string]any{"order": order})
	default:
		return "", fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
}

func argString(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}

func argInt(args map[string]any, key string) int32 {
	switch value := args[key].(type) {
	case float64:
		return int32(value)
	case int:
		return int32(value)
	default:
		return 0
	}
}

func orderServiceInputs(value any) []*orderpb.CreateOrderServiceInput {
	items, _ := value.([]any)
	out := make([]*orderpb.CreateOrderServiceInput, 0, len(items))
	for _, item := range items {
		args, _ := item.(map[string]any)
		out = append(out, orderServiceInput(args))
	}
	return out
}

func orderServiceInput(args map[string]any) *orderpb.CreateOrderServiceInput {
	return &orderpb.CreateOrderServiceInput{
		ServiceId:    argString(args, "service_id"),
		Quantity:     argInt(args, "quantity"),
		AgentName:    argString(args, "agent_name"),
		ScheduleDate: argString(args, "schedule_date"),
		StartDate:    argString(args, "start_date"),
		DueDate:      argString(args, "due_date"),
	}
}

func jsonResult(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}
