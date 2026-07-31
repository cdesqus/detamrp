# Chart, Company Settings, and Transaction Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct the dashboard tooltip, rebuild Company Settings into a stable responsive layout, and add newest-first `created_at` sorting plus Supplier and Created Date filters to Supplier Orders and Receiving.

**Architecture:** Phase 1 separates tooltip content and collision placement from SVG geometry. Phase 2 gives Company Settings a dedicated one-column shell that cannot inherit the generic two-column settings grid. Phase 3 extends both backend list contracts with Jakarta-calendar date bounds and supplier filtering, then presents them through one reusable transaction filter bar.

**Tech Stack:** Go, Gin, PostgreSQL, Next.js 16, React 19, TypeScript, SVG, CSS, Vitest, Testing Library.

## Global Constraints

- Use implementation plans only; do not create a design spec.
- Preserve Ordered blue and Received green.
- Treat Created From and Created To as inclusive Asia/Jakarta calendar dates.
- Sort by `created_at DESC, id DESC`.
- Apply filters in SQL within the authenticated tenant scope.
- Keep all filter controls responsive and visually consistent with the existing enterprise UI.
- Do not change transaction business rules, document generation, or receiving-session behavior.

---

## Phase 1: Precise, Collision-Aware Trend Tooltip

### Task 1: Lock tooltip value association and edge placement with failing tests

**Files:**
- Modify: `frontend/components/dashboard/dashboard-charts.test.tsx`
- Modify: `frontend/components/dashboard/trend-chart-geometry.test.ts`
- Modify: `frontend/components/dashboard/trend-chart-geometry.ts`

**Interfaces:**
- Consumes: the existing `TrendPoint` sequence.
- Produces: `tooltipPlacement(index: number, count: number): 'start' | 'center' | 'end'`.

- [ ] **Step 1: Add a failing placement test**

```ts
import { tooltipPlacement } from './trend-chart-geometry';

it('anchors first and last tooltips inside the chart', () => {
  expect(tooltipPlacement(0, 30)).toBe('start');
  expect(tooltipPlacement(14, 30)).toBe('center');
  expect(tooltipPlacement(29, 30)).toBe('end');
});
```

- [ ] **Step 2: Add a failing semantic tooltip test**

Render:

```tsx
<TrendChart data={[
  { date: '2026-07-28', ordered: 0, received: 20 },
  { date: '2026-07-31', ordered: 21, received: 0 },
]} />
```

Focus the first hit target and assert:

```ts
const tooltip = screen.getByRole('status');
expect(within(tooltip).getByText('Ordered').closest('.trend-tooltip-row')).toHaveTextContent('Ordered0');
expect(within(tooltip).getByText('Received').closest('.trend-tooltip-row')).toHaveTextContent('Received20');
expect(tooltip).toHaveAttribute('data-placement', 'start');
```

Focus the final target and assert `data-placement="end"`.

- [ ] **Step 3: Verify both tests fail**

Run:

```powershell
npm test -- components/dashboard/trend-chart-geometry.test.ts components/dashboard/dashboard-charts.test.tsx
```

Expected: FAIL because placement metadata and explicit tooltip rows do not exist.

- [ ] **Step 4: Implement `tooltipPlacement`**

Return:

- `start` for the first 20% of points.
- `end` for the final 20%.
- `center` otherwise.
- `center` for an empty sequence.

- [ ] **Step 5: Verify the geometry test passes**

Run: `npm test -- components/dashboard/trend-chart-geometry.test.ts`

Expected: PASS.

### Task 2: Rebuild tooltip markup and positioning

**Files:**
- Modify: `frontend/components/dashboard/trend-chart.tsx`
- Modify: `frontend/app/globals.css`
- Test: `frontend/components/dashboard/dashboard-charts.test.tsx`

**Interfaces:**
- Consumes: `tooltipPlacement` from Task 1.
- Produces: explicit `.trend-tooltip-row` elements and `data-placement`.

- [ ] **Step 1: Replace `display: contents` tooltip rows**

Render each series as:

```tsx
<span className="trend-tooltip-row">
  <i className="legend-ordered" />
  <span>Ordered</span>
  <b>{formatTrendValue(active.ordered)}</b>
</span>
```

Do the same for Received. Keep `font-variant-numeric: tabular-nums`.

- [ ] **Step 2: Apply collision-aware transforms**

Use:

```css
.trend-tooltip[data-placement="start"] { transform: translate(0, -100%); }
.trend-tooltip[data-placement="center"] { transform: translate(-50%, -100%); }
.trend-tooltip[data-placement="end"] { transform: translate(-100%, -100%); }
```

Clamp the tooltip top inside the canvas and keep it within the panel at the first and last date.

- [ ] **Step 3: Verify Phase 1**

Run:

```powershell
npm test -- components/dashboard/trend-chart-geometry.test.ts components/dashboard/dashboard-charts.test.tsx components/dashboard/dashboard-view.test.tsx
npm run typecheck
npm run lint
```

Expected: all commands exit 0.

- [ ] **Step 4: Manually inspect representative points**

Check 2026-07-28 (`Ordered 0`, `Received 20`) and 2026-07-31 (`Ordered 21`, `Received 0`) at wide and narrow dashboard widths. Confirm first and last tooltips remain fully visible.

---

## Phase 2: Stable Company Settings Layout

### Task 3: Protect the Company Settings structure from generic grid rules

**Files:**
- Modify: `frontend/components/settings/company-settings.test.tsx`
- Modify: `frontend/components/settings/company-settings.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: the existing company settings API and upload/reset handlers.
- Produces: `.company-settings-shell`, `.company-settings-assets`, `.company-asset-card`, and `.company-settings-footer`.

- [ ] **Step 1: Add failing layout-hook tests**

Assert:

```ts
const form = screen.getByRole('form', { name: 'Company settings' });
expect(form).toHaveClass('company-settings-shell');
expect(screen.getByRole('region', { name: 'Brand assets' })).toHaveClass('company-settings-assets');
expect(screen.getByRole('group', { name: 'Company Logo' })).toHaveClass('company-asset-card');
expect(screen.getByRole('group', { name: 'Login Background' })).toHaveClass('company-asset-card');
expect(screen.getByRole('button', { name: 'Save settings' }).closest('footer')).toHaveClass('company-settings-footer');
```

- [ ] **Step 2: Verify the test fails**

Run: `npm test -- components/settings/company-settings.test.tsx`

Expected: FAIL because the dedicated shell and footer are absent.

- [ ] **Step 3: Simplify the form hierarchy**

Use:

```tsx
<form aria-label="Company settings" className="company-settings-shell">
  <section className="company-settings-identity">...</section>
  <section aria-label="Brand assets" className="company-settings-assets">
    <BrandingField ... />
    <BrandingField ... />
  </section>
  <footer className="company-settings-footer">...</footer>
</form>
```

Each `BrandingField` remains a named fieldset but receives `company-asset-card`.

- [ ] **Step 4: Remove conflicting legacy layout selectors**

Remove Company Settings dependence on `.settings-card`, `.company-branding-card`, and nested `.branding-grid` rules. Do not change SMTP or other Settings layouts.

- [ ] **Step 5: Implement the responsive layout**

Use a single-column shell and:

```css
.company-settings-assets {
  display: grid;
  grid-template-columns: repeat(2, minmax(280px, 1fr));
}
@media (max-width: 780px) {
  .company-settings-assets { grid-template-columns: 1fr; }
}
```

Inside each asset card:

- Heading and file requirement at the top.
- Stable preview area below.
- File input at full available width.
- Upload and Reset in one wrapping action row.
- `min-width: 0` on every grid child.
- Logo uses `object-fit: contain`; background uses `object-fit: cover`.

Footer spans the entire card with feedback left and Save Settings right. At narrow width, place Save Settings below feedback at full width.

- [ ] **Step 6: Verify Phase 2**

Run:

```powershell
npm test -- components/settings/company-settings.test.tsx
npm run typecheck
npm run lint
```

Expected: all commands exit 0.

- [ ] **Step 7: Inspect Company Settings at three widths**

Check 1440px, 1024px, and 390px. Confirm no field, preview, filename, or action overflows the outer card.

---

## Phase 3: Created-At Sorting and Transaction Filters

### Task 4: Extend Purchase Order list filters and newest-first sorting

**Files:**
- Modify: `backend/internal/purchaseorder/domain.go`
- Modify: `backend/internal/purchaseorder/domain_test.go`
- Modify: `backend/internal/purchaseorder/http.go`
- Modify: `backend/internal/purchaseorder/http_test.go`
- Modify: `backend/internal/purchaseorder/store.go`
- Modify: `backend/internal/purchaseorder/store_test.go`

**Interfaces:**
- Extends `purchaseorder.ListQuery` with `CreatedFrom time.Time` and `CreatedToExclusive time.Time`.
- Accepts `createdFrom=YYYY-MM-DD` and `createdTo=YYYY-MM-DD`.
- Interprets dates in `Asia/Jakarta`.

- [ ] **Step 1: Write failing HTTP parsing tests**

Verify:

- `createdFrom=2026-07-01` becomes `2026-07-01T00:00:00+07:00`.
- `createdTo=2026-07-31` becomes the exclusive bound `2026-08-01T00:00:00+07:00`.
- Invalid dates return HTTP 400 with field-specific errors.
- From after To returns HTTP 400.

- [ ] **Step 2: Verify HTTP tests fail**

Run: `go test ./internal/purchaseorder -run 'Test.*List.*(Date|Filter)'`

Expected: FAIL because created-date parsing is absent.

- [ ] **Step 3: Implement date parsing**

Add a package helper that parses `2006-01-02` with `time.LoadLocation("Asia/Jakarta")`, and converts Created To to an exclusive next-day bound.

- [ ] **Step 4: Write failing SQL contract tests**

Assert both count and item queries contain:

```sql
($5::timestamptz IS NULL OR p.created_at >= $5)
($6::timestamptz IS NULL OR p.created_at < $6)
ORDER BY p.created_at DESC,p.id DESC
```

Supplier filtering remains tenant-scoped and uses the existing `supplierId`.

- [ ] **Step 5: Implement SQL filtering and sorting**

Pass nullable timestamp arguments to both count and list queries. Update `LIMIT` and `OFFSET` placeholder positions accordingly.

- [ ] **Step 6: Verify Purchase Order backend**

Run: `go test ./internal/purchaseorder`

Expected: PASS.

### Task 5: Add Receiving list query, response fields, filtering, and sorting

**Files:**
- Modify: `backend/internal/receiving/domain.go`
- Modify: `backend/internal/receiving/domain_test.go`
- Modify: `backend/internal/receiving/http.go`
- Modify: `backend/internal/receiving/http_test.go`
- Modify: `backend/internal/receiving/store.go`
- Modify: `backend/internal/receiving/store_test.go`

**Interfaces:**
- Adds `receiving.ListQuery` with `SupplierID`, `CreatedFrom`, and `CreatedToExclusive`.
- Adds `SupplierID uuid.UUID` and `CreatedAt time.Time` to `Receiving`.
- Changes `Store.List(ctx, actor)` to `Store.List(ctx, actor, query)`.

- [ ] **Step 1: Write failing Receiving HTTP tests**

Cover valid Supplier/Created Date filters, invalid UUID/date responses, and From-after-To validation.

- [ ] **Step 2: Verify Receiving HTTP tests fail**

Run: `go test ./internal/receiving -run 'Test.*List.*(Date|Filter)'`

Expected: FAIL because Receiving has no list query.

- [ ] **Step 3: Implement the Receiving query contract**

Reuse the same Jakarta date semantics as Purchase Orders. Keep package boundaries independent; do not import one domain package from the other.

- [ ] **Step 4: Write failing store tests**

Assert the query:

```sql
WHERE r.tenant_id=$1
AND ($2::uuid='00000000-0000-0000-0000-000000000000' OR p.supplier_id=$2)
AND ($3::timestamptz IS NULL OR r.created_at >= $3)
AND ($4::timestamptz IS NULL OR r.created_at < $4)
ORDER BY r.created_at DESC,r.id DESC
```

Also assert the SELECT scans `p.supplier_id` and `r.created_at`.

- [ ] **Step 5: Implement store filtering and response fields**

Keep joins tenant-scoped. Return an empty array rather than `null`.

- [ ] **Step 6: Verify Receiving backend**

Run: `go test ./internal/receiving`

Expected: PASS.

### Task 6: Build a reusable enterprise transaction filter bar

**Files:**
- Create: `frontend/components/transaction-list-filters.tsx`
- Create: `frontend/components/transaction-list-filters.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**

```ts
type TransactionListFiltersProps = {
  search?: string;
  onSearchChange?: (value: string) => void;
  supplierId: string;
  createdFrom: string;
  createdTo: string;
  suppliers: { id: string; code?: string; name: string }[];
  recordCount: number;
  refreshing?: boolean;
  onApply: () => void;
  onReset: () => void;
};
```

- [ ] **Step 1: Write failing component tests**

Test:

- Supplier labels render as `CODE — Name`.
- Apply invokes only after explicit click.
- From later than To shows an accessible error and does not invoke Apply.
- Reset clears through the parent callbacks.
- Active filter summary is announced.

- [ ] **Step 2: Verify the filter component test fails**

Run: `npm test -- components/transaction-list-filters.test.tsx`

Expected: FAIL because the component does not exist.

- [ ] **Step 3: Implement the filter bar**

Use labeled search, supplier, Created From, and Created To controls. Place Apply, Reset, active summary, refreshing state, and record count without changing table height.

- [ ] **Step 4: Add responsive styling**

Desktop uses a compact aligned row; laptop wraps controls; mobile uses one column with full-width buttons. Focus rings and button hierarchy follow the existing enterprise controls.

- [ ] **Step 5: Verify the reusable component**

Run: `npm test -- components/transaction-list-filters.test.tsx`

Expected: PASS.

### Task 7: Integrate filters into Supplier Orders

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`

**Interfaces:**
- Fetches suppliers from `/api/master-data/suppliers?limit=200` so active and inactive historical suppliers remain filterable.
- Sends applied `search`, `supplierId`, `createdFrom`, and `createdTo` to `/api/purchase-orders`.

- [ ] **Step 1: Write failing integration tests**

Assert:

- Initial response order is rendered unchanged from newest to oldest.
- Applying filters sends all selected query parameters.
- Reset removes filter parameters and returns to page 1.
- A refresh keeps the previous table snapshot visible until the response arrives.

- [ ] **Step 2: Verify Supplier Orders tests fail**

Run: `npm test -- components/supplier-orders/supplier-orders.test.tsx`

Expected: FAIL because the filter bar is absent.

- [ ] **Step 3: Implement draft and applied filter state**

Only applied state triggers the list request. Abort stale requests. Reset page to 1 on Apply/Reset. Keep the current items while `refreshing=true`.

- [ ] **Step 4: Verify Supplier Orders**

Run: `npm test -- components/supplier-orders/supplier-orders.test.tsx`

Expected: PASS.

### Task 8: Integrate filters into Receiving

**Files:**
- Modify: `frontend/components/receiving/receiving-index.tsx`
- Modify: `frontend/components/receiving/receiving.test.tsx`

**Interfaces:**
- Fetches the same historical supplier list.
- Sends `supplierId`, `createdFrom`, and `createdTo` to `/api/receivings`.
- Reads `createdAt` from each Receiving item while continuing to display the existing Receiving Date column.

- [ ] **Step 1: Write failing integration tests**

Assert:

- Newest `createdAt` item appears first from the server response.
- Apply sends Supplier and both Created Date bounds.
- Reset reloads without filter parameters and resets pagination.
- Open receiving sessions remain visible and unaffected by completed-receiving filters.

- [ ] **Step 2: Verify Receiving tests fail**

Run: `npm test -- components/receiving/receiving.test.tsx`

Expected: FAIL because Receiving filters are absent.

- [ ] **Step 3: Implement filter integration**

Keep completed receiving filters independent from session creation and open-session loading. Preserve the existing Create Receiving modal.

- [ ] **Step 4: Verify Receiving**

Run: `npm test -- components/receiving/receiving.test.tsx`

Expected: PASS.

### Task 9: Full verification and three-phase visual audit

**Files:**
- Verify only; modify files solely to fix failures introduced by Tasks 1–8.

- [ ] **Step 1: Run backend verification**

```powershell
go test ./...
```

Run from `backend`.

- [ ] **Step 2: Run frontend verification**

```powershell
npm run typecheck
npm run lint
npm test
npm run build
```

Run from `frontend`.

- [ ] **Step 3: Audit the approved scenarios**

- Dashboard: 2026-07-28 and 2026-07-31 tooltip accuracy and edge visibility.
- Company Settings: 1440px, 1024px, and 390px with long filenames and both preview types.
- Supplier Orders: default newest-first, Supplier filter, Created Date range, Apply, Reset, and refreshing snapshot.
- Receiving: default newest-first, filters, pagination reset, open sessions, and Create Receiving modal.

- [ ] **Step 4: Review repository state**

```powershell
git diff --check
git status --short
```

Confirm there are no unrelated business-logic edits, new dependencies, or generated artifacts.

