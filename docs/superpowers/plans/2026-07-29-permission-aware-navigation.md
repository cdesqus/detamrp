# Permission-Aware Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hide unauthorized modules and empty navigation groups, protect direct frontend routes with an explicit Access Denied state, and give the complete Dashboard its own `dashboard.view` permission.

**Architecture:** Keep authorization metadata and pure filtering/matching helpers in `navigation.ts`; AppShell consumes those helpers after `/api/auth/me` resolves and conditionally mounts route content. Backend RBAC remains authoritative, with the Dashboard endpoint switching from `inventory.view` to the new catalog permission.

**Tech Stack:** Next.js/React/TypeScript, Vitest and Testing Library, Go/Gin RBAC middleware, Go testing.

## Global Constraints

- Backend middleware remains the security boundary.
- `/api/auth/me` permissions remain the frontend source of truth.
- Unauthorized module children must not mount or initiate API requests.
- Unknown routes retain normal Next.js not-found behavior.
- Dashboard data remains complete and requires `dashboard.view`.
- Existing roles are not automatically granted `dashboard.view`.
- No new dependencies or database migration.

---

### Task 1: Dashboard Permission

**Files:**
- Modify: `backend/internal/rbac/catalog.go`
- Modify: `backend/internal/rbac/service_test.go`
- Modify: `backend/internal/dashboard/http.go`
- Modify: `backend/internal/dashboard/http_test.go`

**Interfaces:**
- Produces: RBAC code `dashboard.view`
- Produces: Dashboard GET authorization requiring `dashboard.view`

- [ ] **Step 1: Write failing catalog and HTTP authorization tests**

Add `dashboard.view` to the required catalog cases. In Dashboard HTTP tests, authenticate one request with `dashboard.view` and assert success; authenticate another with only `inventory.view` and assert `403`.

```go
if _, ok := Catalog["dashboard.view"]; !ok {
	t.Fatal("catalog missing dashboard.view")
}
```

- [ ] **Step 2: Run tests and verify RED**

```powershell
Set-Location backend
go test ./internal/rbac ./internal/dashboard -count=1
```

Expected: FAIL because the catalog lacks `dashboard.view` and the endpoint still accepts `inventory.view`.

- [ ] **Step 3: Implement the new permission**

Add `"dashboard.view": "View dashboard"` to `rbac.Catalog` and change the Dashboard route middleware to:

```go
rbac.RequirePermissions("dashboard.view")
```

- [ ] **Step 4: Run tests and verify GREEN**

```powershell
Set-Location backend
go test ./internal/rbac ./internal/dashboard -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/rbac/catalog.go backend/internal/rbac/service_test.go backend/internal/dashboard/http.go backend/internal/dashboard/http_test.go
git commit -m "feat: add dedicated dashboard view permission"
```

### Task 2: Pure Navigation and Route Policy

**Files:**
- Modify: `frontend/components/app-shell/navigation.ts`
- Create: `frontend/components/app-shell/navigation.test.ts`

**Interfaces:**
- Produces: `NavigationItem.requiredPermission: string`
- Produces: `visibleNavigationGroups(permissions: string[]): NavigationGroup[]`
- Produces: `requiredPermissionForPath(pathname: string): string | null`
- Produces: `firstPermittedRoute(permissions: string[]): string | null`

- [ ] **Step 1: Write failing pure-policy tests**

Test literal permission mappings, partial groups, empty-group removal, specific route precedence, and deterministic first route:

```ts
expect(requiredPermissionForPath('/supplier-orders/new')).toBe('po.create');
expect(requiredPermissionForPath('/supplier-orders/123')).toBe('po.view');
expect(visibleNavigationGroups(['role.manage']).flatMap(group => group.items).map(item => item.label))
  .toEqual(['Roles & Permissions']);
expect(firstPermittedRoute(['inventory.view'])).toBe('/inventory');
```

- [ ] **Step 2: Run tests and verify RED**

```powershell
Set-Location frontend
npx vitest run components/app-shell/navigation.test.ts
```

Expected: FAIL because the policy helpers and permission metadata do not exist.

- [ ] **Step 3: Add permission metadata and pure helpers**

Extend every item with the exact mapping from the design spec. Implement filtering without mutating `navigationGroups`:

```ts
export function visibleNavigationGroups(permissions: string[]) {
  const granted = new Set(permissions);
  return navigationGroups
    .map(group => ({ ...group, items: group.items.filter(item => granted.has(item.requiredPermission)) }))
    .filter(group => group.items.length > 0);
}
```

Implement ordered route rules with `/supplier-orders/new` before `/supplier-orders`, exact/prefix boundary matching, and return `null` for unknown routes. Implement `firstPermittedRoute` from the filtered group/item order.

- [ ] **Step 4: Run tests and verify GREEN**

```powershell
Set-Location frontend
npx vitest run components/app-shell/navigation.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/app-shell/navigation.ts frontend/components/app-shell/navigation.test.ts
git commit -m "feat: define frontend route permission policy"
```

### Task 3: Filtered AppShell and Direct-Route Guard

**Files:**
- Modify: `frontend/components/app-shell/app-shell.tsx`
- Modify: `frontend/components/app-shell/app-shell.test.tsx`

**Interfaces:**
- Consumes: `visibleNavigationGroups`
- Consumes: `requiredPermissionForPath`
- Consumes: `firstPermittedRoute`
- Produces: Access Denied content that does not mount unauthorized `children`

- [ ] **Step 1: Write failing navigation visibility tests**

Authenticate fixtures with explicit permissions. Assert a `role.manage` user sees Roles & Permissions but not Users, SMTP, Dashboard, or the empty Data Master group. Assert a `dashboard.view` user sees Dashboard.

```tsx
expect(screen.getByRole('link', { name: 'Roles & Permissions' })).toBeInTheDocument();
expect(screen.queryByRole('link', { name: 'Users' })).not.toBeInTheDocument();
expect(screen.queryByRole('button', { name: 'Data Master' })).not.toBeInTheDocument();
```

- [ ] **Step 2: Write a failing direct-route mount test**

Use a child component whose effect increments a spy. Open `/inventory` with only `role.manage`; assert Access Denied, a link to `/settings/roles`, and zero child effect calls.

```tsx
function ProtectedChild() {
  useEffect(() => mounted(), []);
  return <div>protected inventory</div>;
}
expect(await screen.findByRole('heading', { name: 'Access Denied' })).toBeInTheDocument();
expect(mounted).not.toHaveBeenCalled();
```

- [ ] **Step 3: Run AppShell tests and verify RED**

```powershell
Set-Location frontend
npx vitest run components/app-shell/app-shell.test.tsx
```

Expected: FAIL because AppShell renders the static catalog and always mounts children.

- [ ] **Step 4: Render filtered navigation**

After user resolution, compute `visibleGroups`. Use it for navigation rendering and active/open group behavior. Do not render a collapsible group with zero visible items.

- [ ] **Step 5: Add the Access Denied boundary**

Resolve the pathname permission. When a known route is unauthorized, render the shell and sidebar normally but replace `<main>{children}</main>` with:

```tsx
<main>
  <section className="module-index">
    <div className="table-empty" role="alert">
      <h1>Access Denied</h1>
      <span>You do not have permission to access this module.</span>
      {firstRoute
        ? <Link className="table-action" href={firstRoute}>Go to an available module</Link>
        : <button className="table-action" onClick={() => void logout()}>Logout</button>}
    </div>
  </section>
</main>
```

The conditional must select between the denial element and `children`, ensuring React never mounts unauthorized children.

- [ ] **Step 6: Run focused and complete frontend tests**

```powershell
Set-Location frontend
npx vitest run components/app-shell/navigation.test.ts components/app-shell/app-shell.test.tsx
npm test -- --run
npm run lint
```

Expected: PASS with no lint errors. Update existing AppShell fixtures to use the permissions required by the links they assert.

- [ ] **Step 7: Commit**

```powershell
git add frontend/components/app-shell/app-shell.tsx frontend/components/app-shell/app-shell.test.tsx
git commit -m "feat: hide unauthorized modules in app shell"
```

### Task 4: Full Verification

**Files:**
- Modify only if a regression test exposes a defect in files from Tasks 1–3.

**Interfaces:**
- Verifies the complete permission-aware navigation contract.

- [ ] **Step 1: Run complete frontend verification**

```powershell
Set-Location frontend
npm test -- --run
npm run lint
```

Expected: PASS.

- [ ] **Step 2: Run complete backend verification**

```powershell
Set-Location backend
go test ./... -count=1
go vet ./...
```

Expected: PASS and exit code 0.

- [ ] **Step 3: Inspect repository state**

```powershell
git diff --check
git status --short
git log -6 --oneline
```

Expected: no whitespace errors and no uncommitted implementation changes.
