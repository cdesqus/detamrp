# PO Dropdown Dismiss Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dismiss Supplier Order row menus through normal outside-click and Escape interactions.

**Architecture:** Attach document-level pointer and keyboard listeners only while a menu is open. A component root ref distinguishes inside interactions from outside interactions.

**Tech Stack:** React, TypeScript, Vitest, Testing Library

## Global Constraints

- No new runtime dependency.
- Preserve all existing PO actions and document links.

---

### Task 1: Menu dismissal

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Test: `frontend/components/supplier-orders/supplier-orders.test.tsx`

**Interfaces:**
- Consumes: `menuOrderId`, `docsOrderId`
- Produces: outside-click and Escape dismissal behavior

- [ ] **Step 1: Write the failing test**

Render the index, open Action, click the page heading, and assert `aria-expanded="false"`. Repeat for Docs and Escape.

- [ ] **Step 2: Verify the test fails**

Run `npm test -- components/supplier-orders/supplier-orders.test.tsx`.

- [ ] **Step 3: Implement the minimal behavior**

Add a root ref and document `pointerdown`/`keydown` listeners that clear both open-menu state values when interaction occurs outside the root menu or Escape is pressed.

- [ ] **Step 4: Verify**

Run the focused test, full test suite, typecheck, lint, and production build.

- [ ] **Step 5: Verify in Chrome**

Open Action and Docs, click outside each, and confirm both close.

