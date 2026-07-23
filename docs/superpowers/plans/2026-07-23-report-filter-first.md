# Report Filter-First Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent Receiving Report data from loading before the user supplies a required date range and applies the filters.

**Architecture:** Tighten the existing report filter parser so both dates are required, then convert the React report screen from auto-loading to an explicit submitted-filter state. The existing JSON/PDF endpoints, query, and PDF renderer remain unchanged.

**Tech Stack:** Go, Gin, Next.js, React, TypeScript, Vitest.

## Global Constraints

- From Date and To Date are required.
- Supplier and Reference remain optional.
- No report request occurs on initial page load.
- Results and Export PDF remain hidden until a filter submission succeeds.
- Reset returns to the initial filter-only state.

---

### Task 1: Require report date range

**Files:**
- Modify: `backend/internal/report/service_test.go`
- Modify: `backend/internal/report/service.go`

- [ ] Write tests asserting missing From Date and To Date produce field errors.
- [ ] Run `go test ./internal/report -run TestParseFilter -count=1` and observe failure.
- [ ] Add required-field validation without changing existing ISO date or reversed-range validation.
- [ ] Run `go test ./internal/report -count=1` and observe success.

### Task 2: Make report loading filter-first

**Files:**
- Modify: `frontend/components/report-index.test.tsx`
- Modify: `frontend/components/report-index.tsx`

- [ ] Write tests asserting initial render fetches suppliers only, results/export are absent, Apply is disabled until both dates are supplied, Apply fetches the report, and Reset hides results.
- [ ] Run `npm test -- report-index.test.tsx` and observe failure.
- [ ] Replace automatic report loading with an explicit applied state and display validation near missing dates.
- [ ] Run the focused frontend test and observe success.

### Task 3: Verify and deploy

**Files:**
- Modify only when a failing verification has a regression test.

- [ ] Run backend tests and vet.
- [ ] Run frontend tests, typecheck, lint, and build.
- [ ] Commit the tested change.
- [ ] Merge locally after approval, rebuild the active stack, and smoke-test the report page and endpoints.
