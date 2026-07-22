# Supplier Order PDF Exports Design

## Scope

Provide three real PDF documents from the Supplier Order index and make successful `Save & Send for Approval` return the user to that index. PDF controls are removed from the Supplier Order detail page.

## Supplier Order index

Each PO row exposes compact document actions:

- `PO PDF` is available for every persisted PO status.
- `DN PDF` is available only when the PO is approved and has a generated Delivery Note.
- `Kanban Labels PDF` is available only when the PO is approved and has generated Kanban lots.

Each available action opens a real `application/pdf` response in a new browser tab. An unavailable document is rendered as a neutral dash, not as an enabled or disabled fake button. The index remains horizontally scrollable and compact.

## PDF endpoints and authorization

Authenticated, tenant-scoped backend endpoints generate PDFs on demand:

- `GET /purchase-orders/:id/documents/po.pdf`
- `GET /purchase-orders/:id/documents/delivery-note.pdf`
- `GET /purchase-orders/:id/documents/kanban-labels.pdf`

All endpoints require `po.view`. Price fields are included in the PO PDF only when the current user also has `po.price.view`. DN and Kanban endpoints return a typed conflict/not-found response when operational documents do not exist. Queries use the authenticated actor's tenant and never trust tenant data from the URL.

Responses use `Content-Type: application/pdf`, an inline filename derived from the system document number, and defensive filename sanitization. PDFs are generated in memory and are not persisted in PostgreSQL or the local filesystem.

## PO PDF

The PO PDF uses A4 portrait and includes the PO number/status, supplier, dates, currency, notes, material table, Kanban quantities, total base quantity, and creator. Authorized price viewers also see unit prices and total amount.

## DN PDF

The DN PDF uses A4 portrait and includes the system-issued DN number, PO reference, supplier, expected delivery date, issue date, and every DN line with material, base unit, quantity per Kanban, total Kanban, and total quantity. It is the system document the supplier prints and attaches to the shipment.

## Kanban Labels PDF

The label PDF uses A4 portrait with a compact two-column label grid. Every generated Kanban lot produces exactly one label containing:

- a scannable Code 128 barcode;
- Kanban ID in human-readable text;
- DN and PO numbers;
- raw-material code and name;
- lot quantity and base unit.

Labels are ordered by PO-line sort position and lot number. Page breaks never split an individual label.

## Submit-and-close flow

After the PO save and submit requests both succeed, `Save & Send for Approval` navigates to `/supplier-orders`. If either request fails, the user stays on the form and sees the existing error. Duplicate submission remains blocked while requests are in flight.

## Error handling and tests

- Expected missing-document and permission failures use existing typed HTTP handling.
- PDF rendering failures are logged with tenant, user, PO, document type, and cause while returning a sanitized response.
- Backend tests validate authorization, tenant isolation, content headers, PDF signature, price redaction, multi-line DN data, exact label count, barcode content inputs, and missing-document failures.
- Frontend tests validate document availability by status, new-tab URLs, removal of detail controls, and submit success/error navigation.
