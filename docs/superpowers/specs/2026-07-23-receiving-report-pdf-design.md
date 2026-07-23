# Receiving Report PDF Design

## Scope

Implement one report for the MVP: Receiving Report. Do not add separate Supplier Order, Inventory, or Outgoing reports, and do not add Excel export.

## Screen

The Reports page initially shows only the filter panel and must not fetch or display receiving transactions automatically. Filters are:

- From Date (required)
- To Date (required)
- Supplier
- Reference search covering Receiving Number, DN Number, and PO Number

Apply Filters becomes visually active only after both required dates are filled. Supplier and Reference remain optional. After a successful Apply Filters action, the page displays the matching results table and enables Export PDF. Reset clears the filters, hides the results and export action, and restores the initial filter-only state.

The supplier filter is populated from active master suppliers. The table displays:

- Receiving Number
- Receiving Date
- DN Number
- PO Number
- Supplier
- Raw Material
- Kanban Received
- Received Quantity
- Outstanding Quantity
- Sage Number
- Created By

One receiving can contain multiple raw materials, so the report uses one row per receiving material.

## Backend

Add a focused report module inside the modular monolith. It reads receiving headers, receiving lines, PO/DN references, supplier, raw material, units, Sage receipt number, and creator using tenant-scoped queries.

Endpoints:

- `GET /reports/receiving` returns filtered JSON rows and totals.
- `GET /reports/receiving.pdf` returns the same filtered result as an A4 landscape PDF.

Accepted query parameters are `fromDate`, `toDate`, `supplierId`, and `search`. The report endpoints require both dates. Missing or invalid dates, or a date range where From Date exceeds To Date, return a typed validation response.

## PDF

The PDF is A4 landscape and contains:

- Title: Receiving Report
- Generated timestamp
- Selected period and supplier summary
- Compact transaction table
- Footer totals for Kanban received and received quantity
- Page number on every page

Long reports continue across pages with the table header repeated. Empty filtered results still export a valid PDF showing “No receiving transactions found.”

## Security and Consistency

- Existing session authentication and tenant context are required.
- Report reads are tenant-scoped.
- The PDF and screen use the same backend filtering logic.
- Price information is excluded.
- No database schema change is required.

## Tests

- Repository filtering, tenant isolation, flattened material rows, and totals.
- HTTP query validation and PDF response headers.
- PDF content, empty-state export, and pagination.
- Frontend initial filter-only state, required-date behavior, no automatic report fetch, filtering, reset, data table, error state, and PDF export URL.
