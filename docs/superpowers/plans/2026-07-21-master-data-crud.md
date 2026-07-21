# Master Data CRUD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver tenant-safe, audited, permission-controlled CRUD for Measurements, Suppliers, and Raw Materials with compact drawers and real PostgreSQL data.

**Architecture:** Extend the Go modular monolith with a master-data service/store/HTTP boundary and typed validation errors, backed by RLS transactions. The Next.js frontend consumes these APIs through the existing proxy and uses shared table/drawer primitives with domain-specific forms.

**Tech Stack:** Go 1.26, Gin, pgx, PostgreSQL 17 RLS, Next.js 16, React 19, TypeScript, Vitest, Testing Library, Docker Compose.

## Global Constraints

- Raw Material stores a base unit; Kanban is only a physical lot grouping.
- Records use activate/deactivate and are never hard-deleted.
- Read requires `master_data.view`; mutation requires `master_data.manage`.
- Currency is derived from the selected supplier and never trusted from the browser.
- Standard Unit Price is non-negative and becomes the future PO default snapshot.
- Every write records the authenticated user and tenant through RLS transaction context.
- UI remains compact and contains no fabricated records.

---

### Task 1: Database Price Migration and Master-Data Domain Contracts

**Files:**
- Create: `database/migrations/003_raw_material_price.sql`
- Modify: `backend/internal/masterdata/migration_test.go`
- Create: `backend/internal/masterdata/domain.go`
- Create: `backend/internal/masterdata/domain_test.go`

**Interfaces:**
- Produces: `FieldErrors`, `ListQuery`, `MeasurementInput`, `SupplierInput`, `RawMaterialInput`, normalization/validation methods, and additive price/currency columns.

- [ ] **Step 1: Write failing migration and domain tests**

```go
func TestMeasurementRejectsKanban(t *testing.T) {
  input := MeasurementInput{Code: " kanban ", Name: "Lot"}
  errs := input.NormalizeAndValidate()
  if errs["code"] == "" { t.Fatal("KANBAN was accepted") }
}

func TestRawMaterialRejectsInvalidQuantities(t *testing.T) {
  input := RawMaterialInput{QtyPerKanban: 0, MinimumStock: -1, StandardUnitPrice: -1}
  errs := input.NormalizeAndValidate()
  if len(errs) != 3 { t.Fatalf("got %#v", errs) }
}
```

Migration test must assert `standard_unit_price numeric(20,6)`, non-negative check, `currency char(3)`, and supplier-derived backfill SQL.

- [ ] **Step 2: Verify tests fail**

Run: `go test ./internal/masterdata -run 'TestMeasurementRejectsKanban|TestRawMaterialRejectsInvalidQuantities|TestMasterDataMigration' -v`
Expected: FAIL because the contracts and migration do not exist.

- [ ] **Step 3: Implement domain contracts and migration**

Normalize codes with `strings.ToUpper(strings.TrimSpace(...))`, validate email with `mail.ParseAddress`, enforce the supported currency set, clamp list limit defaults to 50/max 200, and add the additive SQL migration with supplier-currency backfill before `NOT NULL`.

- [ ] **Step 4: Verify focused and package tests**

Run: `go test ./internal/masterdata -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add database/migrations/003_raw_material_price.sql backend/internal/masterdata
git commit -m "feat: define master data CRUD contracts"
```

### Task 2: Measurement CRUD Service, Store, and API

**Files:**
- Create: `backend/internal/masterdata/errors.go`
- Create: `backend/internal/masterdata/store.go`
- Create: `backend/internal/masterdata/service.go`
- Create: `backend/internal/masterdata/service_test.go`
- Create: `backend/internal/masterdata/http.go`
- Create: `backend/internal/masterdata/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Produces: `Service.ListMeasurements`, `CreateMeasurement`, `GetMeasurement`, `UpdateMeasurement`; `RegisterRoutes(router, service, authMiddleware)`.
- Consumes: `database.WithTenant`, authenticated user/permission context, domain contracts from Task 1.

- [ ] **Step 1: Write failing service tests**

Use an in-memory fake store to prove normalization, field errors, audit user forwarding, not-found mapping, and activation updates:

```go
result, err := service.CreateMeasurement(ctx, Actor{TenantID: tenant, UserID: user}, MeasurementInput{Code: " kg ", Name: "Kilogram"})
if err != nil || result.Code != "KG" { t.Fatalf("%+v %v", result, err) }
```

- [ ] **Step 2: Verify service tests fail**

Run: `go test ./internal/masterdata -run TestServiceMeasurement -v`
Expected: FAIL because service/store interfaces do not exist.

- [ ] **Step 3: Implement measurement store and service**

Use tenant-scoped SQL for list/search/active pagination, insert with audit IDs, get, and update. Map PostgreSQL unique violation `23505` to a `ConflictError` field `code`.

- [ ] **Step 4: Write and run failing API tests**

Test GET/POST/PATCH success, 400 field shape, 403 permission denial, 404, and 409 using a fake service. Run `go test ./internal/masterdata -run TestHTTPMeasurement -v`; expected FAIL before route implementation.

- [ ] **Step 5: Implement routes and verify**

Register `/master-data/measurements` routes, bind actor from auth context, enforce RBAC middleware, return list shape `{ "items": [], "total": 0 }`, and stable error JSON. Run `go test ./...`; expected PASS.

- [ ] **Step 6: Commit**

```powershell
git add backend
git commit -m "feat: add measurement CRUD API"
```

### Task 3: Supplier CRUD Service, Store, and API

**Files:**
- Modify: `backend/internal/masterdata/store.go`
- Modify: `backend/internal/masterdata/service.go`
- Modify: `backend/internal/masterdata/service_test.go`
- Modify: `backend/internal/masterdata/http.go`
- Modify: `backend/internal/masterdata/http_test.go`

**Interfaces:**
- Produces: supplier list/create/get/update methods using the same actor, query, response, and error conventions as Measurements.

- [ ] **Step 1: Write failing supplier service tests**

Prove uppercase Supplier ID, supported currency validation, email validation, unique code/Sage-code conflicts, audit forwarding, and deactivate behavior.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/masterdata -run TestServiceSupplier -v`
Expected: FAIL because supplier methods are absent.

- [ ] **Step 3: Implement supplier service/store methods**

Add searchable/paginated list, tenant-scoped insert/get/update, and unique-conflict mapping for `code` and `sage_supplier_code`.

- [ ] **Step 4: Write failing HTTP tests, implement routes, verify green**

Cover `GET/POST /master-data/suppliers` and `GET/PATCH /master-data/suppliers/:id` with success and stable error responses. Run `go test ./...`; expected PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/masterdata
git commit -m "feat: add supplier CRUD API"
```

### Task 4: Raw Material CRUD and Currency Derivation

**Files:**
- Modify: `backend/internal/masterdata/store.go`
- Modify: `backend/internal/masterdata/service.go`
- Modify: `backend/internal/masterdata/service_test.go`
- Modify: `backend/internal/masterdata/http.go`
- Modify: `backend/internal/masterdata/http_test.go`

**Interfaces:**
- Produces: Raw Material list/create/get/update, supplier/unit option endpoints, supplier-derived currency within the write transaction.

- [ ] **Step 1: Write failing raw-material tests**

Prove invalid quantities fail, inactive/missing supplier or unit conflicts, input currency is ignored, store result currency equals supplier currency, and list supplier filtering is forwarded.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/masterdata -run TestServiceRawMaterial -v`
Expected: FAIL because raw-material CRUD methods are absent.

- [ ] **Step 3: Implement atomic raw-material writes**

Inside one tenant transaction select active supplier currency and active measurement, then insert/update Raw Material with derived currency and audit user. Add joined list/get responses with supplier name and unit code.

- [ ] **Step 4: Add HTTP tests and routes**

Cover list filtering, create, edit/deactivate, invalid numeric fields, inactive references, unique conflicts, and response currency. Run `go test ./...; go vet ./...`; expected PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/masterdata
git commit -m "feat: add raw material CRUD API"
```

### Task 5: Reusable Frontend CRUD Shell and Collapsible Groups

**Files:**
- Modify: `frontend/components/app-shell/navigation.ts`
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Create: `frontend/components/master-data/data-table.tsx`
- Create: `frontend/components/master-data/form-drawer.tsx`
- Create: `frontend/components/master-data/api.ts`
- Create: `frontend/components/master-data/master-data-crud.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces: reusable `DataTable`, `FormDrawer`, typed `apiRequest`, and persisted group-state behavior.

- [ ] **Step 1: Write failing navigation tests**

Verify Data Master, Procurement, Logistics, and Settings each expose an `aria-expanded` group button, active groups start open, manual state persists in `order-stock.nav-groups`, and Dashboard/Reports stay direct links.

- [ ] **Step 2: Verify red, implement group state, verify green**

Run focused AppShell tests before and after implementation. Expected initial failure followed by PASS.

- [ ] **Step 3: Write failing shared CRUD tests**

Prove create opens the drawer, Escape closes when clean, server field errors remain visible, deactivate requires confirmation, and successful save triggers reload.

- [ ] **Step 4: Implement shared table/drawer/API primitives**

Use compact semantic tables, accessible dialog markup, field-error mapping, abortable list requests, and the existing `/api` proxy.

- [ ] **Step 5: Verify shared tests and commit**

Run: `npm test -- --run components/app-shell/app-shell.test.tsx components/master-data/master-data-crud.test.tsx`
Expected: PASS.

```powershell
git add frontend/components frontend/app/globals.css
git commit -m "feat: add reusable master data CRUD UI"
```

### Task 6: Measurement, Supplier, and Raw Material Pages

**Files:**
- Create: `frontend/components/master-data/measurement-crud.tsx`
- Create: `frontend/components/master-data/supplier-crud.tsx`
- Create: `frontend/components/master-data/raw-material-crud.tsx`
- Create: `frontend/components/master-data/domain-crud.test.tsx`
- Modify: `frontend/app/measurements/page.tsx`
- Modify: `frontend/app/suppliers/page.tsx`
- Modify: `frontend/app/raw-materials/page.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: Task 5 primitives and backend endpoints from Tasks 2–4.
- Produces: fully active Create/Edit/Deactivate flows for all three master pages.

- [ ] **Step 1: Write failing domain UI tests**

Test Measurement KANBAN error, Supplier required/currency fields, Raw Material searchable supplier/unit controls, read-only derived currency, Qty per Kanban validation, and Created/Updated audit columns.

- [ ] **Step 2: Verify red**

Run: `npm test -- --run components/master-data/domain-crud.test.tsx`
Expected: FAIL because domain components do not exist.

- [ ] **Step 3: Implement domain pages and forms**

Connect real APIs, activate Create buttons, render compact rows/actions, implement edit/deactivate, and refresh filtered lists after writes.

- [ ] **Step 4: Run full frontend verification**

Run: `npm test; npm run lint; npm run typecheck; npm run build`
Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend
git commit -m "feat: activate master data CRUD pages"
```

### Task 7: PostgreSQL, Docker, and Browser CRUD Verification

**Files:**
- Modify only if verification identifies a tested runtime defect.

**Interfaces:**
- Consumes: Docker services frontend `3019`, backend `8091`, PostgreSQL `5445`.
- Produces: verified local CRUD behavior with persistent database records.

- [ ] **Step 1: Rebuild the Docker stack and apply migration**

Run `docker compose -p platform-master-po up -d --build`, apply migration 003 idempotently to the existing volume, and verify PostgreSQL healthy.

- [ ] **Step 2: Run authenticated API smoke flow**

Create a Measurement, create a Supplier, create a Raw Material using both IDs, edit each, deactivate the Raw Material, list/search/filter records, and verify Created/Updated By values.

- [ ] **Step 3: Verify tenant and validation behavior**

Attempt duplicate codes, KANBAN measurement, negative price, and an inaccessible tenant record. Expected: stable 400/409 responses and no cross-tenant data.

- [ ] **Step 4: Verify browser routes and runtime health**

GET all three pages through port 3019, verify auth proxy, containers, database health, disk free space, and clean Git status.

- [ ] **Step 5: Run final verification**

Run backend tests/vet and frontend tests/lint/typecheck/build from the final Docker image. Expected: all PASS.
