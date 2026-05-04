# Ceerat ChatGPT Client UI

This module preserves the legacy ChatGPT-style browser UI assets and provides a tiny redirect helper for anyone who starts the old client command by habit.

## Active Runtime

The active app serves the preserved UI from `apps/ceerat-web-ui`:

```text
http://localhost:3000/chatgpt-client/
```

The trailing slash is intentional. The legacy `index.html` uses relative asset paths like `./assets/js/app.js`.

## Backend Behavior

The UI no longer talks directly to ChatGPT. It posts to:

```text
POST /api/chatgpt-client/get-prompt-result
```

That endpoint is implemented in `apps/ceerat-web-ui` and forwards the authenticated request to `ceerat-agent-service`, so this UI has the same customer, service, and order capabilities as the dashboard AI Agent panel.

## Files

```text
web/                 Preserved legacy UI assets
main.go              Redirect helper to the main web UI
main_test.go         Redirect helper tests
```

## Redirect Helper

This module is not part of `make start-stack`. If run manually, it redirects to the main web app:

```bash
CEERAT_WEB_UI_URL=http://localhost:3000 go run .
```

Default redirect target:

```text
http://localhost:3000/chatgpt-client
```
