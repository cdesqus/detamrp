# Email Delivery and Formatting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Send approval email during Save & Send, render business-friendly numbers and deployment URLs, and prove supplier email attachments are correct.

**Architecture:** Keep the existing email endpoints and SMTP service boundaries. Add presentation-only decimal formatting in the backend template, call the existing approval-email endpoint after a successful PO submission, and inject the public origin through Docker Compose environment expansion.

**Tech Stack:** Go, Gin, PostgreSQL, React, Next.js, Vitest, Docker Compose.

## Global Constraints

- Database decimal values remain unchanged; formatting applies only to email HTML.
- Approval tokens remain single-use and resend invalidates older tokens.
- Supplier email remains manually triggered after PO approval.
- Local `APP_BASE_URL` defaults to `http://localhost:3019`.
- Production sets `APP_BASE_URL=https://deta.kaumtech.com`.

---

### Task 1: Business Number Formatting in Approval Email

**Files:**
- Modify: `backend/internal/emailing/templates.go`
- Test: `backend/internal/emailing/crypto_test.go`

**Interfaces:**
- Consumes: decimal strings loaded into `ApprovalMailData`.
- Produces: `formatEmailNumber(value string) string`, used only by email presentation.

- [ ] **Step 1: Write failing template test**

Add an approval with `TotalAmount: "2000000.000000"` and lines containing
`"5.000000"`, `"5.250000"`, and `"50.000000"`. Assert the HTML contains
`IDR 2.000.000`, `5 PC`, `5,25 BOX`, and `50 PC`, and does not contain
`.000000`.

- [ ] **Step 2: Verify red**

Run: `go test ./internal/emailing -run TestApprovalTemplateFormatsBusinessNumbers -v`
Expected: FAIL because raw decimal strings remain in the HTML.

- [ ] **Step 3: Implement minimal formatter**

Parse the sign, integer, and fractional string without floating-point
conversion; trim trailing fractional zeroes, group integer digits with `.`,
and join a remaining fractional portion using `,`.

- [ ] **Step 4: Verify green**

Run: `go test ./internal/emailing -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/emailing/templates.go backend/internal/emailing/crypto_test.go
git commit -m "fix: format approval email numbers"
```

### Task 2: Send Approval Email After Form Submission

**Files:**
- Modify: `frontend/components/supplier-orders/supplier-order-form.tsx`
- Test: `frontend/components/supplier-orders/supplier-orders.test.tsx`

**Interfaces:**
- Consumes: saved PO ID and successful `/api/purchase-orders/:id/submit`.
- Produces: POST `/api/email/purchase-orders/:id/approval` before index navigation.

- [ ] **Step 1: Write failing success-path test**

Extend the Save & Send test to return a successful email response after the
submit response and assert calls occur in this order:

```text
POST /api/purchase-orders
POST /api/purchase-orders/:id/submit
POST /api/email/purchase-orders/:id/approval
```

- [ ] **Step 2: Verify red**

Run: `npm test -- --run components/supplier-orders/supplier-orders.test.tsx`
Expected: FAIL because the email endpoint is never called.

- [ ] **Step 3: Implement success path**

After successful submission, POST to the existing approval-email endpoint.
Navigate to `/supplier-orders` only after its successful response.

- [ ] **Step 4: Write and verify failure-path test**

Return HTTP 422 from the email endpoint. Assert the submitted PO remains
hydrated, navigation does not occur, and the form displays the backend delivery
message explaining that the order is pending and can be resent.

- [ ] **Step 5: Implement failure handling and verify**

Map the email response message into `_form` without resubmitting or rolling back
the PO. Run the focused frontend test and expect PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/supplier-orders/supplier-order-form.tsx frontend/components/supplier-orders/supplier-orders.test.tsx
git commit -m "fix: send approval email after order submission"
```

### Task 3: Public Deployment URL

**Files:**
- Modify: `docker-compose.yml`
- Modify: `.env.example`

**Interfaces:**
- Consumes: host environment variable `APP_BASE_URL`.
- Produces: backend environment value with local fallback.

- [ ] **Step 1: Add configuration assertion**

Add a shell verification that `docker compose config` resolves
`APP_BASE_URL=https://deta.kaumtech.com` when the variable is supplied.

- [ ] **Step 2: Verify red**

Run:

```powershell
$env:APP_BASE_URL='https://deta.kaumtech.com'
docker compose config
```

Expected: output still contains `http://localhost:3019`.

- [ ] **Step 3: Implement environment expansion**

Set:

```yaml
APP_BASE_URL: ${APP_BASE_URL:-http://localhost:3019}
```

Document the production value in `.env.example`.

- [ ] **Step 4: Verify green**

Run `docker compose config` with and without the variable and confirm the
production value and local fallback respectively.

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml .env.example
git commit -m "fix: configure public application URL"
```

### Task 4: Supplier Email Attachments

**Files:**
- Modify if coverage requires: `backend/internal/emailing/service_test.go`
- Inspect: `backend/internal/emailing/service.go`

**Interfaces:**
- Consumes: approved order and three `purchaseorder.PDFDocument` values.
- Produces: one SMTP message to the supplier with PO, DN, and Kanban-label PDFs.

- [ ] **Step 1: Write failing or coverage test**

Exercise `SendSupplier` with a recording SMTP sender. Assert recipient, PO/DN
references, exactly three attachments, expected filenames, `application/pdf`
content type, and non-empty content.

- [ ] **Step 2: Verify the test**

Run: `go test ./internal/emailing -run TestSendSupplier -v`.
If existing behavior passes, retain the coverage without changing production
code. If it fails, make only the smallest service correction demonstrated by
the failure.

- [ ] **Step 3: Verify logging failure semantics**

Make the recording sender return an error and assert the service returns the
error and persists a `FAILED` log rather than `SENT`.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/emailing/service_test.go backend/internal/emailing/service.go
git commit -m "test: verify supplier email documents"
```

### Task 5: Full Verification and Deployment Handoff

**Files:**
- No production files expected.

**Interfaces:**
- Consumes: Tasks 1–4.
- Produces: verified commits and server deployment commands.

- [ ] **Step 1: Run backend suite**

Run: `go test ./...` from `backend`.
Expected: PASS.

- [ ] **Step 2: Run frontend suite**

Run: `npm test`, `npm run lint`, `npm run typecheck`, and `npm run build` from
`frontend`.
Expected: all exit 0.

- [ ] **Step 3: Verify Compose and build images**

Run:

```powershell
$env:APP_BASE_URL='https://deta.kaumtech.com'
docker compose config
docker compose build backend frontend
```

Expected: configured backend URL is the HTTPS domain and both images build.

- [ ] **Step 4: Push main**

```bash
git push origin main
```

- [ ] **Step 5: Server handoff**

On the application server, set `APP_BASE_URL=https://deta.kaumtech.com`, pull
main, and recreate backend/frontend. Existing approval emails retain their old
URLs and tokens; resend generates a new production-domain link.
