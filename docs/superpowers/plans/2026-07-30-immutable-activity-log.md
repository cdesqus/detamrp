# Immutable Activity Log Implementation Plan

> **For agentic workers:** Execute inline with strict red-green-refactor cycles; do not delegate.

**Goal:** Add a tenant-isolated, immutable Activity Log that records all important state-changing business actions in the same transaction and exposes a permission-protected filter/detail UI.

**Architecture:** PostgreSQL triggers write sanitized before/after JSON snapshots so every covered mutation is atomic with its activity event. The application sets tenant and actor session context for every transaction. A read-only Go module serves filtered events, and a Settings page presents the list and structured detail.

**Tech Stack:** PostgreSQL triggers/RLS, Go/Gin/pgx, Next.js/React/TypeScript.

## Global Constraints

- Page views are never logged.
- Activity rows cannot be updated or deleted.
- Passwords, tokens, SMTP secrets, logo/background bytes, and session data are never stored.
- A failed activity insert rolls back the business mutation.
- `activity_log.view` is required for API and UI access.

---

### Task 1: Atomic audit foundation

**Files:** `database/migrations/016_activity_log.sql`, `backend/internal/database/tenant.go`, `pool.go`, related tests

- [ ] Add failing tests for actor transaction context, activity table/RLS, immutable protection, trigger coverage, action inference, and secret redaction.
- [ ] Set `app.user_id` beside `app.tenant_id` for every tenant transaction.
- [ ] Create `activity_logs`, sanitization/action trigger functions, triggers on business/master/settings tables, indexes, grants, and default permission assignment.
- [ ] Run database/package tests until green.

### Task 2: Read-only Activity Log API

**Files:** create `backend/internal/activitylog/{domain,store,http}.go` and tests; modify `backend/internal/api/server.go`, `backend/cmd/api/main.go`

- [ ] Add failing normalization, tenant-filter, permission, filter, pagination, and JSON-contract tests.
- [ ] Implement date/user/module/action filters, bounded pagination, actor labels, structured before/after data, and authenticated routes.
- [ ] Wire the store into the API without exposing mutation endpoints.
- [ ] Run focused and backend tests until green.

### Task 3: Settings Activity Log UI

**Files:** create `frontend/components/settings/activity-log.{tsx,test.tsx}`, `frontend/app/settings/activity-log/page.tsx`; modify navigation, tests, and `globals.css`

- [ ] Add failing navigation and component tests for permission visibility, filters, table rows, empty/error states, pagination, and structured detail.
- [ ] Implement a compact responsive table with date, actor, module, action, and target columns.
- [ ] Add date/user/module/action filters and a readable before/after detail dialog.
- [ ] Run focused and full frontend verification until green.

### Task 4: Verification and integration

- [ ] Run backend tests/vet, frontend tests/lint/typecheck/build, migration assertions, and `git diff --check`.
- [ ] Audit trigger table coverage and confirm no secret/media fields can enter stored JSON.
- [ ] Merge into `main`, rerun verification, remove only the Phase 6 worktree, delete the feature branch, and push `main`.
