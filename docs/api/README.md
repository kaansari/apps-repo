# API Documentation

API and integration docs live here.

## Current Docs

```text
user-service.md
```

## Runtime APIs

The authoritative transport contracts live in:

```text
packages/ceerat-contracts/proto/
```

Current gRPC areas:

```text
auth
customer
order
patient
service
```

The web UI exposes same-origin browser endpoints for auth, dashboard data, customers, customer services, orders, and AI chat. See `apps/ceerat-web-ui/README.md`.

The AI agent exposes HTTP endpoints documented in `ai/ceerat-agent-service/README.md`.
