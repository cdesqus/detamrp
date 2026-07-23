# Stock Inventory Overview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only Stock Inventory module that includes zero-stock materials and exposes each material's currently available Kanban lots.

**Architecture:** Add a focused Go `inventory` package with tenant-scoped read models and HTTP routes, then add a compact Next.js inventory page consuming those endpoints. Current balances are calculated from `kanban_lots.status = 'IN_STOCK'`; the append-only ledger remains the audit source and no duplicate balance table is introduced.

**Tech Stack:** Go 1.26, Gin, pgx/PostgreSQL 17 with RLS, shopspring/decimal, Next.js 16, React 19, TypeScript, Vitest, Testing Library.

## Global Constraints

- Every active raw material must appear, including materials with zero stock.
- Only Kanban lots with status `IN_STOCK` contribute to available stock.
- The module is read-only and protected by `inventory.view`.
- Stock Taking, Stock Adjustment, Stock Moving, and barcode reprinting are excluded.
- UI must remain compact and use the shared number formatter.
- Existing tenant scoping and PostgreSQL row-level security remain mandatory.

---

### Task 1: Inventory domain and stock classification

**Files:**
- Create: `backend/internal/inventory/domain.go`
- Create: `backend/internal/inventory/domain_test.go`

**Interfaces:**
- Produces: `StockStatus(quantity, minimum decimal.Decimal) string`
- Produces: `Actor`, `Summary`, `StockItem`, `StockResponse`, `KanbanItem`, and `KanbanResponse` JSON read models.

- [ ] **Step 1: Write failing classification tests**

Add table-driven tests proving zero is `OUT_OF_STOCK`, positive quantity at or below minimum is `LOW_STOCK`, and quantity above minimum is `IN_STOCK`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/inventory -run TestStockStatus -count=1`

Expected: FAIL because the package and `StockStatus` do not exist.

- [ ] **Step 3: Implement the minimal domain**

Create the read models from the approved API contract and implement:

```go
func StockStatus(quantity, minimum decimal.Decimal) string {
    if quantity.IsZero() {
        return "OUT_OF_STOCK"
    }
    if quantity.LessThanOrEqual(minimum) {
        return "LOW_STOCK"
    }
    return "IN_STOCK"
}
```

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/inventory -run TestStockStatus -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/inventory/domain.go backend/internal/inventory/domain_test.go
git commit -m "feat: define inventory stock read models"
```

### Task 2: Tenant-scoped inventory queries

**Files:**
- Create: `backend/internal/inventory/store.go`
- Create: `backend/internal/inventory/store_integration_test.go`

**Interfaces:**
- Consumes: Task 1 read models and `StockStatus`.
- Produces: `NewStore(*pgxpool.Pool) *Store`
- Produces: `ListStock(context.Context, Actor, Filters) (StockResponse, error)`
- Produces: `ListKanbans(context.Context, Actor, uuid.UUID) (KanbanResponse, error)`

- [ ] **Step 1: Write failing integration tests**

Create tenant fixtures with active materials covering zero stock, low stock, in stock, consumed lots, and another tenant. Assert:

- zero-stock materials remain in results;
- only `IN_STOCK` lots affect counts and quantities;
- inactive and cross-tenant materials are absent;
- search, supplier, and status filters affect items;
- summary values remain global and unfiltered;
- Kanban details include DN, PO, quantity, unit, and receiving date;
- consumed lots are excluded;
- unknown or inactive material returns `ErrNotFound`.

- [ ] **Step 2: Verify RED**

Run: `$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'; go test ./internal/inventory -run Integration -count=1`

Expected: FAIL because `Store` is missing.

- [ ] **Step 3: Implement overview and detail queries**

Use the project database transaction helper so `app.tenant_id` is set. Start the overview from active `raw_materials`, join suppliers and measurements, then left-join an aggregate of `IN_STOCK` Kanban lots through purchase-order lines. Apply item filters outside the unfiltered summary query.

For details, validate the active material first and query only `IN_STOCK` lots, joining delivery notes, purchase orders, and the completed receiving record to obtain `receivedDate`. Always return `kanbans: []` instead of `null`.

- [ ] **Step 4: Verify GREEN**

Run: `$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'; go test ./internal/inventory -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/inventory/store.go backend/internal/inventory/store_integration_test.go
git commit -m "feat: query current raw material inventory"
```

### Task 3: Inventory HTTP endpoints and server registration

**Files:**
- Create: `backend/internal/inventory/http.go`
- Create: `backend/internal/inventory/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/internal/api/server_test.go`
- Modify: `cmd/api/main.go`

**Interfaces:**
- Consumes: `Store.ListStock` and `Store.ListKanbans`.
- Produces: `GET /inventory/stock`
- Produces: `GET /inventory/stock/:rawMaterialId/kanbans`

- [ ] **Step 1: Write failing HTTP tests**

Test authenticated access with `inventory.view`, `403` without permission, `400` for malformed `supplierId` or unsupported status, `400` for malformed material ID, and `404` for an unknown material.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/inventory ./internal/api -count=1`

Expected: FAIL because routes are not registered.

- [ ] **Step 3: Implement and register routes**

Normalize status to uppercase and accept only `IN_STOCK`, `LOW_STOCK`, and `OUT_OF_STOCK`. Register the inventory store through the existing server configuration and initialize it in `cmd/api/main.go`.

- [ ] **Step 4: Verify GREEN**

Run: `$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'; go test ./internal/inventory ./internal/api -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/inventory backend/internal/api cmd/api/main.go
git commit -m "feat: expose stock inventory endpoints"
```

### Task 4: Inventory navigation and page UI

**Files:**
- Create: `frontend/app/inventory/page.tsx`
- Create: `frontend/components/inventory/inventory-index.tsx`
- Create: `frontend/components/inventory/inventory-index.test.tsx`
- Modify: `frontend/components/app-shell/navigation.ts`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `GET /api/inventory/stock` and `GET /api/inventory/stock/:rawMaterialId/kanbans`.
- Consumes: `formatQuantity` from `frontend/lib/number-format.ts`.
- Produces: `/inventory` UI and Logistics navigation link.

- [ ] **Step 1: Write failing navigation and component tests**

Assert `Stock Inventory` appears before Receiving. Mock overview and detail responses and test:

- all four summary values render;
- zero-stock rows render;
- quantities have no redundant trailing zeroes;
- status pills use green, amber, and red variants;
- filters produce the expected API query;
- `Open` loads a centered modal;
- empty Kanban details show `No in-stock Kanban available`;
- overview and detail failures show English error states.

- [ ] **Step 2: Verify RED**

Run: `npx vitest run components/app-shell/app-shell.test.tsx components/inventory/inventory-index.test.tsx --maxWorkers=1`

Expected: FAIL because the route, navigation item, and component do not exist.

- [ ] **Step 3: Implement the compact inventory page**

Add the navigation item with the existing package icon. Build summary cards, search and native selects, the compact overview table, colored status pills, and the centered read-only Kanban modal. Fetch detail data only when `Open` is clicked and preserve the overview while the modal loads.

- [ ] **Step 4: Verify GREEN**

Run: `npx vitest run components/app-shell/app-shell.test.tsx components/inventory/inventory-index.test.tsx --maxWorkers=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/app/inventory frontend/components/inventory frontend/components/app-shell frontend/app/globals.css
git commit -m "feat: add stock inventory overview"
```

### Task 5: Full verification and local runtime

**Files:**
- Verify only; no production changes expected.

**Interfaces:**
- Consumes the complete feature from Tasks 1–4.
- Produces a merge-ready branch and a running local build.

- [ ] **Step 1: Run backend verification**

Run:

```powershell
$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'
go test ./... -count=1
go vet ./...
```

Expected: all packages PASS and vet exits `0`.

- [ ] **Step 2: Run frontend verification**

Run:

```powershell
npx vitest run --maxWorkers=1
npm run typecheck
npm run build
```

Expected: all tests PASS, TypeScript exits `0`, and the Next.js production build succeeds.

- [ ] **Step 3: Check repository hygiene**

Run: `git diff --check`

Expected: no output and exit code `0`.

- [ ] **Step 4: Rebuild local Docker services after merge**

Run:

```powershell
docker compose -p platform-master-po up -d --build
docker compose -p platform-master-po ps
```

Expected: PostgreSQL is healthy and backend/frontend are running on ports `5445`, `8091`, and `3019`.

- [ ] **Step 5: Smoke test**

Verify `GET http://localhost:8091/health` and `GET http://localhost:3019/inventory` return HTTP `200`, then manually confirm the zero-stock row and Kanban detail modal using an account with `inventory.view`.
