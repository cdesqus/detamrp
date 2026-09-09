# Sales Material Planning Design

## Purpose

Add a simple sales-driven material planning workflow to Deta MRP. The system calculates gross raw-material requirements from confirmed Sales Orders and active BOMs, allows users to adjust the resulting quantities, and produces a finalized Planning Order and PDF for the purchasing team.

Planning Orders are references only. They do not consider inventory, incoming Supplier Orders, or automatically create Supplier Orders.

## Scope

The feature adds five connected areas:

1. Customer master data.
2. Finished Good master data.
3. Versioned Bills of Materials (BOMs).
4. Sales Orders identified with an `SLO` prefix.
5. Material Planning Orders with revisions and PDF output.

The existing Raw Materials and Suppliers remain the authoritative source for material, unit, and supplier information.

## Out of Scope

- Sales prices, taxes, discounts, and Sales Order monetary totals.
- Stock deductions or inventory availability calculations.
- Safety stock, scrap, waste percentages, and incoming-order calculations.
- Production orders, work in progress, routing, and Finished Good inventory.
- Automatic creation or updating of Supplier Orders.
- Splitting one Sales Order across multiple Planning Orders.
- Multiple suppliers or substitute suppliers for one Raw Material.

## Core Relationships

```text
Customer
  -> Finished Goods
      -> BOM revisions
          -> existing Raw Materials
              -> existing Supplier

Customer
  -> Sales Orders (SLO)
      -> Finished Goods
          -> Planning Order
              -> calculated Raw Material requirements
```

Each Finished Good belongs to exactly one Customer. Each Raw Material continues to belong to exactly one Supplier. A BOM references Raw Materials; it does not duplicate supplier or unit ownership.

## Customer Master

Customer records contain:

- Customer code.
- Name.
- Address.
- Contact person.
- Email.
- Phone.
- Active or inactive status.

Inactive Customers remain visible in historical documents but cannot be used in new Finished Goods or Sales Orders.

## Finished Good Master

Finished Good records contain:

- One unique Finished Good code.
- Name.
- Required Customer.
- Unit.
- Description.
- Active or inactive status.

Only one product code is required in the initial version. Separate internal and customer part numbers are not included.

When a Customer is selected on a Sales Order, the Finished Good picker only shows active products belonging to that Customer.

## BOM

### BOM List

The BOM list is organized by Finished Good and shows:

- BOM number.
- Finished Good.
- Customer.
- Revision.
- Material count.
- Status.
- Last updated date.

Users can filter by BOM number, Finished Good, Customer, and status.

### BOM Header

A BOM header contains:

- Automatically generated BOM number.
- Finished Good.
- Customer, derived from the Finished Good and read-only.
- Revision, beginning at `R0`.
- Output quantity, defaulting to `1`.
- Finished Good unit, derived from the Finished Good.
- Effective date.
- Notes.
- Status: `Draft`, `Active`, `Revised`, or `Inactive`.

Only one BOM revision may be `Active` for a Finished Good at a time.

### BOM Materials

Each component row contains:

- Raw Material reference.
- Supplier, derived from Raw Material and read-only.
- Usage quantity per BOM output quantity.
- Unit, derived from Raw Material and read-only.
- Notes.

Usage quantity must be greater than zero. A Raw Material cannot appear twice in the same BOM, and a BOM must contain at least one material before activation.

### BOM Revisions

Draft BOMs can be edited. An Active BOM is changed by creating a copied revision. Activating the new revision changes the previous Active revision to `Revised`.

Historical Planning Orders retain snapshots of the BOM revision and calculated values used at generation time.

## Sales Orders

### Number and Fields

Sales Order numbers use `SLO-YYYYMM-####` and are assigned automatically.

A Sales Order contains:

- SLO number.
- Customer.
- Order date.
- Target delivery date.
- One or more Finished Good lines with quantity and unit.
- Notes.
- Status.
- Planning Order reference when planned.

Prices and order values are excluded.

### Lifecycle

Sales Order statuses are:

- `Draft`: editable and excluded from planning.
- `Confirmed`: locked for planning eligibility.
- `Cancelled`: not valid for planning.

No approval workflow is required. A user explicitly confirms a Draft Sales Order.

A Sales Order can only be confirmed if every line has an active BOM. A confirmed order requiring changes is cancelled and duplicated into a new SLO. The new order records a reference to the cancelled source order.

### Planning Eligibility

A confirmed SLO may belong to only one active Planning Order family. The entire SLO is planned together; individual lines or partial quantities cannot be split across Planning Orders.

The Create Planning Order screen only shows confirmed SLOs that do not belong to a Draft or Finalized Planning Order. Used SLOs are hidden rather than shown as unavailable.

As soon as a Planning Order Draft is saved, its SLOs disappear from other Planning Order selection screens. Cancelling the Planning Order or deleting its Draft releases those SLOs for selection again.

The Sales Order detail page shows the linked Planning Order so users can trace why an SLO is no longer selectable.

## Planning Orders

### Purpose and Numbering

A Planning Order consolidates gross Raw Material requirements from one or more complete Sales Orders. It is an independent purchasing reference and has no automatic Supplier Order integration.

Planning Order numbers use `PLN-YYYYMM-####-R0`. Revisions retain the base number and increment the suffix to `R1`, `R2`, and so on.

### List Page

The list shows:

- Planning number and revision.
- Planning date.
- SLO references.
- Total material count.
- Status.
- Created by.
- Last updated time.
- Contextual actions.

Filters cover planning number, SLO number, date, status, Supplier, and Raw Material.

### Generation Flow

1. The user starts a new Planning Order.
2. The system shows only eligible SLOs.
3. The user selects one or more complete SLOs, including SLOs from different Customers.
4. The system reads the active BOM snapshot for every Finished Good line.
5. It calculates each material requirement as:

   `SLO Finished Good quantity / BOM output quantity * BOM material usage quantity`

6. Requirements for the same Raw Material are summed across Finished Goods, SLOs, and Customers.
7. The resulting Planning Order Draft is saved and its SLOs become unavailable to other Planning Orders.

The calculation does not read current inventory or incoming quantities.

### Material Requirement Table

Each summarized row displays:

- Raw Material code and name.
- Supplier derived from the Raw Material snapshot.
- Requirement breakdown by Customer, SLO, Finished Good, and BOM revision.
- Recommended quantity calculated by the system.
- Final quantity entered by the user.
- Unit.
- Adjustment notes.

Recommended quantity is read-only. Final quantity initially equals Recommended quantity and remains editable while the Planning Order is a Draft. Adjustment notes are required whenever Final quantity differs from Recommended quantity.

### Detail Page

The detail page shows:

- Planning number and revision.
- Planning date and status.
- Source SLOs and their Customers.
- Material summary and calculation breakdown.
- Creator and creation time.
- Finalizing user and finalization time.
- Previous and next revision links.
- Revision and activity history.

### Lifecycle and Revisions

Planning Order revision statuses are:

- `Draft`: editable and deletable.
- `Finalized`: official, active, and locked.
- `Revised`: an older finalized version replaced by a newer finalized revision.
- `Cancelled`: no longer valid.

Finalizing validates all sources, calculations, quantities, and required adjustment notes. A Finalized Planning Order cannot be edited directly.

`Create Revision` copies a Finalized Planning Order into the next Draft revision. The prior version remains Finalized until the new revision is finalized. When that occurs, the prior version becomes `Revised`, and the new version becomes the active Finalized version. A Revised document identifies the revision that replaced it.

Revisions remain in the same Planning Order family and retain the same source SLOs. The `Cancel Planning` action cancels the currently active revision and closes the entire Planning Order family; only that family-level cancellation releases its SLOs. Creating or discarding a revision Draft does not release the SLOs because the preceding Finalized revision remains active. Deleting the initial `R0` Draft releases its SLOs because no finalized revision exists yet.

## PDF Document

Planning Orders can be printed or exported as English-language PDFs.

The PDF contains:

- Company logo and identity.
- Planning number and revision.
- Planning date and status.
- Source SLOs and Customers.
- Material table with code, name, Supplier, Recommended quantity, Final quantity, unit, and adjustment notes.
- Requirement breakdown by SLO and Finished Good.
- Created-by and finalized-by information.
- Print timestamp.

A Draft PDF has a `DRAFT` watermark. A PDF from an older revision has a `REVISED` watermark and identifies the replacement revision. A Cancelled PDF has a `CANCELLED` watermark. Finalized PDFs have no watermark.

## Snapshot and Audit Rules

Planning calculations and PDFs must remain historically stable even when master data changes. Each Planning Order stores the relevant display snapshots, including Customer, Finished Good, Raw Material, Supplier, unit, BOM revision, source quantities, recommended quantities, final quantities, and notes.

Lifecycle changes and revision creation are written to the existing activity-log mechanism with actor and timestamp information.

## Validation and Error Handling

- Prevent confirming an SLO when a Finished Good is inactive or lacks an active BOM.
- Prevent selecting or saving an SLO already held by another active Planning Order family.
- Recheck SLO eligibility transactionally when saving a Planning Order to avoid concurrent duplicate planning.
- Reject BOM material quantities and SLO quantities that are zero or negative.
- Require adjustment notes when Final and Recommended quantities differ.
- Prevent activating multiple BOM revisions for the same Finished Good.
- Prevent editing Finalized, Revised, or Cancelled Planning Orders.
- Show actionable messages identifying the affected SLO, Finished Good, BOM, or material.

## Testing Strategy

Backend tests cover:

- Customer and Finished Good ownership rules.
- BOM validation, activation, and revision lifecycle.
- SLO numbering, confirmation, cancellation, and duplication.
- Planning eligibility and protection against duplicate SLO use.
- Multi-customer and multi-SLO material consolidation.
- BOM output-quantity calculations and decimal quantities.
- Required adjustment notes.
- Planning revision transitions and SLO release on cancellation.
- Historical snapshot stability.
- PDF contents and watermarks by status.

Frontend tests cover:

- Finished Good filtering by Customer.
- BOM material selection with read-only Supplier and unit fields.
- Eligible-SLO-only selection.
- Material breakdown visibility.
- Recommended quantity locking and Final quantity editing.
- Status-specific actions and revision navigation.
- Validation and empty states.

End-to-end coverage verifies the primary path:

`Customer -> Finished Good -> Active BOM -> Confirmed SLO -> Generated Planning Order -> Adjusted quantity -> Finalized PDF -> Revision`

## Acceptance Criteria

- Users can manage Customers and Customer-owned Finished Goods.
- Users can create and revise BOMs using existing Raw Materials, Suppliers, and units.
- Users can create price-free Sales Orders numbered with the `SLO` prefix and confirm them without approval.
- Users can generate one Planning Order from one or more complete, eligible SLOs across multiple Customers.
- The system calculates and consolidates gross material requirements without stock calculations.
- Users can adjust Final quantities while retaining immutable Recommended quantities and calculation details.
- One SLO cannot belong to multiple active Planning Order families.
- Finalized Planning Orders are locked and can be changed through explicit revisions.
- Planning Order PDFs correctly represent Draft, Finalized, Revised, and Cancelled documents.
- No Supplier Order is created or modified by the Planning Order workflow.
