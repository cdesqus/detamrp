# Transaction Table Actions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign Supplier Orders and Receiving as compact paginated operational tables with continuing row numbers and icon-based actions/documents.

**Architecture:** Keep pagination client-side because both existing endpoints currently return the complete tenant result and the prototype data volume is limited. Introduce small reusable pagination and icon-button primitives, then preserve the existing submit/cancel/document endpoints behind the redesigned UI.

**Tech Stack:** Next.js, React, TypeScript, Vitest, existing CSS and SVG icon system.

## Global Constraints

- Default page size 20; choices 20, 50, and 100.
- Numbering continues across pages.
- Supplier Order columns start Number, Actions, Documents, Status, PO Number.
- Receiving has no Actions column.
- Send to Supplier remains disabled with `SMTP is not configured`.
- Keep the compact visual density.

---

### Task 1: Shared compact table controls

**Files:**
- Create: `frontend/components/table-pagination.tsx`
- Create: `frontend/components/table-pagination.test.tsx`
- Modify: `frontend/components/icons.tsx`
- Modify: `frontend/app/globals.css`

- [ ] Write failing tests for page navigation, page-size changes, continuing offsets, accessible icon labels, and tooltips.
- [ ] Run focused tests and observe RED.
- [ ] Implement the pagination control and required icons using existing styling conventions.
- [ ] Run focused tests and observe GREEN.

### Task 2: Supplier Orders operational table

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

- [ ] Write failing tests for column order, continuing numbers, pagination, document icons, envelope popover, Draft edit/submit/cancel states, and disabled supplier send.
- [ ] Run the Supplier Orders tests and observe RED.
- [ ] Implement client-side pagination and compact icon cells while preserving existing request guards and cancellation dialog.
- [ ] Run the Supplier Orders tests and observe GREEN.

### Task 3: Receiving operational table

**Files:**
- Modify: `frontend/components/receiving/receiving-index.tsx`
- Modify: `frontend/components/receiving/receiving.test.tsx`

- [ ] Write failing tests for numbering, pagination, absence of Actions, and the Receiving PDF icon.
- [ ] Run the Receiving tests and observe RED.
- [ ] Implement the compact paginated table without changing receiving session behavior.
- [ ] Run the Receiving tests and observe GREEN.

### Task 4: Verification and deployment

**Files:**
- Modify only if a verification failure has a regression test.

- [ ] Run all frontend tests serially, typecheck, lint, and production build.
- [ ] Run backend tests to confirm no API regression.
- [ ] Commit the implementation.
- [ ] Merge after user approval, rebuild the active frontend, and smoke-test the application.
