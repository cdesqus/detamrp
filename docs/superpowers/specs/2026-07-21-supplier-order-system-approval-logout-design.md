# Supplier Order, In-App Approval, and Logout Design

## Scope

This increment makes Supplier Orders usable from creation through in-app approval and adds a complete logout interaction. Email delivery, PDF generation, DN/Kanban generation, Sage synchronization, and Receiving remain separate follow-up increments.

## Supplier Order User Experience

The Supplier Orders index loads real tenant-scoped data and retains the compact table columns already designed. `Create order` opens a dedicated order page rather than a small modal because one order may contain multiple raw materials.

The form contains:

- System-generated PO number, shown after the first save.
- One searchable supplier selector.
- Order date, defaulting to today.
- Expected delivery date, which cannot precede the order date.
- Currency inherited from the selected supplier.
- Optional notes.
- Raw-material entries added using `+ Raw Material`; the UI never says `Add Line` or displays a line count.

Before a supplier is selected, `+ Raw Material` is disabled. After selection, the searchable material selector contains only active raw materials assigned to that supplier. Each material can occur only once per PO. Changing supplier after materials exist requires confirmation and clears all selected materials.

For each material, the UI shows material code/name, base unit, quantity per Kanban, total Kanban, calculated total quantity, unit price, and calculated amount. Quantity per Kanban and unit price are snapshotted from Raw Material master and read-only in this increment. The user enters a positive integer Total Kanban.

Calculations use decimal-safe server values:

```text
ordered_base_qty = qty_per_kanban_snapshot * total_kanban
line_total       = ordered_base_qty * unit_price_snapshot
order_total      = sum(line_total)
```

Available actions are `Save as Draft` and `Save & Send for Approval`. Drafts may be edited by users with `po.edit_draft`. Submitted orders are immutable in this increment. No hard-delete endpoint is exposed; an unused draft can be cancelled, preserving its audit trail.

## Status and Approval Flow

This increment uses these states:

```text
DRAFT -> PENDING_APPROVAL -> APPROVED
                         -> REJECTED
DRAFT -> CANCELLED
```

Submission requires at least one material and exactly one configured active approver with effective `po.approve` permission. In the same database transaction, the system:

1. Locks the PO draft.
2. Revalidates supplier, material ownership, quantities, prices, and dates.
3. Snapshots approver user ID, display name, and email.
4. Changes the PO to `PENDING_APPROVAL`.
5. Creates one pending approval request tied to the PO version.

Email is deliberately excluded. The pending request appears in the configured approver's header notification center and Approval Inbox. Approve and Reject operate on the same approval request, use row locking, require the configured snapshot user, and are idempotent: a completed request cannot be processed again. Reject requires a reason; approve does not.

Approval in this increment only changes the local PO and approval request status. Automatic PO/DN/Kanban documents, supplier email, and Sage outbox events will be added in the next transaction increment so placeholder documents are never shown as completed artifacts.

## Persistence and Constraints

Migration `005_purchase_orders.sql` adds tenant-scoped, RLS-protected tables:

- `purchase_orders`: supplier, dates, currency snapshot, notes, status, version, total amount, Sage number, approval snapshot, and audit fields.
- `purchase_order_lines`: raw material and unit snapshots, total Kanban, ordered base quantity, unit price, line total, sort position, and audit fields.
- `purchase_order_approvals`: PO/version, approver snapshot, status, decision reason/time/user, and audit fields.

Required constraints include:

- `UNIQUE (tenant_id, id)` on all business tables.
- `UNIQUE (tenant_id, po_number)` on PO headers.
- Composite tenant foreign keys for supplier, material, user, PO, and measurement references.
- `UNIQUE (tenant_id, purchase_order_id, raw_material_id)` for one occurrence of a material per PO.
- `UNIQUE (tenant_id, purchase_order_id, version)` for one approval request per submitted PO version.
- Positive `qty_per_kanban_snapshot`, `total_kanban`, and `ordered_base_qty`; non-negative prices and totals.
- Expected date on or after order date.
- Database quantities and monetary values use `numeric(20,6)`, never floating point.

PO numbers use a tenant-local, collision-safe format `PO-YYYYMM-#####`. Allocation happens in the same transaction as draft creation. Gaps are acceptable; duplicates are not.

## API and Authorization

The backend adds a focused `purchaseorder` package with domain validation, service orchestration, SQL persistence, and HTTP routes:

- `GET /purchase-orders`
- `GET /purchase-orders/:id`
- `POST /purchase-orders`
- `PUT /purchase-orders/:id`
- `POST /purchase-orders/:id/submit`
- `POST /purchase-orders/:id/cancel`
- `GET /purchase-order-approvals`
- `POST /purchase-order-approvals/:id/approve`
- `POST /purchase-order-approvals/:id/reject`

Routes enforce existing RBAC permissions: `po.view`, `po.create`, `po.edit_draft`, `po.submit`, `po.approve`, and `po.reject`. All queries execute inside the existing PostgreSQL tenant context and RLS policies.

## Header Notifications and Logout

The static approval notification source is replaced with the authenticated user's pending approval API. The bell shows a count badge and compact PO/supplier entries linking to Approval Inbox. Users without pending approvals see an empty state.

The display name at the top right becomes an accessible dropdown. It shows display name, username, and `Logout`. Logout calls the existing `POST /api/auth/logout`, clears the server session cookie, closes the menu, and replaces browser history with `/login`. A failed logout shows a small error and does not pretend the session ended.

## Errors and Concurrency

- Duplicate browser submissions are protected by disabled submit controls plus server-side row locking and state checks.
- Editing, cancelling, approving, or rejecting a stale state returns HTTP `409` with a readable field/form error.
- A material deactivated or moved to another supplier before save/submit is rejected server-side.
- A changed master price affects new saves only; stored snapshots remain stable.
- An approver changed after submission does not redirect an existing request because the approver snapshot is authoritative.
- List pages show compact loading, empty, and retry states without oversized cards.

## Verification

Backend tests cover validation, tenant relationships, decimal calculations, draft editing rules, submission snapshots, authorization, row-locked approval decisions, idempotency, and migration/RLS contracts. Frontend tests cover supplier-filtered material selection, calculated values, form actions, real index rendering, approval notifications/inbox, decision errors, user dropdown, and logout redirect. Final verification runs all Go tests, all Vitest tests, the Next.js production build, Docker rebuild, login/API smoke tests, and one browser-level create-submit-approve smoke flow.
