# Receiving DN Scan Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Receiving DN suggestions with exact scanner-first validation and fix empty Kanban sessions crashing the scan page.

**Architecture:** The existing receiving session creation endpoint will accept the human-facing DN number and resolve it atomically inside the tenant transaction. Stable backend error codes drive English UI validation messages. Both backend and frontend normalize an empty scan collection to `[]`.

**Tech Stack:** Go 1.26, Gin, PostgreSQL 17, Next.js 16, React 19, TypeScript, Vitest.

## Global Constraints

- DN matching is complete, trimmed, and case-insensitive; partial matching is forbidden.
- The create modal contains no datalist, select, or suggestion UI.
- All receiving validation text is English.
- Existing tenant isolation and one-open-session locking remain enforced.
- Use test-first changes and do not add a database migration.

---

### Task 1: Exact DN session creation contract

**Files:**
- Modify: `backend/internal/receiving/domain.go`
- Modify: `backend/internal/receiving/store.go`
- Modify: `backend/internal/receiving/http.go`
- Test: `backend/internal/receiving/domain_test.go`
- Test: `backend/internal/receiving/http_test.go`

**Interfaces:**
- Consumes: `POST /receiving-sessions` authenticated tenant context.
- Produces: request `{ "deliveryNoteNumber": "DN-..." }`; errors `{ "error": "...", "code": "DN_INVALID|DN_FULLY_RECEIVED|DN_IN_PROGRESS" }`.

- [ ] **Step 1: Write failing normalization and HTTP contract tests**

Add tests proving whitespace/case normalization, an empty number rejection, the request field `deliveryNoteNumber`, and stable error-code mapping.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `go test ./internal/receiving -run 'TestNormalizeDeliveryNote|TestCreateSession' -count=1`

Expected: FAIL because DN-number creation and stable error codes are missing.

- [ ] **Step 3: Implement exact lookup and typed business errors**

Normalize with `strings.ToUpper(strings.TrimSpace(value))`. Inside one `database.WithTenant` transaction, query `delivery_notes` by `upper(delivery_note_number)=$2`, distinguish zero `ISSUED` Kanbans from unknown DN, reject `ACTIVE` or `PAUSED` sessions, and then insert the session.

- [ ] **Step 4: Run focused and package tests**

Run: `go test ./internal/receiving -count=1`

Expected: PASS.

- [ ] **Step 5: Commit backend contract**

```powershell
git add backend/internal/receiving
git commit -m "fix: validate receiving sessions by DN number"
```

### Task 2: Scanner-first modal and null-safe Kanban session

**Files:**
- Modify: `frontend/components/receiving/receiving-index.tsx`
- Modify: `frontend/components/receiving/receiving-session.tsx`
- Test: `frontend/components/receiving/receiving.test.tsx`

**Interfaces:**
- Consumes: Task 1 request and stable error codes.
- Produces: exact DN submit-on-Enter UI and null-safe scan rendering.

- [ ] **Step 1: Write failing UI tests**

Assert that the modal has `Scan or Type DN Number`, has no `list` attribute or suggestions, POSTs `{deliveryNoteNumber:'DN-1'}` on Enter, maps all three codes to the specified English messages, navigates on success, and renders `Ready to scan.` when `scans:null`.

- [ ] **Step 2: Run focused test and verify RED**

Run: `npx vitest run components/receiving/receiving.test.tsx --maxWorkers=1`

Expected: FAIL against the current datalist flow and null `.length` access.

- [ ] **Step 3: Implement minimal scanner-first UI**

Remove options fetching, `datalist`, preview, and Start button dependency. Submit the normalized field on form Enter, decode the backend `code`, keep the modal open on validation failure, and navigate immediately on success. Normalize session data using `scans: Array.isArray(payload.scans) ? payload.scans : []` before storing it.

- [ ] **Step 4: Run focused UI test and type-check**

Run: `npx vitest run components/receiving/receiving.test.tsx --maxWorkers=1; npm run typecheck`

Expected: PASS.

- [ ] **Step 5: Commit UI behavior**

```powershell
git add frontend/components/receiving
git commit -m "fix: make receiving DN entry scanner first"
```

### Task 3: Regression verification and local deployment

**Files:**
- No production files expected.

**Interfaces:**
- Consumes: completed Tasks 1 and 2.
- Produces: verified master deployment at ports 3019 and 8091.

- [ ] **Step 1: Run full backend verification**

Run: `$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'; go test ./... -count=1; go vet ./...`

Expected: PASS.

- [ ] **Step 2: Run full frontend verification**

Run: `npx vitest run --maxWorkers=1; npm run typecheck; npm run build`

Expected: PASS.

- [ ] **Step 3: Merge the feature branch to master and rebuild Docker**

Run: `git merge --ff-only fix/receiving-dn-scan; docker compose -p platform-master-po up -d --build`

Expected: frontend and backend containers are Up; PostgreSQL is healthy.

- [ ] **Step 4: Smoke-test exact DN and empty scan session**

Use the admin session to confirm invalid DN returns `DN_INVALID`, an open DN returns `DN_IN_PROGRESS`, and a newly created zero-scan session endpoint returns `"scans":[]`. Open the resulting `/receiving/{id}` route and confirm HTTP 200 with a functional Kanban input.

- [ ] **Step 5: Confirm clean completion**

Run: `git status --short; docker compose -p platform-master-po ps`

Expected: clean `master`, frontend/backend Up, PostgreSQL healthy.
