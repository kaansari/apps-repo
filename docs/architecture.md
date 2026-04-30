# Ceerat Platform Architecture

Ceerat Platform uses a monorepo layout.

## Layers

- `packages/ceerat-contracts`: shared API contracts, protobuf-generated Go types, domain DTOs, and mapping helpers.
- `services/ceerat-user-service`: user, patient, and authentication gRPC service.
- `apps`: future frontend applications.
- `ai`: future AI services and agents.
- `analytics`: future data pipelines, reporting, notebooks, and analysis jobs.
- `infra`: future local and cloud orchestration files.

## Dependency rule

Services may depend on packages. Packages must not depend on services.

```text
apps / services / ai / analytics
            |
            v
packages/ceerat-contracts
```
