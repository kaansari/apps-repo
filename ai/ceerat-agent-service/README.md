# Ceerat Agent Service

HTTP AI agent that validates a Ceerat JWT, uses OpenAI tool calling, and executes work through the existing Ceerat gRPC APIs.

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
  "message": "Create a customer named John Smith, email john@example.com, phone 555-1111"
}
```

Response:

```json
{
  "reply": "Customer John Smith has been created.",
  "actions": ["create_customer"]
}
```

## Environment variables

```bash
export OPENAI_API_KEY="sk-..."
export OPENAI_MODEL="gpt-4.1-mini"
export CEERAT_USER_SERVICE_ADDR="localhost:50051"
export PORT="8088"
```

## Run locally

From the repository root:

```bash
go work sync
go run ./services/ceerat-user-service
```

In another terminal:

```bash
export OPENAI_API_KEY="sk-..."
go run ./ai/ceerat-agent-service
```

## Test

Use a token from your login flow:

```bash
curl -X POST http://localhost:8088/agent/chat \
  -H "Authorization: Bearer $CEERAT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"List my customers"}'
```
