# Centered Master Data Modal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Center create/edit master-data forms and make sidebar child modules visibly nested.

**Architecture:** Preserve the shared React component and alter only its presentation contract. Use a size modifier derived from the field count, plus an explicit child-link class supplied by the navigation renderer.

**Tech Stack:** Next.js, React, TypeScript, CSS, Vitest, Testing Library.

## Global Constraints

- Create and edit use the same centered modal.
- Existing CRUD behavior and compact typography remain unchanged.
- Mobile layout must remain usable.

---

### Task 1: Centered create/edit modal

**Files:**
- Modify: `frontend/components/master-data-crud.tsx`
- Modify: `frontend/components/master-data-crud.test.tsx`
- Modify: `frontend/app/globals.css`

- [ ] Add failing assertions that both create and edit dialogs use the centered modal class.
- [ ] Run `npm test -- components/master-data-crud.test.tsx` and confirm failure.
- [ ] Add compact/wide modal modifiers and centered, viewport-constrained CSS.
- [ ] Re-run the focused test and confirm it passes.

### Task 2: Sidebar child hierarchy

**Files:**
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Modify: `frontend/app/globals.css`

- [ ] Add a failing navigation assertion for the child-link class.
- [ ] Render links in labeled collapsible groups with that class.
- [ ] Add indentation and a subtle hierarchy guide, disabled in icon-only mode.
- [ ] Run the full frontend tests and `npm run build`.

### Task 3: Deploy

- [ ] Commit the verified UI changes.
- [ ] Rebuild the `platform-master-po` frontend service.
- [ ] Confirm `/measurements` returns HTTP 200 and all services are running.
