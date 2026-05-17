# Implementation Document: Customer Self-Registration with RBAC

## Problem

Customer portal registration currently fails with:

```text
role is not allowed to access this service
```

Reason:

```text
customer role is not allowed to call CreateCustomer
```

But customer registration needs to create:

```text
1. User record with role = customer
2. Customer profile linked one-to-one with that user
```

This should be allowed only for the registering customer’s own profile, not for creating other customers.

---

# Goal

Support this behavior:

```text
Public user can register as customer
  ↓
System creates user with role=customer
  ↓
System creates customer profile linked to that user
  ↓
Customer can log in
  ↓
Customer can view/update only their own customer profile
  ↓
Customer can create orders only for themselves
  ↓
Customer cannot create or manage other customers
```

---

# Recommended Design

Do **not** allow the normal protected `CreateCustomer` method for the customer role.

Instead, add a dedicated self-service flow.

```text
RegisterCustomerAccount
```

This method should be public or pre-auth because the user does not have a JWT yet.

---

# 1. Add New Auth/User API

Add a dedicated registration method:

```proto
rpc RegisterCustomer(RegisterCustomerRequest) returns (RegisterCustomerResponse);
```

Example proto:

```proto
message RegisterCustomerRequest {
  string first_name = 1;
  string last_name = 2;
  string company = 3;
  string email = 4;
  string password = 5;
  string phone = 6;

  string address_line1 = 7;
  string address_line2 = 8;
  string city = 9;
  string state = 10;
  string postal_code = 11;
  string country = 12;
}

message RegisterCustomerResponse {
  User user = 1;
  customer.Customer customer = 2;
  string token = 3;
}
```

This method should internally create both records in one transaction:

```text
users
customers
```

---

# 2. Make RegisterCustomer Public

Add this method to the JWT/RBAC public allowlist:

```text
/auth.Auth/RegisterCustomer
```

or the actual generated full method name.

Example:

```go
var DefaultPublicMethods = []string{
    "/auth.Auth/Auth",
    "/auth.Auth/Login",
    "/auth.Auth/Register",
    "/auth.Auth/RegisterCustomer",
    "/auth.Auth/ValidateToken",
    "/grpc.health.v1.Health/Check",
}
```

This avoids RBAC blocking registration before a user exists.

---

# 3. Database Relationship

Add a one-to-one relationship between `users` and `customers`.

Preferred:

```sql
ALTER TABLE customers
ADD COLUMN user_id UUID UNIQUE REFERENCES users(id);
```

This enforces:

```text
one customer profile per customer user
```

Also ensure:

```sql
CREATE INDEX idx_customers_user_id ON customers(user_id);
```

---

# 4. RegisterCustomer Transaction

Implementation flow:

```text
1. Validate required fields.
2. Check email does not already exist.
3. Load customer role.
4. Create user:
   - first_name
   - last_name
   - email
   - password_hash
   - role_id = customer role
5. Create customer:
   - user_id = new user id
   - first_name
   - last_name
   - email
   - phone
   - address fields
6. Generate JWT containing user_id and role=customer.
7. Return user, customer, token.
```

All steps must happen in one DB transaction.

If customer creation fails, roll back user creation.

---

# 5. Do Not Use Admin CreateCustomer for Registration

Current failing flow likely does this:

```text
Register user
Login user
Call CreateCustomer
RBAC denies CreateCustomer for customer role
```

Replace that with:

```text
Call RegisterCustomer
```

The customer portal registration page should not call the normal customer creation API.

---

# 6. Customer Self-Service Permissions

Add new protected methods specifically for customer self-profile operations.

Example:

```proto
rpc GetMyCustomerProfile(GetMyCustomerProfileRequest) returns (GetMyCustomerProfileResponse);
rpc UpdateMyCustomerProfile(UpdateMyCustomerProfileRequest) returns (UpdateMyCustomerProfileResponse);
```

These should use JWT context, not request user ID.

```proto
message GetMyCustomerProfileRequest {}

message GetMyCustomerProfileResponse {
  customer.Customer customer = 1;
}

message UpdateMyCustomerProfileRequest {
  string first_name = 1;
  string last_name = 2;
  string phone = 3;
  string address_line1 = 4;
  string address_line2 = 5;
  string city = 6;
  string state = 7;
  string postal_code = 8;
  string country = 9;
}

message UpdateMyCustomerProfileResponse {
  customer.Customer customer = 1;
}
```

RBAC:

```text
customer role can call:
/customer.CustomerService/GetMyCustomerProfile
/customer.CustomerService/UpdateMyCustomerProfile
```

But customer role should still not call:

```text
/customer.CustomerService/CreateCustomer
/customer.CustomerService/ListCustomers
/customer.CustomerService/DeleteCustomer
```

---

# 7. Orders for Customer Role

Customer should be able to create orders only for themselves.

Add self-service order methods:

```proto
rpc CreateMyOrder(CreateMyOrderRequest) returns (CreateMyOrderResponse);
rpc ListMyOrders(ListMyOrdersRequest) returns (ListMyOrdersResponse);
rpc GetMyOrder(GetMyOrderRequest) returns (GetMyOrderResponse);
```

These methods should:

```text
1. Extract user_id from JWT context.
2. Find customer by user_id.
3. Create/list/get orders only for that customer_id.
4. Ignore customer_id from user input.
```

RBAC:

```text
customer role can call:
/order.OrderManager/CreateMyOrder
/order.OrderManager/ListMyOrders
/order.OrderManager/GetMyOrder
```

Customer role should not call general admin/agent order methods unless those methods enforce ownership.

---

# 8. Alternative: Ownership-Aware RBAC

Another possible design is to allow:

```text
customer -> /customer.CustomerService/CreateCustomer
```

but add ownership checks inside the service.

However, this is less safe because `CreateCustomer` sounds like a general privileged method.

Recommended design:

```text
General methods for admin/agent
Self-service methods for customer
```

Example:

```text
CreateCustomer          -> admin/agent only
RegisterCustomer        -> public
GetMyCustomerProfile    -> customer
UpdateMyCustomerProfile -> customer
CreateOrder             -> admin/agent
CreateMyOrder           -> customer
```

---

# 9. Customer Portal UI Changes

Update customer portal registration page.

## Current bad flow

```text
Register user
Call CreateCustomer
Fails due to RBAC
```

## New flow

```text
Submit registration form
Call /api/customer/register
Backend calls Auth.RegisterCustomer
Store returned JWT
Redirect customer to dashboard
```

Registration form fields:

```text
First name
Last name
Company optional
Email
Password
Confirm password
Phone
Address line 1
Address line 2
City
State
Postal code
Country
```

---

# 10. Customer Dashboard UI Changes

After login, customer can access:

```text
/customer/profile
/customer/orders
/customer/services
```

Customer dashboard should not show:

```text
Admin user management
RBAC management
Create customer page
All customers list
```

---

# 11. API Route Changes in Web UI

Add:

```http
POST /api/customer/register
GET  /api/customer/me
PATCH /api/customer/me
POST /api/customer/orders
GET  /api/customer/orders
GET  /api/customer/orders/:id
```

These should call the new gRPC self-service methods.

---

# 12. RBAC Seed Updates

Add permissions:

```sql
-- Public method, not role permission:
-- /auth.Auth/RegisterCustomer

-- Customer role protected permissions:
INSERT INTO role_permissions (...)
VALUES
(customer_role_id, '/customer.CustomerService/GetMyCustomerProfile'),
(customer_role_id, '/customer.CustomerService/UpdateMyCustomerProfile'),
(customer_role_id, '/order.OrderManager/CreateMyOrder'),
(customer_role_id, '/order.OrderManager/ListMyOrders'),
(customer_role_id, '/order.OrderManager/GetMyOrder'),
(customer_role_id, '/service.ServiceManager/ListServices');
```

Keep this denied:

```text
/customer.CustomerService/CreateCustomer
/customer.CustomerService/ListCustomers
```

---

# 13. Security Rules

Service handlers must not trust IDs from customer requests.

For customer role:

```go
authUser := security.MustAuthenticatedUser(ctx)

customer, err := repo.GetCustomerByUserID(ctx, authUser.ID)
```

Never accept this from customer request:

```text
user_id
customer_id
role_id
```

The server derives ownership from JWT.

---

# 14. Tests

Add tests:

```text
RegisterCustomer creates user with customer role.
RegisterCustomer creates linked customer record.
RegisterCustomer rolls back user if customer creation fails.
RegisterCustomer rejects duplicate email.
RegisterCustomer returns JWT.
RegisterCustomer is public and not blocked by RBAC.
Customer cannot call CreateCustomer.
Customer can call GetMyCustomerProfile.
Customer can call UpdateMyCustomerProfile.
Customer cannot update another customer's profile.
Customer can create order for themselves.
Customer cannot create order for another customer.
Customer can list only their own orders.
Admin/agent CreateCustomer still works.
```

---

# 15. Acceptance Criteria

This feature is complete when:

```text
Customer can register from customer portal.
Registration creates both user and customer records.
New user has role=customer.
Customer can log in immediately after registration.
Customer has exactly one linked customer profile.
Customer can update own profile.
Customer cannot create another customer.
Customer cannot list all customers.
Customer can create orders for themselves.
Customer cannot assign orders to another customer.
RBAC no longer blocks registration.
RBAC still protects privileged customer/order APIs.
Tests pass.
```

---

# Suggested Codex Prompt

```text
Implement customer self-registration and self-service authorization.

Currently customer portal registration fails with:
"role is not allowed to access this service"
because the customer role is not allowed to call CreateCustomer.

Do not give customer role access to the general CreateCustomer method.

Instead, add a dedicated public RegisterCustomer flow that creates both:
1. a user with role=customer
2. a linked customer profile

Update auth.proto/user service with:
RegisterCustomer(RegisterCustomerRequest) returns (RegisterCustomerResponse)

RegisterCustomerRequest should collect first_name, last_name, company, email, password, phone, and address fields.

RegisterCustomer must run in one DB transaction:
- validate required fields
- reject duplicate email
- load the customer role
- create user with role=customer
- create customer profile with user_id linked to the created user
- generate JWT
- return user, customer, and token

Add user_id to customers with a UNIQUE constraint so customer-role users have one-to-one customer profiles.

Add RegisterCustomer to the public JWT/RBAC allowlist so it is not blocked before login.

Add self-service customer methods:
GetMyCustomerProfile
UpdateMyCustomerProfile

These must derive user_id from authenticated JWT context and load customer by customers.user_id. They must not trust user_id or customer_id from the request.

Add self-service order methods:
CreateMyOrder
ListMyOrders
GetMyOrder

These must derive customer_id by looking up the authenticated user's linked customer profile. Customer role must be able to create orders only for itself.

Update RBAC seed permissions:
- customer can call GetMyCustomerProfile
- customer can call UpdateMyCustomerProfile
- customer can call CreateMyOrder
- customer can call ListMyOrders
- customer can call GetMyOrder
- customer can call ListServices
- customer cannot call CreateCustomer
- customer cannot call ListCustomers

Update customer portal registration UI so it calls the new RegisterCustomer endpoint instead of registering a user and then calling CreateCustomer.

Update customer dashboard so customers can manage only:
- own profile
- own orders
- available services

Add tests for registration, RBAC, ownership, duplicate email, JWT return, and customer self-service order creation.

Use codes.Unauthenticated for missing/invalid token and codes.PermissionDenied for ownership or role violations.

Do not log passwords, JWT tokens, or authorization headers.
```
