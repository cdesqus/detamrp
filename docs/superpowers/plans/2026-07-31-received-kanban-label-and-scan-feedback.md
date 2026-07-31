# Received Kanban Label and Scan Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Mark received Kanban cards in the labels PDF and return clear feedback for every receiving scan outcome.

**Architecture:** Extend the Kanban label document query with the lot status and render a red visual stamp without changing the QR payload. Introduce typed receiving scan errors and stable HTTP error codes, then map those codes to explicit frontend messages.

**Tech Stack:** Go, PostgreSQL/pgx, go-pdf/fpdf, Gin, React, Vitest.

## Global Constraints

- Keep the QR payload equal to the original Kanban ID.
- Show Kanban ID immediately below the QR, followed by the card counter.
- Render only the word `RECEIVED` in red for received lots.
- Do not change other documents or transaction tables.

---

### Task 1: Kanban Labels PDF

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Test: `backend/internal/purchaseorder/pdf_test.go`

- [ ] Add failing tests for received status loading, caption order, and the red `RECEIVED` stamp.
- [ ] Extend `KanbanLabel` and its SQL query with lot status.
- [ ] Reorder the QR captions and render the received stamp.
- [ ] Run purchase-order PDF tests.

### Task 2: Receiving Scan Error Contract

**Files:**
- Modify: `backend/internal/receiving/domain.go`
- Modify: `backend/internal/receiving/store.go`
- Modify: `backend/internal/receiving/http.go`
- Test: `backend/internal/receiving/domain_test.go`

- [ ] Add failing tests for duplicate-session, already-received, wrong-DN, and unknown-ID codes.
- [ ] Classify the lot state before inserting a scan.
- [ ] Return stable JSON error codes from the scan endpoint.
- [ ] Run receiving backend tests.

### Task 3: Scanner Feedback UI

**Files:**
- Modify: `frontend/components/receiving/receiving-session.tsx`
- Test: `frontend/components/receiving/receiving.test.tsx`

- [ ] Add failing tests for explicit scan failure and success feedback.
- [ ] Map backend codes to clear English messages.
- [ ] Preserve the scanned value on failure and clear it on success.
- [ ] Run receiving frontend tests.

### Task 4: Verification

- [ ] Run `go test ./...` in `backend`.
- [ ] Run frontend tests, typecheck, lint, and build.
- [ ] Review `git diff --check`, commit, and push.
