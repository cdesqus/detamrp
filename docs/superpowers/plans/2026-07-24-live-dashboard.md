# Live Operational Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the dashboard placeholder with filterable, tenant-scoped operational KPIs, charts, and recent activity backed by PostgreSQL.

**Architecture:** Add a focused `dashboard` backend package that validates one filter and executes aggregate queries inside the existing tenant transaction/RLS boundary. Expose one authenticated JSON endpoint and consume it from a client-side dashboard component that owns URL filters, loading/error states, and lightweight SVG/CSS charts.

**Tech Stack:** Go, Gin, pgx/PostgreSQL RLS, Next.js 16, React 19, TypeScript, Vitest, Testing Library, inline SVG/CSS.

## Global Constraints

- Default period is the last 30 calendar days, inclusive.
- Date and supplier filters affect all historical metrics; Current Stock ignores dates but respects supplier.
- Kanban metrics count complete Kanban lots, not base-unit quantities.
- Derived PO statuses must match the Supplier Order module.
- No chart library or cache is added.
- Every database query is tenant-scoped and executed through the existing tenant transaction/RLS pattern.
- Existing compact UI sizing and restrained colors remain unchanged.

---

## File Structure

- `backend/internal/dashboard/domain.go`: filter, response, metric, chart, and activity contracts.
- `backend/internal/dashboard/store.go`: tenant-scoped aggregate queries and response assembly.
- `backend/internal/dashboard/http.go`: authentication, query validation, and `GET /dashboard`.
- `backend/internal/dashboard/store_test.go`: filter validation and aggregate behavior tests.
- `backend/internal/dashboard/http_test.go`: authentication and HTTP contract tests.
- `backend/internal/api/server.go`: dashboard route dependency registration.
- `backend/cmd/api/main.go`: construct and inject the dashboard store.
- `frontend/components/dashboard/dashboard-data.ts`: API-aligned TypeScript contracts and date helpers.
- `frontend/components/dashboard/dashboard-view.tsx`: fetch lifecycle, filters, KPI cards, charts, and activity.
- `frontend/components/dashboard/dashboard-view.test.tsx`: user-visible dashboard behavior.
- `frontend/components/dashboard/dashboard-chart.tsx`: reusable compact panel/empty-state shell.
- `frontend/app/dashboard/page.tsx`: render the live dashboard view.
- `frontend/app/globals.css`: compact filters and chart styling.

### Task 1: Dashboard Domain and Filter Validation

**Files:**
- Create: `backend/internal/dashboard/domain.go`
- Test: `backend/internal/dashboard/store_test.go`

**Interfaces:**
- Produces: `ParseFilter(url.Values, time.Time) (Filter, map[string]string)`
- Produces: `Snapshot`, with JSON fields `filter`, `metrics`, `trend`, `poStatus`, `outstandingBySupplier`, and `activities`.

- [ ] **Step 1: Write failing filter tests**

Cover omitted dates defaulting to `now-29` through `now`, valid explicit dates and supplier UUID, `from > to`, malformed dates, and malformed supplier ID.

```go
filter, fields := ParseFilter(url.Values{}, time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC))
require.Empty(t, fields)
assert.Equal(t, "2026-06-25", filter.From.Format(time.DateOnly))
assert.Equal(t, "2026-07-24", filter.To.Format(time.DateOnly))
```

- [ ] **Step 2: Run the tests and verify failure**

Run: `cd backend && go test ./internal/dashboard -run ParseFilter -v`

Expected: FAIL because the package/contracts do not exist.

- [ ] **Step 3: Implement domain contracts and strict validation**

Use date-only parsing, inclusive dates, `uuid.Nil` for all suppliers, and field keys `from`, `to`, and `supplierId`. Define:

```go
type Filter struct {
    From       time.Time `json:"from"`
    To         time.Time `json:"to"`
    SupplierID uuid.UUID `json:"supplierId,omitempty"`
}

type Metrics struct {
    PendingApproval  int64 `json:"pendingApproval"`
    OpenPO           int64 `json:"openPO"`
    ReceivedKanban   int64 `json:"receivedKanban"`
    OutstandingKanban int64 `json:"outstandingKanban"`
    CurrentStock     int64 `json:"currentStock"`
}
```

Define daily trend, status, supplier, and activity DTOs using stable camelCase JSON names.

- [ ] **Step 4: Run focused tests**

Run: `cd backend && go test ./internal/dashboard -run ParseFilter -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/dashboard
git commit -m "feat: define dashboard filters and contracts"
```

### Task 2: Tenant-Scoped Dashboard Aggregation

**Files:**
- Create: `backend/internal/dashboard/store.go`
- Modify: `backend/internal/dashboard/store_test.go`

**Interfaces:**
- Consumes: `Filter`, `Snapshot`.
- Produces: `NewStore(*database.Pool) *Store`.
- Produces: `(*Store).Snapshot(context.Context, Actor, Filter) (Snapshot, error)`.

- [ ] **Step 1: Add failing aggregate tests**

Seed two tenants with POs, delivery notes, Kanban lots, receiving records, and outgoing documents. Assert:

```go
assert.EqualValues(t, 1, got.Metrics.PendingApproval)
assert.EqualValues(t, 1, got.Metrics.OpenPO)
assert.EqualValues(t, 2, got.Metrics.ReceivedKanban)
assert.EqualValues(t, 3, got.Metrics.OutstandingKanban)
assert.EqualValues(t, 2, got.Metrics.CurrentStock)
```

Also assert supplier filtering, empty datasets, no cross-tenant rows, complete date series, top-ten supplier ordering, and that changing the date range does not change Current Stock.

- [ ] **Step 2: Run tests and verify failure**

Run: `cd backend && go test ./internal/dashboard -run 'Snapshot|Tenant|Supplier' -v`

Expected: FAIL because `Store.Snapshot` is missing.

- [ ] **Step 3: Implement aggregate queries**

Use `database.WithTenant` once per snapshot and parameterized SQL. Reuse Supplier Order derived-status SQL semantics. Build the daily series with `generate_series($from::date,$to::date,'1 day')`, left joining ordered and received lot aggregates. Determine:

```sql
current stock       := kanban_lots.status = 'IN_STOCK'
outstanding         := kanban_lots.status = 'ISSUED'
received in period  := distinct receiving_kanban_lots.kanban_lot_id joined to completed receivings
```

Return the latest 10 unioned activity events ordered by timestamp descending. Verify the selected supplier belongs to the actor tenant before running aggregates.

- [ ] **Step 4: Run focused and backend tests**

Run: `cd backend && go test ./internal/dashboard -v && go test ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/dashboard
git commit -m "feat: aggregate live dashboard data"
```

### Task 3: Authenticated Dashboard API

**Files:**
- Create: `backend/internal/dashboard/http.go`
- Create: `backend/internal/dashboard/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: `Store.Snapshot`.
- Produces: authenticated `GET /dashboard?from=&to=&supplierId=`.

- [ ] **Step 1: Write failing HTTP contract tests**

Test 401 without a session, 422 with `{error:"validation failed",fields:{...}}`, 200 with the full snapshot JSON shape, and 404/422 for a supplier outside the tenant.

```go
request := httptest.NewRequest(http.MethodGet, "/dashboard?from=2026-07-01&to=2026-07-24", nil)
request.AddCookie(&http.Cookie{Name: "session", Value: "valid"})
server.ServeHTTP(recorder, request)
assert.Equal(t, http.StatusOK, recorder.Code)
```

- [ ] **Step 2: Run HTTP tests and verify failure**

Run: `cd backend && go test ./internal/dashboard -run HTTP -v`

Expected: FAIL because routes are not registered.

- [ ] **Step 3: Register route and dependency**

Follow existing module middleware conventions, set a dashboard actor from the authenticated user, require a read permission already shared by operational users, parse filters using the server clock, and return generic 500 copy without leaking SQL errors.

- [ ] **Step 4: Run API and backend verification**

Run: `cd backend && go test ./internal/dashboard ./internal/api ./... && go vet ./...`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/dashboard backend/internal/api/server.go backend/cmd/api/main.go
git commit -m "feat: expose live dashboard endpoint"
```

### Task 4: Live Dashboard View and Filters

**Files:**
- Modify: `frontend/components/dashboard/dashboard-data.ts`
- Create: `frontend/components/dashboard/dashboard-view.tsx`
- Create: `frontend/components/dashboard/dashboard-view.test.tsx`
- Modify: `frontend/app/dashboard/page.tsx`

**Interfaces:**
- Consumes: `GET /api/dashboard`.
- Produces: `DashboardView`, URL query parameters `from`, `to`, `supplierId`.

- [ ] **Step 1: Write failing interaction tests**

Mock dashboard and supplier API calls. Test default loading, all five KPI values, URL-derived filters, Apply, Reset, invalid ranges, request errors with Retry, stale-response protection, and empty states.

```tsx
render(<DashboardView />)
expect(await screen.findByText('12', { selector: 'strong' })).toBeInTheDocument()
await user.selectOptions(screen.getByLabelText('Supplier'), supplierID)
await user.click(screen.getByRole('button', { name: 'Apply' }))
expect(fetchMock).toHaveBeenLastCalledWith(expect.stringContaining(`supplierId=${supplierID}`), expect.anything())
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd frontend && npm test -- dashboard-view.test.tsx`

Expected: FAIL because `DashboardView` does not exist.

- [ ] **Step 3: Implement fetch lifecycle and filter form**

Use `useSearchParams` and `router.replace` to preserve filters. Fetch suppliers and dashboard data with credentials and `AbortController`. Render compact KPI cards, a stable loading skeleton, an inline alert with Retry, and filtered empty states. Reset removes query parameters and reloads the default 30-day period.

- [ ] **Step 4: Replace the static page**

Make `frontend/app/dashboard/page.tsx` render:

```tsx
<AppShell title="Dashboard">
  <DashboardView />
</AppShell>
```

- [ ] **Step 5: Run frontend tests**

Run: `cd frontend && npm test -- dashboard-view.test.tsx dashboard-chart.test.tsx`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/app/dashboard frontend/components/dashboard
git commit -m "feat: load and filter dashboard data"
```

### Task 5: Accessible Compact Charts

**Files:**
- Create: `frontend/components/dashboard/trend-chart.tsx`
- Create: `frontend/components/dashboard/status-donut.tsx`
- Create: `frontend/components/dashboard/supplier-bars.tsx`
- Create: `frontend/components/dashboard/dashboard-charts.test.tsx`
- Modify: `frontend/components/dashboard/dashboard-view.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: typed trend/status/supplier arrays from `DashboardSnapshot`.
- Produces: inline SVG line and donut charts and CSS horizontal supplier bars.

- [ ] **Step 1: Write failing chart tests**

Assert accessible chart labels, legends, exact textual values, no `NaN` coordinates for all-zero or one-day series, top-ten rows, and compact empty-state behavior.

- [ ] **Step 2: Run and verify failure**

Run: `cd frontend && npm test -- dashboard-charts.test.tsx`

Expected: FAIL because the chart components do not exist.

- [ ] **Step 3: Implement charts without dependencies**

Normalize SVG coordinates against non-zero maximums, render ordered/received lines with point titles, use `stroke-dasharray` for donut segments, and show supplier names with numeric labels beside proportional CSS bars. Provide visible legends and `aria-label` summaries.

- [ ] **Step 4: Add compact responsive styling**

Keep existing 10px panel gaps, 7px radii, restrained blue/green/amber/red series, desktop two-column grid, and single-column mobile layout. Avoid oversized headings, cards, or thick chart decoration.

- [ ] **Step 5: Run full frontend verification**

Run: `cd frontend && npm test && npm run typecheck && npm run lint && npm run build`

Expected: all commands PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/dashboard frontend/app/globals.css
git commit -m "feat: add operational dashboard charts"
```

### Task 6: Runtime and Regression Verification

**Files:**
- Modify only if a verification failure identifies a dashboard-specific defect.

**Interfaces:**
- Validates the complete backend-to-browser flow.

- [ ] **Step 1: Run complete automated verification**

```bash
cd backend
go test ./...
go vet ./...
cd ../frontend
npm test
npm run typecheck
npm run lint
npm run build
```

Expected: PASS with no new warnings or failures.

- [ ] **Step 2: Rebuild local Docker services**

Run:

```bash
docker compose -p platform-master-po build backend frontend
docker compose -p platform-master-po up -d
docker compose -p platform-master-po ps
```

Expected: PostgreSQL healthy, migration exited successfully, backend and frontend running on the configured ports.

- [ ] **Step 3: Smoke-test real data**

Log in at `http://localhost:3019`, open `/dashboard`, and verify:

- existing transactions produce non-zero KPIs/charts;
- Apply changes every period-aware section;
- supplier filtering narrows results;
- Reset returns to the default 30-day URL and data;
- Current Stock remains unchanged when only dates change;
- refresh preserves applied URL filters.

- [ ] **Step 4: Inspect repository state and commit any verified fix**

Run: `git status --short && git diff --check`

Expected: clean working tree after any necessary dashboard-only correction is tested and committed.

