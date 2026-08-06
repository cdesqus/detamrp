# Material Planning Order Design

## Purpose

Add a material-planning workflow driven by confirmed Sales Orders (SLOs). The workflow calculates gross raw-material requirements from Finished Goods and their active Bills of Materials (BOMs), lets purchasing users adjust the recommendation, and produces a printable PDF reference. It does not calculate stock availability and does not create Supplier Orders.

## Scope and decisions

- Sales Order numbers use `SLO-YYYYMM-####` and contain no pricing in the initial release.
- A Finished Good belongs to exactly one Customer. The first release uses one Finished Good code; customer part numbers are out of scope.
- A Finished Good has an active BOM. A BOM defines raw material usage per one Finished Good unit.
- BOM rows reference the existing Raw Materials master. Supplier and unit are displayed from Raw Materials and are not duplicated or editable in the BOM.
- Each raw material has one supplier.
- One Planning Order may include complete SLOs from multiple customers.
- An SLO must be planned in full and may belong to only one active Planning Order revision family.
- Planning recommendations are gross requirements only; current stock and incoming orders are intentionally excluded.
- Planning Orders are reference documents only. They do not create, draft, or link to Supplier Orders.
- All user-facing labels and generated PDFs are in English.

## Domain model

### Customer

Stores the customer identity and contact details used by Finished Goods and Sales Orders.

### Finished Good

Stores a product sold to exactly one Customer, its code, name, unit, description, and active status.

### BOM

A BOM belongs to one Finished Good and has a revision (`R0`, `R1`, ...), effective date, notes, and lifecycle status: `Draft`, `Active`, `Revised`, or `Inactive`.

Each BOM component stores a Raw Material reference and `Usage Qty` per one Finished Good unit. A BOM must contain at least one component, must not contain duplicate Raw Materials, and all usage quantities must be greater than zero. Only one BOM revision per Finished Good may be `Active`.

### Sales Order

Sales Orders use the `SLO-YYYYMM-####` number format and contain Customer, order date, delivery date, Finished Goods with quantities, notes, and status: `Draft`, `Confirmed`, or `Cancelled`.

Finished Goods selectable on an SLO are filtered to the selected Customer. An SLO can only be confirmed when every line has an Active BOM.

### Planning Order

Planning Orders use `PLN-YYYYMM-####-R0` numbering, with subsequent revisions `R1`, `R2`, and so on. The document stores its source SLOs, the BOM revisions used for calculation, material requirement rows, adjustment notes, lifecycle status, and revision history.

Statuses:

- `Draft`: editable and reserves its selected SLOs.
- `Finalized`: official active reference; locked for editing.
- `Revised`: an older finalized version replaced by a newer revision.
- `Cancelled`: no longer valid and releases its SLOs.

## Planning workflow

1. User opens **Create Planning Order**.
2. The selector shows only confirmed SLOs that do not belong to an active Planning Order. SLOs already used by a planning order are hidden from this selector.
3. User selects one or more complete SLOs, including SLOs from different customers.
4. The system reads each selected Finished Good's Active BOM and calculates each component requirement:

   `Required Qty = Sales Order Qty x BOM Usage Qty per Unit`

5. The system groups identical Raw Materials across all selected SLOs. Supplier and unit are read from the Raw Materials master.
6. The generated table shows the calculation and creates a `Recommended Qty`. Recommended Qty is read-only.
7. User may enter a `Final Qty`. If Final Qty differs from Recommended Qty, an adjustment note is required.
8. User saves the document as Draft or Finalizes it. Finalization locks the document and marks every source SLO as planned.
9. The user can print or download a PDF reference. No Supplier Order is generated.

An SLO used by a Draft Planning Order is unavailable to other planning documents. If that draft is deleted or the planning order is Cancelled, the SLO becomes available again. A finalized planning order is changed only through **Create Revision**, which copies the current document and increments its revision. The previous version becomes `Revised` after the new revision is finalized.

## User interface

### Planning Order list

Shows Planning Number, Revision, Planning Date, source SLOs, material count, status, creator, last update, and actions. Filters include number, SLO, date, status, supplier, and material.

### Generate/edit screen

The header shows planning number, revision, planning date, selected SLOs, and status. The requirements table contains:

- Raw Material
- Supplier
- Requirement Breakdown by SLO and Finished Good
- Recommended Qty (read-only)
- Final Qty (editable)
- Unit
- Notes

The screen displays only eligible SLOs. Helper text explains that the list contains confirmed Sales Orders that have not been planned.

### Detail screen

Shows source SLOs and customers, Finished Goods, BOM revisions used, material requirements, adjustment notes, revision history, and activity history. Actions depend on status: edit/finalize for Draft, create revision and print/export for Finalized, and view/print for Revised or Cancelled.

### BOM screens

The BOM list is organized by Finished Good and shows BOM number, revision, component count, status, and update time. The edit screen starts with Finished Good, then lists Raw Materials, their read-only Supplier and Unit, Usage Qty, and notes. A new revision is created instead of editing an Active BOM.

## PDF document

The Planning Order PDF contains company branding, Planning Number and Revision, date, status, source SLOs and customers, a material table, requirement breakdown, recommended and final quantities, adjustment notes, creator/finalizer, and print date.

- Draft PDFs carry a `DRAFT` watermark.
- Revised versions carry a `REVISED` watermark.
- Cancelled versions carry a `CANCELLED` watermark.
- Finalized PDFs have no lifecycle watermark.

## Validation and error handling

- Prevent confirmation of SLOs without an Active BOM.
- Prevent duplicate Raw Materials within one BOM.
- Require positive quantities for SLO lines, BOM usage, and Final Qty.
- Require an adjustment note when Final Qty differs from Recommended Qty.
- Prevent selecting an SLO already reserved by a Draft or Finalized Planning Order.
- Prevent edits to Finalized Planning Orders and Active BOM revisions.
- If a selected SLO becomes invalid before save, reject finalization and identify the affected SLO.
- Use a transaction when saving or finalizing a Planning Order so SLO reservation and material rows remain consistent.

## Acceptance criteria

- A user can create an SLO with an automatically generated `SLO-YYYYMM-####` number and confirm it only when all Finished Goods have Active BOMs.
- A user can select complete, eligible SLOs from multiple customers and generate consolidated gross material requirements.
- The system groups the same Raw Material across SLOs and calculates quantities from BOM usage without referencing stock.
- A user can adjust quantities with required notes, finalize the Planning Order, and download an English PDF.
- Planning Orders never create or draft Supplier Orders.
- Used SLOs are hidden from new Planning Order selection and become available again only after their Planning Order is deleted or Cancelled.
- Finalized documents remain immutable; revisions preserve the previous version and clearly mark it `Revised`.
