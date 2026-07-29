# Ink-Efficient PO, Delivery Note, and Kanban PDF Design

## Goal

Redesign the Purchase Order, Delivery Note, and Kanban Card PDFs as clean monochrome operational documents. Replace solid black and gray fills with white backgrounds, black text, borders, and dividers so the documents use less ink without looking empty or becoming harder to scan.

## Shared Document Identity

- All three document types display the tenant's company name from `tenant_settings.company_name`.
- The company name identifies the buyer's company. Supplier names continue to come from the supplier master data.
- If the configured company name is blank, the renderer uses the existing application identity fallback so document generation remains available.
- Headers use white backgrounds, black text, outlined panels, and consistent thin black rules.
- Long company and supplier names wrap or scale within bounded areas and must not overlap adjacent content.
- Existing HTTP endpoints, filenames, authorization, and download flows remain unchanged.

## Kanban Card Layout

The Kanban export retains the current physical layout of three cards per A4 page. Each card remains 186 mm wide by 84 mm high, with the existing dashed `CUT HERE` divider between adjacent cards.

Each card contains:

- Buyer company name from Settings.
- `KANBAN CARD` document title.
- Large Kanban ID.
- Existing black-and-white QR code and the human-readable Kanban ID.
- Part Number from the raw material code.
- Part Name from the raw material name, displayed larger and bolder than supporting metadata.
- Supplier name from the supplier master.
- Order Date from the Purchase Order.
- Quantity per Kanban and its unit.
- Card position in `current/total` form, such as `1/5`, under the label `CARD`.
- Delivery Note and Purchase Order references.

The former `LOT` label is removed completely. Card numbering is scoped to a Purchase Order material line. Labels for a five-card line display `1/5` through `5/5`; the next material line restarts at `1/n`. The sequence is derived from the stable material-line and Kanban-lot ordering and requires no persisted database field.

Information is grouped with outlined boxes and dividers. The intended visual priority is:

1. Part Name
2. Kanban ID and QR code
3. Card position
4. Quantity
5. Supplier, Order Date, PO, and DN references

## Purchase Order Layout

- Replace the solid dark header and gray table fills with white backgrounds.
- Place the buyer company name prominently in the header.
- Keep `PURCHASE ORDER` and the PO number visually clear using typography and outlined regions.
- Group supplier, order date, expected delivery date, currency, and related metadata into aligned bordered panels.
- Use thin black table borders, consistent cell padding, bold white-background column headings, and reliable text wrapping.
- Preserve the existing price-inclusive and price-excluded export modes.
- Keep totals, notes, approval information, and signatures, while correcting alignment and spacing within their current business meaning.
- Repeat the same PO column headings when line items continue onto another page.

## Delivery Note Layout

- Replace all solid black and gray fills with white backgrounds, black text, borders, and dividers.
- Place the buyer company name prominently in the header.
- Make the Delivery Note number the document's most prominent identifier and noticeably larger than supporting references.
- Retain the existing Delivery Note QR code.
- Align supplier, PO number, issue date, and related metadata in bordered panels.
- Use thin black table borders, consistent padding, bold white-background headings, and bounded wrapping for material details.
- Retain totals, remarks, and signature areas while improving alignment and spacing.
- Repeat the same Delivery Note column headings when material rows continue onto another page.

## Data Flow

The document-loading queries extend the existing PDF document models with the data required by their renderers:

- `CompanyName` from `tenant_settings.company_name`.
- `SupplierName` from the Purchase Order's supplier relation.
- `OrderDate` from the Purchase Order.
- Per-label card position and total, derived from each material line's stable Kanban ordering.

No database migration is required. The renderer must not infer the supplier from free text or duplicate the buyer company name as supplier identity.

## Error Handling and Layout Safety

- Existing QR encoding, PDF serialization, request cancellation, and export-size errors retain contextual error handling.
- Missing required identifiers continue to fail before QR encoding.
- Optional or blank display fields render as an em dash where the current document contract permits an empty value.
- Long company names, supplier names, material codes, and part names are measured and fitted within explicit bounds.
- Layout helpers must prevent text from crossing borders or overlapping QR codes, references, totals, and signature blocks.

## Testing and Acceptance Criteria

Automated tests must prove:

- Kanban output keeps exactly three cards per A4 page.
- `LOT` no longer appears and `CARD` does.
- Card positions progress through `1/n` for a material line and reset to `1/n` on the next material line.
- Buyer company name, supplier name, and order date appear on the Kanban.
- Buyer company name appears on PO and Delivery Note PDFs.
- The Delivery Note number remains present and uses the enlarged identifier layout.
- The redesigned headers and table headings do not use solid black or gray fills.
- PO and Delivery Note tables preserve borders, wrap long content, and paginate safely.
- QR payloads and existing price-mode behavior remain unchanged.
- Existing Unicode, cancellation, export-limit, service, HTTP, and database isolation tests remain green.

Before completion, run the focused purchase order PDF tests, the complete backend Go test suite, and `go vet ./...`.

## Out of Scope

- Database schema changes.
- Changes to document numbering formats or QR payloads.
- Changes to Kanban card dimensions, the three-card page capacity, or cut-line behavior.
- Changes to business calculations, PO approval, Receiving, or Outgoing workflows.
- Adding logos, colors, or new organization-profile fields beyond the existing company name.
