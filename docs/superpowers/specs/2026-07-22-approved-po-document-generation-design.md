# Approved PO Document Generation Design

## Scope

This change adds navigation back from every Supplier Order detail and implements automatic Delivery Note and Kanban lot generation for approved purchase orders. It covers future approvals and approved orders that existed before this feature.

## User experience

- Every Supplier Order detail shows a compact `Back to Supplier Orders` action above the title.
- An approved PO detail shows a Documents section containing its DN number, Kanban count, and document actions.
- The Supplier Order index uses the same generated document data for its DN Documents and Kanban Labels columns.
- Pending, rejected, cancelled, and draft orders never expose generated inbound documents.
- Document generation errors must be visible to administrators; an order must not silently appear approved without its required operational records.

## Data model

- `delivery_notes` is the header and belongs to exactly one purchase order. The MVP creates exactly one system-issued DN per approved PO.
- `delivery_note_lines` is the junction between a DN and PO lines. A DN contains all lines of its PO.
- `kanban_lots` belongs to exactly one delivery note line and one PO line. One row represents one physical Kanban lot/barcode.
- Tenant-scoped foreign keys prevent cross-tenant relationships.
- Unique constraints enforce one DN per PO, one DN line per PO line within that DN, unique DN numbers, and unique Kanban IDs.
- Quantity checks require each Kanban lot to use the approved PO line's `qty_per_kanban_snapshot`; generated lots cannot exceed the ordered Kanban quota.

## Numbering

- DN numbers use `DN-YYYYMM-#####` with a tenant/month sequence.
- Kanban IDs use `KB-YYYYMM-######` with a tenant/month sequence.
- Numbers are allocated under database row locks to remain unique during concurrent approvals.

## Approval transaction

Approving a PO performs the following atomically in the same tenant transaction:

1. Lock and validate the pending approval and PO.
2. Mark the approval and PO approved.
3. Create the DN header if it does not already exist.
4. Create one DN line for every approved PO line.
5. Create exactly `total_kanban` Kanban lots for each DN line.
6. Commit all records together.

Generation is idempotent. A retry observes the unique PO/DN relationship and never creates duplicate DN lines or Kanban lots.

## Existing approved orders

A migration backfills every existing `APPROVED` PO using the same structure. The migration is idempotent and preserves the PO's approved quantities. In the current prototype this includes `PO-202607-00001` and `PO-202607-00003`.

## API and UI data flow

- Purchase Order detail responses include a compact generated-document summary.
- The detail page renders the summary directly and refreshes it after an approval decision.
- The Supplier Order index renders DN and Kanban availability from API data rather than placeholder values.
- Initial document actions expose printable browser documents. PDF generation can use the browser print-to-PDF path while keeping the endpoint contract ready for dedicated PDF rendering.

## Failure handling

- Any generation failure rolls back approval, preventing an `APPROVED` PO without DN/Kanban records.
- Database constraints guard duplicate generation and cross-PO mappings.
- Existing approved-order backfill fails the migration rather than leaving a partially generated document set.

## Verification

- Migration tests validate tables, constraints, RLS, and backfill behavior.
- Store integration tests verify multi-line generation, exact Kanban counts, idempotency, tenant isolation, and approval rollback.
- HTTP tests verify the document summary contract.
- Frontend tests verify Back navigation and approved document visibility.
- A local smoke test confirms existing approved POs have DN and Kanban records after migration.
