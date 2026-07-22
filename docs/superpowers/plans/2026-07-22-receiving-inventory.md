# Receiving and Inventory Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement exclusive resumable Receiving sessions, partial no-excess posting, inventory ledger entries, PO receiving states, outbox creation, and Receiving PDFs.

**Architecture:** A new `receiving` backend module owns sessions and posting. PostgreSQL constraints and row locks enforce exclusivity; posting and the transactional outbox commit atomically. The frontend separates the compact create form from a scan-focused session screen.

**Tech Stack:** PostgreSQL 17/RLS, Go 1.26, Gin, pgx, decimal, go-pdf/fpdf, Next.js 16, React 19, TypeScript, Vitest.

## Global Constraints

- Partial receiving is allowed; duplicate and excess receiving are forbidden.
- One resumable session per DN; pause retains scans and allows another authorized operator to resume the same session.
- Local receiving, inventory, PO status, and PDF do not wait for Sage.
- All new tables are tenant-scoped and RLS-protected; ledger entries are append-only.
- Completion is atomic and idempotent.

---

### Task 1: Receiving schema, ledger, lifecycle, and RLS

**Files:**
- Create: `database/migrations/007_receiving_inventory.sql`
- Create: `backend/internal/receiving/migration_test.go`
- Create: `backend/internal/receiving/migration_integration_test.go`
- Modify: `database/migration-bootstrap.sql`

**Interfaces:**
- Produces tables: `receiving_sessions`, `receiving_session_scans`, `receivings`, `receiving_lines`, `receiving_kanban_lots`, `inventory_ledger_entries`, `integration_outbox`.
- Produces PO statuses `PARTIALLY_RECEIVED`, `FULLY_RECEIVED` and Kanban states `ISSUED`, `IN_STOCK`, `CONSUMED` plus session lock columns/state.

- [ ] Write migration contract tests for columns, tenant composite foreign keys, unique active/resumable session per DN, unique receiving per Kanban, append-only ledger trigger, RLS policies, and outbox idempotency key.
- [ ] Run `go test ./internal/receiving -run Migration -count=1`; confirm RED because migration 007 is absent.
- [ ] Add migration with UUID tenant keys, timestamps/actors, session status check (`ACTIVE`,`PAUSED`,`COMPLETED`,`CANCELLED`,`EXPIRED`), immutable snapshot numeric fields, and partial unique indexes.
- [ ] Grant application role only required operations; deny ledger UPDATE/DELETE and cross-tenant access.
- [ ] Add live tests using `SET LOCAL ROLE nextgen_app` for exclusivity, RLS, ledger immutability, and transaction rollback.
- [ ] Run migration tests and commit `feat: add receiving inventory schema`.

### Task 2: Receiving domain and transactional service

**Files:**
- Create: `backend/internal/receiving/domain.go`
- Create: `backend/internal/receiving/errors.go`
- Create: `backend/internal/receiving/store.go`
- Create: `backend/internal/receiving/service.go`
- Create: `backend/internal/receiving/service_test.go`
- Create: `backend/internal/receiving/store_integration_test.go`
- Modify: `backend/internal/purchaseorder/domain.go`
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Produces: `CreateSession(ctx, Actor, CreateSessionInput) (Session, error)`, `ResumeSession`, `PauseSession`, `CancelSession`, `ScanKanban`, `RemoveScan`, `CompleteSession`, `ListReceivings`, `GetReceiving`.
- `CompleteSession` returns immutable `Receiving` snapshots and creates `SAGE_GOODS_RECEIPT_CREATE` once.

- [ ] Write failing domain tests for DN eligibility, empty submit, duplicate/wrong-DN/already-received scans, full-session conflict, pause/resume, expiry, and typed errors.
- [ ] Verify RED with `go test ./internal/receiving -count=1`.
- [ ] Implement normalized inputs and a repository interface small enough for unit fakes.
- [ ] Implement SQL store transactions using `SELECT ... FOR UPDATE` over session/DN/lots; revalidate all scans before any write.
- [ ] During completion insert receiving snapshots, positive ledger deltas, outbox row with `receiving:<id>:sage-goods-receipt`, update lot states, derive DN/PO status, then close session.
- [ ] Add `StatusPartiallyReceived` and `StatusFullyReceived` to the PO domain; update operational-document loaders so approved, partially received, and fully received POs retain DN/Kanban PDF access.
- [ ] Add live concurrency and idempotency tests; verify one winner and no double ledger/outbox.
- [ ] Run package/full tests and vet; commit `feat: implement receiving transactions`.

### Task 3: Receiving HTTP API and application wiring

**Files:**
- Create: `backend/internal/receiving/http.go`
- Create: `backend/internal/receiving/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Produces authenticated routes:
  - `GET /receiving-options?search=`
  - `GET/POST /receiving-sessions`
  - `GET /receiving-sessions/:id`
  - `POST /receiving-sessions/:id/scans`
  - `DELETE /receiving-sessions/:id/scans/:kanbanId`
  - `POST /receiving-sessions/:id/pause|resume|cancel|complete`
  - `GET /receivings` and `GET /receivings/:id`.

- [ ] Write failing handler tests for permissions, payloads, typed 404/409/422 responses, tenant actor propagation, and idempotent completion response.
- [ ] Verify RED, register routes with `receiving.view/create/submit` permissions, and implement compact JSON projections.
- [ ] Wire the SQL store/service into main without changing existing module startup.
- [ ] Run backend tests and vet; commit `feat: expose receiving API`.

### Task 4: Receiving PDF

**Files:**
- Create: `backend/internal/receiving/pdf.go`
- Create: `backend/internal/receiving/pdf_test.go`
- Modify: `backend/internal/receiving/http.go`
- Modify: `backend/internal/receiving/http_test.go`

**Interfaces:**
- Produces: `RenderReceivingPDF(receiving ReceivingDocument) ([]byte, error)` and `GET /receivings/:id/document.pdf`.

- [ ] Write failing tests for `%PDF-`, Unicode, received lot IDs, planned/previous/now/outstanding snapshots, actor, read-only Sage number, safe inline filename, and `private, no-store`.
- [ ] Verify RED.
- [ ] Implement tenant-scoped projection and renderer using the existing embedded fonts/layout helpers (move shared helpers to `backend/internal/pdfdoc` only if compilation demands reuse).
- [ ] Keep generation read-only so render failure cannot affect posting.
- [ ] Run backend tests/vet and commit `feat: generate receiving PDFs`.

### Task 5: Receiving index, create modal, and scan session UI

**Files:**
- Create: `frontend/components/receiving/receiving-index.tsx`
- Create: `frontend/components/receiving/receiving-form.tsx`
- Create: `frontend/components/receiving/receiving-session.tsx`
- Create: `frontend/components/receiving/receiving.test.tsx`
- Create: `frontend/app/receiving/[id]/page.tsx`
- Modify: `frontend/app/receiving/page.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes Task 3 API and Task 4 PDF endpoint.
- Produces compact index, centered DN form, focused scanner, pause/resume, and completed PDF link.

- [ ] Write failing interaction tests for typeable DN selection, material preview, create-session navigation, autofocus/refocus, Enter-to-scan, inline scan errors, remove, pause/resume, exclusive-session conflict, submit confirmation, and PDF link.
- [ ] Verify RED with the focused Vitest file.
- [ ] Implement index using real API data and existing AppShell/table/modal patterns.
- [ ] Update Supplier Order document-link eligibility so `APPROVED`, `PARTIALLY_RECEIVED`, and `FULLY_RECEIVED` orders retain their DN/Kanban links.
- [ ] Implement the scanner with no quantity field, explicit scanned/outstanding counters, stable focus, and keyboard-first actions.
- [ ] Add responsive compact CSS and no oversized cards/headings.
- [ ] Run all frontend tests, typecheck, and build; commit `feat: add focused receiving workflow`.

### Task 6: Receiving live verification

**Files:** Modify only for verified defects.

- [ ] Run fresh backend live/full tests, vet, frontend full tests/typecheck/build, and `git diff --check`.
- [ ] Rebuild Docker.
- [ ] Live-create a partial receiving, verify positive ledger/outbox/PO `PARTIALLY_RECEIVED`, resume the same paused session from another user, complete remaining lots, and verify `FULLY_RECEIVED`.
- [ ] Download the Receiving PDF and verify `%PDF-`, snapshots, safe filename, and no Sage dependency.
