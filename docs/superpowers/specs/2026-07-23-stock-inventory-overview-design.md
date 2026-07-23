# Stock Inventory Overview Design

## Goal

Add a read-only Inventory module that shows current raw-material stock and the individual Kanban lots available for outgoing consumption.

## Scope

This phase includes:

- A `Stock Inventory` submenu under Logistics, before Receiving.
- Real-time stock totals derived from Kanban lots whose status is `IN_STOCK`.
- All active raw materials, including materials with zero stock.
- Summary metrics, stock filters, and an in-stock Kanban detail view.

This phase excludes Stock Taking, Stock Adjustment, Stock Moving, barcode reprinting, and editable stock balances.

## Architecture

The backend adds a focused inventory package with read-only HTTP endpoints protected by `inventory.view`. The stock overview query starts from active `raw_materials` and left-joins the current `IN_STOCK` Kanban population so zero-stock materials remain visible.

The existing append-only `inventory_ledger_entries` remains the audit trail, but it is not used as the balance source in this MVP. Current stock is derived from Kanban lifecycle state to avoid introducing a duplicated balance table or a reconciliation process.

## Stock Rules

- `Available Kanban` is the number of Kanban lots with status `IN_STOCK`.
- `Stock Quantity` is the sum of the quantity of those lots in the raw material's base unit.
- A material with no `IN_STOCK` lots has zero available Kanban and zero stock quantity.
- `Out of Stock` means stock quantity equals zero.
- `Low Stock` means stock quantity is greater than zero and less than or equal to the raw material's minimum stock.
- `In Stock` means stock quantity is greater than minimum stock.
- Only active raw materials appear in the overview.
- Only `IN_STOCK` lots appear in the Kanban detail list. `ISSUED` and `CONSUMED` lots are excluded.

## API

### `GET /inventory/stock`

Permission: `inventory.view`

Query parameters:

- `search`: optional material-code or material-name search.
- `supplierId`: optional supplier UUID.
- `status`: optional `IN_STOCK`, `LOW_STOCK`, or `OUT_OF_STOCK`.

Response:

```json
{
  "summary": {
    "totalRawMaterials": 12,
    "totalInStockKanban": 48,
    "lowStockMaterials": 3,
    "outOfStockMaterials": 2
  },
  "items": [
    {
      "rawMaterialId": "uuid",
      "itemCode": "RM-001",
      "rawMaterialName": "Bolt",
      "supplierId": "uuid",
      "supplierName": "Supplier A",
      "availableKanban": 4,
      "stockQuantity": "2000.000000",
      "baseUnitCode": "PCS",
      "minimumStock": "1000.000000",
      "stockStatus": "IN_STOCK"
    }
  ]
}
```

The summary reflects all active raw materials and is not changed by search, supplier, or status filters. Filters affect only `items`, so the cards remain a stable company-wide snapshot.

### `GET /inventory/stock/:rawMaterialId/kanbans`

Permission: `inventory.view`

Returns only the selected material's `IN_STOCK` Kanban lots:

```json
{
  "rawMaterialId": "uuid",
  "itemCode": "RM-001",
  "rawMaterialName": "Bolt",
  "kanbans": [
    {
      "kanbanLotId": "uuid",
      "kanbanId": "KB-202607-00001",
      "deliveryNoteNumber": "DN-202607-00001",
      "poNumber": "PO-202607-00001",
      "quantity": "500.000000",
      "baseUnitCode": "PCS",
      "receivedDate": "2026-07-23"
    }
  ]
}
```

An unknown, inactive, or inaccessible raw material returns `404`. An active material with no in-stock lots returns `200` with an empty `kanbans` array.

## User Interface

Route: `/inventory`

The page follows the existing compact Linear/Notion-inspired layout:

- Four compact summary cards:
  - Total Raw Materials
  - In-Stock Kanban
  - Low Stock
  - Out of Stock
- A search input.
- Supplier and stock-status filters.
- A compact stock overview table:
  - Item Code
  - Raw Material
  - Supplier
  - Available Kanban
  - Stock Quantity
  - Base Unit
  - Minimum Stock
  - Status
  - Action
- Colored status pills:
  - Green: In Stock
  - Amber: Low Stock
  - Red: Out of Stock
- An `Open` action displays a centered, read-only modal containing the in-stock Kanban table:
  - Kanban ID
  - DN Number
  - PO Number
  - Quantity
  - Received Date

Quantity fields use the shared compact number formatter, removing trailing zeroes while preserving meaningful decimals.

## Navigation and Access

- Add `Stock Inventory` under the collapsible Logistics group before Receiving.
- The navigation item and page require `inventory.view`.
- Unauthorized API access returns `403`.
- Existing tenant scoping and PostgreSQL row-level security remain in force through the authenticated request context.

## Error and Empty States

- Overview load failure shows an English retryable error state.
- A filter with no matching materials shows `No stock records found`.
- A material with no available lots shows `No in-stock Kanban available`.
- Kanban detail loading is contained inside the modal and does not replace the overview.

## Testing

- Backend domain tests cover stock-status classification at zero, below/equal to minimum, and above minimum.
- Backend integration tests prove zero-stock materials are retained, only `IN_STOCK` lots are counted, tenant isolation is preserved, filters work, and detail results exclude consumed lots.
- HTTP tests cover permission enforcement, response shape, invalid filters, and not-found behavior.
- Frontend tests cover navigation, summary rendering, compact number formatting, filtering, colored statuses, modal detail loading, empty states, and API failures.
- Full Go tests, Go vet, frontend tests, TypeScript checking, and the production build must pass before merge.
