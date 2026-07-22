# Approved PO Documents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate one tenant-scoped DN and the exact approved Kanban lots atomically for every approved PO, backfill existing approved POs, expose the documents in Supplier Orders, and make draft navigation behave naturally.

**Architecture:** PostgreSQL owns document identity, relationships, uniqueness, RLS, and legacy backfill. The purchase-order store calls an idempotent document generator inside the existing approval transaction, and order queries return a compact document summary. The Next.js UI consumes that summary, provides consistent Back navigation, and moves draft cancellation to the index.

**Tech Stack:** PostgreSQL 17 migrations and RLS, Go 1.26 with pgx, Gin HTTP APIs, Next.js 16/React 19/TypeScript, Vitest and Go tests, Docker Compose.

## Global Constraints

- The MVP creates exactly one system-issued DN per approved PO and includes all PO lines.
- One Kanban lot row represents one physical barcode and uses the PO line quantity snapshot.
- Approval and generation commit atomically; a failed generation must leave the PO pending.
- Generation and migration backfill must be idempotent and tenant isolated.
- Existing approved POs, including `PO-202607-00001` and `PO-202607-00003`, must be backfilled.
- `Save as Draft` returns to the Supplier Order index; `Cancel Draft` lives in a row `...` menu with confirmation.
- Tables and typography remain compact and consistent with the existing UI.

---

### Task 1: Tenant-scoped inbound document schema and approved-order backfill

**Files:**
- Create: `database/migrations/006_inbound_documents.sql`
- Create: `backend/internal/purchaseorder/inbound_migration_test.go`

**Interfaces:**
- Consumes: `purchase_orders`, `purchase_order_lines`, `users`, and `app.tenant_id` from migrations 001–005.
- Produces: `delivery_note_number_sequences`, `kanban_number_sequences`, `delivery_notes`, `delivery_note_lines`, and `kanban_lots` with tenant-scoped constraints.

- [ ] **Step 1: Write the failing migration contract test**

Add a test that reads migration 006 and requires these concrete contracts:

```go
func TestInboundMigrationContainsGenerationAndIsolationContracts(t *testing.T) {
    sql := readMigration(t, "006_inbound_documents.sql")
    required := []string{
        "create table if not exists delivery_notes",
        "unique (tenant_id, purchase_order_id)",
        "create table if not exists delivery_note_lines",
        "unique (tenant_id, delivery_note_id, purchase_order_line_id)",
        "create table if not exists kanban_lots",
        "unique (tenant_id, kanban_id)",
        "force row level security",
        "where p.status = 'approved'",
        "generate_series",
    }
    for _, fragment := range required {
        if !strings.Contains(strings.ToLower(sql), fragment) { t.Errorf("missing %q", fragment) }
    }
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/purchaseorder -run TestInboundMigration -count=1`

Expected: FAIL because `006_inbound_documents.sql` does not exist.

- [ ] **Step 3: Add the migration**

Create UUID-backed tables with composite tenant foreign keys and these statuses:

```sql
status text not null default 'ISSUED' check (status in ('ISSUED','PARTIALLY_RECEIVED','RECEIVED','CANCELLED'))
```

Use `DN-YYYYMM-#####` and `KB-YYYYMM-######`. Backfill approved POs with `row_number()` partitioned by tenant/month; expand each PO line using:

```sql
CROSS JOIN LATERAL generate_series(1, pol.total_kanban::bigint) lot_no
```

Seed sequence rows to the maximum generated ordinal, enable and force RLS on all five tables, create `tenant_id = current_setting('app.tenant_id')::uuid` policies, and grant only the operations required by `nextgen_app`.

- [ ] **Step 4: Verify migration and integration startup**

Run:

```powershell
go test ./internal/purchaseorder -run 'TestInboundMigration|TestMigration' -count=1
docker compose -p platform-master-po up -d --build migrate backend
docker compose -p platform-master-po ps
```

Expected: tests PASS; migrator exits 0; PostgreSQL healthy; backend running.

- [ ] **Step 5: Commit schema task**

```powershell
git add database/migrations/006_inbound_documents.sql backend/internal/purchaseorder/inbound_migration_test.go
git commit -m "feat: add inbound document schema and backfill"
```

### Task 2: Atomic and idempotent approval document generator

**Files:**
- Create: `backend/internal/purchaseorder/documents.go`
- Modify: `backend/internal/purchaseorder/store.go`
- Modify: `backend/internal/purchaseorder/sql_store_integration_test.go`

**Interfaces:**
- Consumes: `database.TenantTx`, locked PO ID, approval actor, and the five tables from Task 1.
- Produces: `ensureApprovedDocuments(ctx context.Context, tx database.TenantTx, actor Actor, purchaseOrderID uuid.UUID) error`.

- [ ] **Step 1: Write failing integration tests**

Add tests which submit and approve a two-line PO where line quantities are 2 and 3 Kanbans, then assert:

```go
if dnCount != 1 || dnLineCount != 2 || lotCount != 5 {
    t.Fatalf("documents = dn %d, lines %d, lots %d", dnCount, dnLineCount, lotCount)
}
```

Call `ensureApprovedDocuments` again and assert counts remain `1/2/5`. Add a forced generator-error fixture and assert the approval plus PO both remain pending.

- [ ] **Step 2: Run integration test and verify RED**

Run: `go test ./internal/purchaseorder -run TestSQLStoreApprovalGeneratesDocuments -count=1`

Expected: FAIL because the generator and document rows do not exist.

- [ ] **Step 3: Implement the generator**

In `documents.go`, allocate number blocks under sequence-row locks, insert the DN with `ON CONFLICT (tenant_id,purchase_order_id) DO NOTHING`, insert one junction row per PO line, and insert exactly `total_kanban` lots. Validate integral Kanban counts before looping:

```go
if !line.TotalKanban.Equal(line.TotalKanban.Truncate(0)) || !line.TotalKanban.IsPositive() {
    return fmt.Errorf("PO line %s has invalid Kanban count", line.ID)
}
```

Every insert uses tenant ID and actor ID explicitly. Existing rows are loaded and validated before treating a retry as successful.

- [ ] **Step 4: Call the generator in the approval transaction**

In `decide`, after changing the PO to `APPROVED` but before loading the response, call:

```go
if orderStatus == StatusApproved {
    if err = ensureApprovedDocuments(ctx, tx, actor, purchaseOrderID); err != nil { return err }
}
```

Do not call it for rejection.

- [ ] **Step 5: Verify generator tests and all backend tests**

Run:

```powershell
go test ./internal/purchaseorder -run TestSQLStoreApprovalGeneratesDocuments -count=1
go test ./...
```

Expected: targeted test PASS; complete backend suite PASS.

- [ ] **Step 6: Commit generator task**

```powershell
git add backend/internal/purchaseorder/documents.go backend/internal/purchaseorder/store.go backend/internal/purchaseorder/sql_store_integration_test.go
git commit -m "feat: generate DN and Kanban lots on approval"
```

### Task 3: Purchase Order document summary API

**Files:**
- Modify: `backend/internal/purchaseorder/domain.go`
- Modify: `backend/internal/purchaseorder/store.go`
- Modify: `backend/internal/purchaseorder/http_test.go`
- Modify: `backend/internal/purchaseorder/store_test.go`

**Interfaces:**
- Consumes: generated document tables from Task 1.
- Produces: `DocumentSummary` embedded as `documents` in each `Order` JSON response.

- [ ] **Step 1: Write failing serialization and store tests**

Require this exact JSON structure for an approved order:

```json
{
  "documents": {
    "deliveryNoteId": "uuid",
    "deliveryNoteNumber": "DN-202607-00001",
    "kanbanCount": 10,
    "issuedAt": "2026-07-22T00:00:00Z"
  }
}
```

Assert non-approved orders return `documents: null`.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/purchaseorder -run 'Test.*DocumentSummary|TestDomainJSONContract' -count=1`

Expected: FAIL because `Order.Documents` is absent.

- [ ] **Step 3: Add domain type and tenant-safe hydration**

Add:

```go
type DocumentSummary struct {
    DeliveryNoteID uuid.UUID `json:"deliveryNoteId"`
    DeliveryNoteNumber string `json:"deliveryNoteNumber"`
    KanbanCount int64 `json:"kanbanCount"`
    IssuedAt time.Time `json:"issuedAt"`
}
```

Add this field to `Order`:

```go
Documents *DocumentSummary `json:"documents"`
```

Hydrate summaries for order list and detail using a tenant-filtered aggregate over `delivery_notes`, `delivery_note_lines`, and `kanban_lots`; avoid one query per list row by loading summaries for all listed PO IDs in one query.

- [ ] **Step 4: Run backend suite**

Run: `go test ./...`

Expected: all backend tests PASS.

- [ ] **Step 5: Commit API task**

```powershell
git add backend/internal/purchaseorder/domain.go backend/internal/purchaseorder/store.go backend/internal/purchaseorder/http_test.go backend/internal/purchaseorder/store_test.go
git commit -m "feat: expose PO document summaries"
```

### Task 4: Supplier Order detail navigation and document display

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `InitialOrder.documents` from Task 3.
- Produces: compact Back action and approved Documents card.

- [ ] **Step 1: Write failing UI tests**

Render a detail and assert:

```tsx
expect(screen.getByRole('button', { name: 'Back to Supplier Orders' })).toBeInTheDocument();
expect(screen.getByText('DN-202607-00001')).toBeInTheDocument();
expect(screen.getByText('10 Kanban labels')).toBeInTheDocument();
```

Click Back and assert `router.push('/supplier-orders')`. Render a pending PO and assert the Documents card is absent.

- [ ] **Step 2: Run test and verify RED**

Run: `npm test -- components/supplier-orders/supplier-orders.test.tsx`

Expected: FAIL because Back and Documents are missing.

- [ ] **Step 3: Implement compact UI**

Extend `InitialOrder` with:

```ts
documents?: { deliveryNoteId: string; deliveryNoteNumber: string; kanbanCount: number; issuedAt: string } | null;
```

Render Back above the title for any saved detail. Render a compact Documents card only when `status === 'APPROVED' && documents`, containing DN number, issue date, Kanban count, `Print DN`, and `Print Kanban Labels` actions. Use existing button sizes and card spacing; do not introduce large headings.

- [ ] **Step 4: Verify frontend test**

Run: `npm test -- components/supplier-orders/supplier-orders.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit detail UI task**

```powershell
git add frontend/components/supplier-orders/supplier-order-form.tsx frontend/components/supplier-orders/supplier-orders.test.tsx frontend/app/globals.css
git commit -m "feat: show generated documents on PO detail"
```

### Task 5: Draft save redirect and index cancellation menu

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: existing `POST /purchase-orders/:id/cancel` and draft status.
- Produces: save-and-exit behavior plus row `...` cancellation action.

- [ ] **Step 1: Write failing interaction tests**

After successful draft save, assert:

```tsx
await waitFor(() => expect(push).toHaveBeenCalledWith('/supplier-orders'));
expect(screen.queryByRole('button', { name: 'Cancel draft' })).not.toBeInTheDocument();
```

On a draft index row, open `Actions for PO-...`, select `Cancel Draft`, confirm the centered modal, assert POST `/api/purchase-orders/:id/cancel`, and assert the list reloads. Assert non-draft rows do not offer cancellation.

- [ ] **Step 2: Run test and verify RED**

Run: `npm test -- components/supplier-orders/supplier-orders.test.tsx`

Expected: FAIL because draft save stays in the form and cancellation is still inside it.

- [ ] **Step 3: Implement draft behavior**

After `hydrate(order)` on a successful non-submit save, execute:

```ts
router.push('/supplier-orders');
return;
```

Remove the form's `Cancel draft` action and obsolete `cancel()` function. Add a compact `...` row menu to the index, a centered confirmation modal, duplicate-click protection, visible API errors, Escape-to-close, and focus return.

- [ ] **Step 4: Verify focused and complete frontend suites**

Run:

```powershell
npm test -- components/supplier-orders/supplier-orders.test.tsx
npm test
npm run typecheck
npm run build
```

Expected: focused tests PASS; all frontend tests PASS; typecheck and production build exit 0.

- [ ] **Step 5: Commit draft UX task**

```powershell
git add frontend/components/supplier-orders/supplier-order-form.tsx frontend/components/supplier-orders/supplier-order-index.tsx frontend/components/supplier-orders/supplier-orders.test.tsx frontend/app/globals.css
git commit -m "fix: make supplier order draft flow exit-first"
```

### Task 6: Local deployment and live backfill verification

**Files:**
- Modify only if verification exposes a defect in files owned by Tasks 1–5.

**Interfaces:**
- Consumes: complete database, backend, and frontend implementation.
- Produces: running local application with backfilled approved POs.

- [ ] **Step 1: Run clean verification**

Run:

```powershell
Push-Location backend; go test ./...; Pop-Location
Push-Location frontend; npm test; npm run typecheck; npm run build; Pop-Location
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 2: Rebuild and start local stack**

Run:

```powershell
docker compose -p platform-master-po up -d --build
docker compose -p platform-master-po ps
```

Expected: migrator exits 0; PostgreSQL healthy; backend and frontend running.

- [ ] **Step 3: Verify existing data through API and SQL**

Authenticate locally, fetch `PO-202607-00001` and `PO-202607-00003`, and assert both report a non-null DN summary. Query counts and verify each order has one DN, one DN line per PO line, and `sum(total_kanban)` Kanban lots. Do not approve or alter `PO-202607-00002` during smoke testing.

- [ ] **Step 4: Verify HTTP availability**

Run:

```powershell
(Invoke-WebRequest -UseBasicParsing http://localhost:3019/login).StatusCode
(Invoke-RestMethod -UseBasicParsing http://localhost:8091/health).status
```

Expected: `200` and `ok`.

- [ ] **Step 5: Record final repository state**

Run: `git status --short; git log -8 --oneline`

Expected: no unintended working-tree changes; task commits visible on local `master`.
