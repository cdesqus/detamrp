# Modern Delivery Note PDF Design

## Goal

Redesign the generated Delivery Note as a polished A4 operational document with a scannable QR code, clearer material information, totals, remarks, and minimal manual sign-off areas.

## QR Behavior

- The QR payload is exactly the system-issued Delivery Note number, for example `DN-202607-00002`.
- It does not contain a URL, login token, PO number, or other metadata.
- Scanning it in the Receiving flow must yield the exact value already accepted by the DN scanner input.
- The human-readable DN number remains printed near the QR code.

## Layout

The document uses A4 portrait with compact typography and controlled spacing.

### Header

- Product/company label `Order Stock` at the upper left.
- Centered `DELIVERY NOTE` title.
- Delivery Note number directly beneath the title.
- QR code placed prominently in the upper-middle/right header area.
- Operational metadata arranged in two compact groups:
  - Supplier
  - PO Number
  - Expected Delivery
  - Issued Date

### Material Table

Columns:

- No.
- Material Code
- Description
- Unit
- Qty / Kanban
- Kanbans
- Total Qty

All quantities use the shared PDF decimal formatter and omit redundant trailing zeroes. Rows wrap long descriptions without expanding the overall visual scale. The table repeats its heading after a page break.

The table footer shows:

- Total Kanban
- Total Quantity grouped by base unit

When multiple base units exist, totals are displayed separately, for example `PCS 1.500 | KG 250`.

### Manual Area

- A bordered Remarks row for handwritten operational notes.
- Two compact signature boxes:
  - `SUPPLIER` / `Prepared By`
  - `RECEIVER` / `Received By`
- Each box includes blank space plus printed `Name:` and `Date:` fields.
- `Receiver` means the warehouse keeper or receiving operator; it is not a warehouse location.

## Pagination

- The renderer estimates row height from wrapped descriptions.
- If the next row, totals, remarks, or signature section does not fit, it adds a new page.
- Material table headings repeat on each continuation page.
- The QR and full header appear on the first page only.
- The signature section is never split between pages.

## Implementation

- Use `github.com/boombuler/barcode/qr`, already available through the existing barcode dependency.
- Encode and scale the QR as a PNG, register it with gofpdf, and render it without external network calls.
- Keep the existing Delivery Note data model; all required fields are already available.
- Add isolated helpers for QR generation, material-row rendering, totals by unit, and the signature section so the existing PO and Kanban label renderers remain unchanged.

## Error Handling

- QR encoding or PNG generation failure aborts PDF generation with a contextual error.
- Empty Delivery Note numbers are rejected before QR encoding.
- The existing operational-document availability rules remain unchanged.

## Testing

- A QR encoder test decodes the generated QR and proves its payload equals the Delivery Note number exactly.
- PDF tests verify the modern headings, both signature labels, remarks, every material line, and compact formatted quantities.
- A multi-line fixture verifies total Kanban and per-unit total quantity.
- A long document fixture verifies multi-page output and repeated material headings.
- Existing Unicode embedding, PO PDF, and Kanban label tests must remain green.
- Full Go tests, Go vet, and a live Delivery Note PDF response check must pass before merge.
