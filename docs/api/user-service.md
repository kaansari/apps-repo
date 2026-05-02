# API Documentation

Ceerat User Service exposes two gRPC services on `PORT` or `50051` by default.

## Auth service

Defined in `ceerat-contracts/proto/auth/auth.proto`.

### `Create(User) returns (Response)`

Creates a user, hashes the password with bcrypt, stores the user, and returns the created user plus a JWT.

Request:

```json
{
  "name": "Jane Doe",
  "company": "Ceerat",
  "email": "jane@example.com",
  "password": "plain-text-password"
}
```

Response:

```json
{
  "user": {
    "id": "uuid",
    "name": "Jane Doe",
    "company": "Ceerat",
    "email": "jane@example.com"
  },
  "token": {
    "token": "jwt",
    "valid": true
  }
}
```

### `Get(User) returns (Response)`

Looks up a user by `id`.

Request:

```json
{ "id": "uuid" }
```

### `GetAll(Request) returns (Response)`

Returns all users.

Request:

```json
{}
```

### `Auth(User) returns (Token)`

Authenticates either by `email` and `password`, or by `id`.

Request:

```json
{
  "email": "jane@example.com",
  "password": "plain-text-password"
}
```

Response:

```json
{
  "token": "jwt",
  "valid": true
}
```

### `ValidateToken(Token) returns (Token)`

Validates a JWT.

Request:

```json
{ "token": "jwt" }
```

### `UpdateProfile(User) returns (Response)`

Updates editable user profile attributes. The caller must provide the user `id`; service clients should authorize the caller before sending the request.

Request:

```json
{
  "id": "uuid",
  "name": "Jane Doe",
  "company": "Ceerat Health",
  "email": "jane@example.com"
}
```

Response:

```json
{
  "user": {
    "id": "uuid",
    "name": "Jane Doe",
    "company": "Ceerat Health",
    "email": "jane@example.com"
  },
  "token": {
    "token": "jwt",
    "valid": true
  }
}
```

### `UpdatePassword(PasswordUpdate) returns (Response)`

Changes a user's password after validating the current password. The new password is stored as a bcrypt hash and is never returned.

Request:

```json
{
  "id": "uuid",
  "currentPassword": "old-password",
  "newPassword": "new-password"
}
```

Response:

```json
{
  "user": {
    "id": "uuid",
    "name": "Jane Doe",
    "company": "Ceerat",
    "email": "jane@example.com"
  }
}
```

## Patient service

Defined in `ceerat-contracts/proto/patient/patient.proto`.

### `Create(Patient) returns (PatientResponse)`

Creates a patient record.

Request:

```json
{
  "fname": "Jane",
  "lname": "Doe",
  "dob": "1990-01-01",
  "dos": "2026-04-26",
  "location": "NYC",
  "icdcodes": "A00",
  "covidTest": true,
  "covidTestResult": false,
  "rsvTest": false,
  "rsvTestResult": false,
  "strepTest": false,
  "strepTestResult": false,
  "fluTest": false,
  "fluTestResult": false
}
```

### `Get(GetPatientRequest) returns (Patient)`

Looks up a patient by `id`.

Request:

```json
{ "id": "uuid" }
```

### `GetAll(GetAllRequest) returns (GetAllResponse)`

Returns all patients.

Request:

```json
{}
```

### `Auth(Patient)` and `ValidateToken(Token)`

These RPCs are present in the legacy proto contract but are currently placeholders that return `valid: false`.

## grpcurl examples

With the service running locally:

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext -d '{"email":"jane@example.com","password":"secret"}' localhost:50051 auth.Auth/Auth
grpcurl -plaintext -d '{"id":"<uuid>"}' localhost:50051 auth.Auth/Get
grpcurl -plaintext -d '{}' localhost:50051 auth.Auth/GetAll
grpcurl -plaintext -d '{"id":"<uuid>","name":"Jane Doe","company":"Ceerat Health","email":"jane@example.com"}' localhost:50051 auth.Auth/UpdateProfile
grpcurl -plaintext -d '{"id":"<uuid>","currentPassword":"old-secret","newPassword":"new-secret"}' localhost:50051 auth.Auth/UpdatePassword
grpcurl -plaintext -d '{"id":"<uuid>"}' localhost:50051 patient.patient/Get
grpcurl -plaintext -d '{}' localhost:50051 patient.patient/GetAll
```
