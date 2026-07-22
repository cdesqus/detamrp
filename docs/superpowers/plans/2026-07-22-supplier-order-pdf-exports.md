# Supplier Order PDF Exports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Open real PO, DN, and Code-128 Kanban-label PDFs from the Supplier Order index and return successful approval submissions to that index.

**Architecture:** The Go backend loads tenant-scoped immutable document projections and renders PDFs in memory through focused renderer functions. Authenticated Gin endpoints apply existing PO permissions and stream inline PDFs. The Next.js index opens those endpoints in new tabs while the detail form keeps only operational information.

**Tech Stack:** Go 1.26, Gin, pgx, `github.com/go-pdf/fpdf`, `github.com/boombuler/barcode/code128`, Next.js 16, React 19, TypeScript, Go tests, Vitest.

## Global Constraints

- PDF responses are real `application/pdf` documents generated in memory and never persisted.
- All document queries are actor-tenant scoped and endpoints require `po.view`.
- PO prices appear only with `po.price.view`.
- DN and Kanban Labels PDFs exist only for approved POs with generated documents.
- Every Kanban lot produces exactly one two-column A4 label with a scannable Code 128 barcode.
- The Supplier Order list stays compact; unavailable documents render as neutral dashes.
- Successful `Save & Send for Approval` navigates to `/supplier-orders`; failures remain on the form.

---

### Task 1: Tenant-scoped PDF projections and renderers

**Files:**
- Create: `backend/internal/purchaseorder/pdf.go`
- Create: `backend/internal/purchaseorder/pdf_test.go`
- Modify: `backend/go.mod`
- Modify: `backend/go.sum`

**Interfaces:**
- Consumes: `Order`, `OrderLine`, Delivery Note tables, Kanban lot tables, and `Actor`.
- Produces: `RenderPOPDF(order Order, includePrices bool) ([]byte, error)`, `RenderDeliveryNotePDF(document DeliveryNoteDocument) ([]byte, error)`, `RenderKanbanLabelsPDF(document KanbanLabelDocument) ([]byte, error)` and tenant-scoped store loaders.

- [ ] **Step 1: Write failing renderer and projection tests**

Tests require `%PDF-` signatures, price presence/redaction, all DN lines, exact label count, Kanban IDs passed to Code 128, stable label ordering, and tenant-filtered SQL arguments:

```go
if !bytes.HasPrefix(result, []byte("%PDF-")) { t.Fatal("not a PDF") }
if bytes.Contains(redacted, []byte("200000")) { t.Fatal("price leaked") }
if got := countLabelMarkers(result); got != len(document.Labels) { t.Fatalf("labels=%d", got) }
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run 'TestRender.*PDF|TestLoad.*Document' -count=1`

Expected: FAIL because renderer types/functions are absent.

- [ ] **Step 3: Add dependencies and renderers**

Use `go get github.com/go-pdf/fpdf@v0.9.0 github.com/boombuler/barcode@v1.1.0`. Render A4 documents with built-in fonts. Encode each Kanban ID using:

```go
encoded, err := code128.Encode(label.KanbanID)
scaled, err := barcode.Scale(encoded, 420, 72)
```

Convert the barcode image to PNG in memory and register it with fpdf. Keep renderer inputs independent from SQL.

- [ ] **Step 4: Add tenant-scoped document loaders**

Load DN header/lines and labels ordered by PO-line sort position plus lot number. Return `NotFoundError` for another tenant and `ConflictError` when the PO exists but operational documents are unavailable.

- [ ] **Step 5: Verify GREEN and commit**

Run: `go test ./internal/purchaseorder -count=1; go test ./...; go vet ./...`

Expected: all PASS.

```powershell
git add backend/go.mod backend/go.sum backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: render supplier order PDFs"
```

### Task 2: Authenticated PDF HTTP endpoints

**Files:**
- Modify: `backend/internal/purchaseorder/service.go`
- Modify: `backend/internal/purchaseorder/http.go`
- Modify: `backend/internal/purchaseorder/http_test.go`
- Modify: `backend/internal/purchaseorder/service_test.go`

**Interfaces:**
- Consumes: renderers/loaders from Task 1 and existing `po.view`/`po.price.view` permissions.
- Produces: the three `/purchase-orders/:id/documents/*.pdf` GET endpoints.

- [ ] **Step 1: Write failing HTTP contract tests**

Test all routes require `po.view`, content begins `%PDF-`, headers contain `application/pdf` and safe inline filenames, PO prices depend on permission, missing DN/labels return typed responses, and renderer failures produce sanitized responses plus structured logs.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run 'Test.*PDF.*Route|Test.*DocumentEndpoint' -count=1`

Expected: FAIL with missing routes/service methods.

- [ ] **Step 3: Register and implement endpoints**

Register routes beneath the authenticated purchase-order group:

```go
orders.GET("/:id/documents/po.pdf", rbac.RequirePermissions("po.view"), handler)
orders.GET("/:id/documents/delivery-note.pdf", rbac.RequirePermissions("po.view"), handler)
orders.GET("/:id/documents/kanban-labels.pdf", rbac.RequirePermissions("po.view"), handler)
```

Set `Content-Disposition: inline; filename="<safe-name>.pdf"`, stream bytes, and log render failures with tenant/user/PO/document type.

- [ ] **Step 4: Verify GREEN and commit**

Run: `go test ./internal/purchaseorder -count=1; go test ./...; go vet ./...`

Expected: all PASS.

```powershell
git add backend/internal/purchaseorder/service.go backend/internal/purchaseorder/http.go backend/internal/purchaseorder/http_test.go backend/internal/purchaseorder/service_test.go
git commit -m "feat: expose supplier order PDF endpoints"
```

### Task 3: Index PDF actions and submit-and-close UX

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: PDF endpoints from Task 2 and the existing order `documents` summary.
- Produces: compact new-tab PDF links on the list and successful submit redirect.

- [ ] **Step 1: Write failing interaction tests**

Require `PO PDF` for persisted rows, DN/labels links only for approved rows with summaries, target `_blank` plus `rel="noopener noreferrer"`, neutral dashes otherwise, no detail document card, and redirect only after submit success:

```tsx
expect(screen.getByRole('link', { name: 'Open PO PDF for PO-1' })).toHaveAttribute('target', '_blank');
expect(push).toHaveBeenCalledWith('/supplier-orders');
```

- [ ] **Step 2: Verify RED**

Run: `npx vitest run components/supplier-orders/supplier-orders.test.tsx --maxWorkers=1`

Expected: FAIL because PDF links and submit redirect are absent.

- [ ] **Step 3: Implement compact index actions and redirect**

Use authenticated same-origin anchors such as `/api/purchase-orders/${id}/documents/po.pdf`. Remove the detail Documents card. After successful submit hydrate, execute `router.push('/supplier-orders')`; do not redirect on save/submit errors.

- [ ] **Step 4: Verify frontend and commit**

Run:

```powershell
npx vitest run --maxWorkers=1
npm run typecheck
npm run build
```

Expected: all PASS.

```powershell
git add frontend/components/supplier-orders/supplier-order-index.tsx frontend/components/supplier-orders/supplier-order-form.tsx frontend/components/supplier-orders/supplier-orders.test.tsx frontend/app/globals.css
git commit -m "feat: open supplier order PDFs from index"
```

### Task 4: Final live verification and deployment

**Files:**
- Modify only if a verified defect is found in Tasks 1–3.

- [ ] **Step 1: Run fresh complete verification**

Run backend live tests with `TEST_DATABASE_URL`, frontend 1-worker tests, typecheck, build, vet, and `git diff --check`.

- [ ] **Step 2: Rebuild the local stack**

Run: `docker compose -p platform-master-po up -d --build`

Expected: migration exit 0; PostgreSQL healthy; backend/frontend running.

- [ ] **Step 3: Smoke all three PDFs**

Authenticate, request all valid PDFs, assert HTTP 200, `application/pdf`, `%PDF-`, non-empty safe filenames, price redaction by role, and exact label count for an approved PO. Assert pending PO DN/labels are unavailable and its status is unchanged.

- [ ] **Step 4: Record repository state**

Run: `git status --short; git log -6 --oneline`

Expected: clean local `master` with feature commits visible.
