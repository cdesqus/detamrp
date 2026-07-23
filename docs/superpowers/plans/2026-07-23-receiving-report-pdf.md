# Receiving Report PDF Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build one tenant-scoped Receiving Report backed by completed receiving transactions, with screen filters and A4 landscape PDF export.

**Architecture:** Add a focused `internal/report` backend module whose store owns the flattened receiving query and whose service renders the PDF from the same result model returned as JSON. Register authenticated report endpoints through the existing API composition root, then replace the static React report placeholder with a data-driven component.

**Tech Stack:** Go 1.26, Gin, pgx/PostgreSQL, go-pdf/fpdf, Next.js 16, React 19, TypeScript, Vitest/Testing Library.

## Global Constraints

- Only Receiving Report is in scope.
- Export format is PDF only; do not add Excel/CSV.
- Use one row per receiving material.
- Filters are From Date, To Date, Supplier, and a reference search covering Receiving, DN, and PO numbers.
- All reads must be tenant-scoped and require the existing authenticated `receiving.view` permission.
- The JSON screen and PDF export must use the same filter model and query.
- Prices are excluded.
- No database migration is required.

---

### Task 1: Receiving report query and totals

**Files:**
- Create: `backend/internal/report/model.go`
- Create: `backend/internal/report/store.go`
- Create: `backend/internal/report/store_test.go`

**Interfaces:**
- Produces: `Filter`, `Row`, `Totals`, `Result`, `Actor`, and `Store.ListReceiving(context.Context, Actor, Filter) (Result, error)`.
- Consumes: `database.WithTenantTx` and existing receiving, DN, PO, supplier, Kanban, raw-material snapshot, and user tables.

- [ ] **Step 1: Write failing store tests**

Test a pgx query recorder with a result containing two materials. Assert the SQL receives `tenantID`, `fromDate`, `toDate`, `supplierID`, and trimmed `search`; assert rows retain receiving/DN/PO/supplier/material/unit/Sage/creator fields; assert `TotalKanban` and `TotalQuantity` sum the returned rows.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/report -run TestStore -count=1`

Expected: FAIL because `Store`, report models, and `ListReceiving` do not exist.

- [ ] **Step 3: Implement the query**

Create a single grouped query rooted at `receivings`, joining `receiving_kanban_lots`, `kanban_lots`, `purchase_order_lines`, `delivery_notes`, `purchase_orders`, `suppliers`, and `users`. Group by receiving and purchase-order-line snapshot, count Kanban lots, sum received quantity, and use `r.outstanding_kanban * pol.qty_per_kanban_snapshot` only as the material outstanding quantity for lots of that PO line that remain `ISSUED`. Apply all filters with nullable/empty arguments and order by receiving date descending, receiving number, then material code.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `go test ./internal/report -run TestStore -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/report
git commit -m "feat: add receiving report query"
```

### Task 2: Filter validation and landscape PDF

**Files:**
- Create: `backend/internal/report/service.go`
- Create: `backend/internal/report/pdf.go`
- Create: `backend/internal/report/service_test.go`
- Create: `backend/internal/report/pdf_test.go`

**Interfaces:**
- Consumes: `Store.ListReceiving`, `Filter`, and `Result`.
- Produces: `ParseFilter(url.Values) (Filter, FieldErrors)`, `Service.Receiving`, and `Service.ReceivingPDF`.

- [ ] **Step 1: Write failing validation and PDF tests**

Assert ISO dates parse, invalid dates produce field errors, From Date after To Date is rejected, the PDF begins `%PDF-`, uses landscape pages, contains report title and transaction fields, includes total Kanban/quantity, exports a valid empty report, and paginates a long result.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/report -run "TestParseFilter|TestReceivingPDF" -count=1`

Expected: FAIL because validation, service, and renderer do not exist.

- [ ] **Step 3: Implement filter parsing and PDF rendering**

Parse `YYYY-MM-DD` dates, optional UUID supplier, and a trimmed search. Render A4 landscape with compact fonts, repeated table header, generated timestamp, filter summary, totals, page numbering, and an honest empty state. Format quantity without redundant trailing zeroes.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `go test ./internal/report -run "TestParseFilter|TestReceivingPDF" -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/report
git commit -m "feat: render receiving report PDF"
```

### Task 3: Authenticated report HTTP routes

**Files:**
- Create: `backend/internal/report/http.go`
- Create: `backend/internal/report/http_test.go`
- Modify: `backend/internal/api/server.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: existing session `Authenticator`, RBAC middleware, `Service.Receiving`, and `Service.ReceivingPDF`.
- Produces: `GET /reports/receiving` and `GET /reports/receiving.pdf`.

- [ ] **Step 1: Write failing route tests**

Assert unauthenticated requests return 401, users without `receiving.view` return 403, invalid filter fields return 422, JSON returns `{items, totals}`, and PDF returns `application/pdf` with an attachment filename.

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/report -run TestHTTP -count=1`

Expected: FAIL because routes are not registered.

- [ ] **Step 3: Implement and register routes**

Reuse the session cookie authentication pattern, derive tenant/user actor context, require `receiving.view`, and make JSON and PDF pass the same parsed `Filter`. Add `WithReportService` to API options and wire the SQL store/service in `cmd/api/main.go`.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `go test ./internal/report ./internal/api ./cmd/api -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add backend/internal/report backend/internal/api/server.go backend/cmd/api/main.go
git commit -m "feat: expose receiving report endpoints"
```

### Task 4: Data-driven Reports screen and PDF export

**Files:**
- Modify: `frontend/components/report-index.tsx`
- Modify: `frontend/components/report-index.test.tsx`
- Modify: `frontend/app/globals.css`

**Interfaces:**
- Consumes: `/api/reports/receiving`, `/api/reports/receiving.pdf`, and `/api/master-data/suppliers?active=true&limit=200`.
- Produces: interactive filters, compact results table, totals footer, reset, loading/error/empty states, and a PDF export link.

- [ ] **Step 1: Write failing frontend tests**

Mock suppliers and report responses. Assert initial real-data load, filters are encoded in JSON and PDF URLs, Apply reloads, Reset clears inputs and reloads, rows/totals render, errors display, and Export PDF is disabled only while loading or when validation fails.

- [ ] **Step 2: Run tests and verify RED**

Run: `npm test -- report-index.test.tsx`

Expected: FAIL because the current page is a static placeholder with disabled buttons.

- [ ] **Step 3: Implement the component**

Use controlled compact filters, URLSearchParams, credentials-enabled fetches, and a standard anchor opening the authenticated PDF endpoint in a new tab. Render the agreed eleven report columns in a horizontally scrollable compact table and show server field validation near filters.

- [ ] **Step 4: Run tests and verify GREEN**

Run: `npm test -- report-index.test.tsx`

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add frontend/components/report-index.tsx frontend/components/report-index.test.tsx frontend/app/globals.css
git commit -m "feat: build receiving report screen"
```

### Task 5: Full verification and local deployment

**Files:**
- Modify only if verification exposes a tested defect.

- [ ] **Step 1: Verify backend**

Run: `go test -p 1 ./... -count=1` and `go vet ./...` from `backend`.

Expected: all packages PASS and vet exits zero.

- [ ] **Step 2: Verify frontend**

Run: `npm test`, `npm run typecheck`, `npm run lint`, and `npm run build` from `frontend`.

Expected: all commands exit zero.

- [ ] **Step 3: Check repository hygiene**

Run: `git diff --check` and `git status --short`.

Expected: no whitespace errors and only intentional changes.

- [ ] **Step 4: Rebuild the active local stack**

Run: `docker compose -p platform-master-po up -d --build backend frontend` (or reuse successfully built local images if Docker Desktop hits the known Windows paging-file limitation).

Expected: PostgreSQL healthy; backend and frontend running.

- [ ] **Step 5: Smoke test**

Assert `http://localhost:8091/health` and `http://localhost:3019` return HTTP 200, then authenticate and verify the report JSON and PDF endpoints return successful responses.

- [ ] **Step 6: Confirm the implementation commits are complete**

Run: `git log --oneline -5`

Expected: the query, PDF/HTTP, and frontend report commits are present; create no additional commit when verification made no changes.
