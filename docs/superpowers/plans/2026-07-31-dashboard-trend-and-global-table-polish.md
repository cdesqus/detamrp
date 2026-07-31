# Dashboard Trend and Global Table Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair and modernize the dashboard trend chart, then add restrained column separation and row interaction styling to every standard application table.

**Architecture:** Phase 1 keeps the chart dependency-free and isolates geometry/formatting helpers from the React SVG renderer so calculations can be tested directly. Phase 2 applies a shared table treatment through `.table-frame`, with explicit rules for empty states, footers, action columns, nested operational tables, and activity-detail tables.

**Tech Stack:** Next.js 16, React 19, TypeScript, SVG, CSS, Vitest, Testing Library.

## Global Constraints

- Preserve the existing Ordered blue and Received green series colors.
- Do not add a charting or tooltip dependency.
- Keep chart summaries accessible without pointer interaction.
- Use subtle neutral separators; tables must not resemble rigid spreadsheets.
- Apply the treatment to all standard tables while preventing doubled borders in specialized tables.
- Preserve existing table density, horizontal scrolling, pagination, and business behavior.

## Table Module Coverage

- **Data Master:** Suppliers, Raw Materials, Units, Categories, Packings, and Plants via `MasterDataCrud` and `ModuleIndex`.
- **Procurement:** Supplier Orders list, Supplier Order material lines, and Approval Inbox.
- **Logistics:** Stock Inventory and its Kanban/detail table, Receiving list and active session scans, Outgoing Material list and active session lines.
- **Reports:** Receiving Report results.
- **Settings:** Users, Roles & Permissions, Email Log, Activity Log, and Activity Log change-detail table.
- **Excluded because they do not render data tables:** Dashboard metric/chart cards, Company Settings, SMTP Settings, and other form-only pages.

---

## Phase 1: Dashboard Trend Chart

### Task 1: Extract deterministic chart geometry

**Files:**
- Create: `frontend/components/dashboard/trend-chart-geometry.ts`
- Create: `frontend/components/dashboard/trend-chart-geometry.test.ts`
- Modify: `frontend/components/dashboard/trend-chart.tsx`

**Interfaces:**
- Consumes: `DashboardSnapshot['trend']`.
- Produces: `buildTrendGeometry(data, width, height)` returning plot bounds, `maxValue`, Y ticks, X label indices, ordered/received line paths, and ordered/received area paths.

- [ ] **Step 1: Write failing geometry tests**

```ts
import { describe, expect, it } from 'vitest';
import { buildTrendGeometry } from './trend-chart-geometry';

describe('buildTrendGeometry', () => {
  it('keeps line and area paths finite for one day', () => {
    const result = buildTrendGeometry([{ date: '2026-07-24', ordered: 5, received: 3 }], 720, 240);
    expect(result.orderedLine).not.toMatch(/NaN|Infinity/);
    expect(result.receivedLine).not.toMatch(/NaN|Infinity/);
    expect(result.orderedArea.endsWith('Z')).toBe(true);
  });

  it('selects a readable subset of labels for a 30-day range', () => {
    const data = Array.from({ length: 30 }, (_, index) => ({
      date: `2026-07-${String(index + 1).padStart(2, '0')}`,
      ordered: index,
      received: index / 2,
    }));
    expect(buildTrendGeometry(data, 720, 240).xLabelIndices.length).toBeLessThanOrEqual(6);
  });
});
```

- [ ] **Step 2: Verify the geometry test fails**

Run: `npm test -- components/dashboard/trend-chart-geometry.test.ts`

Expected: FAIL because `trend-chart-geometry` does not exist.

- [ ] **Step 3: Implement the geometry helper**

Create typed helpers that:

```ts
export type TrendPoint = { date: string; ordered: number; received: number };
export function buildTrendGeometry(data: TrendPoint[], width: number, height: number) {
  // Use left/right/top/bottom plot padding.
  // Round maxValue to a readable tick ceiling.
  // Center a single data point.
  // Select at most six X labels including first and last.
  // Close area paths along the zero baseline.
}
```

- [ ] **Step 4: Update `TrendChart` to consume geometry**

Remove duplicated `x`, `y`, and `path` calculations from `trend-chart.tsx`. Keep `DashboardSnapshot['trend']` as the public prop type.

- [ ] **Step 5: Run the geometry and existing dashboard chart tests**

Run: `npm test -- components/dashboard/trend-chart-geometry.test.ts components/dashboard/dashboard-charts.test.ts`

Expected: PASS.

- [ ] **Step 6: Commit Phase 1 geometry**

```powershell
git add frontend/components/dashboard/trend-chart-geometry.ts frontend/components/dashboard/trend-chart-geometry.test.ts frontend/components/dashboard/trend-chart.tsx
git commit -m "refactor: stabilize dashboard trend geometry"
```

### Task 2: Render an enterprise trend chart and combined tooltip

**Files:**
- Modify: `frontend/components/dashboard/trend-chart.tsx`
- Modify: `frontend/components/dashboard/dashboard-charts.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: geometry from Task 1.
- Produces: keyboard- and pointer-selectable dates, a combined tooltip, visible axes, gradients, restrained dots, and the existing accessible SVG summary.

- [ ] **Step 1: Write failing interaction tests**

```tsx
it('shows ordered and received values in one tooltip', async () => {
  const user = userEvent.setup();
  render(<TrendChart data={[
    { date: '2026-07-23', ordered: 2, received: 1 },
    { date: '2026-07-24', ordered: 5, received: 3 },
  ]} />);
  await user.tab();
  expect(screen.getByRole('status')).toHaveTextContent('2026-07-23');
  expect(screen.getByRole('status')).toHaveTextContent('Ordered 2');
  expect(screen.getByRole('status')).toHaveTextContent('Received 1');
});
```

Also assert that line paths have dedicated `trend-series-line` classes and area paths have `trend-series-area` classes so point fill cannot override line fill again.

- [ ] **Step 2: Verify the tooltip test fails**

Run: `npm test -- components/dashboard/dashboard-charts.test.tsx`

Expected: FAIL because the combined tooltip and dedicated path classes are absent.

- [ ] **Step 3: Implement active-date interaction**

In `TrendChart`:

- Track `activeIndex` with React state.
- Add a transparent focusable hit target for each date.
- Update the active date on pointer enter, focus, and arrow-key navigation.
- Render one tooltip containing the date and both series values.
- Keep `<title>` elements or an equivalent hidden series summary for non-interactive accessibility.

- [ ] **Step 4: Render the visual hierarchy**

Add:

- Neutral Y-axis tick labels and horizontal grid lines.
- At most six formatted X-axis labels.
- Separate gradient area paths at low opacity.
- Two clean line paths with round caps and joins.
- Small default data dots and an emphasized active pair.
- A crosshair limited to the active date.
- Compact legend indicators matching Ordered blue and Received green.

- [ ] **Step 5: Replace conflicting chart CSS**

Remove the shared rule that gives `.trend-ordered` and `.trend-received` a solid `fill`. Style lines, areas, and dots independently:

```css
.trend-series-line { fill: none; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; }
.trend-series-area { stroke: none; opacity: .08; }
.trend-series-dot { stroke: white; stroke-width: 1.5; }
```

Use a neutral chart surface, soft horizontal grid, responsive tooltip, and reduced-motion guards.

- [ ] **Step 6: Verify Phase 1**

Run:

```powershell
npm test -- components/dashboard/dashboard-charts.test.tsx components/dashboard/dashboard-chart.test.tsx components/dashboard/dashboard-view.test.tsx
npm run typecheck
npm run lint
```

Expected: all commands exit 0.

- [ ] **Step 7: Commit Phase 1 chart polish**

```powershell
git add frontend/components/dashboard/trend-chart.tsx frontend/components/dashboard/dashboard-charts.test.tsx frontend/app/globals.css
git commit -m "feat: polish dashboard trend chart"
```

---

## Phase 2: Global Table Treatment

### Task 3: Add semantic hooks for special and terminal columns

**Files:**
- Modify: `frontend/components/master-data-crud.tsx`
- Modify: `frontend/components/module-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/approvals/approval-inbox.tsx`
- Modify: `frontend/components/inventory/inventory-index.tsx`
- Modify: `frontend/components/receiving/receiving-index.tsx`
- Modify: `frontend/components/receiving/receiving-session.tsx`
- Modify: `frontend/components/outgoing/outgoing-index.tsx`
- Modify: `frontend/components/outgoing/outgoing-session.tsx`
- Modify: `frontend/components/report-index.tsx`
- Modify: `frontend/components/settings/users-settings.tsx`
- Modify: `frontend/components/settings/roles-settings.tsx`
- Modify: `frontend/components/settings/email-log.tsx`
- Modify: `frontend/components/settings/activity-log.tsx`

**Interfaces:**
- Consumes: existing table markup and behavior.
- Produces: consistent `table-column-actions`, `table-column-number`, `table-row-empty`, and `table-detail` hooks only where structural exceptions need them.

- [ ] **Step 1: Add a failing representative structure test**

Extend the existing Supplier Orders, Receiving, and master-data tests to assert:

```ts
expect(screen.getByRole('columnheader', { name: 'Actions' })).toHaveClass('table-column-actions');
expect(screen.getByRole('columnheader', { name: 'No.' })).toHaveClass('table-column-number');
```

For an empty-state fixture, assert the spanning cell uses `table-row-empty`.

- [ ] **Step 2: Verify the representative tests fail**

Run:

```powershell
npm test -- components/supplier-orders/supplier-orders.test.tsx components/receiving/receiving.test.tsx components/master-data-crud.test.tsx
```

Expected: FAIL because the semantic hooks are absent.

- [ ] **Step 3: Add hooks without changing table behavior**

Apply hooks consistently:

- Number columns: `table-column-number`.
- Actions/documents: `table-column-actions`.
- Empty/loading/error spanning cells: `table-row-empty`.
- Operational detail tables nested inside sessions or drawers: `table-detail`.

Do not rename headers, move columns, or alter data rendering.

- [ ] **Step 4: Re-run representative tests**

Run the same command from Step 2.

Expected: PASS.

- [ ] **Step 5: Commit table structure hooks**

```powershell
git add frontend/components
git commit -m "refactor: add table presentation hooks"
```

### Task 4: Apply restrained separators and row interaction globally

**Files:**
- Modify: `frontend/app/globals.css`
- Modify: `frontend/components/module-index.test.tsx`
- Modify: `frontend/components/inventory/inventory-index.test.tsx`
- Modify: `frontend/components/settings/activity-log.test.tsx`

**Interfaces:**
- Consumes: `.table-frame` and semantic hooks from Task 3.
- Produces: one shared table presentation for every module listed under Table Module Coverage.

- [ ] **Step 1: Add regression assertions to representative module tests**

Assert representative tables retain:

- A `.table-frame` ancestor.
- The specialized `table-detail` hook for nested/detail tables.
- Pagination outside the `<table>` but inside `.table-frame`.

These tests protect the selectors that global CSS depends on without asserting exact colors.

- [ ] **Step 2: Verify any new structural assertion fails before its hook exists**

Run:

```powershell
npm test -- components/module-index.test.tsx components/inventory/inventory-index.test.tsx components/settings/activity-log.test.tsx
```

Expected: FAIL only for missing specialized hooks.

- [ ] **Step 3: Implement shared table styling**

Add rules equivalent to:

```css
.table-frame :is(th, td):not(:last-child) {
  border-right: 1px solid rgb(228 228 231 / 52%);
}
.table-frame th:not(:last-child) {
  border-right-color: rgb(212 212 216 / 72%);
}
.table-frame tbody tr {
  transition: background-color 120ms ease;
}
.table-frame tbody tr:hover > td {
  background: #fafafa;
  border-right-color: rgb(212 212 216 / 82%);
}
```

Refine these selectors so:

- Empty/loading/error spanning cells never show internal separators.
- The last cell stays open against the frame.
- `tfoot` totals remain visually distinct.
- Action and number columns use subtly different neutral backgrounds only when helpful.
- Nested `.table-detail` tables do not inherit doubled outer borders.
- Interactive controls retain their own focus rings and backgrounds.
- Hover is disabled for empty-state rows.

- [ ] **Step 4: Verify all table module tests**

Run:

```powershell
npm test -- components/master-data-crud.test.tsx components/module-index.test.tsx components/supplier-orders/supplier-orders.test.tsx components/approvals/approval-inbox.test.tsx components/inventory/inventory-index.test.tsx components/receiving/receiving.test.tsx components/outgoing/outgoing.test.tsx components/report-index.test.tsx components/settings/activity-log.test.tsx
```

Expected: all targeted table suites pass.

- [ ] **Step 5: Inspect every covered module at desktop and narrow widths**

Verify the module list in **Table Module Coverage** against:

- 1440px desktop.
- 1024px laptop.
- 390px mobile or narrow viewport with horizontal scrolling.

Check separator strength, hover continuity, action-button alignment, empty rows, pagination borders, sticky/scroll behavior, and nested detail tables.

- [ ] **Step 6: Commit Phase 2**

```powershell
git add frontend/app/globals.css frontend/components
git commit -m "feat: polish tables across modules"
```

### Task 5: Full verification

**Files:**
- Verify only; modify files solely to fix failures caused by Tasks 1–4.

- [ ] **Step 1: Run the complete frontend suite**

```powershell
npm run typecheck
npm run lint
npm test
npm run build
```

Expected: all commands exit 0 and all Vitest tests pass.

- [ ] **Step 2: Review the final diff**

```powershell
git diff --check
git status --short
```

Confirm there are no whitespace errors, unintended business-logic edits, new dependencies, or unrelated file changes.

