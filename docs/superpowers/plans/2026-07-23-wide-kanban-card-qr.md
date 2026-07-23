# Wide Kanban Card QR Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current small Code128 labels with three wide QR-based Kanban Cards per A4 page.

**Architecture:** Keep the existing Kanban document model and endpoint, but replace the Kanban rendering branch with focused QR and card-layout helpers. The renderer uses fixed card slots so cards cannot split and pagination is deterministic.

**Tech Stack:** Go 1.26, gofpdf, boombuler/barcode QR, shopspring/decimal, Go tests.

## Global Constraints

- Exactly three wide cards per A4 portrait page.
- QR payload is exactly the Kanban ID.
- Code128 is not rendered on Kanban Cards.
- No unavailable operational fields are invented.
- Existing 1,000-label export limit remains.
- Quantities omit redundant trailing zeroes.

---

### Task 1: Kanban QR generation

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Produces: `encodeKanbanQR(value string) (barcode.Barcode, error)`
- Produces: `kanbanQRPNG(value string) ([]byte, error)`

- [ ] **Step 1: Write failing QR tests**

Assert:

- `encodeKanbanQR("KB-202607-00028").Content()` equals exactly `KB-202607-00028`;
- empty Kanban IDs return an error;
- PNG output starts with the PNG signature.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run TestKanbanQR -count=1`

Expected: FAIL because QR helpers do not exist.

- [ ] **Step 3: Implement QR helpers**

Encode with QR error correction level M, scale to a square image, and PNG-encode locally. Return contextual errors containing the Kanban ID.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run TestKanbanQR -count=1`

Expected: PASS.

### Task 2: Wide three-card renderer

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: Task 1 QR PNG helper.
- Produces: `writeKanbanCard(pdf, document, label, index, x, y, width, height) error`.

- [ ] **Step 1: Write failing layout tests**

Render one label and verify extracted PDF text contains:

```text
KANBAN CARD
KANBAN ID
RAW MATERIAL
QUANTITY
LOT
DELIVERY NOTE
PURCHASE ORDER
KB-202607-00028
```

Verify quantity `5.000000` renders as `5 PC`, and the legacy `KANBAN LABEL` title is absent.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run TestRenderWideKanbanCard -count=1`

Expected: FAIL because the old label layout is still active.

- [ ] **Step 3: Implement card layout**

Use:

```go
const cardsPerPage = 3
const cardX = 12.0
const cardWidth = 186.0
const cardHeight = 84.0
const cardGap = 5.0
const firstCardY = 12.0
```

Create bordered sections for the title/ID, QR, material details, quantity/lot, and DN/PO references. Wrap description text and render the human-readable Kanban ID next to the QR.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run TestRenderWideKanbanCard -count=1`

Expected: PASS.

### Task 3: Pagination, cut lines, and regression safety

**Files:**
- Modify: `backend/internal/purchaseorder/pdf.go`
- Modify: `backend/internal/purchaseorder/pdf_test.go`

**Interfaces:**
- Consumes: Task 2 card renderer.
- Produces deterministic three-card pagination.

- [ ] **Step 1: Write failing pagination tests**

Render four labels and assert:

- the PDF has exactly two pages;
- the first three Kanban IDs appear on page one data;
- the fourth label is retained on page two;
- cut-line caption `CUT HERE` is present;
- the legacy Code128 encoder hook is never called.

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/purchaseorder -run "TestKanbanCardPagination|TestKanbanCardsDoNotUseCode128" -count=1`

Expected: FAIL under the old two-column renderer.

- [ ] **Step 3: Implement pagination and cut lines**

Add a page when `index % 3 == 0`. Compute slot using `index % 3`. Draw a dashed separator and `CUT HERE` between slots one/two and two/three. Never draw an unused placeholder card.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./internal/purchaseorder -run "TestKanbanCardPagination|TestKanbanCardsDoNotUseCode128" -count=1`

Expected: PASS.

- [ ] **Step 5: Run full verification**

Run:

```powershell
$env:TEST_DATABASE_URL='postgres://nextgen:nextgen@localhost:5445/nextgen?sslmode=disable'
$env:GOMAXPROCS='2'
go test -p 1 ./... -count=1
go vet ./...
git diff --check
```

Expected: all packages PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/purchaseorder/pdf.go backend/internal/purchaseorder/pdf_test.go
git commit -m "feat: redesign Kanban cards with QR"
```

### Task 4: Merge, rebuild, and live PDF check

**Files:**
- Verify only.

- [ ] **Step 1: Complete the branch workflow**

Use finishing-a-development-branch and merge only after user selection.

- [ ] **Step 2: Rebuild local Docker**

Run:

```powershell
docker compose -p platform-master-po up -d --build
docker compose -p platform-master-po ps
```

- [ ] **Step 3: Check a live Kanban PDF**

Authenticate locally, request a PO Kanban-label endpoint, and verify HTTP `200`, `application/pdf`, `%PDF-` prefix, and non-trivial size.
