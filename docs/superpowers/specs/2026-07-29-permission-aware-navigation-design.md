# Permission-Aware Navigation and Route Guard Design

## Goal

Make the authenticated interface reflect backend RBAC consistently. Users must only see modules they can access, empty navigation groups must disappear, and opening an unauthorized frontend route directly must show a clear Access Denied state without mounting the module or starting its API requests.

## Permission Model

Add `dashboard.view` to the backend RBAC catalog with the display label `View dashboard`. The Dashboard API changes from requiring `inventory.view` to requiring `dashboard.view`.

Dashboard data remains complete. A role must explicitly receive `dashboard.view` to see the Dashboard menu, open `/dashboard`, or call the Dashboard API. Existing roles are not silently granted the new permission by a migration; administrators control access through Roles & Permissions.

The authenticated `/api/auth/me` response remains the frontend source of truth for the current user's effective permission codes.

## Navigation Metadata

Each navigation item declares one required permission:

| Navigation item | Route | Required permission |
|---|---|---|
| Dashboard | `/dashboard` | `dashboard.view` |
| Measurements | `/measurements` | `master_data.view` |
| Suppliers | `/suppliers` | `master_data.view` |
| Raw Materials | `/raw-materials` | `master_data.view` |
| Supplier Orders | `/supplier-orders` | `po.view` |
| Stock Inventory | `/inventory` | `inventory.view` |
| Receiving | `/receiving` | `receiving.view` |
| Outgoing Material | `/outgoing-material` | `inventory.view` |
| Reports | `/reports` | `receiving.view` |
| Users | `/settings/users` | `user.manage` |
| Roles & Permissions | `/settings/roles` | `role.manage` |
| SMTP Settings | `/settings/smtp` | `smtp_settings.view` |
| Email Log | `/settings/email-log` | `email_log.view` |

The AppShell filters navigation items after `/api/auth/me` resolves. A collapsible group is omitted when none of its children remain. Active-state calculation and initial group expansion use the filtered navigation rather than the unfiltered catalog.

## Direct-Route Guard

The AppShell applies a centralized route-to-permission policy before rendering its `children`. This guard is a user-experience boundary; backend middleware remains the security boundary.

Route requirements are:

| Route pattern | Required permission |
|---|---|
| `/dashboard` | `dashboard.view` |
| `/measurements`, `/suppliers`, `/raw-materials` | `master_data.view` |
| `/supplier-orders/new` | `po.create` |
| `/supplier-orders` and `/supplier-orders/:id` | `po.view` |
| `/approvals` | `po.approve` |
| `/delivery-notes` | `dn.view` |
| `/inventory` | `inventory.view` |
| `/receiving` and `/receiving/:id` | `receiving.view` |
| `/outgoing-material` and `/outgoing-material/:id` | `inventory.view` |
| `/reports` | `receiving.view` |
| `/settings/users` | `user.manage` |
| `/settings/roles` | `role.manage` |
| `/settings/smtp` | `smtp_settings.view` |
| `/settings/email-log` | `email_log.view` |

More specific routes are evaluated before their parent prefixes, so `/supplier-orders/new` requires `po.create` rather than `po.view`.

When permission is missing:

- AppShell still renders the authenticated shell and the filtered sidebar.
- The module component is not mounted.
- The content area displays `Access Denied`, a short explanation, and a link to the first permitted navigation route.
- No automatic redirect occurs, so the denial is explicit and understandable.
- If the user has no permitted navigation route, the denial state provides Logout as the available action.

Unknown routes are not assigned a permission by this policy; normal Next.js routing and not-found behavior remain responsible for them.

## Components and Boundaries

`frontend/components/app-shell/navigation.ts` owns declarative navigation metadata, route permission rules, permission checks, filtered group construction, and first-permitted-route selection. These are pure functions that can be tested without rendering React.

`frontend/components/app-shell/app-shell.tsx` owns authentication loading, uses the pure policy functions, renders filtered navigation, and replaces unauthorized content with the Access Denied state.

`backend/internal/rbac/catalog.go` owns the new permission definition. `backend/internal/dashboard/http.go` owns enforcement for the Dashboard API.

No individual module component duplicates route authorization logic.

## Error and State Handling

- While the current user is loading, retain the existing loading page.
- Authentication failure retains the existing login redirect behavior.
- A valid authenticated user with insufficient permission receives Access Denied, not an empty table and not a login redirect.
- Backend `403` responses remain unchanged and continue protecting APIs from direct requests or stale clients.
- Permission matching is exact and case-sensitive, consistent with the backend catalog.

## Testing and Acceptance Criteria

Frontend pure-policy tests prove:

- Each navigation route maps to the expected permission.
- A limited user sees only permitted items.
- Groups with no permitted children are removed.
- `supplier-orders/new` uses `po.create` before the `po.view` parent rule.
- First-permitted-route selection is deterministic.

AppShell tests prove:

- A user without a module permission cannot see its link.
- A partially authorized group contains only its permitted links.
- A fully unauthorized group is absent.
- An unauthorized direct route shows Access Denied and does not mount its child.
- The Access Denied state links to the first permitted route.
- A user with `dashboard.view` sees and can render Dashboard content.
- A user without `dashboard.view` does not see Dashboard.
- Existing collapse, mobile drawer, notification, user-menu, and logout behavior remains green.

Backend tests prove:

- `dashboard.view` exists in the RBAC catalog.
- The Dashboard endpoint accepts `dashboard.view`.
- `inventory.view` without `dashboard.view` is rejected by the Dashboard endpoint.

Before completion, run the focused navigation/AppShell tests, the complete frontend test suite, the focused backend RBAC/Dashboard tests, the complete backend test suite, frontend lint, and `go vet ./...`.

## Out of Scope

- Field-level or row-level authorization within an otherwise permitted module.
- Replacing backend RBAC middleware with frontend checks.
- Automatically granting `dashboard.view` to existing roles.
- Adding a dedicated Access Denied URL.
- Redesigning the Roles & Permissions interface beyond displaying the new catalog permission.
