# Outgoing Material Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement full-Kanban Outgoing Material sessions that consume received stock, append negative ledger entries, and produce an internal PDF.

**Architecture:** A focused `outgoing` backend module uses the inventory and lifecycle foundation from the Receiving plan. A compact header followed by a scan-focused UI posts immutable full-lot consumption locally, with no Sage outbox event.

**Tech Stack:** PostgreSQL 17/RLS, Go 1.26, Gin, pgx, decimal, go-pdf/fpdf, Next.js 16, React 19, TypeScript, Vitest.

## Global Constraints

- Every scan consumes one complete `IN_STOCK` Kanban; partial quantity is impossible.
- Destination is a typeable suggestion field with free-text input; no Production Line master is added.
- Completed Outgoing transactions are immutable and never pushed to Sage in the MVP.
- Ledger, session, and document data remain tenant-scoped and auditable.

---

### Task 1: Outgoing schema and RLS

**Files:**
- Create: `database/migrations/008_outgoing_material.sql`
- Create: `backend/internal/outgoing/migration_test.go`
- Create: `backend/internal/outgoing/migration_integration_test.go`
- Modify: `database/migration-bootstrap.sql`

**Interfaces:**
- Produces: `outgoing_sessions`, `outgoing_session_scans`, `outgoing_documents`, `outgoing_lines`, and uniqueness/lock constraints over Kanban lots.

- [ ] Write failing migration tests for tenant composite keys, document number uniqueness, session statuses, unique active scan lock, RLS, and immutable completed rows.
- [ ] Verify RED.
- [ ] Add migration and grants; allow free-text destination but enforce normalized non-empty length at the domain layer.
- [ ] Add live role/RLS/lock tests and run them.
- [ ] Commit `feat: add outgoing material schema`.

### Task 2: Full-Kanban outgoing service

**Files:**
- Create: `backend/internal/outgoing/domain.go`
- Create: `backend/internal/outgoing/errors.go`
- Create: `backend/internal/outgoing/store.go`
- Create: `backend/internal/outgoing/service.go`
- Create: `backend/internal/outgoing/service_test.go`
- Create: `backend/internal/outgoing/store_integration_test.go`

**Interfaces:**
- Produces `CreateSession`, `ScanKanban`, `RemoveScan`, `CancelSession`, `CompleteSession`, `ListOutgoing`, `GetOutgoing`.
- Completion consumes stored lot quantity; API accepts no quantity field.

- [ ] Write failing tests for normalized destination, missing notes allowance, not-received/consumed/duplicate/locked lot rejection, removal, empty completion, idempotency, and multi-material documents.
- [ ] Verify RED.
- [ ] Implement domain types with no quantity input in scan commands.
- [ ] Implement atomic SQL completion: lock lots, require `IN_STOCK`, create immutable header/lines, set `CONSUMED`, append exact negative full-lot ledger delta, close session.
- [ ] Assert no `integration_outbox` insert occurs in unit and live tests.
- [ ] Run package/full tests and vet; commit `feat: implement full Kanban outgoing`.

### Task 3: Outgoing API and PDF

**Files:**
- Create: `backend/internal/outgoing/http.go`
- Create: `backend/internal/outgoing/http_test.go`
- Create: `backend/internal/outgoing/pdf.go`
- Create: `backend/internal/outgoing/pdf_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Produces `GET/POST /outgoing-sessions`, scan/remove/cancel/complete subroutes, `GET /outgoing-material`, `GET /outgoing-material/:id`, and `GET /outgoing-material/:id/document.pdf`.

- [ ] Write failing permission and contract tests using `inventory.view` and `inventory.consume`; verify scan requests contain only `kanbanId`.
- [ ] Write failing PDF tests for `%PDF-`, destination, notes, operator, all lot/material/full quantities, Unicode, inline filename, and no-store.
- [ ] Verify RED.
- [ ] Register endpoints, map typed errors, stream the in-memory PDF, and wire module startup.
- [ ] Run backend tests/vet; commit `feat: expose outgoing documents`.

### Task 4: Outgoing index and focused scanner

**Files:**
- Create: `frontend/components/outgoing/outgoing-index.tsx`
- Create: `frontend/components/outgoing/outgoing-form.tsx`
- Create: `frontend/components/outgoing/outgoing-session.tsx`
- Create: `frontend/components/outgoing/outgoing.test.tsx`
- Create: `frontend/app/outgoing-material/[id]/page.tsx`
- Modify: `frontend/app/outgoing-material/page.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes Task 3 API.
- Produces compact index, typeable/free-text destination, keyboard scanner, full-lot preview, and PDF link.

- [ ] Write failing tests for suggestions plus free text, session navigation, autofocus, scan response details, no quantity input, duplicate/error feedback, remove, submit, immutable completed view, and PDF link.
- [ ] Verify focused RED.
- [ ] Implement real-data index and centered header form.
- [ ] Implement focused scanner showing Kanban, material, full quantity/unit, warehouse/location; never render an editable quantity.
- [ ] Run full frontend tests, typecheck, build; commit `feat: add outgoing material workflow`.

### Task 5: Outgoing live verification

**Files:** Modify only for verified defects.

- [ ] Receive test Kanban into stock, create Outgoing with a free-text destination, scan and submit whole lots.
- [ ] Verify lots become `CONSUMED`, negative ledger amounts exactly match received full quantities, and no Sage outbox job is created.
- [ ] Verify repeat/partial/non-stock attempts fail without side effects.
- [ ] Verify PDF and compact list in the running Docker application.
- [ ] Run all backend/frontend verification commands and `git diff --check`.
