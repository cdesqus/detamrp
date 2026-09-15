# Production Planning Implementation Plan

**Goal:** Implement the user's approved standalone monthly planning workflow, SQL → API → UI, without Sales Orders.

**Architecture:** Extend the existing production PlanRepository using tenant-scoped PostgreSQL transactions and row locks. Gin routes enforce production permissions; Next.js pages use existing shell and table styles. Approved plans are immutable; no override permission is introduced until a separate revision workflow is specified.

**Tech Stack:** Go, pgx, PostgreSQL, Next.js, React, Vitest.

**Spec:** User-provided Production Planning design in this conversation.

- [x] SQL: Add snapshot columns and audit triggers in migration 029. Implement PlanRepository in backend/internal/production/store.go with automatic numbering, active master checks, draft-only changes, guarded transitions, linked orders and progress. Test workflow against PostgreSQL when available.
- [x] API: Add production/http.go, wire server.go and cmd/api/main.go, register permissions. Test authentication, invalid inputs and transitions. Server determines number, status and units.
- [x] UI: Add production-planning list/new/detail/edit routes and components, permission-aware navigation/actions, part lookup, totals, progress and audit history. Test creation, errors and status actions with Vitest.
- [x] Verify: Run Go tests, frontend tests, TypeScript and production build; inspect diff. Preserve restored baseline files and report any infrastructure limitations.

Production Order creation creates one released order per planning line, once per plan, inside the same locked transaction. Actual progress uses good quantities from the final operation only to avoid double-counting intermediate processing. Execution entry, costing and dashboard screens are outside this CRUD increment.

## Deployment and verification

Apply all pending migrations through `029_production_planning_details.sql` using the existing migration runner, then deploy the API and frontend together. Migration 028 grants production permissions to existing ADMIN roles. Other roles require `production.view`, `production.plan`, and/or `production.order` as appropriate.

Production Planning is available at `/production-planning`. Numbers use a transactional per-tenant/month counter, e.g. `PP-202609-000001`.

The SQL workflow integration test requires `PLANNING_TEST_DATABASE_URL` pointing to a **disposable** database with all migrations applied. It retains its tenant fixtures because audit logs are immutable. It exercises the actual Go store as `nextgen_app`, including FG/process outputs, active references, tenant isolation, rollback, snapshots, numbering, workflow guards, orders, progress and history. This session used PGlite with pgcrypto and its TCP adapter; use `default_query_exec_mode=exec` with that adapter to avoid its shared prepared-statement cache. This does not substitute for a multi-connection concurrency test on native PostgreSQL.

Frontend verification: 155 tests pass, TypeScript passes, targeted ESLint passes, and Next.js production build passes. The pre-existing report test was updated to select its report via the existing URL parameter instead of a removed tab button.
