# Consumable Material and Inventory Reporting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Raw Material classification, consumable receiving without Kanban, Supplier Order type separation, inventory filtering, and PDF/Excel Inventory Movement Reports.

**Architecture:** Keep one Raw Materials master, Supplier Order module, Receiving module, and inventory ledger. Add a `material_type` classification that controls receiving validation and report filtering; expose movement reports from the existing ledger with shared filters and separate PDF/Excel renderers.

**Tech Stack:** Go, PostgreSQL migrations/pgx, Gin HTTP API, Next.js/React/TypeScript, Vitest, Go PDF renderer, and Excel export library/pattern approved by the existing project.

## Global Constraints

- `Material Type` is `Raw Material` or `Consumable`; `Category` remains a separate configurable classification.
- Raw Material receiving requires Kanban; Consumable receiving uses Delivery Note quantity without Kanban.
- Both types use the same Supplier Order, Receiving, and inventory ledger modules.
- A submitted Supplier Order cannot mix Material Types.
- Inventory Movement Report is a Reports submodule with PDF and Excel export.
- No opening/closing balance calculation is required for the selected period; show transaction-time balance.

---

## Task 1: Add Material Type to Raw Materials

**Files:** `database/migrations/018_consumable_material_type.sql`, `backend/internal/masterdata/models.go`, `service.go`, `store.go`, `http.go`, raw-material tests, and `frontend/app/raw-materials/` files.

- [ ] Write migration/domain tests for required `Raw Material` default, valid `Consumable`, invalid values, and existing-row compatibility.
- [ ] Add `material_type text NOT NULL DEFAULT 'Raw Material'` with a check constraint and seed existing rows as Raw Material.
- [ ] Add API and UI field with English labels and filter support.
- [ ] Run targeted master-data Go/Vitest tests.
- [ ] Commit `feat: classify raw materials by material type`.

## Task 2: Separate Supplier Order Types

**Files:** `backend/internal/purchaseorder/domain.go`, `service.go`, `store.go`, `http.go`, purchase-order tests, and supplier-order frontend components/tests.

- [ ] Add `MaterialType` to order input and response snapshots.
- [ ] Add failing tests rejecting mixed material types and accepting Standard Material or Consumable orders.
- [ ] Derive/validate order type from selected Raw Materials; do not trust a conflicting client value.
- [ ] Add an order-type filter and label to list/detail/PDF views.
- [ ] Run targeted purchase-order tests.
- [ ] Commit `feat: separate supplier order material types`.

## Task 3: Add the direct-quantity receiving schema

**Files:** `database/migrations/020_consumable_receiving.sql`, `backend/internal/receiving/domain.go`, `store.go`, migration/integration tests.

- [ ] Add a receiving-session line table that supports either `kanban_lot_id` or direct `raw_material_id + received_quantity`, with a check constraint requiring exactly one mode.
- [ ] Add nullable `kanban_lot_id` and material/quantity snapshots to receiving ledger/summary tables where required; preserve all existing Kanban constraints.
- [ ] Add indexes and row-level security for the new tenant-scoped rows.
- [ ] Write migration tests against a clean PostgreSQL database and verify both modes can coexist in the schema.
- [ ] Commit `feat: add direct consumable receiving schema`.

## Task 4: Add Consumable Receiving Flow

**Files:** `backend/internal/receiving/domain.go`, `service.go`, `store.go`, `http.go`, receiving tests, `frontend/components/receiving/` files.

- [ ] Write tests proving Raw Material requires Kanban and Consumable accepts direct received quantity.
- [ ] Load material type from the Raw Material snapshot when validating receiving lines.
- [ ] Keep Delivery Note required for both flows; bypass Kanban validation only for Consumables.
- [ ] Record both flows in the same inventory ledger, leaving Kanban ID null for Consumables.
- [ ] Add conditional UI copy and quantity inputs based on material type.
- [ ] Run targeted receiving/inventory tests.
- [ ] Commit `feat: receive consumables without kanban`.

## Task 5: Inventory Material-Type Filters

**Files:** `backend/internal/inventory/domain.go`, `store.go`, `http.go`, inventory tests, `frontend/components/inventory/inventory-index.tsx`, and tests.

- [ ] Add material-type filter to stock and ledger queries.
- [ ] Add All Materials, Raw Materials, and Consumables filter controls.
- [ ] Verify balances remain in one ledger and only presentation/filtering differs.
- [ ] Run targeted inventory tests and frontend tests.
- [ ] Commit `feat: filter inventory by material type`.

## Task 6: Inventory Movement Report API and PDF

**Files:** `backend/internal/report/domain.go`, `store.go`, `http.go`, `pdf.go`, report tests, `frontend/components/report-index.tsx`, and tests.

- [ ] Write tests for date, material type, category, material, supplier, and movement-type filters.
- [ ] Add movement query columns: date, material, type, category, movement, qty in, qty out, transaction balance, reference, Kanban ID, and user.
- [ ] Add Reports submodule route and UI filter form.
- [ ] Implement English PDF with branding, filters, generation date, summary, and movement table.
- [ ] Add PDF download link and test non-empty output/watermark-free finalized report.
- [ ] Commit `feat: add inventory movement PDF report`.

## Task 7: Inventory Movement Excel Export

**Files:** `backend/internal/report/excel.go`, `excel_test.go`, `http.go`, and report frontend/tests.

- [ ] Write an export test asserting headers and representative transaction rows, including blank Kanban ID for Consumables.
- [ ] Implement `.xlsx` export using the approved project dependency and the same query/filter contract as PDF.
- [ ] Add Excel export action beside PDF in the Reports submodule.
- [ ] Run report tests and frontend tests.
- [ ] Commit `feat: add inventory movement excel export`.

## Task 8: End-to-End Verification

- [ ] Run migrations 019 and 020 on a clean PostgreSQL database.
- [ ] Run backend targeted suites serially with the configured Go cache and frontend typecheck/Vitest.
- [ ] Verify Standard Material receiving rejects missing Kanban.
- [ ] Verify Consumable receiving succeeds without Kanban and appears in inventory movement PDF/Excel.
- [ ] Verify mixed-type Supplier Orders are rejected.
- [ ] Verify existing Raw Material flows remain unchanged.
- [ ] Commit final fixes as `test: verify consumable material reporting workflow`.
