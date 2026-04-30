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
make clean         # remove generated build output
```

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
