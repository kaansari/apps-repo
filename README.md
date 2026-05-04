# Ceerat Platform

Ceerat Platform is a Go monorepo for the Ceerat business web app, shared protobuf contracts, gRPC backend services, AI agent tooling, and local stack automation.

## Repository Layout

```text
ceerat-platform/
  packages/
    ceerat-contracts/          Shared protobuf contracts, generated Go code, domain DTOs, mappers
  services/
    ceerat-user-service/       Auth, users, customers, services, customer services, orders
  apps/
    ceerat-web-ui/             Authenticated web UI and same-origin browser API proxy
  ai/
    ceerat-agent-service/      OpenAI tool-calling agent backed by Ceerat gRPC APIs
    ceerat-chatgpt-client/     Preserved legacy chat UI assets and redirect helper
  analytics/                   Placeholder for analytics and reporting work
  infra/                       Placeholder for deployment and infrastructure assets
  docs/
    api/                       API documentation
  scripts/                     Local stack scripts
  go.work                      Local Go workspace
  Makefile                     Platform-level automation
```

## Current Features

- Login, registration, HttpOnly session cookie handling, profile updates, password changes, and logout.
- Dashboard for customers, seeded construction services, and assigned customer services.
- Orders page for creating, listing, viewing, updating status, adding services, and removing services from orders.
- AI Agent panel on the dashboard.
- ChatGPT-style chat page at `/chatgpt-client/` that keeps the legacy UI look but uses `ceerat-agent-service` as its backend.
- Structured JSON request/activity logging with secret redaction.

## First-Time Setup

```bash
cd /Users/kaansari/go/src/github.com/kaansari/ceerat-platform
make setup
```

The local stack expects PostgreSQL 14 tools by default under `/usr/local/opt/postgresql@14/bin`. Override `PG_CTL`, `INITDB`, or `PSQL` if your install path is different.

## Common Commands

```bash
make tidy          # run go mod tidy in active Go modules
make test          # run Go tests for contracts, user service, web UI, and agent service
make test-orders   # run order feature tests
make build         # build platform binaries into ./bin
make start-user    # start ceerat-user-service directly
make start-agent   # start ceerat-agent-service directly
make start-stack   # build and start Postgres, user service, agent service, and web UI
make stop-stack    # stop the local Ceerat stack
make status-stack  # show local stack process status
make agent-chat MSG='List my customers'
make logs          # tail Postgres, user-service, web-ui, and agent logs
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

Runtime logs are written under `logs/`, and PID files are written under `.run/`.

## Local URLs

```text
Web UI:              http://localhost:3000
Orders page:         http://localhost:3000/orders
ChatGPT-style UI:    http://localhost:3000/chatgpt-client/
Agent service:       http://localhost:8088
User service gRPC:   localhost:50051
Postgres:            localhost:55434
```

The ChatGPT-style UI is served by `ceerat-web-ui`; there is no separate browser app on port `3010` in the active stack.

## Environment

The local stack reads `.env` when present. Common variables:

```text
CEERAT_WEB_UI_PORT=3000
CEERAT_SERVICE_PORT=50051
CEERAT_AGENT_PORT=8088
CEERAT_AGENT_BASE_URL=http://localhost:8088
CEERAT_DB_HOST=localhost
CEERAT_DB_PORT=55434
CEERAT_DB_USER=postgres
CEERAT_DB_PASSWORD=postgres
CEERAT_DB_NAME=postgres
CEERAT_JWT_SECRET=dev-secret
CEERAT_ENV=development
OPENAI_API_KEY=...
OPENAI_MODEL=gpt-4.1-mini
```

Do not commit real secrets. Development logs redact passwords, tokens, and secret-looking fields.

## Architecture Boundaries

- Browser requests go through `apps/ceerat-web-ui` same-origin endpoints.
- `ceerat-web-ui` calls the gRPC user service and proxies AI chat requests to `ceerat-agent-service`.
- `ceerat-agent-service` validates the Ceerat JWT and uses platform gRPC APIs for all customer, service, and order work.
- `ceerat-user-service` owns persistence and migrations.
- `packages/ceerat-contracts` stays free of database and service implementation logic.

## AI Agent

The agent supports tool-backed operations for:

- `create_customer`
- `list_customers`
- `list_services`
- `assign_service_to_customer`
- `create_order`
- `list_orders`
- `get_order`
- `update_order_status`
- `add_service_to_order`
- `remove_service_from_order`

Manual test:

```bash
make agent-chat MSG='List my customers'
```

## Orders

Orders belong to a user and customer and can contain one or more services. The order feature is implemented across:

```text
packages/ceerat-contracts/proto/order/
services/ceerat-user-service/orders/
apps/ceerat-web-ui/web/templates/orders.html
apps/ceerat-web-ui/web/static/app.js
ai/ceerat-agent-service/internal/agent/tools.go
```

Web routes:

```text
GET    /orders
GET    /api/orders
POST   /api/orders
GET    /api/orders/{id}
PATCH  /api/orders/{id}/status
POST   /api/orders/{id}/services
DELETE /api/orders/{id}/services/{orderServiceId}
```

## GitHub

Repository:

```text
github.com/kaansari/ceerat-platform
```

Module paths:

```text
github.com/kaansari/ceerat-platform/packages/ceerat-contracts
github.com/kaansari/ceerat-platform/services/ceerat-user-service
github.com/kaansari/ceerat-platform/apps/ceerat-web-ui
github.com/kaansari/ceerat-platform/ai/ceerat-agent-service
```
