# Ink-Efficient Operational PDFs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign Purchase Order, Delivery Note, and three-per-page Kanban PDFs with white backgrounds, black outlined sections, buyer company identity, richer Kanban metadata, and per-material `CARD current/total` numbering.

**Architecture:** Extend the existing purchase-order document models and tenant-filtered queries with company identity, order date, and explicit card-position fields. Keep PDF generation in `backend/internal/purchaseorder/pdf.go`, introduce small monochrome layout helpers there, and update each renderer independently under focused PDF tests.

**Tech Stack:** Go 1.24, `github.com/go-pdf/fpdf`, PostgreSQL/pgx query contracts, `shopspring/decimal`, Go `testing`.

## Global Constraints

- Keep existing HTTP endpoints, filenames, authorization, download flows, numbering formats, and QR payloads unchanged.
- Keep exactly three 186 mm × 84 mm Kanban Cards per A4 page with existing `CUT HERE` behavior.
- Use `tenant_settings.company_name` for the buyer company and supplier master data for supplier identity.
- Use `"Order Stock"` when the configured company name is blank.
- Use white backgrounds, black text, black borders, and dividers; do not introduce logos or colors.
- Do not add a database migration or persisted card-position field.
- Preserve PO price-inclusive and price-excluded modes.
- Preserve Unicode, cancellation, export-limit, tenant-isolation, and multi-page behavior.

---

### Task 1: Document Data and Card-Position Plumbing

**Files:**
- Modify: `backend/internal/purchaseorder/domain.go`
- Modify: `backend/internal/purchaseorder/store.go`
- Modify: `backend/internal/purchaseorder/pdf.go`
- Test: `backend/internal/purchaseorder/store_test.go`
- Test: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Produces: `Order.CompanyName string`
- Produces: `DeliveryNoteDocument.CompanyName string`
- Produces: `DeliveryNoteDocument.OrderDate time.Time`
- Produces: `KanbanLabelDocument.CompanyName string`
- Produces: `KanbanLabelDocument.OrderDate time.Time`
- Produces: `KanbanLabel.CardNumber int` and `KanbanLabel.CardTotal int`
- Produces: `pdfCompanyName(string) string`

- [ ] **Step 1: Write failing company-name query contract tests**

Update the order query tests so `orderSelect` must select `COALESCE(NULLIF(BTRIM(ts.company_name),''),'Order Stock')` and join `tenant_settings ts` by tenant. Extend `documentHeader`, `TestLoadDeliveryNoteDocumentUsesTenantFilteredQueriesAndAllLines`, and `TestLoadKanbanLabelDocumentUsesStableTenantFilteredOrdering` fixtures to expect company name and order date.

```go
if !strings.Contains(strings.ToLower(orderSelect), "join tenant_settings ts on ts.tenant_id=p.tenant_id") {
	t.Fatal("orderSelect does not load tenant company identity")
}
if document.CompanyName != "Buyer PT" || !document.OrderDate.Equal(orderDate) {
	t.Fatalf("document identity = %#v", document)
}
```

- [ ] **Step 2: Write a failing card-position reset test**

Make the Kanban row fixture return `lot_number` plus `total_kanban`, with two labels for material A and one for material B. Assert the result is `1/2`, `2/2`, then `1/1`.

```go
got := [][2]int{
	{document.Labels[0].CardNumber, document.Labels[0].CardTotal},
	{document.Labels[1].CardNumber, document.Labels[1].CardTotal},
	{document.Labels[2].CardNumber, document.Labels[2].CardTotal},
}
want := [][2]int{{1, 2}, {2, 2}, {1, 1}}
if !reflect.DeepEqual(got, want) {
	t.Fatalf("card positions = %#v, want %#v", got, want)
}
```

- [ ] **Step 3: Run focused tests and verify RED**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test(OrderSelect|LoadDeliveryNoteDocument|LoadKanbanLabelDocument)' -count=1
```

Expected: FAIL because the models and SQL scans do not yet include company name, order date, or card total.

- [ ] **Step 4: Add the model fields and fallback helper**

Add `CompanyName string` to `Order`; add `CompanyName string` and `OrderDate time.Time` to both operational document models; replace `KanbanLabel.LotNumber` with `CardNumber int` and `CardTotal int`.

```go
func pdfCompanyName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Order Stock"
	}
	return value
}
```

- [ ] **Step 5: Extend tenant-filtered SQL and scans**

Join `tenant_settings` in `orderSelect` and scan `CompanyName` in the same position selected by the query. Extend `loadOperationalDocumentHeader` to select and scan:

```sql
COALESCE(NULLIF(BTRIM(ts.company_name),''),'Order Stock'),
p.po_number,
s.name,
p.order_date,
p.expected_delivery_date,
p.status
```

Join `tenant_settings ts ON ts.tenant_id=p.tenant_id`. Map these fields into both operational document models. Change the Kanban label query to select `kl.lot_number, pol.total_kanban::integer`, scan them into `CardNumber` and `CardTotal`, and retain `ORDER BY pol.sort_position,kl.lot_number`.

- [ ] **Step 6: Update fixture helpers and run focused tests**

Update `setKanbanLabel` to accept `cardNumber, cardTotal int`, update existing literals from `LotNumber` to the two card fields, and run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test(OrderSelect|LoadDeliveryNoteDocument|LoadKanbanLabelDocument)' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add backend/internal/purchaseorder/domain.go backend/internal/purchaseorder/store.go backend/internal/purchaseorder/store_test.go backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: load operational PDF identity and card positions"
```

### Task 2: Outlined Three-Per-Page Kanban Cards

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Test: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: `KanbanLabelDocument.CompanyName`, `.SupplierName`, `.OrderDate`
- Consumes: `KanbanLabel.CardNumber`, `.CardTotal`
- Produces: `formatCardPosition(KanbanLabel) string`
- Preserves: `renderKanbanLabelsPDF(context.Context, KanbanLabelDocument) ([]byte, error)`

- [ ] **Step 1: Rewrite the Kanban content test for the approved copy**

Extend `TestRenderWideKanbanCard` with buyer, supplier, and order date. Require the new labels and values, and reject `LOT`.

```go
document.CompanyName = "PT Buyer Indonesia"
document.SupplierName = "PT Supplier Sentosa"
document.OrderDate = time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
document.Labels[0].CardNumber = 1
document.Labels[0].CardTotal = 5

for _, text := range []string{
	"PT Buyer Indonesia", "PART NUMBER", "PART NAME", "PT Supplier Sentosa",
	"ORDER DATE", "21 Jul 2026", "CARD", "1/5",
} {
	if !pdfContainsText(result, text) {
		t.Errorf("missing Kanban Card text %q", text)
	}
}
if pdfContainsText(result, "LOT") {
	t.Fatal("legacy LOT label is still present")
}
```

- [ ] **Step 2: Add failing monochrome and pagination assertions**

Keep the four-card/two-page test and assert three cards remain on page one through the existing page count. Add `pdfContainsSolidHeaderFill`, which checks the uncompressed PDF content for the legacy dark fill color operator, and assert it is false for a rendered Kanban PDF.

```go
func pdfContainsLegacyDarkFill(document []byte) bool {
	return bytes.Contains(document, []byte("0.094 0.094 0.106 rg"))
}
```

Expected test: `if pdfContainsLegacyDarkFill(result) { t.Fatal("Kanban still uses a dark solid fill") }`.

- [ ] **Step 3: Run Kanban tests and verify RED**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test(RenderWideKanbanCard|KanbanCardPagination)' -count=1
```

Expected: FAIL because the current card uses a dark filled header, `RAW MATERIAL`, and `LOT`, and omits identity metadata.

- [ ] **Step 4: Implement card-position formatting**

```go
func formatCardPosition(label KanbanLabel) string {
	return strconv.Itoa(label.CardNumber) + "/" + strconv.Itoa(label.CardTotal)
}
```

The database contract guarantees positive values; renderer fixtures must set both fields.

- [ ] **Step 5: Redraw `writeKanbanCard` with outlined sections**

Keep `cardsPerPage = 3`, `cardWidth = 186`, `cardHeight = 84`, QR size, QR payload, and cut-line logic. Replace the filled header with an outlined 15 mm header. Place buyer company name and `KANBAN CARD` at left, large Kanban ID at right, and divide the detail area into bounded boxes for:

```text
PART NUMBER | PART NAME
SUPPLIER | ORDER DATE
QUANTITY | CARD
DELIVERY NOTE | PURCHASE ORDER
```

Use `fitPDFTextLines`/`writePDFTextLines` for company, supplier, and part name. Give Part Name the largest detail font that fits within two lines. Use only black draw/text colors and white cell fills.

- [ ] **Step 6: Run all Kanban PDF tests**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test.*Kanban' -count=1
```

Expected: PASS, including QR-only, export limit, cancellation, stable ordering, and three-card pagination tests.

- [ ] **Step 7: Commit**

```powershell
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: redesign Kanban cards for ink-efficient printing"
```

### Task 3: Monochrome Purchase Order

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Test: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: `Order.CompanyName`
- Produces: `writeOutlinedSectionTitle(*fpdf.Fpdf, string)`
- Preserves: `RenderPOPDF(Order, bool) ([]byte, error)`

- [ ] **Step 1: Write failing PO identity and monochrome tests**

Extend the PO fixture with `CompanyName: "PT Buyer Indonesia"`. Require the company name and existing business sections in both price modes. Assert the PDF does not contain the legacy dark or gray fill operators.

```go
if !pdfContainsText(result, "PT Buyer Indonesia") {
	t.Fatal("buyer company is missing")
}
if pdfContainsLegacyDarkFill(result) || bytes.Contains(result, []byte("0.957 0.957 0.961 rg")) {
	t.Fatal("PO still uses a solid dark or gray section fill")
}
```

- [ ] **Step 2: Run PO tests and verify RED**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'TestRenderPOPDF' -count=1
```

Expected: FAIL because the PO currently renders a dark banner, gray section fills, and hard-coded `ORDER STOCK`.

- [ ] **Step 3: Add the outlined section-title helper**

Replace `writePDFSectionTitle` with a white-background helper that uses a black border and no fill:

```go
func writeOutlinedSectionTitle(pdf *fpdf.Fpdf, title string) {
	ensurePDFPageRoom(pdf, 9)
	pdf.SetDrawColor(24, 24, 27)
	pdf.SetTextColor(24, 24, 27)
	pdf.SetFont(pdfFontFamily, "B", 8)
	pdf.CellFormat(0, 7, "  "+title, "1", 1, "L", false, 0, "")
	pdf.Ln(1)
}
```

- [ ] **Step 4: Redraw the PO header and table headings**

Use an outlined header panel containing `pdfCompanyName(order.CompanyName)`, `PURCHASE ORDER`, PO number, and status. Put supplier and order metadata in aligned bordered regions. Change `writePDFTable` headings to `fill=false`, preserve black cell borders, existing widths, line wrapping, price-mode columns, totals, notes, and approval content.

- [ ] **Step 5: Run PO and shared table tests**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test(RenderPOPDF|FitPDFTextLines|WritePDFTable)' -count=1
```

Expected: PASS for price inclusion/redaction, Unicode text, business sections, wrapping, and pagination.

- [ ] **Step 6: Commit**

```powershell
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: redesign purchase order PDF with outlined sections"
```

### Task 4: Monochrome Delivery Note with Dominant Number

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Test: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: `DeliveryNoteDocument.CompanyName`
- Consumes: `DeliveryNoteDocument.OrderDate`
- Preserves: `RenderDeliveryNotePDF(DeliveryNoteDocument) ([]byte, error)`
- Preserves: QR payload equal to `DeliveryNoteNumber`

- [ ] **Step 1: Write failing DN identity, metadata, and monochrome tests**

Extend `TestRenderDeliveryNotePDFUsesModernLayout` with company and order dates. Require the buyer identity and Order Date, preserve QR assertions, and reject legacy fill colors.

```go
for _, text := range []string{"PT Buyer Indonesia", "ORDER DATE", "21 Jul 2026"} {
	if !pdfContainsText(result, text) {
		t.Errorf("missing Delivery Note text %q", text)
	}
}
if pdfContainsLegacyDarkFill(result) || bytes.Contains(result, []byte("0.957 0.957 0.961 rg")) {
	t.Fatal("Delivery Note still uses a solid dark or gray fill")
}
```

Add a package-level constant `deliveryNoteNumberFontSize = 16.0` and assert it is larger than the title metadata font size constant, so the hierarchy is explicit and regression-testable.

- [ ] **Step 2: Run Delivery Note tests and verify RED**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test(RenderDeliveryNote|DeliveryNote)' -count=1
```

Expected: FAIL because the header and table/footer headings still use dark/gray fills and buyer/order-date content is missing.

- [ ] **Step 3: Redraw the Delivery Note header**

Replace the dark banner with a bordered white header. Render `pdfCompanyName(document.CompanyName)`, `DELIVERY NOTE`, and the DN number using `deliveryNoteNumberFontSize`. Keep the QR at the right with the unchanged payload. Add bordered metadata rows for Supplier, PO Number, Order Date, Expected Delivery, and Issued Date.

- [ ] **Step 4: Remove gray fills from DN table, totals, remarks, and signatures**

Set all header `CellFormat` calls to `fill=false`; preserve `"1"` borders and bold headings. Keep current row-height measurement, repeated headings after `pdf.AddPage()`, unit totals, remarks, supplier/receiver signature areas, and footer room calculation.

- [ ] **Step 5: Run all Delivery Note tests**

Run:

```powershell
Set-Location backend
go test ./internal/purchaseorder -run 'Test.*DeliveryNote' -count=1
```

Expected: PASS, including every-line, QR payload, long-content wrapping, continued-page heading, unit totals, and tenant-filtered loading tests.

- [ ] **Step 6: Commit**

```powershell
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: redesign delivery note PDF for monochrome printing"
```

### Task 5: Full Regression and Static Verification

**Files:**
- Modify only if a regression is found: `backend/internal/purchaseorder/pdf.go`
- Modify only if a test contract needs correction: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Verifies all interfaces produced by Tasks 1–4.

- [ ] **Step 1: Format touched Go files**

```powershell
Set-Location backend
gofmt -w internal/purchaseorder/domain.go internal/purchaseorder/store.go internal/purchaseorder/store_test.go internal/purchaseorder/pdf.go internal/purchaseorder/pdf_test.go
```

- [ ] **Step 2: Run the focused purchase-order package**

```powershell
Set-Location backend
go test ./internal/purchaseorder -count=1
```

Expected: PASS.

- [ ] **Step 3: Run the complete backend suite**

```powershell
Set-Location backend
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run static analysis**

```powershell
Set-Location backend
go vet ./...
```

Expected: exit code 0 with no diagnostics.

- [ ] **Step 5: Inspect the final diff**

```powershell
git diff --check
git status --short
git log -5 --oneline
```

Expected: no whitespace errors; only intentional implementation/test changes remain, and Tasks 1–4 commits are present.

- [ ] **Step 6: Commit any verification-only corrections**

If Steps 2–4 required a code or test correction, stage only those explicit files and commit:

```powershell
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "fix: close operational PDF regression gaps"
```

If no correction was required, do not create an empty commit.
