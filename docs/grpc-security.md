# gRPC Security

Ceerat protects non-auth gRPC methods with a shared JWT interceptor in `packages/ceerat-contracts/security`.

## Request Auth

Protected calls must include metadata:

```text
authorization: Bearer <jwt-token>
```

The interceptor also accepts `x-auth-token: <jwt-token>` for compatibility. It never logs token values.

## Public Methods

The default public allowlist is:

```text
/auth.Auth/Auth
/auth.Auth/Create
/auth.Auth/Register
/auth.Auth/Login
/auth.Auth/ValidateToken
/grpc.health.v1.Health/Check
/health.Health/Check
```

`Register` and `Login` are included for compatibility with future proto names. In the current generated auth service, registration is `/auth.Auth/Create` and login is `/auth.Auth/Auth`.

## Authenticated Context

After validation, handlers can read:

```go
user, ok := security.AuthenticatedUserFromContext(ctx)
```

Customer, customer-service, order, profile, and password handlers prefer this authenticated user ID over request `user_id` fields. Request IDs remain accepted for backward-compatible direct handler tests, but production gRPC traffic is blocked by the interceptor before handlers run.

## Enforcement

`ceerat-user-service` applies the interceptor when `JWT_AUTH_ENABLED` is not `false`, `0`, or `no`. The service uses its local token service to validate JWTs, avoiding a recursive gRPC call to its own public `ValidateToken` method.

Other gRPC servers can use `security.NewUserServiceTokenValidator(authpb.NewAuthClient(conn))` and the same interceptor package.

## Clients

The web UI stores the JWT in an HttpOnly session cookie and attaches it as outgoing gRPC metadata for protected calls.

The AI agent validates the incoming HTTP bearer token, stores it on the agent session, and attaches it to platform gRPC tool calls.

## Errors

```text
Missing token      -> codes.Unauthenticated
Invalid token      -> codes.Unauthenticated
Wrong resource     -> codes.PermissionDenied
```

Safe messages are used, such as `authentication required`, `invalid token`, and `access denied`.

## Local Environment

```env
USER_SERVICE_ADDR=localhost:50051
JWT_AUTH_ENABLED=true
```

`JWT_AUTH_ENABLED=false` is available only as a temporary local development bypass.
