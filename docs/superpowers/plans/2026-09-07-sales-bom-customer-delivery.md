# Sales, BOM Calculation, and Customer Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking. This document authorizes planning only; implementation follows user instruction.

**Goal:** Deliver customer/FG masters, independent nested BOMs, snapshot-based SO calculations, immutable customer deliveries, and Excel reports using existing Raw Materials.

**Architecture:** Add bounded Go domain modules behind existing Gin, PostgreSQL tenant transactions, RBAC and audit conventions. A pure BOM calculator feeds an immutable SO snapshot; delivery and report modules consume that snapshot without affecting inventory. Reuse Next.js application shell and established forms/tables.

**Tech Stack:** Existing Go 1.26, Gin, pgx/PostgreSQL 17, shopspring/decimal, fpdf; Next.js 16/React 19/TypeScript, Vitest. Select and pin an XLSX writer at implementation time after checking its official documentation and current maintenance; this planning task installs nothing.

**Spec:** `docs/superpowers/specs/2026-09-07-sales-bom-customer-delivery-design.md`

## Global Constraints

- Use the visible term **Item Code**. FG and Raw Material remain separate masters; no Part master.
- Quantity basis is one base unit of output; Kanban is not a BOM unit.
- No production tracking or stock mutation. No customer binding or price override on FG sales.
- One Delivery per SO source, no delivery Draft/edit/cancel/delete.
- Existing supplier workflows and tenant protections remain operational.
- Preserve the 329 pre-existing tracked deletions; work in an intact isolated checkout for implementation.
- Migration numbers are allocated against the selected checkout's complete history, never hardcoded from main's 016 alone.

## Delivery slices and order

1. Compatibility inventory and Customer/FG masters.
2. BOM authoring and calculator.
3. Sales Order and immutable requirements.
4. Customer Delivery.
5. Multi-order Reports/Excel and release verification.

Each slice yields independently testable software. No separate Planning Orders module is introduced. Complete dependent slices in order; do not merge the historical planning branch as a substitute for this design.

## Task 0: Establish an intact baseline and compatibility decisions

**Files read:** `backend/internal/masterdata/{models,domain,store}.go`, `frontend/components/master-data-crud.tsx`, `database/migrations`, branch `planning-order` migration 017 and implementation, branch `consumable-report` migrations, existing app shell/auth routes. Record findings in the design's existing-code section.

- [ ] Inspect `git worktree list`, `git status --short`, and committed histories. Use a new isolated checkout without restoring the user's missing files. Read applicable AGENTS.md there before edits.
- [ ] Determine target branch/migration set; compare existing customer/FG/BOM/SO tables and APIs. For a main-only base create missing objects; for existing schemas append transformations (remove FG customer restriction, add RM BOM output, prices/snapshots, new statuses) without destroying rows. Document the exact migration filenames before implementation.
- [ ] Confirm Item Code field mapping, master price per base unit, and whether existing self-made RM records satisfy supplier/packing/Kanban requirements. Stop dependent master changes if actual schema cannot represent those records without false data; draft a narrow compatibility adjustment rather than creating another master.
- [ ] Run baseline `go test ./...` from backend, `npm ci`, `npm run test`, `npm run typecheck` from frontend, using a disposable PostgreSQL database for integration tests and existing migration runner. Record pre-existing failures separately.

## Task 1: Customer and FG masters with authorization

**Create:** `backend/internal/salesmaster/{domain,service,store,http}.go`, corresponding `domain_test.go`, `http_test.go`, `store_integration_test.go`; `frontend/app/customers/page.tsx`, `frontend/app/finished-goods/page.tsx`, `frontend/components/sales-master/{customers,finished-goods}.tsx`, `sales-master.test.tsx`. Add one append-only migration for these entities/permissions using filename allocated in Task 0.

**Modify:** `backend/cmd/api/main.go`, `backend/internal/api/server.go`, `backend/internal/rbac/catalog.go`, `frontend/components/app-shell/navigation.ts`. Reuse existing customer/FG components if Task 0 locates compatible versions at these routes.

**Interfaces:** `Customer{id,code,name,address,contact,email,phone,active}`; `FinishedGood{id,itemCode,name,baseUnitId,salesPrice,currency,priceVersion,active}`. Decimal JSON values are strings. `GET/POST /api/customers`, `GET/PATCH /api/customers/:id`, equivalent `/api/finished-goods`. Creation/edit schemas do not contain a required customerId. Price changes increment priceVersion server-side and require `fg.price.manage` in addition to management access.

- [ ] Add domain tests for valid FG without customer, nonpositive sales price, missing unit, invalid currency, duplicate Item Code per tenant; API tests deny price updates by users lacking price permission.
- [ ] Run `go test ./internal/salesmaster -v` and verify new behavior tests fail before implementation.
- [ ] Implement tenant-scoped schema/repository and HTTP handlers using existing tenant transactions, audit helpers, pagination and error envelopes. Use composite tenant FKs and forced RLS on every new table. Archive referenced masters instead of deleting historical identities.
- [ ] Build master forms following existing CRUD UI. Display Item Code, master price and unit; no customer ownership field on FG. Add tests rendering one FG for orders by two different customers and verifying restricted price controls.
- [ ] Run backend package tests including disposable-database isolation tests; run `npm run test -- sales-master` and `npm run typecheck`. Commit only task files after diff review.

## Task 2: BOM storage, activation and nested editor

**Create:** `backend/internal/bom/{domain,service,store,http}.go`, `{domain,service,http}_test.go`, `store_integration_test.go`; `frontend/app/boms/page.tsx`, `frontend/app/boms/[id]/page.tsx`; `frontend/components/bom/{bom-index,bom-editor,bom-tree}.tsx`, `bom.test.tsx`. Add append-only BOM migration.

**Interfaces:** `OutputRef{kind:"FG"|"RAW_MATERIAL",id}`; `Component{rawMaterialId,usageQty}`; `BOM{id,output,revision,status,components}`. `GET/POST /api/boms`, `GET/PATCH /api/boms/:id`, `POST /api/boms/:id/activate`. `LoadActiveGraph(ctx, tenant, output) (Graph,error)` supplies the complete consistent recursive graph to Task 3. An absent BOM differs from an existing but inactive-only BOM.

- [ ] Write table-driven validation tests: empty components, duplicate RM line, zero/negative usage, own-output component, fractional discrete units. Add graph tests for A->B->A and valid shared-child diamond A->X/B->X.
- [ ] Run `go test ./internal/bom -v`; verify tests fail on missing validation/activation behavior.
- [ ] Create schema with exactly one output FK and partial unique indexes for active FG and RM outputs. Persist activated revisions immutably; edit copied drafts. Serialize activations with a tenant-scoped transaction advisory lock, validate proposed effective graph and atomically replace the previous active revision.
- [ ] Add a concurrent activation integration test ensuring two individually proposed edges cannot commit a cycle; add cross-tenant component/output rejection.
- [ ] Build output selector for FG/RM, component search from existing RM, base-unit usage fields, Draft/Activate actions, and nested chevrons with accessible buttons. Plus adds a component only. Child detail reads its own BOM; editing a parent must not mutate the shared child BOM.
- [ ] Run Go BOM tests, `npm run test -- bom`, typecheck. Test Expand All/Collapse All and per-parent unit labels. Commit task files.

## Task 3: Pure calculator and snapshot contract

**Create:** `backend/internal/bom/calculator.go`, `calculator_test.go`, `snapshot.go`, `snapshot_test.go`.

**Interfaces:** `BuildSnapshot(graph Graph) (Snapshot,error)` produces immutable per-unit nested nodes. `Calculate(snapshot Snapshot, quantity decimal.Decimal) (RequirementResult,error)` returns path-preserving nodes, intermediate recap, terminal requirements and currency-grouped values. Snapshot stores versioned schema, output identity/unit, every child usage, labels, BOM revisions, terminal price/currency and Kanban factor. Required amount multiplication never uses floating point.

Algorithm:

```text
walk(node, parentQuantity, recursionStack):
  reject node identity already on current stack
  quantity = parentQuantity * node.usagePerParent
  record path and quantity
  if node has children: walk every child with quantity
  else: append terminal quantity, snapshot price/currency, source path
aggregate terminal quantities by master identity and unit
sum terminal quantity * snapshot price by currency, retaining source contributions
kanban equivalent = grouped quantity / compatible snapshot factor
suggested purchase kanban = ceiling(equivalent), only after grouping
```

- [ ] Implement fixture tests using FG A=2X+Y; FG B=X; X=0.3kg Plate. Assert 10A+5B gives X=25, Y=10, Plate=7.5; only Y+Plate count in material value. Assert direct material BOM also works.
- [ ] Test decimal precision/overflow, incompatible units, missing active child BOM, absent terminal BOM, zero-price completeness flags, multiple currencies, repeated shared descendants, and Kanban requirement 120/50=2.4 with suggestion 3 while valued quantity stays 120.
- [ ] Run `go test ./internal/bom -run 'Calculate|Snapshot' -v` and confirm red before implementing the recursive calculation and serialization.
- [ ] Implement exact-decimal traversal using a path stack (not a global deduplication set), deterministic grouping and explicit rounding at persistence boundaries. Keep the complete per-unit structure so remaining requirements can later be recalculated without dividing rounded totals.
- [ ] Run tests and snapshot round-trip tests; mutate original graph/master fixtures and verify persisted snapshot results do not change. Commit calculator files.

## Task 4: Sales Order creation, submit and calculation UI

**Create:** `backend/internal/salesorder/{domain,service,store,http}.go`, `{domain,service,http}_test.go`, `store_integration_test.go`; `frontend/app/sales-orders/{page.tsx,new/page.tsx,[id]/page.tsx}`; `frontend/components/sales-orders/{order-form,order-index,order-detail,material-requirements}.tsx`, `sales-orders.test.tsx`; append-only SO/snapshot migration.

**Interfaces:** `CreateOrderInput{customerId,orderDate,deliveryDate,customerPoReference,notes,lines:[{finishedGoodId,quantity}]}`; no client price. Draft review carries only server-provided `priceVersion` tokens. `Submit(ctx,actor,orderId,expectedVersion)` atomically writes status SUBMITTED and Task 3 snapshot. `GET/POST /api/sales-orders`, `GET/PATCH /api/sales-orders/:id`, `POST /api/sales-orders/:id/submit`, `GET /api/sales-orders/:id/requirements?basis=TOTAL|REMAINING`. Draft price version changes return conflict with review details.

- [ ] Write tests for one customer/multiple FG, FG reused across customers, tampered client price rejected, stale price version review, missing FG BOM, inactive masters, date/quantity validation, tenant boundaries and repeat submit.
- [ ] Run `go test ./internal/salesorder -v`, observing meaningful failures.
- [ ] Implement draft lifecycle and submission in one consistent tenant transaction; load prices and the full active graph together. Lock draft/version, generate number using existing transactional numbering, capture snapshot, append audit, commit once. Submitted order and snapshots cannot be edited by application APIs.
- [ ] Build form with read-only price/subtotal, permission-aware costs, validation and preview refresh on stale prices. Detail has Order Details, Material Requirements tree/recap, Delivery History placeholder backed by empty data until Task 5. Add TOTAL/REMAINING selector; initially both are equal without deliveries.
- [ ] Verify changing nested BOM or FG/RM price after submit leaves calculation unchanged. Verify restricted monetary fields are absent in JSON, not merely hidden in CSS.
- [ ] Run Go package/integration tests, `npm run test -- sales-orders`, typecheck. Commit task files.

## Task 5: Immediate final customer delivery and printable document

**Create:** `backend/internal/customerdelivery/{domain,service,store,http,pdf}.go`, `{service,http,pdf}_test.go`, `store_integration_test.go`; `frontend/app/customer-deliveries/{page.tsx,new/page.tsx,[id]/page.tsx}`; `frontend/components/customer-deliveries/{delivery-form,delivery-index,delivery-detail}.tsx`, `customer-deliveries.test.tsx`; append-only delivery migration.

**Modify:** SO detail/history and remaining-requirement query; API wiring and navigation.

**Interfaces:** `CreateDeliveryInput{salesOrderId,deliveryDate,address,notes,lines:[{salesOrderLineId,quantity}]}` plus `Idempotency-Key` header. `POST /api/customer-deliveries` finalizes immediately. Read/list and `GET /api/customer-deliveries/:id/pdf` only; no update/cancel/delete endpoints. `DeliveredQuantities(ctx,tx,soId)` returns decimal sums per line. Remaining = ordered - sum(deliveries).

- [ ] Add test cases: 100 ordered, deliver 60, remaining 40; deliver 41 fails; zero/negative/empty payload fails; line from another SO fails; different customer/tenant fails; draft SO fails.
- [ ] Add concurrent integration tests: two requests for final 40 cannot both succeed; identical idempotency retries return the same ID/number; reused key with a different payload conflicts. Assert no inventory/kanban changes and no delivery draft rows.
- [ ] Run `go test ./internal/customerdelivery -v`, verify failures before implementation.
- [ ] Implement single transaction with stable SO/line locks, unique tenant idempotency key, request hash, final records/audit, and saved address/item labels. Revoke UPDATE/DELETE on finalized delivery tables from the application role. Derive delivery progress without changing SO submission state.
- [ ] Build form selecting exactly one SO and remaining lines; confirmation summary is form UI only, not a persisted draft. Preserve key across retry, disable repeated click, show success document. Use existing fpdf layout/branding conventions for a price-free customer delivery note; reprint snapshots, not current master labels.
- [ ] Calculate REMAINING requirements from original per-unit snapshots and current delivered sums in a consistent read transaction. For example fixture, delivery of 6 A leaves X=13,Y=4,Plate=3.9.
- [ ] Run integration/PDF tests, `npm run test -- customer-deliveries`, typecheck; test absent edit/cancel controls and rejected corresponding HTTP methods. Commit task files.

## Task 6: Multi-order reporting and Excel export

**Create:** `backend/internal/materialreport/{domain,service,http,xlsx}.go`, `{service,http,xlsx}_test.go`; `frontend/app/reports/material-requirements/page.tsx`, `frontend/components/material-requirements-report.tsx`, corresponding `.test.tsx`.

**Interfaces:** `ReportInput{salesOrderIds,basis:"TOTAL"|"REMAINING"}`. `BuildReport(ctx,actor,input)` loads saved per-unit snapshots and delivery sums in one consistent read; aggregates Task 3 results retaining SO/line/path provenance. `POST /api/reports/material-requirements/query` and `/export` take the same input. Reject duplicate SO IDs and inaccessible/draft orders.

- [ ] Write report tests for common components from multiple orders, differing snapshot prices and Kanban factors, grouped currencies, fully delivered remaining=0, and delivery changing during report generation. Total material value must equal the sum of source contributions, never aggregate quantity times one arbitrary/latest price.
- [ ] Run `go test ./internal/materialreport -v` and verify red.
- [ ] Implement report service using existing tenant transaction patterns. Provide operational intermediate recap separately from terminal valuation, source drilldown, basis and generated-at metadata. Include currency on every amount and unit on every quantity; never sum different units into a quantity grand total.
- [ ] Choose a maintained Go XLSX library from official docs and pin dependency. Export sheets `Orders`, `BOM Breakdown`, `Material Summary`, `Source Detail`. Store user labels as literal strings, not spreadsheet formulas. Preserve Item Code leading zeros, decimal precision and currency distinctions. Do not claim SAGE import compatibility.
- [ ] Parse generated workbook in tests: verify leading-zero codes, quantities, basis, source orders, absence of restricted prices, and literal formula-looking labels. Check exported numbers match the report service result.
- [ ] Build SO selection, basis switch, search and export UI using existing report patterns. Optional manual SAGE reference tracking must be a separate audited export annotation, with no external writes or changes to final SO/delivery payloads.
- [ ] Run Go report tests, `npm run test -- material-requirements`, typecheck. Commit task files.

## Task 7: Permissions, integration and release review

**Modify/test:** existing permission catalog, role provisioning/migration, API registration, navigation tests, activity-log hooks; all new domain integration suites and UI tests.

- [ ] Verify each endpoint/price field/export is permission-gated and tenant-scoped, including direct ID access. Apply explicit permissions from the spec, preserving existing administrator setup and ordinary-role grants.
- [ ] Test entire fixture: create customer/FG and nested BOM; submit 10A+5B; view requirements; deliver 6A; compare TOTAL vs REMAINING; export; mutate masters and verify historical totals; reprint delivery with original labels.
- [ ] Run `go test ./...` from backend with integration database configured as existing tests require; run `npm run test`, `npm run typecheck`, `npm run lint`, `npm run build` from frontend. Run migrations on fresh and populated disposable databases, then repeat runner to check idempotent migration tracking. Record exact results and versions.
- [ ] Check original PO fixture: 2 Kanban * 50 pcs = 100 pcs, valued at 100 * base unit price; receiving/outgoing still change their existing inventory only. Verify customer delivery does not invoke those services.
- [ ] Review diff/migration integrity and requirement coverage. No commit should stage unrelated missing files. Prepare release notes including data defaults and manual SAGE limitations. Deployment is a separate user instruction after target/server compatibility is established.

## Planning review completed

Coverage: master reuse and FG independence (Tasks 0-1), BOM tree/cycles/common parts (2-3), immutable prices/requirements (4), final partial deliveries (5), report bases and exports (6), permissions/regressions (7). Compatibility checks explicitly cover legacy branch schemas, deleted workspace files, required raw-material metadata, currency grouping and base-unit price semantics. The only conditional schema work is selected by evidence from the intended implementation checkout; no production migration is executed by this plan.
