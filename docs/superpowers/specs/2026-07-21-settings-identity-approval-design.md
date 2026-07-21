# Settings Identity and Approval Configuration Design

## Objective

Activate the identity-management dependency required to test Supplier Order approval: Users, Roles & Permissions, and one tenant-scoped Default Approver. Administrators must configure everything through the application UI without database or terminal access.

## Delivery sequence

This is the first independently testable package in the larger transaction roadmap:

1. Users, Roles & Permissions, and Default Approver.
2. Supplier Order, system/email approval, automatic PO/DN/Kanban documents, and supplier email.
3. Receiving, inventory movements, reports, SMTP, and Email Log.

The current package does not create Supplier Orders; it makes the required Director account and approval authority real first.

## User management

Administrators can search, create, edit, lock, and unlock users. No hard delete is permitted.

Fields:

- Username: required, tenant-unique, trimmed and stored lowercase.
- Display Name: required.
- Email: required and valid; it becomes the approval-email destination when the user is configured as Default Approver.
- Password: required on create, optional on edit, minimum eight characters.
- Roles: at least one active role is required.
- Status: Active or Locked.

The API never returns password hashes. An administrator cannot lock their own current account. The final active Administrator cannot be locked or stripped of the Administrator role.

## Roles and permissions

The seeded roles remain visible. Administrators can create custom roles and edit role names and permission assignments. Seeded role codes are immutable; custom role codes are immutable after creation. Roles use activate/deactivate instead of hard delete.

Permission selection is grouped by module: Procurement, Logistics, Inventory, Integration, Settings, and Master Data. Deactivating or changing a role does not destroy historical user or transaction ownership.

## Default Approver

The Users page contains a compact Approval Configuration card above the user table. It has one searchable Default Approver selector and Save action.

Only an active, unlocked user whose effective roles grant `po.approve` can be selected. The backend revalidates this rule transactionally. A configured Default Approver cannot be locked, lose all `po.approve` permission, or be assigned an inactive approval role until another approver is selected.

The configuration is tenant-scoped even though the current prototype has one company. Future Supplier Order submission snapshots the configured approver user ID and email onto the approval request, so later configuration changes do not redirect an existing approval.

## Database

An additive migration:

- adds `email text` to `users`, backfilling the initial admin with a deterministic local placeholder before enforcing non-null;
- adds `active boolean`, creator/updater audit user IDs, and timestamps to `roles`;
- adds `default_approver_user_id` to `tenant_settings` with a tenant-safe composite foreign key to `users`;
- adds case-insensitive tenant email uniqueness for users.

All new application tables and queries remain protected by tenant RLS. Existing sessions, admin login, master data, and seeded roles are preserved.

## Backend modules and API

The modular monolith adds a focused `settings` package with HTTP handlers, services, and PostgreSQL stores.

Endpoints:

- `GET/POST /settings/users`
- `GET/PUT /settings/users/:id`
- `GET/POST /settings/roles`
- `GET/PUT /settings/roles/:id`
- `GET /settings/permissions`
- `GET/PUT /settings/approval-config`

Authorization:

- User endpoints require `user.manage`.
- Role and permission endpoints require `role.manage`.
- Approval configuration requires `configuration.manage`.

Errors follow the existing stable field-error shape. Uniqueness and protected-account conflicts return HTTP 409. Invalid input returns 400; unauthorized access returns 401/403.

## Frontend

Users and Roles use the established compact table and centered create/edit modal pattern. User role selection and Role permission selection use compact checkbox groups with clear module headings. Status and creator columns remain small. The Default Approver card shows the configured user's name, email, and approval-role status.

## Testing flow

1. Login as Administrator.
2. Open Settings > Users and create a Director user with the Director role and a real demo email.
3. Set that user as Default Approver.
4. Log out and log in as Director using the new username/password.
5. Confirm the Director can authenticate and has `po.approve` but cannot manage Users/Roles.
6. Log back in as Administrator and confirm protected-account constraints.

This establishes both identities needed for the next Supplier Order approval package.

## Deferred to the next package

- Supplier Order creation and approval requests.
- One-click signed email token.
- SMTP delivery and Email Log persistence.
- PO, DN, and Kanban PDFs.
