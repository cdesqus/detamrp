# Wide Kanban Card QR Design

## Goal

Replace the current small two-column barcode labels with wide, operational Kanban Cards that are easier to read, cut, attach, and scan in Receiving and Outgoing.

## Page Layout

- A4 portrait.
- Three cards per page in one vertical column.
- Each card spans the printable page width and is approximately 85 mm high.
- Cards are separated by a dashed cut line with a small scissors/cut indicator.
- No card is split between pages.
- A partial final page leaves unused card slots blank.

## QR Behavior

- Each card contains one large QR code.
- The QR payload is exactly the Kanban ID, for example `KB-202607-00028`.
- It contains no URL, DN number, PO number, login token, or JSON.
- The human-readable Kanban ID is printed below or beside the QR as a scanner fallback.
- The existing Code128 barcode is removed from Kanban Cards.

## Card Content

Each card shows:

- `KANBAN CARD` heading.
- Kanban ID as the primary identifier.
- Large QR code.
- Raw Material Code.
- Raw Material Description.
- Quantity and base unit.
- Lot number.
- Delivery Note number.
- Purchase Order number.

The most important scanning information—Kanban ID, material code, quantity, and QR—uses the strongest visual hierarchy. Supporting references use smaller compact text.

## Visual Style

- Compact black, white, and neutral-gray business-document styling.
- Strong bordered sections inspired by the supplied reference without copying its unavailable operational fields.
- No cycle, area, location, packing, rit, conveyance, or warehouse data is invented.
- Quantities use the shared compact formatter and omit redundant trailing zeroes.
- Long material descriptions wrap within their section.

## Implementation

- Continue using `github.com/boombuler/barcode`, replacing the Code128 encoder with `github.com/boombuler/barcode/qr`.
- Generate QR images locally and embed them as PNG with gofpdf.
- Keep the existing Kanban document model and endpoint.
- Set the renderer to three cards per page.
- Preserve the existing export limit and cancellation checks.

## Error Handling

- Empty Kanban IDs are rejected before QR generation.
- QR encoding, scaling, or PNG failures return contextual errors naming the Kanban ID.
- Oversized exports still fail before any QR is encoded.

## Testing

- QR tests prove `Content()` equals the exact Kanban ID and empty IDs fail.
- PDF tests verify the new headings and all operational fields.
- Tests verify old Code128 encoding is no longer invoked.
- A four-label fixture verifies the fourth card starts a second page.
- Long descriptions remain present and the generated PDF is valid.
- Existing PO, Delivery Note, Unicode, cancellation, and export-limit tests remain green.
- Full Go tests, Go vet, and a live Kanban PDF endpoint check must pass before merge.
