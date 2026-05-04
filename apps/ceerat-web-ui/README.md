# ceerat-web-ui

Authenticated web application for Ceerat.

The app serves login, registration, dashboard, preferences, orders, the dashboard AI Agent panel, and the preserved ChatGPT-style chat UI. Browser requests stay same-origin; the web server calls `ceerat-user-service` over gRPC and proxies AI chat to `ceerat-agent-service`.

## Routes

Pages:

```text
GET /login
GET /register
GET /
GET /preferences
GET /orders
GET /chatgpt-client
GET /chatgpt-client/
```

Browser API endpoints:

```text
POST   /api/login
POST   /api/register
GET    /api/me
POST   /api/profile
POST   /api/logout
POST   /api/change-password
GET    /api/dashboard
POST   /api/customers
POST   /api/customers/update
POST   /api/customer-services
POST   /api/customer-services/update
GET    /api/orders
POST   /api/orders
GET    /api/orders/{id}
PATCH  /api/orders/{id}/status
POST   /api/orders/{id}/services
DELETE /api/orders/{id}/services/{orderServiceId}
POST   /api/agent/chat
POST   /api/chatgpt-client/get-prompt-result
```

## ChatGPT-Style UI

The legacy chat UI files are served from:

```text
web/chatgpt-client/
```

Open it through the app at:

```text
http://localhost:3000/chatgpt-client/
```

The trailing slash matters because `index.html` uses relative asset paths such as `./assets/js/app.js`. `/chatgpt-client` redirects to `/chatgpt-client/`.

The UI posts prompts to `/api/chatgpt-client/get-prompt-result`; the web server converts that request into the `ceerat-agent-service` chat format and forwards the authenticated session token.

## Run

From this directory:

```bash
CEERAT_WEB_UI_PORT=3000 \
CEERAT_API_BASE_URL=localhost:50051 \
CEERAT_AGENT_BASE_URL=http://localhost:8088 \
go run .
```

From the repository root, prefer:

```bash
make start-stack
```

## Configuration

```text
CEERAT_WEB_UI_PORT   Web server port, default 3000
CEERAT_API_BASE_URL  Ceerat user service gRPC address, default http://localhost:8080
CEERAT_AGENT_BASE_URL Agent service URL, default http://localhost:8088
CEERAT_ENV           Runtime environment, default development
```

## Security

- The backend JWT is stored in an HttpOnly cookie named `ceerat_session`.
- Authenticated pages redirect to `/login` when there is no valid session.
- Browser JavaScript never receives the raw JWT.
- Passwords, tokens, and secrets are redacted from logs.

## Logging

The web server writes structured JSON logs to stdout. Every HTTP request includes method, path, status, duration, remote address, user agent, and response bytes.

Application activities such as login, registration, current-session lookup, profile update, password update, logout, and agent proxy failures are logged as `app.activity` events. Development logging includes redacted request/response payloads; production-style environments omit payloads.

## Test

```bash
go test ./...
```
