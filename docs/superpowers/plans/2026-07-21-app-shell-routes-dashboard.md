# App Shell, Module Routes, and Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a shared collapsible authenticated shell, live navigation to every MVP module index, and a real-data-only dashboard with compact empty charts.

**Architecture:** Next.js App Router pages share a client-side `AppShell` that owns session loading and sidebar state. Route metadata and module index definitions remain typed configuration, while reusable chart and table-shell components provide consistent empty states without fabricated transactions.

**Tech Stack:** Next.js 16, React 19, TypeScript, CSS, Vitest, Testing Library, Docker Compose.

## Global Constraints

- Use only persisted application data; never fabricate metrics or transactions.
- Expanded desktop sidebar is approximately 220 px and collapsed width approximately 56 px.
- Narrow-screen navigation is a temporary drawer.
- UI typography, controls, cards, and table rows remain compact.
- Every sidebar target resolves to a real page.
- Raw Material CRUD is deferred to the next functional slice.

---

### Task 1: Typed navigation and collapsible shared shell

**Files:**
- Create: `frontend/components/app-shell/navigation.ts`
- Create: `frontend/components/app-shell/app-shell.tsx`
- Create: `frontend/components/app-shell/app-shell.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces: `AppShell({ title, children }: { title: string; children: ReactNode })` and `navigationGroups`.
- Consumes: `/api/auth/me`, `usePathname`, `useRouter`, and `localStorage['order-stock.sidebar-collapsed']`.

- [ ] **Step 1: Write failing component tests**

Test that the shell renders navigation, marks the current path active, toggles `aria-expanded`, and persists the collapsed value:

```tsx
render(<AppShell title="Dashboard"><div>content</div></AppShell>);
expect(screen.getByRole('link', { name: 'Dashboard' })).toHaveAttribute('aria-current', 'page');
await user.click(screen.getByRole('button', { name: 'Collapse sidebar' }));
expect(localStorage.getItem('order-stock.sidebar-collapsed')).toBe('true');
```

- [ ] **Step 2: Verify the focused test fails**

Run: `npm test -- --run components/app-shell/app-shell.test.tsx`
Expected: FAIL because `AppShell` does not exist.

- [ ] **Step 3: Implement navigation metadata and shell**

Define groups for Dashboard, Procurement, Logistics, and Data Master. Use links with icon, label, pathname-derived `aria-current`, an accessible toggle button, and authenticated user loading. Store only the boolean collapsed preference; mobile drawer state remains transient.

- [ ] **Step 4: Verify the focused test passes**

Run: `npm test -- --run components/app-shell/app-shell.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/app-shell frontend/app/globals.css
git commit -m "feat: add collapsible authenticated app shell"
```

### Task 2: Reusable module index and all live routes

**Files:**
- Create: `frontend/components/module-index.tsx`
- Create: `frontend/components/module-index.test.tsx`
- Create: `frontend/app/supplier-orders/page.tsx`
- Create: `frontend/app/approvals/page.tsx`
- Create: `frontend/app/delivery-notes/page.tsx`
- Create: `frontend/app/receiving/page.tsx`
- Create: `frontend/app/outgoing-material/page.tsx`
- Create: `frontend/app/measurements/page.tsx`
- Create: `frontend/app/suppliers/page.tsx`
- Create: `frontend/app/raw-materials/page.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `AppShell` from Task 1.
- Produces: `ModuleIndex` accepting title, description, actionLabel, columns, searchPlaceholder, and emptyMessage.

- [ ] **Step 1: Write failing index-shell test**

```tsx
render(<ModuleIndex title="Suppliers" description="Supplier master" actionLabel="New supplier" columns={['Code','Name']} searchPlaceholder="Search suppliers" emptyMessage="No suppliers yet" />);
expect(screen.getByRole('table')).toBeInTheDocument();
expect(screen.getByText('No suppliers yet')).toBeInTheDocument();
expect(screen.getByRole('button', { name: 'New supplier' })).toBeDisabled();
```

- [ ] **Step 2: Verify test fails**

Run: `npm test -- --run components/module-index.test.tsx`
Expected: FAIL because `ModuleIndex` does not exist.

- [ ] **Step 3: Implement compact index shell and pages**

Render search, filter, column control, future primary action, semantic compact table headings, and module-specific empty copy. Each route exports its own configuration and wraps it in `AppShell`.

- [ ] **Step 4: Verify test and route build pass**

Run: `npm test -- --run components/module-index.test.tsx; npm run typecheck`
Expected: both commands PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/module-index* frontend/app
git commit -m "feat: add live module index routes"
```

### Task 3: Real-data-only dashboard charts

**Files:**
- Create: `frontend/components/dashboard/dashboard-chart.tsx`
- Create: `frontend/components/dashboard/dashboard-chart.test.tsx`
- Create: `frontend/components/dashboard/dashboard-data.ts`
- Modify: `frontend/app/dashboard/page.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `AppShell` and a typed `DashboardSnapshot` with KPI values and chart series.
- Produces: `DashboardChart({ title, children, empty })` and `emptyDashboardSnapshot` whose numeric values and arrays are all zero/empty.

- [ ] **Step 1: Write failing empty-data test**

```tsx
render(<DashboardChart title="PO trend" empty><svg aria-label="PO trend chart" /></DashboardChart>);
expect(screen.getByText('Belum ada data transaksi')).toBeInTheDocument();
```

Also assert every KPI in `emptyDashboardSnapshot` equals zero and every series is empty.

- [ ] **Step 2: Verify test fails**

Run: `npm test -- --run components/dashboard/dashboard-chart.test.tsx`
Expected: FAIL because the dashboard components do not exist.

- [ ] **Step 3: Implement compact dashboard**

Build four KPI cards and panels for PO/receiving trend, PO status, outstanding Kanban by supplier, and recent activity. Use lightweight accessible SVG/CSS primitives and show the approved empty message when arrays are empty.

- [ ] **Step 4: Verify focused and complete frontend checks**

Run: `npm test; npm run lint; npm run typecheck; npm run build`
Expected: all commands PASS and every route appears in the Next.js route list.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/dashboard frontend/app/dashboard frontend/app/globals.css
git commit -m "feat: add operational dashboard charts"
```

### Task 4: Docker and browser-path verification

**Files:**
- Modify only if verification exposes a defect in files owned by Tasks 1–3.

**Interfaces:**
- Consumes: Docker Compose services on frontend `3019`, backend `8091`, and PostgreSQL `5445`.
- Produces: a running local build with authenticated module routes.

- [ ] **Step 1: Rebuild the frontend service**

Run: `docker compose up -d --build frontend`
Expected: frontend image builds and all three containers are running; PostgreSQL is healthy.

- [ ] **Step 2: Authenticate through the frontend proxy**

POST `http://localhost:3019/api/auth/login` using `admin` and the configured local password, retain the cookie, then GET `/api/auth/me`.
Expected: username `admin`, one session cookie, and 32 permissions.

- [ ] **Step 3: Smoke-test every browser route**

GET all nine approved routes on port 3019.
Expected: every response is HTTP 200 and contains its module title.

- [ ] **Step 4: Check runtime and disk health**

Run: `docker compose ps` and check drive C free space.
Expected: containers are running, PostgreSQL is healthy, and free space remains sufficient for Docker writes.

- [ ] **Step 5: Record final verification commit if fixes were required**

```powershell
git status --short
git commit -am "fix: harden app shell runtime behavior"
```
