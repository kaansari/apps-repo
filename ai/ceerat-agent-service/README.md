# Ceerat Agent Service

HTTP AI agent that validates a Ceerat JWT, uses OpenAI tool calling, and executes work through the existing Ceerat gRPC APIs.

The agent does not access the database directly. Customer, service, customer-service, and order work all flows through `ceerat-user-service`.

## Endpoints

### `GET /healthz`

Returns service health.

### `POST /agent/chat`

Headers:

```http
Authorization: Bearer <ceerat-jwt>
Content-Type: application/json
```

Body:

```json
{
  "message": "List my customers",
  "session_id": "optional-conversation-id"
}
```

Response:

```json
{
  "reply": "Here are your customers...",
  "actions": ["list_customers"]
}
```

`session_id` is optional. The web UI uses it to keep the preserved ChatGPT-style UI conversation scoped without persisting OpenAI tool protocol messages across turns.

## Tools

The model can call these platform tools:

```text
create_customer
list_customers
list_services
assign_service_to_customer
create_order
list_orders
get_order
update_order_status
add_service_to_order
remove_service_from_order
```

When creating or updating orders, the agent must use existing customer and service IDs resolved through platform APIs. It should not invent IDs.

## Environment Variables

```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="gpt-4.1-mini"
export CEERAT_USER_SERVICE_ADDR="localhost:50051"
export PORT="8088"
```

## Run Locally

From the repository root, the easiest path is:

```bash
make start-stack
```

To run only the agent after the user service is already running:

```bash
cd ai/ceerat-agent-service
OPENAI_API_KEY="$OPENAI_API_KEY" \
OPENAI_MODEL="${OPENAI_MODEL:-gpt-4.1-mini}" \
CEERAT_USER_SERVICE_ADDR=localhost:50051 \
PORT=8088 \
go run .
```

## Test

Use a token from your login flow:

```bash
curl -X POST http://localhost:8088/agent/chat \
  -H "Authorization: Bearer $CEERAT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"List my customers"}'
```

Or from the repository root:

```bash
make agent-chat MSG='List my customers'
```

## Web UI Integration

The dashboard AI Agent panel calls:

```text
POST /api/agent/chat
```

The preserved ChatGPT-style UI calls:

```text
POST /api/chatgpt-client/get-prompt-result
```

Both endpoints are served by `apps/ceerat-web-ui` and forward authenticated requests to this service.
