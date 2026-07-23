# Outgoing Direct Scan and Number Formatting Design

## Goal

Simplify the prototype by starting Outgoing scanning immediately without collecting a destination, and display quantities and monetary values without database-scale trailing zeroes.

## Outgoing Flow

1. The operator opens **Outgoing Material** and clicks **Create outgoing**.
2. The system immediately creates an active Outgoing session and navigates to its Kanban scan page.
3. The create modal, Destination field, Destination suggestions, and Notes field are removed.
4. The backend accepts an empty destination and notes value. Existing database rows and historical documents remain unchanged.
5. New Outgoing records store an empty destination. Screens and PDFs render an empty destination as `—` rather than exposing a placeholder value.
6. Only complete Kanban lots in `IN_STOCK` state can be scanned. Completion continues to change their state to `CONSUMED` and create the negative inventory-ledger entries.

This is intentionally an MVP simplification. A managed Destination master for production lines and other areas can be introduced later without changing the Kanban scan lifecycle.

## Number Formatting

Database precision remains unchanged. Formatting is applied only at the presentation layer.

- Monetary values use Indonesian digit grouping and never show unnecessary trailing zeroes.
- IDR values normally render with zero decimal places, for example `IDR 10.000.000`.
- Non-IDR values render with up to two decimal places.
- Physical quantities and prices render with up to six decimal places but trim trailing zeroes, for example `5.000000` becomes `5` and `5.250000` becomes `5,25`.
- The same shared formatter is used in Supplier Order list/detail/create views and generated PO PDFs so values remain consistent.

## Kanban Label Lifecycle

No received stamp or replacement label is added. The supplier-issued physical Kanban label stays attached to its box, roll, pallet, or other material container throughout warehouse storage. Receiving scans that barcode and changes the system state from `ISSUED` to `IN_STOCK`; Outgoing later scans the same physical barcode and changes the state to `CONSUMED`.

## Error Handling

- If session creation fails, the operator remains on the Outgoing index and sees `Outgoing session could not be created.`
- Empty destination is valid only for new prototype Outgoing sessions; existing destination values are preserved.
- PDF generation displays `—` for empty destination and must not fail.

## Tests

- Frontend tests verify one click creates a session, no Destination or Notes fields are rendered, and successful creation navigates to the scanner.
- Backend tests verify empty destination validation succeeds while overlength destination input remains rejected.
- Formatter tests cover IDR, non-IDR, integers, fractional values, and trailing-zero trimming.
- PDF tests verify an empty destination can be rendered.
- Full backend, frontend, type-check, production build, and Docker smoke checks run before completion.
