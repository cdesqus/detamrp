# Unit, Category, Packing, and Plant Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the existing Measurements master to Unit and add tenant-isolated Category, Packing, and Plant CRUD modules with the approved nested navigation.

**Architecture:** A forward PostgreSQL migration renames `measurements` to `units` and introduces three reference tables. The Go master-data package exposes focused services over the shared SQL store, while Next.js reuses `MasterDataCrud` for four pages and extends navigation to one nested submenu level.

**Tech Stack:** PostgreSQL migrations and RLS, Go 1.24 with Gin/pgx, Next.js 15 with React 19 and TypeScript, Go tests, Vitest.

## Global Constraints

- Existing migrations are immutable; all schema work goes into `database/migrations/012_master_reference_data.sql`.
- Existing tenant isolation and `master_data.view` / `master_data.manage` permissions remain mandatory.
- Measurements becomes Unit throughout the database, backend API, frontend route, copy, and tests.
- Category and Packing fields are code, name, description, active, and existing audit metadata.
- Plant fields are code, name, address, active, and existing audit metadata.
- Deactivation replaces hard deletion.
- No legacy `/master-data/measurements` endpoint is retained.
- This plan does not yet attach Category/Packing to Raw Material or Plant to PO; those belong to the next independently testable vertical slice.

---

### Task 1: Forward database migration

**Files:**
- Create: `database/migrations/012_master_reference_data.sql`
- Modify: `backend/internal/masterdata/migration_test.go`
- Test: `backend/internal/database/migration_runner_test.go`

**Interfaces:**
- Produces: tables `units`, `categories`, `packings`, and `plants`
- Produces: tenant-scoped unique `(tenant_id, code)` constraints and RLS policies
- Consumes: `tenants`, `users`, and the existing `measurements` table

- [ ] **Step 1: Write the failing migration contract tests**

Add assertions that migration 012 contains the Unit rename, all three new tables, RLS enable/force/policies, audit foreign keys, unique tenant/code constraints, and grants:

```go
func TestMasterReferenceMigrationDefinesUnitsCategoriesPackingsAndPlants(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "database", "migrations", "012_master_reference_data.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(content)
	required := []string{
		"ALTER TABLE measurements RENAME TO units",
		"CREATE TABLE categories",
		"CREATE TABLE packings",
		"CREATE TABLE plants",
		"ALTER TABLE categories ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE packings ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE plants ENABLE ROW LEVEL SECURITY",
		"UNIQUE (tenant_id, code)",
		"GRANT SELECT, INSERT, UPDATE ON categories, packings, plants TO nextgen_app",
	}
	for _, fragment := range required {
		if !strings.Contains(sql, fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run: `go test ./internal/masterdata ./internal/database`

Expected: FAIL because migration 012 does not exist.

- [ ] **Step 3: Add the forward migration**

Create `012_master_reference_data.sql` with:

```sql
ALTER TABLE measurements RENAME TO units;
ALTER POLICY measurements_isolation ON units RENAME TO units_isolation;

CREATE TABLE categories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  code text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  active boolean NOT NULL DEFAULT true,
  created_by_user_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_by_user_id uuid NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, id),
  UNIQUE (tenant_id, code),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, id),
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, id)
);
```

Repeat the same audit/tenant structure for `packings`; use `address text NOT NULL DEFAULT ''` instead of description for `plants`. Enable and force RLS, add `tenant_id = current_setting('app.tenant_id', true)::uuid` policies, and grant `SELECT, INSERT, UPDATE` to `nextgen_app`.

- [ ] **Step 4: Run migration tests**

Run: `go test ./internal/masterdata ./internal/database`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add database/migrations/012_master_reference_data.sql backend/internal/masterdata/migration_test.go
git commit -m "feat: add unit and reference master migration"
```

### Task 2: Rename Measurement backend contract to Unit

**Files:**
- Modify: `backend/internal/masterdata/models.go`
- Modify: `backend/internal/masterdata/domain.go`
- Modify: `backend/internal/masterdata/service.go`
- Modify: `backend/internal/masterdata/store.go`
- Modify: `backend/internal/masterdata/http.go`
- Modify: `backend/internal/masterdata/models_test.go`
- Modify: `backend/internal/masterdata/domain_test.go`
- Modify: `backend/internal/masterdata/service_test.go`
- Modify: `backend/internal/masterdata/http_test.go`
- Modify: `backend/internal/masterdata/store_test.go`

**Interfaces:**
- Produces: `Unit`, `UnitInput`, `UnitRepository`, `UnitService`, `NewUnitService`
- Produces: `GET|POST /master-data/units` and `GET|PUT /master-data/units/:id`
- Consumes: the renamed `units` database table

- [ ] **Step 1: Rename tests and expected API paths first**

Change test fixtures and contracts to use:

```go
input := UnitInput{Code: "KG", Name: "Kilogram", DecimalAllowed: true}
service := NewUnitService(repo)
request := httptest.NewRequest(http.MethodGet, "/master-data/units?active=true", nil)
```

Assert `KANBAN` remains invalid with the message `KANBAN is a physical lot, not a base unit`.

- [ ] **Step 2: Run focused tests and verify compilation/test failures**

Run: `go test ./internal/masterdata`

Expected: FAIL because Unit types and routes are not defined.

- [ ] **Step 3: Rename production types and methods**

Apply the complete semantic rename:

```go
type Unit struct {
	ID             uuid.UUID `json:"id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	DecimalAllowed bool      `json:"decimalAllowed"`
	Active         bool      `json:"active"`
	Audit
}

type UnitInput struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DecimalAllowed bool   `json:"decimalAllowed"`
	Active         *bool  `json:"active,omitempty"`
}
```

Rename repository/service/store methods from `Measurement` to `Unit`, query `units u`, return `NotFoundError{Resource: "unit"}`, and register only `/master-data/units`.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/masterdata`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/masterdata
git commit -m "refactor: rename measurement master to unit"
```

### Task 3: Add Category, Packing, and Plant backend CRUD

**Files:**
- Modify: `backend/internal/masterdata/models.go`
- Modify: `backend/internal/masterdata/domain.go`
- Modify: `backend/internal/masterdata/service.go`
- Modify: `backend/internal/masterdata/store.go`
- Modify: `backend/internal/masterdata/http.go`
- Modify: `backend/internal/masterdata/models_test.go`
- Modify: `backend/internal/masterdata/domain_test.go`
- Modify: `backend/internal/masterdata/service_test.go`
- Modify: `backend/internal/masterdata/http_test.go`
- Modify: `backend/internal/masterdata/store_test.go`

**Interfaces:**
- Produces: `Category`, `Packing`, `Plant` and matching input/service/repository contracts
- Produces: `/master-data/categories`, `/master-data/packings`, `/master-data/plants`
- Consumes: common `ListQuery`, `Audit`, `activeValue`, `writeError`, and master-data permissions

- [ ] **Step 1: Add failing validation and HTTP contract tests**

Cover normalization, required fields, list envelopes, create/update status codes, permission denial, duplicate-code conflicts, inactive filters, and tenant-safe actor forwarding:

```go
input := CategoryInput{Code: " chemical ", Name: " Chemical ", Description: "  Controlled material "}
if fields := input.NormalizeAndValidate(); len(fields) != 0 {
	t.Fatalf("unexpected fields: %#v", fields)
}
if input.Code != "CHEMICAL" || input.Name != "Chemical" || input.Description != "Controlled material" {
	t.Fatalf("input was not normalized: %#v", input)
}
```

Use equivalent tests for Packing and Plant; Plant validates `code` and `name` and trims `address`.

- [ ] **Step 2: Run focused tests and verify failure**

Run: `go test ./internal/masterdata`

Expected: FAIL because the new master types and services do not exist.

- [ ] **Step 3: Implement focused domain/service contracts**

Add:

```go
type Category struct { ID uuid.UUID; Code, Name, Description string; Active bool; Audit }
type Packing struct { ID uuid.UUID; Code, Name, Description string; Active bool; Audit }
type Plant struct { ID uuid.UUID; Code, Name, Address string; Active bool; Audit }
```

Give every JSON field explicit camelCase tags. Create corresponding inputs whose `NormalizeAndValidate` uppercases code, trims text, and requires code/name. Add repository interfaces and services with `List`, `Get`, `Create`, and `Update`.

- [ ] **Step 4: Implement SQL store and routes**

Use the existing list pattern:

```sql
WHERE x.tenant_id=$1
  AND ($2='' OR x.code ILIKE '%'||$2||'%' OR x.name ILIKE '%'||$2||'%')
  AND ($3::boolean IS NULL OR x.active=$3)
ORDER BY x.code
LIMIT $4 OFFSET $5
```

Translate PostgreSQL `23505` to `ConflictError{Fields: FieldErrors{"code": "Already in use"}}`. Register GET/POST/PUT endpoints with `master_data.view` and `master_data.manage`.

- [ ] **Step 5: Run master-data tests**

Run: `go test ./internal/masterdata`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add backend/internal/masterdata
git commit -m "feat: add category packing and plant services"
```

### Task 4: Wire services into the application server

**Files:**
- Modify: `backend/internal/api/server.go`
- Modify: `backend/internal/api/server_test.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `*masterdata.UnitService`, `*masterdata.CategoryService`, `*masterdata.PackingService`, `*masterdata.PlantService`
- Produces: `WithUnitService`, `WithCategoryService`, `WithPackingService`, `WithPlantService`

- [ ] **Step 1: Add failing server wiring tests**

Build a server with fake authentication and each service option, then assert authenticated requests reach `/master-data/units`, `/categories`, `/packings`, and `/plants` instead of returning route-level 404.

- [ ] **Step 2: Run API tests and verify failure**

Run: `go test ./internal/api`

Expected: FAIL because the server options are absent.

- [ ] **Step 3: Add server options and main wiring**

Replace `measurementService` with `unitService`, add the three new service fields/options, register their routes when non-nil, and initialize all four services from the existing shared `masterDataStore` in `cmd/api/main.go`.

- [ ] **Step 4: Run backend tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/api backend/cmd/api/main.go
git commit -m "feat: expose reference master APIs"
```

### Task 5: Add nested master-data navigation and CRUD pages

**Files:**
- Modify: `frontend/components/app-shell/navigation.ts`
- Modify: `frontend/components/app-shell/navigation.test.ts`
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Delete: `frontend/app/measurements/page.tsx`
- Create: `frontend/app/units/page.tsx`
- Create: `frontend/app/categories/page.tsx`
- Create: `frontend/app/packings/page.tsx`
- Create: `frontend/app/plants/page.tsx`
- Modify: `frontend/components/master-data-crud.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `/master-data/units`, `/categories`, `/packings`, `/plants`
- Produces: routes `/units`, `/categories`, `/packings`, `/plants`
- Produces: nested `Measurements` navigation parent containing Unit, Category, Packing

- [ ] **Step 1: Write failing navigation and page tests**

Assert the visible tree is:

```ts
{
  label: 'Data Master',
  items: [
    {
      label: 'Measurements',
      items: [
        { label: 'Unit', href: '/units' },
        { label: 'Category', href: '/categories' },
        { label: 'Packing', href: '/packings' }
      ]
    },
    { label: 'Plants', href: '/plants' },
    { label: 'Suppliers', href: '/suppliers' },
    { label: 'Raw Materials', href: '/raw-materials' }
  ]
}
```

Also assert all four paths require `master_data.view` and nested items disappear when permission is absent.

- [ ] **Step 2: Run frontend tests and verify failure**

Run: `npm test -- --run`

Working directory: `frontend`

Expected: FAIL because nested navigation and new pages are absent.

- [ ] **Step 3: Extend navigation types and rendering**

Add an explicit nested node type instead of overloading leaf items:

```ts
export type NavigationLeaf = {
  label: string;
  href: string;
  icon: IconName;
  requiredPermission: string;
};

export type NavigationNode = NavigationLeaf | {
  label: string;
  icon: IconName;
  requiredPermission: string;
  items: NavigationLeaf[];
};
```

Update permission filtering, first-route selection, active state, keyboard-accessible expand/collapse rendering, and CSS indentation for one nested level only.

- [ ] **Step 4: Add the four CRUD pages**

Use `MasterDataCrud` with these endpoint/field contracts:

```tsx
<MasterDataCrud
  title="Units"
  endpoint="/master-data/units"
  initial={{ code: '', name: '', decimalAllowed: false, active: true }}
  fields={[
    { key: 'code', label: 'Code', required: true },
    { key: 'name', label: 'Name', required: true },
    { key: 'decimalAllowed', label: 'Allow Decimal', type: 'checkbox' },
    { key: 'active', label: 'Active', type: 'checkbox' }
  ]}
/>
```

Category and Packing use code, name, description textarea, and active. Plant uses code, name, address textarea, and active. Delete the old `/measurements` page.

- [ ] **Step 5: Run frontend checks**

Run: `npm test -- --run`

Working directory: `frontend`

Expected: PASS.

Run: `npm run build`

Working directory: `frontend`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add frontend
git commit -m "feat: add nested measurement master navigation"
```

### Task 6: Full foundation verification

**Files:**
- Modify only files required to fix regressions found by the commands below

**Interfaces:**
- Produces: a deployable first vertical slice with no legacy Measurements UI/API

- [ ] **Step 1: Verify no stale user-facing or runtime Measurement contract remains**

Run:

```powershell
rg -n "Measurement|Measurements|measurement|measurements" backend frontend database --glob "!database/migrations/002_master_data.sql" --glob "!docs/**"
```

Expected: only intentional historical migration-test references to the rename statement remain.

- [ ] **Step 2: Run backend verification**

Run: `go test ./...`

Working directory: `backend`

Expected: PASS.

- [ ] **Step 3: Run frontend verification**

Run: `npm test -- --run`

Working directory: `frontend`

Expected: PASS.

Run: `npm run build`

Working directory: `frontend`

Expected: PASS.

- [ ] **Step 4: Review migration ordering and worktree**

Run:

```powershell
git diff --check
git status --short
git log -6 --oneline
```

Expected: no whitespace errors; only intended changes are present; stage commits are visible in order.

