# Supplier Order, System Approval, and Logout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver real tenant-scoped Supplier Order creation, draft management, in-app approval/rejection and notification flows, plus a reliable user-menu logout.

**Architecture:** Add a focused Go `purchaseorder` package following the existing domain/service/store/http boundaries, backed by an RLS migration and decimal-safe snapshots. Replace placeholder Next.js pages with compact client components that consume the API, while AppShell owns authenticated approval notifications and logout.

**Tech Stack:** PostgreSQL 17 with RLS, Go/Gin/pgx/shopspring-decimal, Next.js 16/React 19/TypeScript, Vitest/Testing Library, Docker Compose.

## Global Constraints

- One PO belongs to exactly one supplier; its materials must belong to that supplier.
- UI copy uses `+ Raw Material`, never `Add Line`, `Line Items`, or `Total Lines`.
- Quantity per Kanban and unit price are read-only snapshots from Raw Material master.
- Total Kanban is a positive integer; quantities/prices use `numeric(20,6)` and decimal strings.
- `Expected Delivery Date` applies to the whole PO and cannot precede Order Date.
- Email, PDF, DN/Kanban generation, Sage sync, and Receiving are out of scope.
- Tables, forms, badges, notifications, and menus remain compact.

---

### Task 1: Purchase Order Migration and Domain Rules

**Files:**
- Create: `database/migrations/005_purchase_orders.sql`
- Create: `backend/internal/purchaseorder/domain.go`
- Create: `backend/internal/purchaseorder/errors.go`
- Create: `backend/internal/purchaseorder/domain_test.go`
- Create: `backend/internal/purchaseorder/migration_test.go`

**Interfaces:**
- Produces: `Actor`, `Status`, `Order`, `OrderLine`, `OrderInput`, `LineInput`, `Approval`, `DecisionInput`, `ListQuery` and typed validation/conflict/not-found errors.

- [ ] Write failing domain tests proving date validation, at least one material on submission, unique material IDs, positive integer Kanban counts, and exact decimal calculations.
- [ ] Run `go test ./internal/purchaseorder -run 'TestOrder|TestMigration'` and verify failure because the package/migration is absent.
- [ ] Add migration tables, tenant composite keys/FKs, checks, indexes, grants, RLS policies, and tenant PO sequence state.
- [ ] Implement normalization/validation and decimal calculation without floats.
- [ ] Re-run focused tests and verify pass.
- [ ] Commit `feat: define purchase order persistence and domain`.

### Task 2: Purchase Order Service and Transactional Store

**Files:**
- Create: `backend/internal/purchaseorder/service.go`
- Create: `backend/internal/purchaseorder/service_test.go`
- Create: `backend/internal/purchaseorder/store.go`

**Interfaces:**
- Consumes: domain types from Task 1 and existing `database.WithTenant`.
- Produces: `Service.List`, `Get`, `Create`, `Update`, `Submit`, `Cancel`, `ListApprovals`, `Approve`, and `Reject`; `NewSQLStore(*database.Pool)`.

- [ ] Write failing service tests with a fake repository for normalization, draft-only edits, submit validation, decision reason requirements, and repository error propagation.
- [ ] Run `go test ./internal/purchaseorder -run TestService` and verify expected failures.
- [ ] Implement the service boundary and repository interface.
- [ ] Implement SQL transactions that generate `PO-YYYYMM-#####`, load snapshot values from active tenant master data, calculate totals, replace draft materials atomically, and return audit names.
- [ ] Implement submission with `FOR UPDATE`, configured approver eligibility validation, approver snapshot, PO version increment, and exactly one pending approval.
- [ ] Implement cancel/approve/reject with locked state transitions, snapshot-user enforcement, and idempotent conflict responses.
- [ ] Run all `purchaseorder` tests and verify pass.
- [ ] Commit `feat: implement purchase order transactions`.

### Task 3: Purchase Order and Approval HTTP API

**Files:**
- Create: `backend/internal/purchaseorder/http.go`
- Create: `backend/internal/purchaseorder/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `purchaseorder.Service` and existing Authenticator/RBAC context.
- Produces: the nine `/purchase-orders` and `/purchase-order-approvals` routes specified by the design.

- [ ] Write failing HTTP tests for authentication, each RBAC permission, JSON/UUID validation, success response codes, and structured validation/conflict errors.
- [ ] Run focused HTTP tests and verify routes return missing/not-found responses.
- [ ] Register authenticated routes with the existing cookie and permission middleware pattern.
- [ ] Wire SQL store and service in `cmd/api/main.go` and expose a `WithPurchaseOrderService` server option.
- [ ] Run `go test ./...` and verify all backend tests pass.
- [ ] Commit `feat: expose purchase order approval API`.

### Task 4: Supplier Order Index and Dedicated Editor

**Files:**
- Create: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Create: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Create: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/supplier-orders/page.tsx`
- Create: `frontend/app/supplier-orders/new/page.tsx`
- Create: `frontend/app/supplier-orders/[id]/page.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: purchase-order endpoints and existing master-data supplier/material endpoints.
- Produces: real compact index, create route, draft edit/detail route, save/submit/cancel actions.

- [ ] Write failing UI tests for real list rendering, `Create order` navigation, searchable supplier and supplier-filtered materials, disabled material action before supplier, supplier-change confirmation, unique material choices, calculated totals, server error rendering, and both save actions.
- [ ] Run `npm test -- --run components/supplier-orders/supplier-orders.test.tsx` and verify failure because components are absent.
- [ ] Implement typed API models and compact loading/error/empty table states.
- [ ] Implement dedicated form with searchable native datalist/select behavior, `+ Raw Material`, read-only snapshots, positive integer entry, decimal-safe display, and sticky compact actions.
- [ ] Implement draft detail/edit/cancel and read-only submitted details.
- [ ] Re-run focused tests and verify pass.
- [ ] Commit `feat: activate supplier order UI`.

### Task 5: Real Approval Inbox and Notifications

**Files:**
- Create: `frontend/components/approvals/approval-inbox.tsx`
- Create: `frontend/components/approvals/approval-inbox.test.tsx`
- Modify: `frontend/app/approvals/page.tsx`
- Modify: `frontend/components/notifications/notification-center.tsx`
- Modify: `frontend/components/notifications/notification-center.test.tsx`
- Modify: `frontend/components/app-shell/app-shell.tsx`

**Interfaces:**
- Consumes: `GET /purchase-order-approvals` and approval/rejection decision endpoints.
- Produces: approver-only pending list, notification count/details, approve confirmation, reject-reason dialog, and refreshed state.

- [ ] Write failing tests for pending approval fetch, bell count, PO links, empty state, approve, required reject reason, and backend conflict display.
- [ ] Run focused tests and verify the static notification implementation fails expectations.
- [ ] Fetch approvals once per authenticated shell load and refresh after decisions via a browser event.
- [ ] Replace placeholder Approval Inbox with compact real rows and centered decision modal.
- [ ] Re-run focused and full frontend tests.
- [ ] Commit `feat: activate in-app purchase order approval`.

### Task 6: User Menu and Logout

**Files:**
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: current `/api/auth/me` payload and existing `POST /api/auth/logout`.
- Produces: accessible user dropdown with display name, username, logout progress/error, and `/login` history replacement.

- [ ] Write failing tests that open the user menu, display username, POST logout with credentials, redirect after HTTP 204, close on Escape/outside click, and retain session UI on failure.
- [ ] Run the AppShell test and verify logout controls are absent.
- [ ] Implement the compact dropdown and async logout behavior.
- [ ] Re-run AppShell and full frontend tests.
- [ ] Commit `feat: add authenticated user logout menu`.

### Task 7: Verification, Deployment, and Smoke Flow

**Files:**
- Modify only files required by verification defects.

**Interfaces:**
- Produces: merged master branch and runnable Docker services on ports 3019, 8091, and 5445.

- [ ] Run `go test ./...` in `backend` and require zero failures.
- [ ] Run `npm test -- --run` and `npm run build` in `frontend` and require zero failures.
- [ ] Run `git diff --check` and inspect `git status --short`.
- [ ] Rebuild with `docker compose -p platform-master-po up -d --build` and require all three services Up/healthy.
- [ ] Smoke login as admin, create a PO draft from existing supplier/material data, submit it, login as `director_demo`, approve it, verify status `APPROVED`, then logout and verify `/auth/me` returns 401.
- [ ] Merge the isolated feature branch into local `master` with a fast-forward merge and report exact test/build/smoke evidence.
