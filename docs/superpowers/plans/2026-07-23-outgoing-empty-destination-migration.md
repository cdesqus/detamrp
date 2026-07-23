# Outgoing Empty Destination Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow Outgoing sessions to be created without a destination while retaining the 120-character database limit.

**Architecture:** Add one forward-only PostgreSQL migration that replaces the obsolete minimum-length constraint. Protect the behavior with a migration regression test and verify it through the live authenticated API after Docker applies the migration.

**Tech Stack:** PostgreSQL 17 migrations, Go 1.26 tests, Docker Compose.

## Global Constraints

- `outgoing_sessions.destination` remains `NOT NULL`.
- Empty and whitespace-only destinations are valid after trimming.
- Destinations longer than 120 characters remain invalid.
- No placeholder destination and no destination input are added.

---

### Task 1: Add the database regression fix

**Files:**
- Create: `database/migrations/009_outgoing_optional_destination.sql`
- Modify: `backend/internal/outgoing/migration_test.go`

**Interfaces:**
- Produces: an idempotent migration replacing `outgoing_sessions_destination_check`.

- [ ] **Step 1: Write the failing migration test**

Read migration `009_outgoing_optional_destination.sql` and assert it contains:

```go
required := []string{
    "drop constraint if exists outgoing_sessions_destination_check",
    "check (length(trim(destination)) <= 120)",
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/outgoing -run TestOptionalDestinationMigration -count=1`

Expected: FAIL because migration `009` does not exist.

- [ ] **Step 3: Add the migration**

Create:

```sql
BEGIN;
ALTER TABLE outgoing_sessions
  DROP CONSTRAINT IF EXISTS outgoing_sessions_destination_check;
ALTER TABLE outgoing_sessions
  ADD CONSTRAINT outgoing_sessions_destination_check
  CHECK (length(trim(destination)) <= 120);
COMMIT;
```

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/outgoing -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add database/migrations/009_outgoing_optional_destination.sql backend/internal/outgoing/migration_test.go
git commit -m "fix: allow outgoing sessions without destination"
```

### Task 2: Verify and deploy locally

**Files:**
- Verify only.

**Interfaces:**
- Consumes the migration from Task 1.
- Produces a working local Outgoing creation flow.

- [ ] **Step 1: Run full backend verification**

Run:

```powershell
$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'
go test ./... -count=1
go vet ./...
```

Expected: all packages PASS.

- [ ] **Step 2: Rebuild Docker**

Run:

```powershell
docker compose -p platform-master-po up -d --build
docker compose -p platform-master-po ps
```

Expected: migration exits successfully and all services run.

- [ ] **Step 3: Smoke-test the exact failure**

Authenticate as the local administrator and send:

```json
{"destination":"","notes":""}
```

to `POST http://localhost:8091/outgoing-sessions`.

Expected: HTTP `201` with a new active session.

- [ ] **Step 4: Clean smoke-test data**

Delete the newly created empty test session directly through the local PostgreSQL admin connection after verifying it has no scans. This cleanup is limited to the exact session ID returned by Step 3.
