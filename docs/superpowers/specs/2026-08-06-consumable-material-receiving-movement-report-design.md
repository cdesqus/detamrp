# Consumable Material and Inventory Movement Report Design

## Purpose

Support consumable materials without creating a second material master. Consumables use the existing Raw Materials master, receive through Delivery Notes without Kanban tracking, remain visible in inventory, and can be separated from standard raw-material purchasing and reporting.

## Material classification

Add a required `Material Type` field to Raw Materials:

- `Raw Material` — standard materials that use Kanban-based receiving.
- `Consumable` — materials that use quantity-based receiving without Kanban.

Keep `Category` as a separate configurable classification. Category may contain values such as Metal, Packaging, Safety, or Stationery and must not control receiving behavior.

## Supplier Orders

Keep one Supplier Order module. Add an order type or material-type filter so purchasing can distinguish:

- Standard Material Order
- Consumable Order

An order containing mixed Material Types should be rejected or split before submission so the document flow remains unambiguous. Supplier, approval, email, and PDF behavior remain shared.

## Receiving behavior

Receiving remains one module and continues to use Delivery Notes.

### Raw Material

- Kanban validation remains required.
- Each received lot is recorded with Kanban identity and quantity.
- Inventory movement includes Kanban ID.

### Consumable

- Kanban validation is not required.
- User selects the consumable and records received quantity directly.
- Inventory quantity is increased through the same ledger.
- Inventory movement has no Kanban ID.

The system determines the flow from the Raw Material's `Material Type`; users do not choose a receiving mode manually.

## Inventory Movement Report

The report belongs under the existing Reports module and reads the inventory ledger. It does not create a second inventory system.

Filters:

- Date range
- Material Type
- Category
- Raw Material
- Supplier
- Movement Type

Movement types include Receiving, Outgoing, Adjustment, Return, and Opening Balance when applicable. The report does not require a separate opening/closing balance calculation for the selected period.

Columns:

| Date | Material | Material Type | Category | Movement Type | Qty In | Qty Out | Balance | Reference Document | Kanban ID | User |
|---|---|---|---|---|---:|---:|---:|---|---|---|

Kanban ID is populated for Raw Material movements and blank for Consumable movements.

### PDF export

Formal printable report with company branding, applied filters, generation date, summary counts, and the movement table.

### Excel export

Tabular export using the same filters, preserving full transaction detail for sorting and analysis.

## Validation and acceptance criteria

- Every Raw Material has exactly one Material Type.
- Raw Material receiving rejects records without valid Kanban information.
- Consumable receiving accepts quantity directly and does not require Kanban.
- Both flows update the same inventory ledger and stock balance.
- Supplier Orders can be identified as Standard Material or Consumable orders and cannot mix types in one submitted document.
- Inventory Movement Report supports all stated filters and exports to valid English PDF and Excel files.
- Existing Raw Material, Supplier Order, Receiving, and Inventory workflows continue to work for `Raw Material` records.
