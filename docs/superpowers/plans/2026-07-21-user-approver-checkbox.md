# User Approver Checkbox Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Default Approver configuration into User create/edit with a single approver checkbox.

**Architecture:** Extend the existing User DTO/input and transactional user store so user save and approver replacement happen atomically. Remove the standalone frontend card and represent the selected approver with a checkbox in the modal and badge in the table.

**Tech Stack:** Go, PostgreSQL, React, TypeScript, Vitest.

## Global Constraints

- At most one Purchase Order Approver exists per tenant.
- The selected user must be active and effectively grant `po.approve`.
- Selecting a new approver atomically replaces the previous user after UI confirmation.

---

### Task 1: Backend user approver transaction

**Files:** `backend/internal/settings/domain.go`, `service_test.go`, `store.go`.

- [ ] Add failing tests requiring `isPurchaseOrderApprover` on user input/output.
- [ ] Extend user create/update so checked eligible users update `tenant_settings.default_approver_user_id` in the same transaction.
- [ ] Return the approver flag in user list responses and run `go test ./...`.

### Task 2: User modal and table

**Files:** `frontend/components/settings/users-settings.tsx`, `users-settings.test.tsx`, `frontend/app/globals.css`.

- [ ] Add failing tests for checkbox, replacement confirmation, badge, and removal of the external card.
- [ ] Implement the checkbox and confirmation using the existing centered modal.
- [ ] Run frontend tests, typecheck, and production build.

### Task 3: Deploy and smoke test

- [ ] Rebuild backend/frontend and verify the existing `director_demo` appears as PO Approver.
- [ ] Confirm API and Settings Users page return HTTP 200.
