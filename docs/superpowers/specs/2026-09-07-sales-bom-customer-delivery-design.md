# Sales, BOM Calculation, and Customer Delivery Design

Status: consolidated design for review; application implementation has not started.

## Purpose

Extend the existing application to enter customer sales orders manually, calculate component requirements through multi-level BOMs, and record customer deliveries. Excel output is a reference for manual entry into SAGE, not a promised SAGE import format. Existing supplier purchasing, receiving, Kanban inventory, and outgoing flows remain operational.

## Agreed scope

- Use the existing Raw Material master. Its items can be unprocessed materials, stamped parts, or welded components; do not create a separate Part master or require manufacturing process entry.
- Add a separate Finished Goods master and Customer master. Both Raw Materials and FG use the visible term **Item Code**. FG is not restricted to one customer.
- FG has one master sales price, copied automatically into SO and read-only in the SO UI and API.
- Manage BOMs in their own module. A BOM produces either one FG or one existing Raw Material. Components are existing Raw Materials.
- BOM usage is expressed per one base unit of its output. Active child BOMs expand recursively; a component without a BOM is a terminal requirement.
- Common materials share master identity and aggregate across FG/SO paths. Show each path separately in the tree.
- Draft BOMs can be edited and activated. One active BOM per output item. Expand with chevrons, reserve plus for adding components, and provide Expand All/Collapse All.
- One SO belongs to one customer and contains multiple FG lines. Draft -> Submitted, with no approval workflow. Submission captures the complete calculation, BOM versions, item labels, units, Kanban factors, and prices.
- SO detail has Order Details, Material Requirements, and Delivery History. Requirement views show a tree and consolidated material rows.
- Reports can select multiple submitted SOs and use Total Order or Remaining Undelivered quantities. The latter is not unproduced quantity.
- Calculate material value only from terminal component requirements. Do not add intermediate item values to their descendants. Sales value is separate; do not label the difference profit or full HPP.
- BOM requirements use base units (pcs/kg/etc.). Kanban equivalent = required base quantity / quantity per Kanban. Optional purchase suggestion rounds Kanban upward; do not change gross requirement or its value.
- Customer Delivery belongs to exactly one submitted SO; partial deliveries are allowed. Submission creates the final transaction directly, with no persisted Draft, edit, cancellation, or deletion action.
- Delivery reduces remaining SO quantities only. No FG stock, production receipt, production consumption, raw-material stock deduction, or netting against inventory.
- Customer delivery note can be viewed/reprinted and excludes prices. Export calculations to Excel.
- Extend existing tenant isolation, permissions, and activity logs. Do not send email or write to SAGE as part of this feature.

## Proposed implementation defaults

These fill gaps without claiming additional user decisions.

1. Number documents using existing transactional counter conventions: `SLO-YYYYMM-####` and `CDN-YYYYMM-####`. Internal BOM revisions are stored but need not dominate the UI.
2. SO records become immutable at submission; no post-submit amendments/cancellation are included in this release. FG price changes invalidate a draft's acknowledged price version; submission returns a review-needed response rather than silently replacing it.
3. Customer Delivery form displays a confirmation summary before its single final submit. Server idempotency makes retries safe; different concurrent submissions cannot over-deliver.
4. Only active, complete FG BOMs are eligible for SO submission. A raw material with no BOM is a legitimate terminal; one with only a draft/inactive BOM is reported as unresolved instead of silently treated as purchased material.
5. BOM revisions are immutable after activation. Replacement is a copied draft revision activated atomically. Cycle checks use the proposed active graph, including ancestors. Serialize activation per tenant to prevent concurrent A->B and B->A activation.
6. FG sales prices must be positive and have a currency. Existing zero material prices remain visible as zero with an incomplete-cost indication; do not invent prices or call such totals complete estimates. Missing/negative prices are errors for valuation. Requirements can still be inspected without valuation permission.
7. Use decimal arithmetic and existing numeric(20,6) storage. Respect discrete units on SO, delivery, and BOM usage. Do not silently round fractional pieces; show validation on unrepresentable usage. Compute using exact decimals, round persisted quantities/amounts at explicit existing precision boundaries, and reject overflow.
8. Preserve each price's currency; monetary totals group by currency with no currency conversion. Multi-SO reports retain price/quantity provenance when a material has different snapshot prices. Kanban suggestions group by material identity, unit, and snapshot Kanban factor; incompatible factors remain separate.
9. Mark manual SAGE entry with a reference field and audited action on a saved export record, if enabled in the reporting slice. This annotation changes neither SO quantities nor finalized deliveries and does not assert successful SAGE integration.

## Data and calculation contracts

Customer: code, name, address, contact, email, phone, active, tenant/audit identity.

FG: Item Code, name, base unit, one sales price/currency, description, active, price version, audit fields. No mandatory customer_id. Raw Materials remain in their existing table. References use master UUID plus item kind, never Item Code string alone.

BOM header: exactly one of finished_good_id/raw_material_id, revision, status, notes, tenant/audit fields. Lines: raw_material_id and usage_qty (>0) unique within a BOM. Parent base unit comes from its master. No process/cost-center fields.

SO header: number, customer, order/delivery dates, optional customer PO reference, notes, currency policy, status, audit. Lines: FG identity, quantity, unit, displayed price version. Submit snapshots include full nested per-unit requirements with source BOM IDs/revisions, ancestor/path identity, decimal quantities, Item Codes/names, base units, terminal flags, terminal material prices/currencies, Kanban factors, FG sales price/currency. Preserve historical display even when masters change.

Calculation is a pure operation over the saved per-unit snapshot, multiplied by requested line quantity. Total basis uses ordered quantity; remaining basis uses ordered minus finalized deliveries within a consistent database read. Do not query current BOMs or current prices to re-render submitted orders. Aggregate quantities by tenant/material UUID/unit, with provenance retaining order line and source path. A common leaf appearing in multiple paths is summed, not skipped by a global visited set. Detect cycles with the active recursion stack.

Example: FG A uses 2 X + 1 Y; FG B uses 1 X. X uses 0.3 kg Plate. For 10 A + 5 B: X=25 pcs, Y=10 pcs, Plate=7.5 kg. X is visible in the tree/operational recap, but only Plate and Y contribute material value. With 6 A delivered: remaining X=13 pcs, Y=4 pcs, Plate=3.9 kg.

Delivery header: one SO, number, delivery date, customer/address snapshot, notes, immutable audit information, idempotency key and request hash. Lines reference SO line IDs and positive delivered quantities. Submission locks parent SO and affected lines in stable order, validates remaining quantities, writes header/lines/audit in one transaction. Same key+same payload returns original document; same key+different payload returns conflict. Finalized delivery tables are append-only for the application role. PDF reprint uses saved labels/address, with prices excluded.

## Access

Proposed permission codes: `customer.view/manage`, `fg.view/manage/price.manage`, `bom.view/manage/activate`, `sales_order.view/create/edit_draft/submit/price.view`, `customer_delivery.view/create/print`, `material_requirements.view/cost.view/export`. Use fully prefixed strings for each action. Apply API checks and navigation filtering. Restricted users must not receive monetary values through JSON or exports. New grants follow existing administrator provisioning rather than enabling every role.

## Existing-code findings and implementation prerequisites

- Baseline inspected: main at 438e0ad; tracked application directories are locally deleted (329 deletions observed). Read via git show only. The user previously declined restoration and Docker installation. Do not restore, checkout over, or stage these deletions during planning/execution. Use a separate intact checkout at implementation time.
- Server version has not been inspected. Do not deploy or assume local main equals production.
- `planning-order` branch already contains customers, FG, one-level BOMs and SO schemas in migration 017, but FG is customer-bound and a separate Planning Order workflow exists. This design supersedes those restrictions; inspect/adapt useful code rather than blindly merging the branch. Do not add the old Planning Orders UI to this scope.
- `consumable-report` branch contains later migrations including 020. Determine actual target migration history before assigning new numeric filenames. Never edit already-applied migrations or collide with either branch's migration numbers.
- Existing `RawMaterial` has code and SageItemCode fields, supplier/base unit/category/packing requirements, positive QtyPerKanban, StandardUnitPrice, currency. Existing PO computes base quantity = Kanban quantity * QtyPerKanban, value = base quantity * StandardUnitPrice.
- Before changing raw-material validation, inspect actual intended records: self-made components may lack supplier/packing/Kanban values. Reuse valid existing records. Do not fabricate suppliers/factors; if needed, design a scoped optional-metadata extension that keeps PO/receiving eligibility checks strict. This is a compatibility checkpoint, not permission to replace the master or redesign inventory.
- Check which existing field is visibly labeled Item Code and preserve its mapping to code/SageItemCode. Do not rename identifiers based only on English labels.

## Acceptance

1. Existing purchase/receiving/inventory behavior remains unchanged.
2. One FG may be ordered by different customers at the same master price.
3. Direct and nested BOMs calculate correctly; common components aggregate; cyclic graphs cannot activate.
4. Submit captures stable recursive requirements and prices; later master changes do not alter historical calculations or exports.
5. User cannot override SO price by manipulating HTTP requests.
6. Partial deliveries and retries are safe; over-delivery is rejected; no draft/edit/cancel/delete delivery API exists.
7. Total and remaining reports use saved requirements, not live BOMs; different currencies/prices are not silently merged into an incorrect value.
8. Excel output contains basis, generation time, SO references, quantities, units, currencies and provenance; delivery PDFs exclude prices.
9. Cross-tenant access and permission bypass fail, including exports and price fields.

## Release boundary

No production tracking, FG inventory, SAGE API integration/import guarantees, process costing, labor/overhead, stock netting, invoices, taxes, returns, or customer-specific price lists. These are separate changes if requested.
