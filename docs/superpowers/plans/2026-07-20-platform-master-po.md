# Platform, Master Data, and Supplier Order Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menghasilkan vertical slice runnable untuk login username/password, RBAC, tenant-safe master data, dan pembuatan PO draft/submission dengan dependent raw-material picker.

**Architecture:** npm-workspaces monorepo dengan Next.js web dan NestJS API. PostgreSQL diakses melalui Prisma; service layer selalu menerima tenant context dan database memiliki composite tenant constraints. Web memakai compact design tokens, server-side API client, dan permission-aware navigation.

**Tech Stack:** Node.js 24, TypeScript, npm workspaces, Next.js App Router, NestJS, Prisma, PostgreSQL 17, Vitest/Jest, Playwright, Docker Compose.

## Global Constraints

- UI table row 36–40px dan body font 13–14px.
- `tenant_id` tidak pernah ditampilkan pada UI prototype.
- Semua tenant-owned relations memakai composite tenant foreign keys.
- Kanban bukan measurement unit.
- Harga hanya diinput pada PO; `qty_per_kanban` berasal dari Raw Material master.
- Satu PO memiliki tepat satu supplier dan hanya dapat memakai material supplier tersebut.
- Tidak ada `Required Date` per material dan tidak ada status `PLANNED`.
- Semua index table menyediakan search, filters, show/hide columns, dan audit columns.
- Gunakan TDD untuk setiap domain behavior.

---

### Task 1: Monorepo runtime dan quality gates

**Files:**
- Create: `package.json`, `tsconfig.base.json`, `.editorconfig`, `.gitignore`, `.env.example`
- Create: `docker-compose.yml`
- Create: `apps/api/package.json`, `apps/web/package.json`
- Create: `apps/api/src/main.ts`, `apps/web/app/page.tsx`

**Interfaces:**
- Produces: npm scripts `dev`, `build`, `lint`, `test`, `typecheck`; services `postgres:5432`, `redis:6379`, `minio:9000`.

- [ ] Create root npm workspace scripts and shared strict TypeScript configuration.
- [ ] Scaffold NestJS API with `GET /health` returning `{ "status": "ok" }`.
- [ ] Scaffold Next.js App Router page rendering `NextGen Logistics`.
- [ ] Add Docker Compose services with named volumes and health checks.
- [ ] Add API health test and run `npm test` expecting PASS.
- [ ] Run `npm run typecheck && npm run build` expecting exit 0.
- [ ] Commit `chore: scaffold logistics monorepo`.

### Task 2: Prisma tenant-safe foundation

**Files:**
- Create: `apps/api/prisma/schema.prisma`
- Create: `apps/api/prisma/migrations/0001_foundation/migration.sql`
- Create: `apps/api/src/database/prisma.service.ts`
- Create: `apps/api/src/tenancy/tenant-context.ts`
- Test: `apps/api/test/tenancy.e2e-spec.ts`

**Interfaces:**
- Produces: `TenantContext { tenantId: string; userId: string }`, `PrismaService.withTenant<T>(context, callback): Promise<T>`.

- [ ] Write an integration test creating two tenants and proving tenant A cannot read tenant B records.
- [ ] Run the tenancy test and verify failure before schema/policies exist.
- [ ] Define `Tenant`, audit columns, composite unique keys, app database role, RLS policies, and `FORCE ROW LEVEL SECURITY` migration SQL.
- [ ] Implement transaction-local `set_config('app.tenant_id', tenantId, true)` in `withTenant`.
- [ ] Apply migration and rerun test expecting PASS.
- [ ] Commit `feat: add tenant-safe database foundation`.

### Task 3: Local authentication and sessions

**Files:**
- Create: `apps/api/src/auth/auth.module.ts`, `auth.service.ts`, `auth.controller.ts`, `password.service.ts`, `session.guard.ts`
- Create: `apps/api/src/auth/dto/login.dto.ts`
- Modify: `apps/api/prisma/schema.prisma`
- Test: `apps/api/src/auth/auth.service.spec.ts`, `apps/api/test/auth.e2e-spec.ts`

**Interfaces:**
- Produces: `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`; httpOnly secure session cookie; `AuthenticatedUser { id, tenantId, username, displayName, permissions }`.

- [ ] Write failing tests for valid login, invalid credentials, locked account, logout, and protected route.
- [ ] Add User and Session models scoped by tenant.
- [ ] Implement Argon2id password hashing and constant-shape invalid-login response.
- [ ] Implement opaque hashed session tokens, expiry, secure cookie settings, and authentication guard.
- [ ] Add development seed for tenant `OUR_COMPANY` and admin user using env-provided initial password.
- [ ] Run auth unit/e2e tests expecting PASS.
- [ ] Commit `feat: add local username authentication`.

### Task 4: RBAC permissions

**Files:**
- Create: `apps/api/src/rbac/rbac.module.ts`, `permissions.decorator.ts`, `permissions.guard.ts`
- Create: `apps/api/src/rbac/permission-catalog.ts`
- Modify: `apps/api/prisma/schema.prisma`, seed
- Test: `apps/api/src/rbac/permissions.guard.spec.ts`

**Interfaces:**
- Produces: `@RequirePermissions(...permissions: PermissionKey[])`; roles ADMIN, DIRECTOR, PURCHASING, LOGISTICS_PLANNER, WAREHOUSE, FINANCE, VIEWER.

- [ ] Write failing guard tests for allow, deny, multi-role union, and tenant isolation.
- [ ] Add Role, Permission, UserRole, and RolePermission models with tenant-aware keys.
- [ ] Define exact permission catalog from the approved spec and seed default mappings.
- [ ] Implement decorator and guard using session user permissions.
- [ ] Run tests expecting PASS.
- [ ] Commit `feat: enforce permission-based access control`.

### Task 5: Master data API

**Files:**
- Create: `apps/api/src/master-data/measurements/*`
- Create: `apps/api/src/master-data/suppliers/*`
- Create: `apps/api/src/master-data/raw-materials/*`
- Create: `apps/api/src/master-data/warehouses/*`
- Create: `apps/api/src/master-data/locations/*`
- Modify: `apps/api/prisma/schema.prisma`
- Test: `apps/api/test/master-data.e2e-spec.ts`

**Interfaces:**
- Produces: paginated CRUD endpoints under `/measurements`, `/suppliers`, `/raw-materials`, `/warehouses`, `/locations`; common response `{ data, page, pageSize, total }`.

- [ ] Write failing e2e tests for CRUD, searchable pagination, duplicate code, inactive records, audit actor, and forbidden cross-tenant reference.
- [ ] Add tenant-scoped models and composite foreign keys; require supplier email and positive `qtyPerKanban`.
- [ ] Implement focused repository/service/controller files per aggregate with DTO validation.
- [ ] Implement deactivate instead of delete for referenced master records.
- [ ] Implement raw-material search filtered by `supplierId` and active status.
- [ ] Run master-data e2e tests expecting PASS.
- [ ] Commit `feat: add tenant-safe master data APIs`.

### Task 6: Compact web shell, login, and reusable data table

**Files:**
- Create: `apps/web/app/login/page.tsx`, `apps/web/app/(app)/layout.tsx`
- Create: `apps/web/components/app-shell/*`, `components/data-table/*`, `components/combobox/*`
- Create: `apps/web/lib/api-client.ts`, `lib/session.ts`, `lib/column-preferences.ts`
- Create: `apps/web/app/globals.css`
- Test: `apps/web/components/data-table/data-table.test.tsx`, `apps/web/e2e/login.spec.ts`

**Interfaces:**
- Produces: `DataTable<T>`, `SearchableCombobox<T>`, collapsed sidebar, permission-filtered navigation, local persisted column preferences.

- [ ] Write failing component tests for 38px rows, column visibility, persisted preferences, and combobox keyboard selection.
- [ ] Implement compact design tokens and responsive application shell.
- [ ] Implement login form and authenticated API client with credentials.
- [ ] Implement permission-aware menu and user/logout control.
- [ ] Implement reusable server-search combobox and data table filters/columns controls.
- [ ] Run component and login e2e tests expecting PASS.
- [ ] Commit `feat: add compact authenticated application shell`.

### Task 7: Master data web modules

**Files:**
- Create: `apps/web/app/(app)/master-data/measurements/*`
- Create: `apps/web/app/(app)/master-data/suppliers/*`
- Create: `apps/web/app/(app)/master-data/raw-materials/*`
- Create: `apps/web/app/(app)/master-data/warehouses/*`
- Create: `apps/web/app/(app)/master-data/locations/*`
- Test: `apps/web/e2e/master-data.spec.ts`

**Interfaces:**
- Consumes: master APIs, `DataTable<T>`, `SearchableCombobox<T>`.
- Produces: compact list/create/edit flows with audit columns and dependent warehouse/location or supplier/material selectors.

- [ ] Write failing Playwright flow creating measurement, supplier, raw material, warehouse, and location.
- [ ] Implement shared compact page header, drawer/form layout, validation messages, filters, and Columns control.
- [ ] Implement each master module without permanent delete action.
- [ ] Verify inactive material is absent from PO picker API.
- [ ] Run Playwright flow expecting PASS.
- [ ] Commit `feat: add master data management UI`.

### Task 8: Purchase Order domain and API

**Files:**
- Create: `apps/api/src/purchase-orders/purchase-orders.module.ts`, `purchase-orders.service.ts`, `purchase-orders.controller.ts`, `purchase-order-number.service.ts`
- Create: `apps/api/src/purchase-orders/dto/*`
- Modify: `apps/api/prisma/schema.prisma`
- Test: `apps/api/src/purchase-orders/purchase-orders.service.spec.ts`, `apps/api/test/purchase-orders.e2e-spec.ts`

**Interfaces:**
- Produces: `POST /purchase-orders`, `PATCH /purchase-orders/:id`, `POST /purchase-orders/:id/submit`, `GET /purchase-orders`, `GET /purchase-orders/:id`; statuses DRAFT and PENDING_APPROVAL in this slice.

- [ ] Write failing domain tests for decimal formulas, positive integer Kanban count, supplier/material mismatch, immutable submitted PO, and concurrent PO numbering.
- [ ] Add PurchaseOrder and PurchaseOrderLine models with numeric snapshots and optimistic `version`.
- [ ] Implement tenant-scoped PO numbering and transactional create/update.
- [ ] Implement material ownership validation in service and database trigger/constraint migration.
- [ ] Implement draft submission that records submitter/time and freezes commercial fields.
- [ ] Run PO unit/e2e tests expecting PASS.
- [ ] Commit `feat: add supplier order domain and APIs`.

### Task 9: Create PO and Order List web flows

**Files:**
- Create: `apps/web/app/(app)/procurement/orders/page.tsx`
- Create: `apps/web/app/(app)/procurement/orders/new/page.tsx`
- Create: `apps/web/app/(app)/procurement/orders/[id]/page.tsx`
- Create: `apps/web/features/purchase-orders/*`
- Test: `apps/web/e2e/purchase-order.spec.ts`

**Interfaces:**
- Consumes: PO/master APIs and reusable table/combobox.
- Produces: `Save as Draft`, `Save & Send for Approval`, `+ Raw Material`, calculated totals, supplier-dependent material search.

- [ ] Write failing Playwright tests proving supplier is chosen once, material results belong only to that supplier, totals calculate correctly, and submission freezes edits.
- [ ] Implement compact header form with Supplier, Order Date, Expected Delivery Date, Currency, and Notes.
- [ ] Implement `+ Raw Material` picker disabled until supplier selection; do not render supplier again inside material editor.
- [ ] Implement Qty/Kanban read-only snapshot, Total Kanban and Unit Price inputs, and decimal totals.
- [ ] Implement supplier-change confirmation that clears selected materials.
- [ ] Implement Order List filters, show/hide audit columns, and Sage Number placeholder column.
- [ ] Run Playwright PO flow expecting PASS.
- [ ] Commit `feat: add compact supplier order workflow`.

### Task 10: Slice verification and operator documentation

**Files:**
- Create: `README.md`
- Create: `docs/runbooks/local-development.md`
- Create: `docs/runbooks/initial-admin.md`
- Modify: `.env.example`

**Interfaces:**
- Produces: reproducible local startup and seeded prototype walkthrough.

- [ ] Document exact environment keys, Docker startup, migration, seed, dev, test, and build commands.
- [ ] Start clean dependencies and apply migrations.
- [ ] Run `npm run lint`, `npm run typecheck`, `npm test`, and `npm run build`; all must exit 0.
- [ ] Run Playwright master-data and PO journeys; all must pass.
- [ ] Manually verify compact table density at desktop and responsive navigation at mobile viewport.
- [ ] Commit `docs: add platform slice runbooks`.
