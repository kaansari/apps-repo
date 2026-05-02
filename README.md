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

## Next update requirments
NA
