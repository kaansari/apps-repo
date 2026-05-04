# Ceerat AI Agent Integration

This repository now includes an AI agent service at:

```text
ai/ceerat-agent-service
```

The agent is an HTTP service that:
1. receives chat requests from the web app or API clients,
2. validates the current Ceerat JWT with `auth.Auth/ValidateToken`,
3. extracts the authenticated `user.id` from the JWT payload,
4. uses OpenAI tool calling to decide which platform action to run,
5. executes work through the existing Ceerat gRPC APIs.

It does **not** write directly to the database.

## Architecture

```text
Browser / Web UI
  |
  | POST /api/agent/chat
  v
ceerat-web-ui
  |
  | POST /agent/chat with Authorization: Bearer <JWT>
  v
ceerat-agent-service
  |
  | auth.ValidateToken
  | customer.CreateCustomer
  | customer.ListCustomers
  | service.ListServices
  | service.AssignServiceToCustomer
  v
ceerat-user-service gRPC
  |
  v
Postgres
```

## Environment variables

Agent service:

```bash
export OPENAI_API_KEY="sk-your-key"
export OPENAI_MODEL="gpt-4.1-mini"
export CEERAT_USER_SERVICE_ADDR="localhost:50051"
export PORT="8088"
```

Web UI hook:

```bash
export CEERAT_AGENT_BASE_URL="http://localhost:8088"
```

Stack script variables:

```bash
export OPENAI_API_KEY="sk-your-key"
export OPENAI_MODEL="gpt-4.1-mini"
export CEERAT_AGENT_PORT="8088"
```

## Run the full stack

```bash
export OPENAI_API_KEY="sk-your-key"
make start-stack
```

Open:

```text
http://localhost:3000
```

Log in, then use the new **AI Agent** panel on the dashboard.

## Run manually

Terminal 1:

```bash
go run ./services/ceerat-user-service
```

Terminal 2:

```bash
export OPENAI_API_KEY="sk-your-key"
export CEERAT_USER_SERVICE_ADDR="localhost:50051"
go run ./ai/ceerat-agent-service
```

Terminal 3:

```bash
export CEERAT_AGENT_BASE_URL="http://localhost:8088"
go run ./apps/ceerat-web-ui
```

## Direct API test

After logging in through the platform, copy the Ceerat JWT into `CEERAT_TOKEN`.

```bash
export CEERAT_TOKEN="your-ceerat-jwt"

curl -X POST http://localhost:8088/agent/chat \
  -H "Authorization: Bearer $CEERAT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Create a customer named John Smith, email john@example.com, phone 555-1111, 1 Main St, Dallas TX 75001"}'
```

For local development, you can also put `CEERAT_TOKEN` in `.env` and use:

```bash
./scripts/agent-chat.sh "Create a customer named John Smith, email john@example.com"
make agent-chat MSG="List my customers"
```

Assign a service:

```bash
curl -X POST http://localhost:8088/agent/chat \
  -H "Authorization: Bearer $CEERAT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Assign lawn cleaning to John Smith for next Friday"}'
```

## Current agent tools

The first version exposes these tools:

```text
create_customer
list_customers
list_services
assign_service_to_customer
```

You can add more tools in:

```text
ai/ceerat-agent-service/internal/agent/tools.go
```

Each tool should call `internal/platform/client.go`, and that client should call the existing gRPC APIs.

## Security notes

The agent should not ask for or store passwords. The expected flow is:

```text
User logs in normally -> Web app gets Ceerat JWT -> Agent receives JWT -> Agent validates JWT -> Agent acts for that user
```

This keeps credentials out of the chat flow and uses your existing platform authorization model.
