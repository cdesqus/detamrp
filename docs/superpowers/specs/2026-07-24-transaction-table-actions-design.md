# Supplier Order and Receiving Table Actions Design

## Scope

Redesign the Supplier Orders and Receiving index tables using the compact operational reference supplied by the user. This phase does not implement SMTP delivery or Email Log persistence.

## Supplier Orders

Column order begins with:

1. Number
2. Actions
3. Docs
4. Status
5. PO Number

The remaining existing business columns follow: Supplier, Order Date, Expected Date, optional Total according to RBAC, Sage Number, and Created By.

### Number and Pagination

- Default page size is 20.
- Available page sizes are 20, 50, and 100.
- Row numbering continues across pages; page two begins at 21 when the page size is 20.
- Search resets the current page to one.

### Actions

The Actions cell contains one compact `Action ▾` button. It opens a single menu containing every action so the column stays narrow and users do not need to interpret an ambiguous plus, ellipsis, or gear icon.

- Open Detail is always available.
- Edit Order is enabled only for Draft orders and users with `po.edit_draft`.
- Cancel Draft is enabled only for Draft orders and users with `po.edit_draft`; it retains the existing confirmation dialog and is visually separated from routine actions.
- The menu always renders both messaging choices:
  - Send to Approval is enabled only for Draft orders and users with `po.submit`; it uses the existing submit endpoint.
  - Send to Supplier is applicable after approval, including Partially Received and Fully Received orders, but remains disabled until the supplier email endpoint exists.
  - After SMTP and Email Log are implemented, the same action becomes Send or Resend to Supplier without restructuring the table.
- The popover does not display SMTP configuration copy.
- Disabled actions remain visible in muted styling so the operational sequence is understandable.
- No expand, close-order, print, or download actions are added.

### Documents

One compact `Docs ▾` button opens a popover containing three clearly named entries:

- Purchase Order PDF is always available.
- Delivery Note PDF is available after approval and document generation.
- Kanban Labels PDF is available after approval and within the existing safe label export limit.

Unavailable documents remain visible but disabled. Available entries open the PDF in a new tab. The popover closes after selection, outside click, or Escape.

## Receiving

- Add a continuing row number.
- Move Document directly after Number, before Status and receiving identifiers.
- Add pagination with page sizes 20, 50, and 100.
- Do not add an Actions column.
- Keep the Receiving PDF as a compact `RCV` document control with a tooltip and direct new-tab behavior; no popover is needed for one document.
- Preserve all current receiving business columns and open-session banner behavior.

## Visual Rules

- Tables remain dense and modern; do not enlarge rows, headings, buttons, or typography.
- Actions are positioned on the left after the row number.
- `Action ▾` and `Docs ▾` are compact text controls; generic plus, ellipsis, and gear icons are not used.
- Tooltips and accessible names describe every icon.
- Horizontal scrolling remains available for wide tables.

## Future Email Integration

SMTP Settings will provide encrypted live configuration and a connection test. Email Log will record recipient, type, reference, subject, delivery state, error, sender, and timestamps. The future supplier action will call a dedicated idempotent send/resend endpoint that attaches PO, DN, and Kanban PDFs. PO transaction status remains independent from email delivery state.

## Tests

- Continuous numbering and page-size changes.
- Search resets pagination.
- Icon order, accessible labels, and tooltips.
- Draft action enablement and post-approval disabled states.
- Existing submit and cancel flows remain functional.
- Documents open the correct endpoints.
- Receiving pagination, numbering, and document icon.
