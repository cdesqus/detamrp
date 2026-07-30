# Professional Email Templates Implementation Plan

> **For agentic workers:** Execute inline with strict red-green-refactor cycles; do not delegate.

**Goal:** Deliver responsive, light, company-branded emails with clear PO, plant, material, decision, and shipping information.

**Architecture:** Build a shared email view model and shell in `backend/internal/emailing`, with branding loaded once per send. Embed the company logo as a CID inline MIME part so it works without authentication. Preserve the existing SMTP transport and PDF attachment flow, and add a decision-result notification after public approve/reject actions.

**Tech Stack:** Go, PostgreSQL, HTML email with inline styles, MIME multipart, existing SMTP transport.

## Global Constraints

- Company logo falls back to aligned company-name and DETA MRP text when absent or invalid.
- Approval URLs and all user/database content remain HTML escaped.
- Supplier attachments remain capped at 20 MB.
- A completed approval/rejection is never rolled back because notification delivery fails.

---

### Task 1: Shared branded email shell and inline logo

**Files:** `backend/internal/emailing/domain.go`, `templates.go`, `smtp.go`, `crypto_test.go`, `service_test.go`, `store.go`

- [ ] Add failing template and MIME tests for company name, DETA MRP fallback, responsive presentation attributes, CID logo reference, inline disposition, and safe escaping.
- [ ] Run focused tests and confirm failures describe the missing branding behavior.
- [ ] Add a `CompanyBranding` store contract, shared shell view model, optional CID attachment fields, and MIME inline encoding.
- [ ] Run focused tests until green and commit.

### Task 2: Detailed approval and SMTP test emails

**Files:** `backend/internal/emailing/domain.go`, `templates.go`, `service.go`, `store.go`, corresponding tests

- [ ] Add failing tests for Supplier, Plant, dates, status, total, Category, Packing, quantity/card, cards, total quantity, prices, notes, and action hierarchy.
- [ ] Extend approval snapshot queries and render the compact summary/material tables.
- [ ] Route SMTP test through the same branded shell with a concise configuration-success summary.
- [ ] Run focused and package tests until green and commit.

### Task 3: Detailed supplier delivery email

**Files:** `backend/internal/emailing/templates.go`, `service.go`, `service_test.go`

- [ ] Add a failing supplier-message test covering Plant, PO/DN references, material Category/Packing data, quantities, attachment list, and print/ship instructions.
- [ ] Render supplier data from the existing PO snapshots so later master changes cannot alter the email.
- [ ] Preserve the three PDF attachments and 20 MB validation.
- [ ] Run focused and package tests until green and commit.

### Task 4: Approval/rejection result notification

**Files:** `database/migrations/015_email_decision_results.sql`, `backend/internal/emailing/domain.go`, `store.go`, `service.go`, `http.go`, tests

- [ ] Add failing migration, service, and template tests for a `DECISION` email log type and result details: PO, Supplier, Plant, status, decision actor, time, and rejection reason.
- [ ] Add the result data query and branded result template.
- [ ] Send the result to the PO creator after token use; keep the business decision successful if notification delivery fails and show an accurate public-page message.
- [ ] Run focused and package tests until green and commit.

### Task 5: Verification and integration

**Files:** all changed Phase 5 files

- [ ] Run `go test ./...`, `git diff --check`, and inspect rendered HTML/MIME assertions for escaping and secret leakage.
- [ ] Merge into `main`, rerun the backend suite, remove only the Phase 5 worktree, delete the feature branch, and push `main`.
