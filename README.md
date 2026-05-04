# Ceerat Platform

Ceerat Platform is a monorepo for services, shared contracts, UI apps, AI components, analytics, and infrastructure.

## Repository layout

```text
ceerat-platform/
  packages/
    ceerat-contracts/          # Shared protobuf contracts, generated Go code, domain DTOs, mappers
  services/
    ceerat-user-service/       # User/patient/auth gRPC service
  apps/                        # Future web/admin UI applications
  ai/                          # Future AI services and agents
  analytics/                   # Future data analysis services, jobs, notebooks
  infra/                       # Future Docker, Kubernetes, Terraform, Helm, deployment config
  docs/
    api/                       # API documentation
  go.work                      # Local Go workspace linking platform modules
  Makefile                     # Platform-level automation
```

## First-time setup

```bash
cd /Users/kaansari/go/src/github.com/kaansari/ceerat-platform
make setup
```

## Common commands

```bash
make tidy          # run go mod tidy in all Go modules
make test          # run all Go tests
make build         # build all Go services/packages
make start-user    # start ceerat-user-service
make start-stack   # build and start Postgres, ceerat-user-service, and web UI
make stop-stack    # stop the local Ceerat stack
make status-stack  # show local stack process status
make logs          # tail all local stack logs
make logs-api      # tail ceerat-user-service logs
make logs-web      # tail ceerat-web-ui logs
make logs-db       # tail Postgres logs
make clean         # remove generated build output
```

The stack scripts can also be run directly:

```bash
./scripts/start.sh
./scripts/status.sh
./scripts/logs.sh web
./scripts/stop.sh
```

Local runtime logs are written under `logs/`, and PID files are written under `.run/`.

## GitHub setup

This monorepo should be pushed to:

```text
github.com/kaansari/ceerat-platform
```

Module paths are monorepo-based:

```text
github.com/kaansari/ceerat-platform/packages/ceerat-contracts
github.com/kaansari/ceerat-platform/services/ceerat-user-service
```

## Suggested GitHub commands

```bash
git init
git add .
git commit -m "Initial Ceerat platform monorepo"
git branch -M main
git remote add origin git@github.com:kaansari/ceerat-platform.git
git push -u origin main
```

## Add a new Go service

```bash
mkdir -p services/ceerat-product-service
cd services/ceerat-product-service
go mod init github.com/kaansari/ceerat-platform/services/ceerat-product-service
cd ../..
go work use ./services/ceerat-product-service
```


## Last code change implemented May 1st 2026 9PM:

1 - `ceerat-user-service` now exposes `UpdateProfile(User) returns (Response)` and `UpdatePassword(PasswordUpdate) returns (Response)` in the auth protobuf contract.

2 - The user service persists profile updates, validates the current password before password changes, stores new passwords with bcrypt, and omits passwords from responses.

3 - `ceerat-web-ui` preferences now calls the real profile and password update endpoints instead of returning a placeholder 501.

4 - Tests and API documentation were updated for the new contract.

---

1 - Fixed profile updates so `UpdateProfile` returns a refreshed JWT containing the updated profile values.

2 - Updated `ceerat-web-ui` to store the refreshed session cookie after profile updates.

3 - Added a web server test proving company changes are forwarded with the authenticated user ID and the refreshed cookie is written.

4 - Bumped the PWA service worker cache name so browsers fetch the updated JavaScript instead of keeping a stale cached app shell.



Implemented:

1 - Added structured JSON web logs for every HTTP request using standard web server fields: method, path, query, status, duration, bytes, remote address, and user agent.

2 - Added web application activity logs for login, registration, current-session lookup, profile update, password update, and logout.

3 - Added structured JSON gRPC logs for every user-service unary request with service name, method, status, and duration.

4 - Development logging includes redacted request/response payloads. Production-style logging omits payloads. Passwords, tokens, and secrets are redacted in all environments.

5 - Added `CEERAT_ENV` to the stack scripts and README files so logging mode is explicit.

## Last code change implemented May 3rd 2026

Implemented:

1 - Added idempotent construction service seed data in `ceerat-user-service` startup for bathroom plumbing, bathroom faucets, bathroom toilet, electrical trim, door trim, baseboard trim, kitchen plumbing, and light fixture install.

2 - Added customer and service dashboard support in `ceerat-web-ui`: customers are listed in the top section and assigned customer services are listed in the bottom section.

3 - Added web workflows to create customers and update customer information.

4 - Added web workflow to assign seeded services to customers.

5 - Added web workflow to update assigned service status and service date.

## Next update requirements

Below is a README/Codex-ready implementation brief.

---

# Orders Service Update

## Goal

Add a new **Orders** feature to the platform.

An order belongs to one customer and contains one or more services.

Users should be able to:

1. Create an order.
2. Assign the order to a customer.
3. Add one or more services to the order.
4. Set schedule/due/start dates.
5. View orders on a separate Orders page.
6. Manage orders through the AI Agent.

---

# 1. Domain Model

## Relationship

```text
User
 └── Customers
      └── Orders
           └── Order Services
                └── Services
```

Rules:

```text
One customer can have many orders.
One order can contain many services.
One service can appear in many orders.
One order belongs to one user through the customer/user relationship.
```

---

# 2. Proto Contract

Create:

```text
protos/order.proto
```

## `order.proto`

```proto
syntax = "proto3";

package order;

option go_package = "github.com/your-org/ceerat-platform/gen/orderpb;orderpb";

import "customer.proto";
import "service.proto";

message Order {
  string id = 1;
  string customer_id = 2;
  string user_id = 3;

  string order_number = 4;
  string status = 5;

  string schedule_date = 6;
  string start_date = 7;
  string due_date = 8;

  double subtotal = 9;
  double tax = 10;
  double total = 11;

  string notes = 12;

  customer.Customer customer = 13;
  repeated OrderService services = 14;

  string created_at = 15;
  string updated_at = 16;
}

message OrderService {
  string id = 1;
  string order_id = 2;
  string service_id = 3;

  string service_name = 4;
  string category = 5;
  string type = 6;

  double unit_price = 7;
  int32 quantity = 8;
  double total_price = 9;

  string agent_name = 10;
  string schedule_date = 11;
  string start_date = 12;
  string due_date = 13;

  service.Service service = 14;

  string created_at = 15;
  string updated_at = 16;
}

message CreateOrderRequest {
  string customer_id = 1;
  string user_id = 2;
  string schedule_date = 3;
  string start_date = 4;
  string due_date = 5;
  string notes = 6;
  repeated CreateOrderServiceInput services = 7;
}

message CreateOrderServiceInput {
  string service_id = 1;
  int32 quantity = 2;
  string agent_name = 3;
  string schedule_date = 4;
  string start_date = 5;
  string due_date = 6;
}

message CreateOrderResponse {
  Order order = 1;
}

message GetOrderRequest {
  string id = 1;
  string user_id = 2;
}

message GetOrderResponse {
  Order order = 1;
}

message ListOrdersRequest {
  string user_id = 1;
  string customer_id = 2;
  string status = 3;
}

message ListOrdersResponse {
  repeated Order orders = 1;
}

message UpdateOrderStatusRequest {
  string id = 1;
  string user_id = 2;
  string status = 3;
}

message UpdateOrderStatusResponse {
  Order order = 1;
}

message AddServiceToOrderRequest {
  string order_id = 1;
  string user_id = 2;
  CreateOrderServiceInput service = 3;
}

message AddServiceToOrderResponse {
  Order order = 1;
}

message RemoveServiceFromOrderRequest {
  string order_id = 1;
  string order_service_id = 2;
  string user_id = 3;
}

message RemoveServiceFromOrderResponse {
  Order order = 1;
}

service OrderManager {
  rpc CreateOrder(CreateOrderRequest) returns (CreateOrderResponse);
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (UpdateOrderStatusResponse);
  rpc AddServiceToOrder(AddServiceToOrderRequest) returns (AddServiceToOrderResponse);
  rpc RemoveServiceFromOrder(RemoveServiceFromOrderRequest) returns (RemoveServiceFromOrderResponse);
}
```

---

# 3. Database Tables

Add migration:

```text
migrations/xxxx_create_orders.sql
```

```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id),
    user_id UUID NOT NULL REFERENCES users(id),

    order_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'draft',

    schedule_date TIMESTAMP NULL,
    start_date TIMESTAMP NULL,
    due_date TIMESTAMP NULL,

    subtotal NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax NUMERIC(12,2) NOT NULL DEFAULT 0,
    total NUMERIC(12,2) NOT NULL DEFAULT 0,

    notes TEXT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE order_services (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id),

    service_name TEXT NOT NULL,
    category TEXT NULL,
    type TEXT NULL,

    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    quantity INT NOT NULL DEFAULT 1,
    total_price NUMERIC(12,2) NOT NULL DEFAULT 0,

    agent_name TEXT NULL,
    schedule_date TIMESTAMP NULL,
    start_date TIMESTAMP NULL,
    due_date TIMESTAMP NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_order_services_order_id ON order_services(order_id);
```

---

# 4. Backend Service

Create:

```text
services/order-service
```

Suggested files:

```text
services/order-service/
  main.go
  internal/
    handler/
      order_handler.go
    model/
      order.go
      order_service.go
    repository/
      order_repository.go
    service/
      order_service.go
    logger/
      logger.go
  tests/
    order_service_test.go
    order_repository_test.go
```

## Responsibilities

The order service should:

```text
Validate authenticated user ID.
Validate that the customer belongs to the user.
Validate that selected services exist.
Copy service name/category/type/price into order_services.
Calculate subtotal, tax, and total.
Create order number.
Return full order with services.
```

## Order status values

Use:

```text
draft
scheduled
in_progress
completed
cancelled
```

---

# 5. Backend Logic

## CreateOrder flow

```text
1. Receive customer_id, user_id, dates, notes, services.
2. Check customer exists and belongs to user.
3. Create order row.
4. For each selected service:
   - Load service by service_id.
   - Copy name/category/type/price.
   - Calculate quantity * price.
   - Insert into order_services.
5. Calculate order subtotal.
6. Calculate tax if needed.
7. Calculate total.
8. Return full order.
```

## Order number

Generate something readable:

```text
ORD-2026-000001
```

Implementation can use:

```go
fmt.Sprintf("ORD-%d-%06d", time.Now().Year(), sequenceNumber)
```

or UUID-based fallback:

```go
"ORD-" + strings.ToUpper(shortID)
```

---

# 6. Web UI

Add a separate Orders page:

```text
/apps/ceerat-web-ui/pages/orders
```

or, depending on current structure:

```text
/apps/ceerat-web-ui/internal/server/templates/orders.html
```

## Navigation

Add a new nav item:

```text
Customers | Services | Orders | AI Agent
```

## Orders page features

Page URL:

```text
/orders
```

The page should include:

```text
Create Order form
Order list
Order detail panel
Customer selector
Service multi-select
Date fields
Status selector
Notes field
Submit button
```

## Create Order form fields

```text
Customer
Services
Quantity per service
Schedule date
Start date
Due date
Agent name
Notes
Status
```

## UI API endpoints

Add web routes:

```http
GET  /orders
GET  /api/orders
GET  /api/orders/:id
POST /api/orders
PATCH /api/orders/:id/status
POST /api/orders/:id/services
DELETE /api/orders/:id/services/:orderServiceId
```

---

# 7. AI Agent Integration

Update:

```text
ai/ceerat-agent-service
```

Add order tools.

## New agent tools

```text
create_order
list_orders
get_order
add_service_to_order
remove_service_from_order
update_order_status
```

## Example tool schemas

### `create_order`

```json
{
  "name": "create_order",
  "description": "Create an order for a customer with one or more services.",
  "parameters": {
    "type": "object",
    "properties": {
      "customer_id": {
        "type": "string"
      },
      "customer_name": {
        "type": "string"
      },
      "services": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "service_id": { "type": "string" },
            "service_name": { "type": "string" },
            "quantity": { "type": "integer" },
            "agent_name": { "type": "string" },
            "schedule_date": { "type": "string" },
            "start_date": { "type": "string" },
            "due_date": { "type": "string" }
          }
        }
      },
      "schedule_date": { "type": "string" },
      "start_date": { "type": "string" },
      "due_date": { "type": "string" },
      "notes": { "type": "string" }
    },
    "required": ["services"]
  }
}
```

## Agent behavior

Update the system prompt:

```text
You can now manage orders.

An order belongs to one customer and contains one or more services.

When creating an order:
- Identify the customer.
- If customer is ambiguous, ask the user to clarify.
- Identify the requested services.
- If service is ambiguous, list matching services.
- Ask for missing date or quantity only when needed.
- Never invent customer IDs or service IDs.
- Use list_customers and list_services before create_order when only names are provided.
```

## Example AI prompts

```text
Create an order for John Smith with lawn cleaning and pest control for next Friday.
```

```text
Show me all orders for John Smith.
```

```text
Add window cleaning to order ORD-2026-000012.
```

```text
Mark order ORD-2026-000012 as completed.
```

---

# 8. Tests

Add tests for:

## Proto/client tests

```text
Order proto compiles.
OrderManager client can be generated.
```

## Repository tests

```text
Create order.
Create order with multiple services.
List orders by user.
List orders by customer.
Get order by ID.
Add service to order.
Remove service from order.
Update order status.
Reject access when user does not own customer/order.
```

## Service tests

```text
CreateOrder calculates subtotal.
CreateOrder calculates total.
CreateOrder copies service price at order time.
CreateOrder rejects unknown customer.
CreateOrder rejects unknown service.
CreateOrder rejects empty service list.
```

## AI Agent tests

```text
Agent can call create_order.
Agent asks for customer when missing.
Agent resolves customer by name.
Agent resolves service by name.
Agent refuses to create order for unknown customer.
Agent lists orders.
Agent updates order status.
```

---

# 9. Logging

Add structured logs to the order service.

Log:

```text
order.create.started
order.create.completed
order.create.failed
order.list.started
order.get.started
order.status.updated
order.service.added
order.service.removed
```

Example fields:

```json
{
  "event": "order.create.completed",
  "order_id": "uuid",
  "customer_id": "uuid",
  "user_id": "uuid",
  "service_count": 2,
  "total": 250.00
}
```

Do not log:

```text
passwords
tokens
full security credentials
```

---

# 10. Makefile / Docker Updates

Add order service to Docker Compose.

Example service:

```yaml
order-service:
  build:
    context: .
    dockerfile: services/order-service/Dockerfile
  environment:
    DATABASE_URL: ${DATABASE_URL}
    GRPC_PORT: 50054
  ports:
    - "50054:50054"
  depends_on:
    - postgres
```

Update Makefile:

```makefile
proto:
	buf generate

test-orders:
	go test ./services/order-service/...

logs-orders:
	docker logs -f ceerat-order-service
```

---

# 11. Web App Configuration

Add env vars:

```env
ORDER_SERVICE_ADDR=order-service:50054
```

Local dev:

```env
ORDER_SERVICE_ADDR=localhost:50054
```

The web app should create an OrderManager gRPC client using this address.

---

# 12. AI Agent Configuration

Add env var:

```env
ORDER_SERVICE_ADDR=order-service:50054
```

The AI agent should include an order client:

```go
type PlatformClient struct {
    Auth      authpb.AuthClient
    Customers customerpb.CustomerServiceClient
    Services  servicepb.ServiceManagerClient
    Orders    orderpb.OrderManagerClient
}
```

---

# 13. Acceptance Criteria

This update is complete when:

```text
/order.proto exists and compiles.
Order service starts successfully.
Orders tables are created by migration.
User can open /orders page.
User can create an order from the UI.
User can select a customer.
User can assign multiple services.
Order total is calculated correctly.
User can list orders.
User can view order details.
AI agent can create an order.
AI agent can list orders.
AI agent can add services to an order.
Tests pass.
Logs show order lifecycle events.
```

---

# 14. Suggested Codex Prompt

Use this prompt with Codex:

```text
Implement an Orders feature in this repository.

Add a new order.proto with an OrderManager gRPC service. An order belongs to a customer and contains many services through order_services. Generate Go proto clients.

Add database migrations for orders and order_services.

Create a new order-service with model, repository, service, gRPC handler, logging, tests, and Dockerfile.

Expose web UI routes for /orders and APIs for creating, listing, viewing, updating status, adding services, and removing services.

Add an Orders link/page to the web UI. The Orders page should let a logged-in user select a customer, select one or more services, set quantity, schedule/start/due dates, assign agent name, add notes, and submit the order.

Update the AI agent service with order tools: create_order, list_orders, get_order, add_service_to_order, remove_service_from_order, update_order_status. The agent must resolve customers and services by name using existing list APIs and must not invent IDs.

Update docker-compose, Makefile, environment config, logs, and documentation.

Add unit and integration tests for repository, service, gRPC handler, web routes, and AI tools.

Preserve existing architecture: the web UI and AI agent must call gRPC services and must not directly access the database.
```

