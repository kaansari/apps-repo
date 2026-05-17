Add this clarification to the implementation document/Codex prompt:

```text
Important compatibility requirement:

Do not break admin or agent workflows.

Admin and agent users must continue to be able to create and manage multiple customers using the existing general customer APIs.

The new one-to-one user/customer relationship applies only to users with role=customer.

Rules:

1. customer role:
   - can have exactly one linked customer profile
   - can register through RegisterCustomer
   - can view/update only their own customer profile
   - cannot call general CreateCustomer
   - cannot list all customers
   - cannot create customers for other people

2. admin role:
   - can create multiple customers
   - can list all customers
   - can update any customer
   - can assign roles/users where permitted
   - can use existing CreateCustomer flow

3. agent role:
   - can create multiple customers if RBAC allows CreateCustomer
   - can list/update customers according to RBAC
   - can use existing CreateCustomer flow

Database requirement:

Add customers.user_id as nullable and unique.

Example:

ALTER TABLE customers
ADD COLUMN user_id UUID NULL REFERENCES users(id);

CREATE UNIQUE INDEX idx_customers_user_id_unique
ON customers(user_id)
WHERE user_id IS NOT NULL;

This allows:
- customer-role users to have one linked customer row
- admin/agent-created customers to exist without a linked user
- no duplicate linked customer profile for the same customer user

Do not make customers.user_id NOT NULL, because that would break admin/agent-created customer records.

Service behavior:

RegisterCustomer:
- creates user with role=customer
- creates customer with user_id set to new user ID

CreateCustomer:
- remains available for admin/agent based on RBAC
- does not require user_id
- may optionally accept user_id only for admin use
- must not be available to customer role

GetMyCustomerProfile / UpdateMyCustomerProfile:
- only for customer role
- lookup customer by authenticated user's ID

Admin/agent customer APIs:
- must continue to work with customers where user_id is NULL
```

This is the key part:

```sql
CREATE UNIQUE INDEX idx_customers_user_id_unique
ON customers(user_id)
WHERE user_id IS NOT NULL;
```

That preserves admin/agent multi-customer creation while enforcing one-to-one only for actual customer portal users.
