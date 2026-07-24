# Live Operational Dashboard Design

## Goal

Replace the static empty dashboard snapshot with tenant-scoped operational data from PostgreSQL. The dashboard loads the last 30 calendar days by default and lets users filter the same view by date range and supplier.

## User Experience

- Initial load uses `today - 29 days` through `today`, in the application timezone.
- Filters: From Date, To Date, Supplier, Apply, and Reset.
- Supplier is optional; an empty supplier means all suppliers.
- Applied filters are stored in the URL query string so refresh and shared links preserve the view.
- Apply rejects an invalid range where From Date is after To Date.
- A loading state preserves the dashboard layout. A request error shows a retry action without replacing the entire application shell.
- Individual charts show a compact empty state when their filtered dataset has no rows.

## Dashboard Content

### KPI cards

1. **Pending Approval**: count of POs currently in `PENDING_APPROVAL`. Supplier filter applies; date filter uses `order_date`.
2. **Open PO**: count of POs currently in `APPROVED` or operationally derived `PARTIALLY_RECEIVED`. Supplier and PO order-date filters apply.
3. **Received Kanban**: count of distinct Kanban lots received during the selected period. Supplier filter applies; date filter uses `receiving_date`.
4. **Outstanding Kanban**: issued Kanban lots not yet received for POs in scope.
5. **Current Stock**: count of Kanban lots currently in `IN_STOCK`. This is a current-state metric and intentionally ignores the date filter, while still respecting the supplier filter.

All Kanban metrics count complete Kanban lots, not base-unit quantities.

### Charts

- **PO & Receiving Trend**: daily ordered Kanban versus received Kanban across the selected period. Missing dates return zero values.
- **PO Status**: PO count grouped by the user-facing status already used by the Supplier Order module.
- **Outstanding by Supplier**: outstanding Kanban grouped by supplier, ordered descending, limited to the top 10.
- **Latest Activity**: latest 10 events from PO creation/submission/approval, completed receiving, and completed outgoing. Date and supplier filters apply where the event has a supplier relationship. Outgoing activity without a supplier filter may appear; when a supplier is selected, outgoing is included only when its scanned lots originated from that supplier.

## API

Add one authenticated endpoint:

`GET /dashboard?from=YYYY-MM-DD&to=YYYY-MM-DD&supplierId=UUID`

The response contains:

- normalized filter values;
- all five KPI metrics;
- trend rows;
- PO status rows;
- outstanding-supplier rows;
- recent activity rows.

The backend supplies default dates when they are omitted. It validates dates, range order, supplier ownership, authentication, and tenant scope. The endpoint uses the existing tenant transaction/RLS pattern. The first version does not cache results because the prototype dataset is small and users expect immediate transaction visibility.

## Data and Status Rules

- PO commercial values and historical snapshots remain sourced from PO records; dashboard queries never recalculate old POs from current master-data prices.
- Derived PO statuses must use the same logic as the Supplier Order list so counts cannot disagree between modules.
- Cancelled and rejected POs do not contribute to ordered, received, outstanding, or stock quantities unless an existing received lot already exists; current inventory truth always comes from `kanban_lots.status`.
- A receiving is counted only when its transaction is completed.
- Tenant isolation is enforced in every query and by PostgreSQL RLS.

## Frontend Visualization

Use lightweight SVG/CSS charts inside the existing compact dashboard panels:

- two-series line chart for the trend;
- donut chart for PO status;
- horizontal bars for outstanding suppliers.

No chart library is added for the MVP. Tooltips expose exact values, legends use the existing restrained color palette, and text summaries/labels keep the charts understandable without relying only on color.

## Testing and Acceptance

- Store/API tests cover default period, explicit dates, supplier filtering, zero-data responses, invalid filters, cross-tenant isolation, and the current-stock date exception.
- Frontend tests cover initial fetch, URL filters, apply/reset, loading/error/empty states, KPI rendering, and chart rendering.
- A runtime smoke test verifies that existing local PO, receiving, Kanban, and outgoing data produces non-empty dashboard values.
- Existing backend and frontend test suites, lint, typecheck, production build, and Docker startup must remain successful.

