# Modern Delivery Note PDF Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the plain Delivery Note PDF with a polished A4 operational document containing a DN-only QR code, compact material table, totals, remarks, and manual Supplier/Receiver sign-off.

**Architecture:** Keep the existing Delivery Note read model and HTTP route. Refactor only the Delivery Note rendering path into focused helpers for QR generation, header, paginated material table, totals, and signature boxes while leaving PO and Kanban label renderers untouched.

**Tech Stack:** Go 1.26, gofpdf, boombuler/barcode QR, shopspring/decimal, Go tests.

## Global Constraints

- QR payload is exactly the Delivery Note number.
- A4 portrait with compact typography.
- No external network calls or new runtime service.
- No database or API contract changes.
- Quantities omit redundant trailing zeroes.
- Signature boxes are manual paper fields labelled Supplier and Receiver.

---

### Task 1: Delivery Note QR generation

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Produces: `encodeDeliveryNoteQR(value string) (barcode.Barcode, error)`
- Produces: `deliveryNoteQRPNG(value string) ([]byte, error)`

- [ ] **Step 1: Write failing QR tests**

Add tests asserting:

- an empty DN number returns an error;
- `encodeDeliveryNoteQR("DN-202607-00002").Content()` equals exactly `DN-202607-00002`;
- the PNG helper returns bytes beginning with the PNG signature.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run TestDeliveryNoteQR -count=1`

Expected: FAIL because the QR helpers do not exist.

- [ ] **Step 3: Implement minimal QR helpers**

Use:

```go
encoded, err := qr.Encode(strings.TrimSpace(value), qr.M, qr.Auto)
scaled, err := barcode.Scale(encoded, 220, 220)
```

PNG-encode the scaled image and return contextual errors.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run TestDeliveryNoteQR -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: generate delivery note QR codes"
```

### Task 2: Modern first-page layout and material table

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: Task 1 QR PNG helper.
- Produces: modern `RenderDeliveryNotePDF`.
- Produces: `deliveryNoteUnitTotals([]DeliveryNoteLine) []string`.

- [ ] **Step 1: Write failing layout tests**

Render a two-line, mixed-unit document and assert extracted PDF text contains:

```text
Order Stock
DELIVERY NOTE
SCAN FOR RECEIVING
MATERIAL DETAILS
REMARKS
SUPPLIER
Prepared By
RECEIVER
Received By
Total Kanban
Total Quantity
PCS 20
KG 15
```

Also assert line quantities render as `10`, `2`, and `20`, not `10.000000`, `2.000000`, or `20.000000`.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run "TestRenderDeliveryNotePDFUsesModernLayout|TestDeliveryNoteUnitTotals" -count=1`

Expected: FAIL because modern sections and totals are absent.

- [ ] **Step 3: Implement the layout**

Create focused helpers:

- `writeDeliveryNoteHeader`
- `writeDeliveryNoteTableHeader`
- `writeDeliveryNoteLine`
- `deliveryNoteUnitTotals`
- `writeDeliveryNoteFooter`

Use table widths totaling the 186 mm printable width:

```go
[]float64{10, 29, 55, 16, 27, 22, 27}
```

Render QR at approximately 30 mm square, use `formatPDFDecimal` for every quantity, and sort unit totals by unit code for deterministic output.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run "TestRenderDeliveryNotePDF|TestDeliveryNoteUnitTotals" -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: redesign delivery note PDF"
```

### Task 3: Pagination and regression verification

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: Task 2 table helpers.
- Produces: stable multi-page Delivery Notes with repeated table headings and unsplit footer.

- [ ] **Step 1: Write a failing pagination test**

Render at least 45 material rows with long descriptions. Assert:

- the PDF has more than one page;
- `MATERIAL DETAILS` or the material table heading appears on continuation pages;
- the final `REMARKS`, `SUPPLIER`, and `RECEIVER` section appears together.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run TestDeliveryNotePagination -count=1`

Expected: FAIL because the current renderer does not repeat headings or reserve the footer.

- [ ] **Step 3: Implement pagination**

Before every row, estimate wrapped-description height and add a page when the row would cross the bottom margin. On continuation pages, write a compact continuation title and repeat the table header. Before totals/footer, reserve the complete footer height and move it to a new page when needed.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run TestDeliveryNotePagination -count=1`

Expected: PASS.

- [ ] **Step 5: Run full verification**

Run:

```powershell
$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'
go test ./... -count=1
go vet ./...
```

Expected: all packages PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "test: cover delivery note pagination"
```

### Task 4: Merge, rebuild, and live PDF check

**Files:**
- Verify only.

**Interfaces:**
- Produces a locally deployed modern Delivery Note.

- [ ] **Step 1: Merge the verified branch**

Use the finishing-a-development-branch workflow and merge to `master` only after user selection.

- [ ] **Step 2: Rebuild Docker**

Run:

```powershell
docker compose -p platform-master-po up -d --build
docker compose -p platform-master-po ps
```

- [ ] **Step 3: Live PDF check**

Authenticate locally, request an approved PO's Delivery Note endpoint, and verify HTTP `200`, content type `application/pdf`, a `%PDF-` prefix, and non-trivial file size.
