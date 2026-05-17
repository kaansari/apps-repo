# Customer Self-Registration Upgrade Notes

This update adds a customer-facing registration flow.

## What changed

- Public registration now creates both:
  - a `users` record
  - a linked `customers` record
- New users registered from the web UI are assigned the `customer` role.
- `auth.User` now includes a `role` field.
- User/domain/model/security context types now carry `Role`.
- JWT claims include the role because the token embeds the domain user.
- Customer users are restricted to a one-to-one user/customer relationship.
- A customer-role user cannot create additional customer records.
- Existing order logic already validates that an order customer belongs to the authenticated user, so a customer can create orders only for their own customer profile.

## Web UI behavior

The registration page now asks for:

- first name
- last name
- company
- email
- phone
- password
- address fields

After successful registration:

1. `Auth.Create` creates the user.
2. The returned JWT is used to call `CustomerService.CreateCustomer`.
3. The customer profile is linked to the new user's ID.
4. The user is logged in.

## Security behavior

`CustomerService.CreateCustomer` now checks the authenticated user's role.

If the authenticated role is `customer`:

- the service forces `customer.user_id` to the authenticated user ID
- the service checks whether the user already has a customer profile
- if one exists, it returns `PermissionDenied`

Agents/admin-style users may still create multiple customers, depending on RBAC permissions.

## Important proto note

`packages/ceerat-contracts/proto/auth/auth.proto` now includes:

```proto
string role = 7;
```

Regenerate protobuf files in your normal development environment if your workflow uses `protoc` or `buf`.

## Manual test

1. Start the stack.
2. Open `/register`.
3. Enter customer details.
4. Submit registration.
5. Confirm a user row is created with `role = customer`.
6. Confirm a customer row is created with `customers.user_id = users.id`.
7. Log in as that customer.
8. Confirm the dashboard lists only that user's customer profile.
9. Try to create another customer through the API as that customer.
10. Expected result: `PermissionDenied`.
11. Create an order for the customer's own customer profile.
12. Expected result: order is created.
13. Try to create an order for another customer's ID.
14. Expected result: access denied / not found / permission denied.

## Test command

The repository requires Go `1.26.2`. In an environment with that toolchain available, run:

```bash
go test ./...
```
