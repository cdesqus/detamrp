# Global Success Toast Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add compact success confirmation that survives navigation for purchase-order email actions.

**Architecture:** Mount a toast context provider inside the authenticated App Shell. Supplier Order Form and Supplier Order Index consume its hook and emit predefined success messages only after successful API completion.

**Tech Stack:** React, Next.js, TypeScript, Vitest, Testing Library, CSS.

## Global Constraints

- Show one compact top-right success toast.
- Auto-dismiss after 4 seconds; manual close is available.
- A newer toast replaces the current toast and resets its timer.
- Failed requests never emit success.

---

### Task 1: Toast Provider

**Files:**
- Create: `frontend/components/toast/toast-provider.tsx`
- Create: `frontend/components/toast/toast-provider.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces: `ToastProvider({ children })` and `useToast(): { showSuccess(message: string): void }`.

- [ ] Write failing tests for rendering, replacement, close, and 4-second dismissal.
- [ ] Run `npm test -- --run components/toast/toast-provider.test.tsx` and verify red.
- [ ] Implement context, one toast state, timer reset/cleanup, and accessible markup.
- [ ] Add compact fixed-position styling below the header.
- [ ] Run focused tests and verify green.
- [ ] Commit as `feat: add global success toast`.

### Task 2: App Shell Integration

**Files:**
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Test: `frontend/components/app-shell/app-shell.test.tsx`

**Interfaces:**
- Consumes: `ToastProvider`.
- Produces: toast context throughout authenticated module pages.

- [ ] Add a failing App Shell test proving descendants can emit a toast.
- [ ] Mount the provider without changing login/auth behavior.
- [ ] Run the App Shell test and verify green.
- [ ] Commit as `feat: mount toast provider in app shell`.

### Task 3: Supplier Order Success Messages

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Test: `frontend/components/supplier-orders/supplier-orders.test.tsx`

**Interfaces:**
- Consumes: `useToast().showSuccess`.
- Emits:
  - `PO submitted and approval email sent.`
  - `Approval email sent successfully.`
  - `Supplier email sent successfully.`

- [ ] Add failing tests for Save & Send, approval resend, and supplier delivery.
- [ ] Assert failed email responses do not emit success.
- [ ] Emit success immediately before navigation for Save & Send.
- [ ] Emit success after successful resend/delivery in the index.
- [ ] Run supplier-order tests and typecheck.
- [ ] Commit as `feat: confirm supplier order email actions`.

### Task 4: Verification and Push

**Files:**
- No production files expected.

- [ ] Run `npm test`, `npm run lint`, `npm run typecheck`, and `npm run build`.
- [ ] Run `git diff --check` and confirm a clean worktree.
- [ ] Build and recreate the local frontend container.
- [ ] Push `main` to origin and provide server deployment commands.
