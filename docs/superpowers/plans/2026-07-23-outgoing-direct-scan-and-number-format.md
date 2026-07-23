# Outgoing Direct Scan and Number Formatting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Start Outgoing sessions with one click and present PO quantities and money without database-scale trailing zeroes.

**Architecture:** A small shared TypeScript formatter owns browser display rules, while the existing Go PDF renderer gets an equivalent decimal formatter. Outgoing session creation keeps the existing schema but permits an empty destination and removes the unnecessary create modal.

**Tech Stack:** Go 1.26, Gin, PostgreSQL 17, React 19, Next.js 16, TypeScript, Vitest, go-pdf/fpdf.

## Global Constraints

- Database numeric precision and existing records must not change.
- New Outgoing records store empty destination and notes values.
- Empty destination is rendered as `—` in lists and PDFs.
- No Kanban label stamp or reprint behavior is added.
- All behavior changes follow RED-GREEN-REFACTOR.

---

### Task 1: Shared browser number formatting

**Files:**
- Create: `frontend/lib/number-format.ts`
- Create: `frontend/lib/number-format.test.ts`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`

**Interfaces:**
- Produces: `formatQuantity(value, maximumFractionDigits?)` and `formatMoney(value, currency)`.
- Consumers: Supplier Order list and form.

- [ ] **Step 1: Write failing formatter tests**

Cover `formatMoney('10000000.000000','IDR') === 'IDR 10.000.000'`, `formatMoney('12.500000','USD') === 'USD 12,5'`, `formatQuantity('5.000000') === '5'`, and `formatQuantity('5.250000') === '5,25'`.

- [ ] **Step 2: Run focused tests and verify RED**

Run: `npx vitest run lib/number-format.test.ts --maxWorkers=1`

Expected: FAIL because the formatter module does not exist.

- [ ] **Step 3: Implement decimal-safe display formatters**

Parse the decimal string without floating-point arithmetic, trim trailing fraction zeroes, group the whole part with Indonesian separators, use comma as the display decimal separator, render IDR with zero fraction digits, and cap other money at two digits.

- [ ] **Step 4: Replace raw PO display values**

Use `formatMoney` for list totals, line unit prices, line amounts, and order total. Use `formatQuantity` for quantity-per-Kanban and total quantity while leaving editable total-Kanban inputs unchanged.

- [ ] **Step 5: Run formatter, PO component tests, and type-check**

Run: `npx vitest run lib/number-format.test.ts components/supplier-orders/supplier-orders.test.tsx --maxWorkers=1; npm run typecheck`

Expected: PASS.

- [ ] **Step 6: Commit browser formatting**

```powershell
git add frontend/lib frontend/components/supplier-orders
git commit -m "fix: format purchase order values for display"
```

### Task 2: One-click Outgoing creation

**Files:**
- Modify: `backend/internal/outgoing/domain.go`
- Modify: `backend/internal/outgoing/domain_test.go`
- Modify: `backend/internal/outgoing/pdf.go`
- Modify: `backend/internal/outgoing/pdf_test.go`
- Modify: `frontend/components/outgoing/outgoing-index.tsx`
- Modify: `frontend/components/outgoing/outgoing.test.tsx`

**Interfaces:**
- Consumes: existing `POST /outgoing-sessions` request.
- Produces: request `{ "destination": "", "notes": "" }` and immediate navigation to `/outgoing-material/{id}`.

- [ ] **Step 1: Write failing backend tests**

Assert `normalizeDestination("")` succeeds, a 121-character destination still returns `ErrValidation`, and `RenderPDF(Document{Destination:""})` produces a valid PDF with the display fallback `—`.

- [ ] **Step 2: Run backend tests and verify RED**

Run: `go test ./internal/outgoing -count=1`

Expected: FAIL because empty destination is currently rejected.

- [ ] **Step 3: Permit empty destination and render the fallback**

Trim destination, reject only values longer than 120 characters, and pass `emptyDash(d.Destination)` plus `emptyDash(d.Notes)` to the PDF metadata rows.

- [ ] **Step 4: Write failing one-click UI test**

Click `Create outgoing`, assert the POST body equals `{destination:'',notes:''}`, assert no Destination or Notes controls exist, and verify successful navigation to `/outgoing-material/out-1`.

- [ ] **Step 5: Run the UI test and verify RED**

Run: `npx vitest run components/outgoing/outgoing.test.tsx --maxWorkers=1`

Expected: FAIL because the current button opens a modal.

- [ ] **Step 6: Replace the modal with immediate creation**

Remove destination, notes, open-modal state, datalist, and modal markup. Make `Create outgoing` call the existing endpoint directly, disable itself while creating, preserve the existing error message, and navigate on success. Render historical empty destination values as `—` in the list.

- [ ] **Step 7: Run focused backend/frontend tests**

Run: `go test ./internal/outgoing -count=1; npx vitest run components/outgoing/outgoing.test.tsx --maxWorkers=1; npm run typecheck`

Expected: PASS.

- [ ] **Step 8: Commit direct-scan Outgoing**

```powershell
git add backend/internal/outgoing frontend/components/outgoing
git commit -m "fix: start outgoing sessions without destination"
```

### Task 3: PO PDF number formatting

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Produces: `formatPDFDecimal(decimal.Decimal, maximumFractionDigits int)` and `formatPDFMoney(decimal.Decimal, currency string)`.
- Consumers: PO PDF line quantities, prices, amounts, and totals.

- [ ] **Step 1: Write failing PDF formatter tests**

Assert IDR `10000000.000000` formats as `IDR 10.000.000`, quantity `5.250000` formats as `5,25`, and a generated uncompressed test PDF contains the formatted total.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/purchaseorder -run 'TestFormatPDF|TestRenderPurchaseOrderPDF' -count=1`

Expected: FAIL against raw decimal `.String()` output.

- [ ] **Step 3: Implement and apply PDF formatting**

Create a decimal-string formatter with Indonesian grouping, trimmed fractions, and no float conversion. Apply it to quantity-per-Kanban, Kanban count, total quantity, unit price, line total, total base quantity, and total amount.

- [ ] **Step 4: Run purchase-order package tests**

Run: `go test ./internal/purchaseorder -count=1`

Expected: PASS.

- [ ] **Step 5: Commit PDF formatting**

```powershell
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "fix: format purchase order PDF values"
```

### Task 4: Full verification and deployment

**Files:**
- No production files expected.

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: verified local deployment on ports 3019 and 8091.

- [ ] **Step 1: Run all backend checks**

Run: `$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'; go test ./... -count=1; go vet ./...`

Expected: PASS.

- [ ] **Step 2: Run all frontend checks and production build**

Run: `npx vitest run --maxWorkers=1; npm run typecheck; npm run build`

Expected: PASS.

- [ ] **Step 3: Merge locally and rebuild Docker**

Run: `git merge --ff-only fix/outgoing-direct-scan-format; docker compose -p platform-master-po up -d --build`

Expected: frontend and backend are Up; PostgreSQL is healthy.

- [ ] **Step 4: Live smoke test and cleanup**

Verify the PO index renders `IDR 10.000.000`-style totals, click Create outgoing and confirm an active zero-destination session is returned, confirm its scan page loads, then cancel or remove the smoke-test session so no misleading active transaction remains.

- [ ] **Step 5: Confirm clean state**

Run: `git status --short; docker compose -p platform-master-po ps`

Expected: clean `master` and healthy containers.
