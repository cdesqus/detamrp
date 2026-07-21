# Settings Identity and Approval Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Users, Roles & Permissions, and one Default Approver fully configurable from the application.

**Architecture:** Add an additive PostgreSQL migration and a focused Go `settings` module split into domain, service, store, and HTTP responsibilities. Register it with the existing authenticated Gin server, then replace placeholder Settings pages with compact React CRUD screens using centered modals.

**Tech Stack:** PostgreSQL 17, Go 1.26, Gin, pgx, Next.js 16, React 19, TypeScript, Vitest.

## Global Constraints

- No hard delete; users are locked/unlocked and roles activated/deactivated.
- Password hashes never leave the backend.
- The final active Administrator and current signed-in user are protected.
- Default Approver must be active, unlocked, and effectively grant `po.approve`.
- All access is tenant-scoped with PostgreSQL RLS and existing RBAC.

---

### Task 1: Add identity and approver schema

**Files:**
- Create: `database/migrations/004_settings_identity.sql`
- Create: `backend/internal/settings/migration_test.go`

**Produces:** `users.email`; audited/active `roles`; `tenant_settings.default_approver_user_id`.

- [ ] Add a failing migration contract test that requires the new columns, unique email index, and tenant-safe approver foreign key.
- [ ] Run `go test ./internal/settings -run TestMigration -v`; expect failure because migration `004` is absent.
- [ ] Add idempotent SQL that backfills admin email as `admin@local.invalid`, adds role audit/status fields, and adds the composite approver FK.
- [ ] Re-run the focused test; expect PASS.
- [ ] Commit with `feat: add identity settings schema`.

### Task 2: Implement settings domain and service rules

**Files:**
- Create: `backend/internal/settings/domain.go`
- Create: `backend/internal/settings/errors.go`
- Create: `backend/internal/settings/service.go`
- Create: `backend/internal/settings/service_test.go`

**Produces:** `UserInput`, `RoleInput`, `ApprovalConfigInput`, `Service`, and a repository interface.

- [ ] Write failing tests for normalized lowercase usernames, valid email, eight-character create passwords, required roles, immutable role codes, self-lock prevention, final-admin protection, and approver eligibility.
- [ ] Run `go test ./internal/settings -run TestService -v`; expect undefined settings types.
- [ ] Implement normalization and orchestration methods `ListUsers`, `CreateUser`, `UpdateUser`, `ListRoles`, `CreateRole`, `UpdateRole`, `ListPermissions`, `GetApprovalConfig`, and `UpdateApprovalConfig`.
- [ ] Re-run focused tests; expect PASS.
- [ ] Commit with `feat: define settings identity rules`.

### Task 3: Implement tenant-safe PostgreSQL store

**Files:**
- Create: `backend/internal/settings/store.go`
- Create: `backend/internal/settings/store_integration_test.go`
- Modify: `backend/internal/auth/password.go`

**Produces:** `SQLStore` implementing the Task 2 repository and a public password-hash helper used only for writes.

- [ ] Write integration tests covering user/role list and write queries, password replacement, effective permissions, protected final admin, and eligible Default Approver.
- [ ] Implement all operations inside `database.WithTenant`, mapping pgx uniqueness/FK errors to stable settings conflicts.
- [ ] Run `go test ./internal/settings -v`; expect PASS (integration tests skip unless `TEST_DATABASE_URL` is present).
- [ ] Commit with `feat: persist settings identity data`.

### Task 4: Expose authenticated RBAC API

**Files:**
- Create: `backend/internal/settings/http.go`
- Create: `backend/internal/settings/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Produces:** `/settings/users`, `/settings/roles`, `/settings/permissions`, and `/settings/approval-config` endpoints.

- [ ] Write failing HTTP tests for authentication, permission denial, list/create/update success, validation 400, conflict 409, and protected account behavior.
- [ ] Register settings routes with the existing session authenticator and per-route permission middleware.
- [ ] Run `go test ./...`; expect PASS.
- [ ] Commit with `feat: expose settings identity API`.

### Task 5: Activate Users UI and Default Approver

**Files:**
- Create: `frontend/components/settings/users-settings.tsx`
- Create: `frontend/components/settings/users-settings.test.tsx`
- Modify: `frontend/app/settings/users/page.tsx`
- Modify: `frontend/app/globals.css`

**Produces:** searchable Users table, centered create/edit modal, and Approval Configuration card.

- [ ] Write failing tests for list loading, create/edit modal, role selection, optional edit password, lock state, and Default Approver save.
- [ ] Implement typed API calls, compact fields, field-error mapping, and automatic list/config refresh.
- [ ] Run the focused test; expect PASS.
- [ ] Commit with `feat: activate users and default approver settings`.

### Task 6: Activate Roles & Permissions UI

**Files:**
- Create: `frontend/components/settings/roles-settings.tsx`
- Create: `frontend/components/settings/roles-settings.test.tsx`
- Modify: `frontend/app/settings/roles/page.tsx`
- Modify: `frontend/app/globals.css`

**Produces:** Roles table and centered role editor with grouped permission checkboxes.

- [ ] Write failing tests for seeded role rendering, create/edit modal, permission groups, immutable code on edit, and status toggle.
- [ ] Implement typed role/permission API calls and compact grouped checkboxes.
- [ ] Run `npm test`; expect all tests PASS.
- [ ] Run `npm run typecheck` and `npm run build`; expect exit 0.
- [ ] Commit with `feat: activate roles and permissions settings`.

### Task 7: Apply migration and verify two-account flow

**Files:**
- Modify only generated Docker images and the running database schema; no source files.

- [ ] Apply migration `004` to the existing PostgreSQL volume with `psql -v ON_ERROR_STOP=1`.
- [ ] Rebuild backend and frontend using Docker Compose project `platform-master-po`.
- [ ] Login as admin, create a Director user, configure it as Default Approver, then login as Director.
- [ ] Verify Director has `po.approve` and receives 403 for `/settings/users`.
- [ ] Verify frontend Users/Roles pages return HTTP 200 and all three services are running.
