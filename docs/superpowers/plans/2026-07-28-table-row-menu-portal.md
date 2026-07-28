# Supplier Order Row Menu Portal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render Supplier Order Action and Docs menus outside the horizontally scrollable table so short result sets never require vertical scrolling.

**Architecture:** Add a focused portal menu component that owns fixed-position measurement, viewport flipping/clamping, and reposition listeners. The Supplier Order index continues to own which menu is open and the menu content, while passing the active trigger and content to the portal.

**Tech Stack:** React 19, TypeScript, `createPortal`, Vitest, Testing Library, CSS.

## Global Constraints

- Apply only to Supplier Orders Action and Docs menus.
- Preserve every existing permission, action, link, disabled item, and cancellation flow.
- Close on outside interaction and Escape; Escape restores trigger focus.
- Prefer below, flip above near the viewport bottom, and clamp to an 8-pixel horizontal margin.
- Keep horizontal table scrolling and compact styling.
- Add no dependency.

---

### Task 1: Reusable Fixed Row Menu Portal

**Files:**
- Create: `frontend/components/supplier-orders/row-menu-portal.tsx`
- Create: `frontend/components/supplier-orders/row-menu-portal.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces:

```ts
type RowMenuPortalProps = {
  trigger: HTMLElement;
  ariaLabel: string;
  onClose: (restoreFocus: boolean) => void;
  children: ReactNode;
};

export function RowMenuPortal(props: RowMenuPortalProps): ReactPortal;
```

- [ ] **Step 1: Write failing positioning and dismissal tests**

Mock `getBoundingClientRect()` and viewport dimensions. Assert the menu is appended under `document.body`, uses fixed coordinates below the trigger, flips above when lower space is insufficient, clamps its left coordinate, closes on outside pointer input, closes on Escape, and requests focus restoration only for Escape.

```tsx
render(<RowMenuPortal trigger={trigger} ariaLabel="Actions for PO-001" onClose={onClose}>
  <button>Open Detail</button>
</RowMenuPortal>);
expect(document.body.querySelector('[data-row-menu-portal]')).toHaveStyle({ position: 'fixed' });
fireEvent.keyDown(document, { key: 'Escape' });
expect(onClose).toHaveBeenCalledWith(true);
```

- [ ] **Step 2: Verify tests fail**

Run:

```bash
cd frontend
npm test -- row-menu-portal.test.tsx
```

Expected: FAIL because the portal component does not exist.

- [ ] **Step 3: Implement the portal**

Use `createPortal`, `useLayoutEffect`, a menu ref, and a `requestAnimationFrame` measurement pass. Read trigger/menu rectangles, calculate fixed `top`/`left`, and update on capture-phase `scroll` plus `resize`. Outside pointer events must ignore both the portal and its trigger. Escape calls `onClose(true)`; other dismissal paths call `onClose(false)`.

- [ ] **Step 4: Add portal styling**

Move visual menu styling to `.supplier-order-row-menu-portal`, retaining 158px content width, compact 28px rows, current disabled/danger styles, border, radius, and shadow. Set a z-index above the application shell.

- [ ] **Step 5: Verify focused tests**

Run:

```bash
cd frontend
npm test -- row-menu-portal.test.tsx
npm run typecheck
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/supplier-orders/row-menu-portal.tsx frontend/components/supplier-orders/row-menu-portal.test.tsx frontend/app/globals.css
git commit -m "feat: add fixed table row menu portal"
```

### Task 2: Integrate Action and Docs Menus

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-index.tsx`
- Modify: `frontend/components/supplier-orders/supplier-orders.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `RowMenuPortal`.
- Preserves: existing `menuOrderId`, `docsOrderId`, action callbacks, document links, and permission checks.

- [ ] **Step 1: Add failing one-row integration tests**

Render a single Draft PO. Open Action and assert the menu is under `document.body`, the table frame does not receive a menu-open/overflow workaround class, all actions are visible, Docs replaces Action, outside click closes it, and Escape closes then focuses the trigger.

```tsx
await user.click(screen.getByRole('button', { name: 'Actions for PO-001' }));
expect(document.body.querySelector('[data-row-menu-portal]')).toHaveTextContent('Send to Approval');
expect(document.querySelector('.table-frame')).not.toHaveClass('row-menu-open');
```

- [ ] **Step 2: Verify the integration test fails**

Run:

```bash
cd frontend
npm test -- supplier-orders.test.tsx
```

Expected: FAIL because menus are still rendered inside the table.

- [ ] **Step 3: Integrate the portal**

Track Action and Docs trigger elements by PO ID. Render menu content through `RowMenuPortal`, close the competing menu before opening, and restore focus on Escape. Remove `row-menu-open` row classes, upward-menu CSS logic, and obsolete table z-index rules.

- [ ] **Step 4: Run complete frontend verification**

Run:

```bash
cd frontend
npm test
npm run typecheck
npm run lint
npm run build
```

Expected: all commands PASS.

- [ ] **Step 5: Build and smoke-test Docker**

Run:

```bash
docker compose -p platform-master-po build frontend
docker compose -p platform-master-po up -d frontend
```

Open a one-row Supplier Orders list, verify the full menu overlays the page without a vertical table scrollbar, and confirm Action/Docs, outside click, Escape, and horizontal table scroll.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/supplier-orders/supplier-order-index.tsx frontend/components/supplier-orders/supplier-orders.test.tsx frontend/app/globals.css
git commit -m "fix: keep supplier order menus outside table overflow"
```

