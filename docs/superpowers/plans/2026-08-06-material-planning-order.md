# Material Planning Order Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build English-language Sales Order, Finished Good, BOM, and standalone Material Planning Order workflows that calculate adjustable gross material requirements and export a PDF reference.

**Architecture:** Extend the existing Go backend with a master-data layer for Customers, Finished Goods, and BOM revisions, plus a dedicated planning-order domain/service/store/API. Add SQL migrations with immutable calculation snapshots and SLO reservation rules. Add Next.js routes/components for CRUD, planning generation, revision lifecycle, and PDF download while reusing existing auth, navigation, toast, table, and PDF patterns.

**Tech Stack:** Go, PostgreSQL, existing migration runner, Next.js/React/TypeScript, Vitest, Go tests, and the existing PDF generation stack.

## Global Constraints

- Sales Order numbers use `SLO-YYYYMM-####` and contain no pricing in the initial release.
- Finished Goods belong to exactly one Customer and use one Finished Good code in the first release.
- BOM rows reference the existing Raw Materials master; Supplier and Unit are read-only projections from Raw Materials.
- One Planning Order may include complete SLOs from multiple customers, but an SLO belongs to only one active Planning Order revision family.
- Requirements are gross quantities from `Sales Order Qty x BOM Usage Qty per Unit`; do not subtract stock or incoming orders.
- Planning Orders are reference documents only and never create or draft Supplier Orders.
- Finalized Planning Orders and Active BOM revisions are immutable; changes use revisions.
- Used SLOs are hidden from new Planning Order selection and released only when their planning order is deleted or Cancelled.
- All user-facing labels and PDFs are in English.

---

## File map

- Create a new SQL migration under `database/migrations/017_material_planning.sql` for customer, finished-good, BOM, SLO, planning-order, revision, snapshot, and reservation tables.
- Extend `backend/internal/masterdata/` for Customer and Finished Good models/services/handlers and BOM revision behavior, following existing `models.go`, `domain.go`, `service.go`, `store.go`, `http.go`, and test conventions.
- Create `backend/internal/salesorder/` for SLO domain, store, service, HTTP handlers, and tests.
- Create `backend/internal/planningorder/` for requirement calculation, lifecycle/revision rules, persistence, HTTP handlers, PDF generation, and tests.
- Update `backend/internal/api/server.go` to register routes and `backend/internal/rbac/catalog.go` if module permissions are required.
- Add `frontend/app/customers/page.tsx`, `frontend/app/finished-goods/page.tsx`, `frontend/app/boms/page.tsx`, `frontend/app/sales-orders/page.tsx`, `frontend/app/sales-orders/new/page.tsx`, `frontend/app/planning-orders/page.tsx`, `frontend/app/planning-orders/new/page.tsx`, and `frontend/app/planning-orders/[id]/page.tsx`.
- Add focused frontend components under `frontend/components/customers/`, `finished-goods/`, `boms/`, `sales-orders/`, and `planning-orders/`, with colocated Vitest tests.
- Update `frontend/components/app-shell/navigation.ts` and its tests for English module labels and permissions.

## Task 1: Add the relational schema and migration contracts

**Files:**
- Create: `database/migrations/017_material_planning.sql`
- Create: `backend/internal/masterdata/migration_test.go`
- Create: `backend/internal/salesorder/migration_test.go`
- Create: `backend/internal/planningorder/migration_test.go`

**Interfaces:**
- Produces tables and constraints consumed by all later backend tasks.

- [ ] Write migration tests that apply the migration to an isolated database and assert required tables, foreign keys, unique constraints, status checks, and the one-active-BOM-per-Finished-Good rule.
- [ ] Add tables for `customers`, `finished_goods`, `boms`, `bom_components`, `sales_orders`, `sales_order_lines`, `planning_orders`, `planning_order_slos`, `planning_order_materials`, and `planning_order_material_breakdowns`.
- [ ] Store BOM revision and material identity snapshots on planning rows so later Raw Material or BOM edits cannot change historical documents.
- [ ] Add unique numbering constraints for `SLO-YYYYMM-####` and `PLN-YYYYMM-####-R#` values.
- [ ] Add a database-level uniqueness rule preventing one SLO from belonging to more than one non-Cancelled Planning Order revision family.
- [ ] Run the migration tests and the full migration suite.
- [ ] Commit: `git add database/migrations/017_material_planning.sql backend/internal/*/migration_test.go; git commit -m "feat: add sales and material planning schema"`.

## Task 2: Implement Customer and Finished Good master data

**Files:**
- Modify: `backend/internal/masterdata/models.go`, `domain.go`, `service.go`, `store.go`, `http.go`
- Create/modify: `backend/internal/masterdata/customer_test.go`, `finished_good_test.go`
- Modify: `frontend/components/app-shell/navigation.ts`
- Create: `frontend/components/customers/customer-index.tsx`, `customer-index.test.tsx`
- Create: `frontend/components/finished-goods/finished-good-index.tsx`, `finished-good-index.test.tsx`
- Create: `frontend/app/customers/page.tsx`, `frontend/app/finished-goods/page.tsx`

**Interfaces:**
- Produces Customer and FinishedGood CRUD APIs used by BOM and SLO forms.
- `FinishedGood.CustomerID` is required and selectable Finished Goods are filterable by CustomerID.

- [ ] Write backend tests for create/update/list validation, required customer ownership, active/inactive behavior, and duplicate code rejection.
- [ ] Implement store/service methods for Customer and FinishedGood CRUD using existing master-data error and tenant patterns.
- [ ] Add HTTP endpoints and JSON contracts for list, create, update, activate, and deactivate operations.
- [ ] Write frontend tests for customer CRUD, Finished Good customer filtering, and empty/error states.
- [ ] Implement the two English master-data screens with existing table/modal/toast patterns.
- [ ] Run Go master-data tests and targeted Vitest tests.
- [ ] Commit: `git add backend/internal/masterdata frontend/components/customers frontend/components/finished-goods frontend/app/customers frontend/app/finished-goods frontend/components/app-shell/navigation.ts; git commit -m "feat: add customer and finished good masters"`.

## Task 3: Implement BOM revisions

**Files:**
- Modify: `backend/internal/masterdata/reference_domain.go`, `reference_service.go`, `reference_store.go`, `reference_http.go`
- Create: `backend/internal/masterdata/bom_test.go`
- Create: `frontend/components/boms/bom-index.tsx`, `bom-index.test.tsx`, `bom-form.tsx`, `bom-form.test.tsx`
- Create: `frontend/app/boms/page.tsx`

**Interfaces:**
- `ActiveBOM(finishedGoodID)` returns one immutable active revision with components.
- BOM components contain `RawMaterialID`, projected supplier/unit, and positive `UsageQty`.
- Produces BOM data consumed by SLO confirmation and Planning Order calculation.

- [ ] Write failing tests for required Finished Good, at least one component, positive usage, duplicate material rejection, supplier/unit projection, one active revision, and Create Revision behavior.
- [ ] Implement BOM draft creation and component replacement through the existing Raw Materials master references.
- [ ] Implement activation transaction that deactivates/revises the prior active BOM and preserves revision history.
- [ ] Implement read-only supplier and unit projection from Raw Materials; reject attempts to override them.
- [ ] Add HTTP endpoints for list, detail, draft save, activate, and create revision.
- [ ] Build Finished Good-first BOM list/detail/editor UI with English labels and raw-material usage table.
- [ ] Run targeted Go and Vitest tests.
- [ ] Commit: `git add backend/internal/masterdata frontend/components/boms frontend/app/boms; git commit -m "feat: add finished good BOM revisions"`.

## Task 4: Implement Sales Orders and SLO lifecycle

**Files:**
- Create: `backend/internal/salesorder/domain.go`, `store.go`, `service.go`, `http.go`, `errors.go`
- Create: `backend/internal/salesorder/domain_test.go`, `service_test.go`, `http_test.go`, `store_test.go`
- Create: `frontend/components/sales-orders/sales-order-index.tsx`, `sales-order-index.test.tsx`, `sales-order-form.tsx`, `sales-order-form.test.tsx`
- Create: `frontend/app/sales-orders/page.tsx`, `frontend/app/sales-orders/new/page.tsx`

**Interfaces:**
- `CreateSalesOrder`, `ConfirmSalesOrder`, `CancelSalesOrder`, `ListEligibleSalesOrders`, and `GetSalesOrder` service methods.
- Confirmed SLOs expose complete lines and Active BOM references to Planning Order generation.

- [ ] Write tests for automatic `SLO-YYYYMM-####` numbering, customer-filtered Finished Goods, no-pricing payloads, positive line quantities, confirmation requiring Active BOMs, and cancellation.
- [ ] Implement SLO store/service/HTTP endpoints and transaction-safe status transitions.
- [ ] Add planning-status projection (`Unplanned`, `In Planning`, `Planned`) without allowing duplicate planning membership.
- [ ] Build list/form pages with Customer, order/delivery dates, Finished Good lines, quantities, notes, and English status labels.
- [ ] Add tests for the empty eligible-list state and hiding SLOs already reserved by a Planning Order.
- [ ] Run targeted Go and Vitest tests.
- [ ] Commit: `git add backend/internal/salesorder frontend/components/sales-orders frontend/app/sales-orders; git commit -m "feat: add sales order lifecycle"`.

## Task 5: Implement gross material requirement calculation

**Files:**
- Create: `backend/internal/planningorder/domain.go`, `calculator.go`, `calculator_test.go`
- Modify: `backend/internal/masterdata/reference_store.go` only if a read query is needed for Active BOM snapshots.

**Interfaces:**
- `CalculateRequirements(ctx, salesOrderIDs) (CalculationResult, error)`.
- `CalculationResult` contains grouped material rows and per-SLO/Finished-Good breakdowns.

- [ ] Write failing calculator tests for one SLO, multiple SLOs/customers, duplicate raw-material consolidation, BOM revision capture, and the exact formula `SalesOrderQty x UsageQty`.
- [ ] Implement a pure calculator that does not query stock, incoming orders, or Supplier Orders.
- [ ] Preserve material code/name, supplier, unit, BOM revision, and source breakdown snapshots in the result.
- [ ] Run calculator tests and verify no purchase-order package dependency is introduced.
- [ ] Commit: `git add backend/internal/planningorder backend/internal/masterdata/reference_store.go; git commit -m "feat: calculate gross material requirements"`.

## Task 6: Implement Planning Order persistence and lifecycle

**Files:**
- Modify: `backend/internal/planningorder/domain.go`
- Create: `backend/internal/planningorder/store.go`, `service.go`, `http.go`, `errors.go`
- Create: `backend/internal/planningorder/service_test.go`, `store_test.go`, `http_test.go`

**Interfaces:**
- `CreateDraft(ctx, salesOrderIDs)`, `UpdateDraft(ctx, id, adjustments)`, `Finalize(ctx, id)`, `Cancel(ctx, id)`, `CreateRevision(ctx, id)`, `Get(ctx, id)`, and `List(ctx, filters)`.

- [ ] Write tests for eligible SLO selection, complete-SLO enforcement, hidden already-planned SLOs, Draft reservation, Finalized locking, Cancelled release, revision numbering, Revised status, and adjustment-note requirements.
- [ ] Implement transaction-safe draft creation that reserves selected SLOs and persists calculation snapshots.
- [ ] Implement update validation: Recommended Qty is read-only, Final Qty positive, and note required when adjusted.
- [ ] Implement Finalized immutability and Create Revision copy semantics (`R0`, `R1`, ...).
- [ ] Ensure Planning Order code never calls Supplier Order creation or persistence.
- [ ] Add list/detail filters for number, SLO, date, status, supplier, and material.
- [ ] Run targeted Go tests and integration tests against the migration.
- [ ] Commit: `git add backend/internal/planningorder; git commit -m "feat: add planning order lifecycle"`.

## Task 7: Add Planning Order HTTP UI

**Files:**
- Create: `frontend/components/planning-orders/planning-order-index.tsx`, `planning-order-index.test.tsx`, `planning-order-form.tsx`, `planning-order-form.test.tsx`, `planning-order-detail.tsx`, `planning-order-detail.test.tsx`
- Create: `frontend/app/planning-orders/page.tsx`, `frontend/app/planning-orders/new/page.tsx`, `frontend/app/planning-orders/[id]/page.tsx`
- Modify: `frontend/components/app-shell/navigation.ts`, `navigation.test.ts`

**Interfaces:**
- Consumes Sales Order eligibility and Planning Order APIs from Tasks 4 and 6.
- Produces draft, finalized, revised, cancelled, and PDF actions using the exact lifecycle labels from the spec.

- [ ] Write tests for eligible-SLO-only selection, multi-customer selection, grouped material table, read-only Recommended Qty, editable Final Qty, required adjustment notes, status-dependent actions, and revision flow.
- [ ] Implement the create screen with helper text: `Showing confirmed Sales Orders that have not been planned.`
- [ ] Implement the list/detail screens with source SLOs, customers, Finished Goods, BOM revisions, breakdowns, activity/revision history, and status actions.
- [ ] Add navigation entry and module permission behavior consistent with existing app-shell tests.
- [ ] Run targeted Vitest tests and the frontend typecheck/lint commands.
- [ ] Commit: `git add frontend/components/planning-orders frontend/app/planning-orders frontend/components/app-shell; git commit -m "feat: add planning order screens"`.

## Task 8: Generate the Planning Order PDF

**Files:**
- Create: `backend/internal/planningorder/pdf.go`, `pdf_test.go`
- Modify: `backend/internal/planningorder/http.go`, `service.go`
- Create/modify: `frontend/app/planning-orders/[id]/page.test.tsx`

**Interfaces:**
- `RenderPDF(ctx, planningOrderID) ([]byte, error)` returns the English PDF bytes.
- HTTP download endpoint returns `application/pdf` with a safe filename containing Planning Number and Revision.

- [ ] Write PDF tests for company branding, number/revision, date/status, source SLO/customer table, material requirements, breakdown, recommended/final quantities, notes, creator/finalizer, print date, and lifecycle watermarks.
- [ ] Implement PDF rendering using the existing purchase-order/report PDF conventions; finalized PDFs have no watermark, Draft uses `DRAFT`, Revised uses `REVISED`, and Cancelled uses `CANCELLED`.
- [ ] Add authenticated download route and frontend download action.
- [ ] Run PDF tests and verify generated output is non-empty and has the expected content markers.
- [ ] Commit: `git add backend/internal/planningorder frontend/app/planning-orders; git commit -m "feat: add planning order PDF"`.

## Task 9: End-to-end verification and documentation

**Files:**
- Modify: `docs/superpowers/specs/2026-08-06-material-planning-order-design.md` only if implementation decisions require a documented correction.
- Create if needed: focused integration tests under `backend/internal/planningorder/` and `frontend/app/`.

- [ ] Run the complete backend test suite, migration tests, frontend Vitest suite, lint, and typecheck.
- [ ] Exercise the acceptance path: Customer → Finished Good → Active BOM → Confirmed SLO → Draft Planning Order → adjusted Final Qty → Finalized → PDF → Create Revision → Revised old version.
- [ ] Verify that no stock or Supplier Order data affects calculation and no Supplier Order is created.
- [ ] Verify SLO reuse behavior after Draft deletion and Planning Order cancellation.
- [ ] Fix any failures, then commit the final verification changes with `git commit -m "test: verify material planning order workflow"`.
