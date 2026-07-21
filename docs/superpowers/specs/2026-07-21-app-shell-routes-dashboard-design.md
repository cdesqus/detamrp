# App Shell, Module Routes, and Dashboard Design

## Objective

Make every MVP navigation item open a real index page, provide a consistent collapsible application shell, and upgrade the dashboard with compact operational charts backed only by real application data.

This slice establishes navigation and reusable page structure. Transactional CRUD remains out of scope; CRUD implementation will begin afterward with Raw Materials.

## Application Shell

All authenticated pages use one shared `AppShell` rather than duplicating sidebar and header markup.

- Expanded desktop sidebar width: approximately 220 px.
- Collapsed desktop sidebar width: approximately 56 px, showing icons with accessible tooltips.
- A header control toggles the sidebar without navigating away.
- The preference is stored in browser `localStorage` and restored on later visits.
- The content area automatically consumes the released width when collapsed.
- On narrow screens the sidebar becomes a temporary overlay/drawer instead of permanently consuming screen width.
- Navigation active state is derived from the current pathname.
- The authenticated user's display name remains visible in the header.
- Unauthenticated API responses redirect the user to `/login`.

The UI remains compact: restrained typography, short controls, tight table rows, subtle borders, and no oversized cards or bold headings.

## Routes and Navigation

The following routes will be implemented and linked from the sidebar:

| Navigation | Route |
| --- | --- |
| Dashboard | `/dashboard` |
| Supplier Orders | `/supplier-orders` |
| Approval Inbox | `/approvals` |
| Delivery Notes | `/delivery-notes` |
| Receiving | `/receiving` |
| Outgoing Material | `/outgoing-material` |
| Measurements | `/measurements` |
| Suppliers | `/suppliers` |
| Raw Materials | `/raw-materials` |

Each non-dashboard route initially renders a usable index shell containing:

- module title and concise operational description;
- compact primary action appropriate to the module;
- search and filter controls;
- column visibility control;
- compact table headers matching the future domain records;
- honest empty state explaining that no records exist yet.

Controls that require future CRUD functionality may render disabled with a clear `Coming next` indication. They must not silently fail or navigate to dead links.

## Dashboard

The dashboard uses only persisted application data. It must not generate sample transactions or fabricate metrics.

### KPI cards

- Pending Approval
- Expected Today
- Received Today
- Outstanding Kanban

### Visual panels

- PO and receiving trend over time;
- PO status distribution;
- outstanding Kanban grouped by supplier;
- latest operational activity.

Before transactional APIs contain records, each chart renders its axes/title and a compact empty-state message. KPI values remain zero. Once transaction modules are implemented, the same components will consume dashboard API results without redesigning the page.

Charts should be responsive, accessible, and visually quiet. The initial implementation may use lightweight SVG/CSS chart primitives to avoid adding a large dependency before the required chart interactions are known.

## Component Boundaries

- `AppShell`: authentication boundary and responsive page structure.
- `Sidebar`: navigation, active route, expanded/collapsed presentation.
- `PageHeader`: breadcrumbs/title area and sidebar toggle.
- `ModuleIndex`: reusable filter bar, table shell, and empty state.
- `DashboardChart`: consistent chart panel and no-data behavior.

Module pages provide configuration and column definitions to shared components. Domain-specific data fetching and mutations will be added within each module as its CRUD slice is implemented.

## State and Data Flow

1. `AppShell` requests `/api/auth/me` when an authenticated page loads.
2. A rejected session redirects to `/login`; a valid session renders navigation filtered by permissions when required.
3. Sidebar state is initialized from `localStorage` on the client and updated on toggle.
4. Dashboard components request real aggregate data once dashboard endpoints exist. Until then, they receive explicit zero/empty data from a typed local adapter, not invented records.
5. Module index shells display empty collections until their domain APIs are connected.

## Error and Empty States

- Authentication failure: redirect to login.
- Dashboard request failure: show a compact retryable error inside the affected panel without breaking navigation.
- Empty chart data: show `Belum ada data transaksi`.
- Empty module table: show a module-specific empty message and the intended next action.
- Disabled future action: identify it as not yet available rather than accepting input.

## Verification

- Component tests cover sidebar toggling, persistence, and active navigation state.
- Route tests confirm every sidebar target resolves to an index page.
- Dashboard tests confirm empty datasets render zeros and empty states, never fabricated values.
- Responsive behavior is verified for expanded desktop, collapsed desktop, and narrow-screen drawer states.
- Frontend lint, typecheck, unit tests, and production build must pass before Docker is rebuilt.
- Browser-path smoke tests verify `/login`, `/dashboard`, every module route, and authenticated API proxy behavior through port 3019.

## Deferred Work

- CRUD forms and mutations.
- Real dashboard aggregation endpoints.
- User-configurable table columns persisted server-side.
- Advanced chart drill-down and exports.
- Raw Material CRUD is the next planned functional slice.
