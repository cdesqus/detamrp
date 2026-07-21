# Navigation, Notification Center, Reports, and Settings Design

## Objective

Refine the MVP information architecture so operational documents stay with their parent Supplier Order, approvals are surfaced through a modern notification center, and Reports and Settings remain first-class MVP areas. Replace text abbreviations with consistent icons and move the desktop sidebar control into the sidebar itself.

## Sidebar Structure

The sidebar order is:

1. Dashboard
2. Data Master
   - Measurements
   - Suppliers
   - Raw Materials
3. Procurement
   - Supplier Orders
4. Logistics
   - Receiving
   - Outgoing Material
5. Reports
6. Settings
   - Users
   - Roles & Permissions
   - SMTP Settings
   - Email Log

Settings uses an expandable/collapsible group to keep the navigation compact. Its open state follows the active Settings route and may also be toggled manually.

Approval Inbox and Delivery Notes are not sidebar modules.

## Icons and Sidebar Control

- Navigation uses internal SVG icons with a consistent Lucide-like 16 px visual style.
- Text abbreviations such as `D`, `PO`, `DN`, and `RM` are removed.
- Icons include accessible menu labels; icons themselves are hidden from screen readers.
- The desktop collapse/expand control moves into the sidebar brand area.
- In collapsed mode the brand mark, navigation icons, tooltips, and expand control remain available.
- The content header does not contain a desktop collapse button.
- The mobile hamburger remains in the content header because the mobile sidebar is a drawer.

## Supplier Order Document Ownership

Delivery Notes and Kanban labels are documents generated from an approved Supplier Order, not a separate navigation domain.

- `/supplier-orders` is the document access center.
- Its future row/detail actions expose `View/Download PO`, `View/Download DN`, and `View/Download Kanban`.
- The index includes a compact Documents column indicating which generated document sets exist.
- One PO may own multiple DN documents, and each DN may contain multiple PO material allocations and Kanban lots according to the approved relational model.
- The legacy `/delivery-notes` route redirects to `/supplier-orders` so bookmarks do not fail.
- No standalone Delivery Notes link remains in navigation.

The current no-data index exposes the Documents column but does not fabricate documents or enable download actions before a PO exists.

## Notification Center and Approvals

The application header contains a notification bell near the current user.

- The bell is visible to every authenticated user.
- A badge shows the unread count and is hidden when the count is zero.
- Clicking the bell opens a compact dropdown.
- Notification contents are filtered by permission and user relevance.
- Users with approval permission receive pending PO approval notifications.
- Future notification types may include PO status changes, completed receiving, email delivery failure, and Sage integration failure.
- The dropdown ends with `View all notifications`.
- `/approvals` remains a real work page reached from an approval notification or the all-notifications view, but it is not listed in the sidebar.

Until transaction and notification APIs exist, the center consumes an explicit empty snapshot, shows `Belum ada notifikasi`, and never displays a fabricated badge or event.

## Reports

`/reports` is an MVP index route with compact real-data-only filters for:

- date range;
- supplier;
- raw material;
- PO reference;
- receiving reference;
- operational status.

The initial page provides the filter and export layout with an honest empty state. Report results and export become active as transactional APIs are implemented. Supplier filtering is a required report capability.

## Settings

The following MVP index routes are available:

- `/settings/users`: username/password user administration and status.
- `/settings/roles`: RBAC roles and permission assignments.
- `/settings/smtp`: SMTP server configuration and test-email action.
- `/settings/email-log`: sent/failed email history for director and supplier communication.

The pages initially use the shared compact index/form shell. Controls that are not yet connected remain clearly disabled rather than silently accepting changes. Sage outbound-agent configuration can be added under Settings when that integration slice begins.

## Component Boundaries

- `Icon`: typed internal SVG icon renderer.
- `navigationGroups`: ordered navigation and expandable Settings metadata.
- `Sidebar`: active routing, group expansion, tooltips, and collapse control.
- `NotificationCenter`: trigger, unread badge, dropdown, permission-filtered items, and empty state.
- `SupplierOrdersPage`: PO list plus document availability columns.
- `ReportsPage`: report filter/export shell.
- Settings pages: focused route configurations using existing shared index patterns.

## State and Data Flow

1. `AppShell` obtains the authenticated user and permissions from `/api/auth/me`.
2. Navigation renders the approved order and uses the pathname to expand Settings when appropriate.
3. Sidebar collapsed preference continues to use `localStorage['order-stock.sidebar-collapsed']`.
4. `NotificationCenter` receives an empty typed notification snapshot until its API is implemented.
5. Selecting a future approval notification navigates to `/approvals` or a PO-specific approval detail.
6. The legacy Delivery Notes route performs a server redirect to Supplier Orders.

## Error and Empty States

- Notification API unavailable: show a compact retryable state inside the dropdown without affecting navigation.
- No notifications: show `Belum ada notifikasi`; render no badge.
- No report data: preserve selected filters and show an empty report state.
- No PO documents: show neutral unavailable indicators and no active downloads.
- A user without a Settings permission must not receive actionable Settings navigation when permission filtering is connected.

## Verification

- Navigation tests verify group order, absence of Approval Inbox and Delivery Notes, SVG icons, and Settings expansion.
- App shell tests verify the desktop toggle exists inside the sidebar and not the content header.
- Notification tests verify zero notifications hide the badge and render the approved empty copy.
- Supplier Order tests verify PO, DN, and Kanban document headings/actions belong to the PO index configuration.
- Route tests verify Reports and all four Settings pages resolve.
- A redirect test verifies `/delivery-notes` resolves to `/supplier-orders`.
- Frontend tests, lint, typecheck, production build, Docker rebuild, authentication, and route smoke tests must pass.

## Deferred Work

- Notification persistence, read/unread mutations, and real-time delivery.
- Report query endpoints and downloadable report files.
- Settings CRUD mutations and secret encryption.
- PO/DN/Kanban document generation and downloads.
- Sage outbound-agent configuration UI.
