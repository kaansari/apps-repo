# ceerat-web-ui

Progressive web application for Ceerat business, authentication, user preferences, and product listing.

The preferences page uses the user service `UpdateProfile` and `UpdatePassword` gRPC methods through same-origin web endpoints. Profile updates refresh the session cookie when the backend returns an updated JWT.

## Run

```bash
CEERAT_WEB_UI_PORT=3000 CEERAT_API_BASE_URL=localhost:50051 go run .
```

Configuration:

- `CEERAT_WEB_UI_PORT`: web server port, defaults to `3000`
- `CEERAT_API_BASE_URL`: Ceerat user service gRPC address, defaults to `http://localhost:8080`
- `CEERAT_ENV`: runtime environment, defaults to `development`

The web app keeps the backend JWT in an HttpOnly cookie and proxies browser requests through same-origin endpoints.

## Logging

The web server writes structured JSON logs to stdout. Every HTTP request includes method, path, status, duration, remote address, user agent, and response bytes.

Application activities such as login, registration, current-session lookup, profile update, password update, and logout are logged as `app.activity` events. In development, these logs include redacted request/response payloads. Passwords, tokens, and secrets are never logged.
