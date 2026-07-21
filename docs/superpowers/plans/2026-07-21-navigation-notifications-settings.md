# Navigation, Notifications, Reports, and Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refine navigation with SVG icons and sidebar-owned collapse controls, move approvals into a header notification center, consolidate PO documents under Supplier Orders, and expose Reports and Settings MVP routes.

**Architecture:** Extend the existing shared Next.js `AppShell` with typed internal SVG icons, ordered navigation metadata, expandable Settings navigation, and an empty real-data notification adapter. Keep route pages declarative through existing index components, while the legacy Delivery Notes route becomes a server redirect.

**Tech Stack:** Next.js 16, React 19, TypeScript, CSS, Vitest, Testing Library, Docker Compose.

## Global Constraints

- Do not fabricate PO, notification, report, email, or document data.
- Use internal 16 px SVG icons without adding a package dependency.
- Keep UI controls, type, dropdowns, and table rows compact.
- Desktop sidebar collapse belongs inside the sidebar; mobile navigation remains a header drawer control.
- Delivery Notes and Approval Inbox must not appear as sidebar modules.
- Reports must retain supplier filtering as an MVP requirement.

---

### Task 1: SVG Icons and Revised Sidebar Information Architecture

**Files:**
- Create: `frontend/components/icons.tsx`
- Modify: `frontend/components/app-shell/navigation.ts`
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces: `Icon({ name, size? })` and navigation items with typed `IconName`.
- Consumes: existing `AppShell`, `usePathname`, and sidebar local-storage preference.

- [ ] **Step 1: Write failing navigation and control tests**

```tsx
expect(screen.getByRole('navigation')).toHaveTextContent(/Data Master.*Procurement.*Logistics.*Reports.*Settings/s);
expect(screen.queryByRole('link', { name: 'Delivery Notes' })).not.toBeInTheDocument();
expect(screen.queryByRole('link', { name: 'Approval Inbox' })).not.toBeInTheDocument();
expect(within(screen.getByLabelText('Main navigation')).getByRole('button', { name: 'Collapse sidebar' })).toBeInTheDocument();
```

Also verify menu icon nodes render as `svg`, Settings expands on `/settings/users`, and the desktop header does not own the collapse button.

- [ ] **Step 2: Run the test and confirm the intended failure**

Run: `npm test -- --run components/app-shell/app-shell.test.tsx`
Expected: FAIL because the old order, abbreviations, and header toggle remain.

- [ ] **Step 3: Implement the minimal icon/navigation/sidebar revision**

Add internal SVG paths for dashboard, units, supplier, package, clipboard, receiving, outgoing, report, settings, users, shield, mail, and history. Move the toggle beside the brand, reorder groups, remove Delivery Notes and Approval Inbox links, and implement Settings expansion based on pathname plus manual toggle.

- [ ] **Step 4: Run focused tests**

Run: `npm test -- --run components/app-shell/app-shell.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/icons.tsx frontend/components/app-shell frontend/app/globals.css
git commit -m "feat: refine sidebar navigation and icons"
```

### Task 2: Header Notification Center

**Files:**
- Create: `frontend/components/notifications/notification-data.ts`
- Create: `frontend/components/notifications/notification-center.tsx`
- Create: `frontend/components/notifications/notification-center.test.tsx`
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Produces: `NotificationCenter({ items }: { items: NotificationItem[] })` and `emptyNotificationSnapshot`.
- Consumes: header placement from `AppShell` and internal bell icon.

- [ ] **Step 1: Write failing empty-state tests**

```tsx
render(<NotificationCenter items={[]} />);
expect(screen.queryByText(/unread/i)).not.toBeInTheDocument();
await user.click(screen.getByRole('button', { name: 'Notifications' }));
expect(screen.getByText('Belum ada notifikasi')).toBeInTheDocument();
expect(screen.getByRole('link', { name: 'View all notifications' })).toHaveAttribute('href', '/approvals');
```

- [ ] **Step 2: Verify red**

Run: `npm test -- --run components/notifications/notification-center.test.tsx`
Expected: FAIL because the component does not exist.

- [ ] **Step 3: Implement notification trigger and compact dropdown**

Use button state for open/closed behavior, render no badge for zero items, show the approved Indonesian empty copy, and place the component beside the authenticated display name.

- [ ] **Step 4: Verify green**

Run: `npm test -- --run components/notifications/notification-center.test.tsx components/app-shell/app-shell.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/notifications frontend/components/app-shell/app-shell.tsx frontend/app/globals.css
git commit -m "feat: add header notification center"
```

### Task 3: Supplier Order Documents, Reports, Settings, and Redirect Routes

**Files:**
- Modify: `frontend/app/supplier-orders/page.tsx`
- Create: `frontend/app/supplier-orders/supplier-order-config.ts`
- Create: `frontend/app/supplier-orders/supplier-order-config.test.ts`
- Modify: `frontend/app/delivery-notes/page.tsx`
- Create: `frontend/app/reports/page.tsx`
- Create: `frontend/app/settings/users/page.tsx`
- Create: `frontend/app/settings/roles/page.tsx`
- Create: `frontend/app/settings/smtp/page.tsx`
- Create: `frontend/app/settings/email-log/page.tsx`
- Create: `frontend/components/report-index.tsx`
- Create: `frontend/components/report-index.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `AppShell`, `ModuleIndex`, and Next.js `redirect`.
- Produces: live Reports and four Settings routes; Supplier Orders columns `PO Document`, `DN Documents`, and `Kanban Labels`.

- [ ] **Step 1: Write failing report and PO document tests**

```tsx
render(<ReportIndex />);
expect(screen.getByLabelText('Supplier')).toBeInTheDocument();
expect(screen.getByText('Belum ada data report')).toBeInTheDocument();
```

Add a pure exported `supplierOrderColumns` assertion containing `PO Document`, `DN Documents`, and `Kanban Labels`.

- [ ] **Step 2: Verify red**

Run: `npm test -- --run components/report-index.test.tsx app/supplier-orders/supplier-order-config.test.ts`
Expected: FAIL because reports and exported columns do not exist.

- [ ] **Step 3: Implement routes and index shells**

Build real-data-only report filters for date, supplier, raw material, PO, receiving, and status. Configure compact Settings pages for Users, Roles & Permissions, SMTP Settings, and Email Log. Replace the Delivery Notes page with `redirect('/supplier-orders')`.

- [ ] **Step 4: Run route and frontend verification**

Run: `npm test; npm run lint; npm run typecheck; npm run build`
Expected: all checks PASS; Next route output includes Reports and four Settings routes.

- [ ] **Step 5: Commit**

```powershell
git add frontend
git commit -m "feat: add reports settings and PO document access"
```

### Task 4: Docker and Browser-Path Verification

**Files:**
- Modify only when a verified runtime defect requires a fix.

**Interfaces:**
- Consumes: frontend port `3019`, backend port `8091`, PostgreSQL port `5445`.
- Produces: verified local runtime for the revised navigation slice.

- [ ] **Step 1: Check free space and rebuild frontend**

Run: inspect drive C, then `docker compose up -d --build frontend`.
Expected: sufficient writable space, image builds, and PostgreSQL remains healthy.

- [ ] **Step 2: Authenticate through the frontend proxy**

POST `/api/auth/login`, retain the cookie, then GET `/api/auth/me` through port 3019.
Expected: `admin`, one cookie, and 32 permissions.

- [ ] **Step 3: Smoke-test approved routes**

GET Dashboard, Data Master, Supplier Orders, Receiving, Outgoing Material, Reports, Approvals, and all Settings routes.
Expected: HTTP 200. Request Delivery Notes without automatic redirect following and expect a redirect whose location is `/supplier-orders`.

- [ ] **Step 4: Verify Git and runtime status**

Run: `git status --short`, `docker compose ps`, and recheck drive C.
Expected: clean worktree, running frontend/backend, healthy PostgreSQL, and writable free space.
