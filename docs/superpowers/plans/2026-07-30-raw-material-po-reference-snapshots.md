# Raw Material and PO Reference Snapshots Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Attach Category and Packing masters to Raw Materials, select one destination Plant per PO, and persist stable snapshots through PO lists and document data.

**Architecture:** A forward migration adds nullable master relations for legacy rows and snapshot columns for transaction history. Master-data CRUD requires active Category/Packing for every new or edited material. PO draft input requires one active Plant and snapshots Plant plus line Category/Packing values so later master changes cannot alter history.

**Tech Stack:** PostgreSQL migrations/RLS, Go 1.24 with Gin/pgx, Next.js/React/TypeScript, Go tests, Vitest.

## Global Constraints

- Existing migrations remain immutable; schema changes use migration 013.
- One PO has exactly one destination Plant.
- Category and Packing are selected only on Raw Material, never independently edited on PO.
- Existing legacy material and PO rows remain readable.
- New/edited Raw Materials require active Category and Packing.
- Draft PO save and submit reject missing/inactive Plant or missing material Category/Packing.
- Category, Packing, and Plant names/codes are snapshotted into transactions.
- PDF layout changes remain Phase 3; Phase 2 supplies the fields to document loaders.

---

### Task 1: Migration 013 reference relations and snapshots

**Files:**
- Create: `database/migrations/013_material_po_reference_snapshots.sql`
- Modify: `backend/internal/masterdata/migration_test.go`
- Modify: `backend/internal/purchaseorder/migration_test.go`

**Interfaces:**
- Produces: `raw_materials.category_id`, `raw_materials.packing_id`
- Produces: `purchase_orders.plant_id`, `plant_code_snapshot`, `plant_name_snapshot`, `plant_address_snapshot`
- Produces: PO-line Category/Packing snapshot columns

- [ ] Add failing contract tests for all columns, tenant-scoped foreign keys, indexes, and grants.

```go
for _, fragment := range []string{
	"ADD COLUMN category_id uuid",
	"ADD COLUMN packing_id uuid",
	"ADD COLUMN plant_id uuid",
	"category_code_snapshot text",
	"packing_code_snapshot text",
	"plant_code_snapshot text",
} {
	if !strings.Contains(sql, strings.ToLower(fragment)) {
		t.Errorf("migration missing %q", fragment)
	}
}
```

- [ ] Run `go test ./internal/masterdata ./internal/purchaseorder` and confirm migration 013 is missing.
- [ ] Create migration 013 with nullable relations for legacy rows, non-null snapshot text columns defaulting to `''`, composite tenant foreign keys, and lookup indexes.

```sql
ALTER TABLE raw_materials
  ADD COLUMN category_id uuid,
  ADD COLUMN packing_id uuid;
ALTER TABLE purchase_orders
  ADD COLUMN plant_id uuid,
  ADD COLUMN plant_code_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN plant_name_snapshot text NOT NULL DEFAULT '',
  ADD COLUMN plant_address_snapshot text NOT NULL DEFAULT '';
```

- [ ] Run focused tests and commit:

```powershell
git add database/migrations/013_material_po_reference_snapshots.sql backend/internal/masterdata/migration_test.go backend/internal/purchaseorder/migration_test.go
git commit -m "feat: add material and PO reference snapshots"
```

### Task 2: Raw Material Category/Packing backend and UI

**Files:**
- Modify: `backend/internal/masterdata/models.go`
- Modify: `backend/internal/masterdata/domain.go`
- Modify: `backend/internal/masterdata/store.go`
- Modify: `backend/internal/masterdata/domain_test.go`
- Modify: `backend/internal/masterdata/service_test.go`
- Modify: `frontend/app/raw-materials/page.tsx`
- Modify: `frontend/components/master-data-crud.test.tsx`

**Interfaces:**
- Adds `categoryId`, `categoryCode`, `categoryName`, `packingId`, `packingCode`, `packingName` to Raw Material JSON.
- Adds required `categoryId` and `packingId` to `RawMaterialInput`.
- Consumes active `/master-data/categories`, `/packings`, `/units`, and `/suppliers`.

- [ ] Write failing Go validation/store-contract tests and a frontend test asserting four active option requests.

```go
input := RawMaterialInput{}
fields := input.NormalizeAndValidate()
if fields["categoryId"] == "" || fields["packingId"] == "" {
	t.Fatalf("missing reference validation: %#v", fields)
}
```

- [ ] Confirm tests fail because Category/Packing fields are absent and Raw Material still calls `/measurements`.
- [ ] Extend Raw Material domain and SQL queries; create/update succeeds only when Supplier, Unit, Category, and Packing are active in the same tenant.

```go
type RawMaterialInput struct {
	CategoryID uuid.UUID `json:"categoryId"`
	PackingID  uuid.UUID `json:"packingId"`
}
```

- [ ] Update Raw Material form/list with required Category/Packing selects and correct `/master-data/units` endpoint.

```tsx
{ key: 'categoryId', label: 'Category', type: 'select', required: true }
{ key: 'packingId', label: 'Packing', type: 'select', required: true }
```

- [ ] Run `go test ./internal/masterdata` and focused frontend tests, then commit:

```powershell
git add backend/internal/masterdata frontend/app/raw-materials/page.tsx frontend/components/master-data-crud.test.tsx
git commit -m "feat: attach category and packing to raw materials"
```

### Task 3: PO Plant and immutable snapshots backend

**Files:**
- Modify: `backend/internal/purchaseorder/domain.go`
- Modify: `backend/internal/purchaseorder/store.go`
- Modify: `backend/internal/purchaseorder/domain_test.go`
- Modify: `backend/internal/purchaseorder/service_test.go`
- Modify: `backend/internal/purchaseorder/store_test.go`
- Modify: `backend/internal/purchaseorder/sql_store_integration_test.go`
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Adds `PlantID uuid.UUID` to `OrderInput`.
- Adds Plant ID/code/name/address fields to `Order`.
- Adds Category/Packing ID/code/name snapshot fields to `OrderLine`.

- [ ] Write failing tests for required Plant, active Plant lookup, line snapshot capture, stable reads, and submit rejection of incomplete legacy data.

```go
input := validOrderInput()
input.PlantID = uuid.Nil
if fields := input.NormalizeAndValidate(true); fields["plantId"] == "" {
	t.Fatalf("Plant must be required: %#v", fields)
}
```

- [ ] Confirm focused tests fail for missing fields and SQL fragments.
- [ ] Extend create/update/header scans with active Plant snapshot lookup.

```go
type Order struct {
	PlantID              uuid.UUID `json:"plantId"`
	PlantCodeSnapshot    string    `json:"plantCode"`
	PlantNameSnapshot    string    `json:"plantName"`
	PlantAddressSnapshot string    `json:"plantAddress"`
}
```

- [ ] Extend line snapshot and insert/scan paths with active Category/Packing joins.

```go
type OrderLine struct {
	CategoryCodeSnapshot string `json:"categoryCode"`
	CategoryNameSnapshot string `json:"categoryName"`
	PackingCodeSnapshot  string `json:"packingCode"`
	PackingNameSnapshot  string `json:"packingName"`
}
```

- [ ] Extend stored-order submission validation to require valid Plant and non-empty Category/Packing snapshots while preserving historical non-draft reads.
- [ ] Expose the new snapshot fields through document data structs without redesigning PDF layout.
- [ ] Run `go test ./internal/purchaseorder` and commit:

```powershell
git add backend/internal/purchaseorder
git commit -m "feat: snapshot plant category and packing on purchase orders"
```

### Task 4: PO form and transaction lists

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/supplier-orders/supplier-order-config.ts`
- Modify: `frontend/app/supplier-orders/supplier-order-config.test.ts`

**Interfaces:**
- Consumes active `/master-data/plants`.
- Sends `plantId` in PO create/update payload.
- Displays Plant on PO form/list and Category/Packing on selected material rows.

- [ ] Write failing form/list tests for Plant loading, required validation, payload, hydration, read-only detail, and snapshot columns.

```tsx
expect(fetchMock).toHaveBeenCalledWith('/api/master-data/plants?active=true&limit=200', expect.anything());
expect(JSON.parse(String(request.body))).toMatchObject({ plantId: 'plant-1' });
```

- [ ] Confirm focused Vitest tests fail for absent Plant and snapshot UI.
- [ ] Add Plant selector to editable PO header and read-only Plant output after locking.

```tsx
<select aria-label="Plant" value={plantId} disabled={!editable} onChange={event => setPlantId(event.target.value)}>
  <option value="">Select Plant...</option>
  {plants.map(plant => <option key={plant.id} value={plant.id}>{plant.code} — {plant.name}</option>)}
</select>
```

- [ ] Add Category/Packing columns to material rows and Plant column to Supplier Orders list.

```tsx
<td>{line.categoryCode} — {line.categoryName}</td>
<td>{line.packingCode} — {line.packingName}</td>
```
- [ ] Run focused tests, `npm test -- --run`, and `npm run build`.
- [ ] Commit:

```powershell
git add frontend/components/supplier-orders frontend/app/supplier-orders
git commit -m "feat: add plant and material snapshots to supplier orders"
```

### Task 5: Full verification and integration

- [ ] Run `go test ./...` in `backend`.
- [ ] Run `npm test -- --run` in `frontend`.
- [ ] Run `npm run build` in `frontend`.
- [ ] Run `git diff --check` and confirm the feature worktree is clean.
- [ ] Merge the feature branch into `main`, repeat all three verification commands on `main`, and push `origin main`.
