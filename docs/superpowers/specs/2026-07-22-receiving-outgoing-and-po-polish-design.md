# Receiving, Outgoing, and Supplier Order Polish Design

## Objective

Complete the next operational vertical slice after Supplier Orders: improve the PO document and order-status presentation, implement partial Receiving with exclusive resumable scan sessions, and implement full-Kanban Outgoing Material backed by an auditable inventory ledger.

The prototype continues to use one configured tenant in the UI, while every new business table remains tenant-scoped for future SaaS deployment.

## Delivery Order

Implementation follows a vertical sequence:

1. Polish PO PDF, status badges, and read-only Sage PO Number presentation.
2. Add the inventory ledger and Kanban lifecycle foundation.
3. Implement Receiving end-to-end, including PDF and PO status projection.
4. Implement Outgoing Material on top of received stock.

Each stage must be usable and tested before the next depends on it.

## Supplier Order Polish

### PO PDF

The PO PDF keeps the existing endpoint and authorization behavior but adopts a less rigid business-document layout:

- compact company/document header;
- PO identity and supplier information in a balanced two-column block;
- readable material table with measured wrapping and repeated headers;
- totals aligned on the right;
- notes and approval trail in a restrained footer section;
- embedded Unicode fonts, in-memory generation, and existing cache protections remain mandatory.

The result should look modern and professional without decorative clutter.

### Colored Status Badges

Supplier Order index and detail use compact semantic badges:

| Status | Color |
|---|---|
| `DRAFT` | neutral gray |
| `PENDING_APPROVAL` | amber |
| `APPROVED` | green |
| `PARTIALLY_RECEIVED` | blue |
| `FULLY_RECEIVED` | dark green |
| `REJECTED` | red |
| `CANCELLED` | red |

Text remains visible so color is never the only status indicator.

### Sage PO Number

`sagePurchaseOrderNumber` remains read-only and empty until the Sage outbound integration supplies a value. It appears:

- as an optional show/hide column in Supplier Orders;
- in Supplier Order detail.

The MVP does not provide manual editing or placeholder/demo values.

## Inventory Foundation

### Kanban Lifecycle

Operational Kanban states are:

```text
ISSUED
  -> RECEIVING_SESSION
  -> IN_STOCK
  -> OUTGOING_SESSION
  -> CONSUMED
```

Paused or cancelled sessions return their locks to the prior durable state without deleting the session audit record.

### Append-Only Ledger

`inventory_ledger_entries` is append-only and records:

- tenant, material, Kanban lot, warehouse, and location;
- event type (`RECEIVING`, `OUTGOING`);
- signed base-unit quantity delta;
- business document type and identifier;
- event timestamp and actor.

Receiving writes a positive entry. Outgoing writes the full Kanban quantity as a negative entry. Posted transactions are not edited or deleted; later corrections must use Stock Adjustment.

## Receiving

### Index

The compact Receiving index shows:

- Receiving Number;
- DN Number;
- PO Number;
- Supplier;
- Receiving Date;
- Kanban received now;
- Outstanding Kanban;
- Status;
- Sage Receipt Number;
- Created By.

Search, filters, and show/hide columns follow the existing compact table pattern.

### Create Form

`Create Receiving` opens a centered form. The DN selector supports dropdown selection and typing, and only returns issued DNs with outstanding Kanban.

After selection, the form displays PO and supplier data plus a per-material preview of planned, previously received, and outstanding Kanban. A system-generated Receiving Number and receiving date are shown before session creation.

### Exclusive Scan Session

Creating the receiving starts a focused full-screen scan session. Database constraints guarantee no more than one active session per DN.

A session contains its scanned Kanban list and supports:

- scan/type Kanban ID;
- remove before submit;
- pause;
- resume by the same or another authorized operator;
- cancel with a required reason when scanned data exists.

Pausing retains scans and releases the DN for continuation through that same session; it does not create a second session. Sessions also expire after a configurable inactivity timeout so abandoned browser sessions do not hold locks indefinitely.

### Scan Validation

Each scan must be:

- assigned to the selected DN;
- still outstanding;
- not previously received;
- not already present in the session;
- not locked by another active transaction.

Partial receiving is allowed. Excess and duplicate receiving are rejected. Scan errors appear inline in the focus zone with compact visual feedback.

### Atomic Completion

Submit performs one local database transaction that:

1. locks and revalidates the session, DN, and all scanned Kanban lots;
2. creates the completed receiving header and lines;
3. changes scanned Kanban lots to `IN_STOCK`;
4. appends positive inventory ledger entries;
5. stores immutable planned, previously received, received-now, and outstanding snapshots;
6. updates the DN and derived PO receiving status;
7. inserts a transactional outbox event `SAGE_GOODS_RECEIPT_CREATE`;
8. closes the session.

If any Kanban fails final validation, nothing is partially posted.

Receiving completion does not wait for Sage. The Sage Receipt Number stays read-only and empty until the integration returns it.

### Receiving PDF

A real PDF is immediately available after local completion and does not wait for Sage. It contains:

- receiving, DN, PO, supplier, and date;
- received Kanban identifiers and material quantities;
- planned, previously received, received now, and outstanding values;
- outstanding base quantity;
- DN/PO receiving status;
- creator and completer;
- Sage Receipt Number when available.

PDF failure never rolls back a successfully completed receiving; the document can be regenerated from immutable snapshots.

### PO Status Projection

After receiving:

- some but not all ordered Kanban received: `PARTIALLY_RECEIVED`;
- all ordered Kanban received: `FULLY_RECEIVED`.

The commercial PO content remains read-only after approval.

## Outgoing Material

### Index and Header

The compact index shows Document Number, Date, Destination, Kanban count, distinct material count, Status, and Created By.

`Create Outgoing` captures:

- system-generated document number;
- transaction date;
- destination through a typeable dropdown with suggested values and free-text input;
- optional notes.

No Production Line master module is added in this increment.

### Full-Kanban Scan Session

Outgoing uses a focused scan session. Quantity cannot be typed or split. Every accepted scan represents one complete Kanban lot currently in `IN_STOCK`.

Reject scans when the Kanban:

- has not been received;
- is already consumed;
- is duplicated in the session;
- is locked by another active operation.

The preview shows Kanban ID, material, full Kanban quantity, base unit, warehouse, and location. Scans may be removed before submit.

### Atomic Completion

Submit revalidates all scans and atomically:

- creates the completed Outgoing document and lines;
- changes lots to `CONSUMED`;
- appends full-quantity negative ledger entries;
- closes the session.

No Sage event is produced for Outgoing in the MVP. Completed transactions are immutable.

### Outgoing PDF

A compact internal PDF is available immediately after completion and includes header, destination, notes, operator, and all consumed Kanban/material quantities.

## Authorization and Audit

Existing permissions govern the feature:

- `receiving.view`, `receiving.create`, `receiving.submit`;
- `inventory.view`, `inventory.consume`;
- `po.view` for linked PO information.

Every header, session, scan, ledger entry, and state transition is tenant-scoped and records its actor and timestamps. Completion endpoints are idempotent and protect against double submission.

## Error Handling

- Validation failures return typed client errors and preserve the active session.
- Concurrency conflicts state which DN or Kanban is already active without exposing another tenant's data.
- Database/outbox failure rolls back the entire posting transaction.
- Sage delivery failures are retried asynchronously and never reverse local receiving.
- PDF rendering errors are logged with tenant, actor, and document identifiers and return sanitized responses.

## Testing and Acceptance

Automated coverage must include:

- PO PDF layout regression and every colored badge mapping;
- Sage PO Number visible but not editable;
- one active receiving session per DN under concurrent requests;
- pause/resume retaining scans;
- inactivity expiry;
- valid partial receiving and subsequent completion;
- duplicate, wrong-DN, already-received, and excess rejection;
- atomic rollback when one scanned lot becomes invalid;
- positive receiving ledger and negative full-Kanban outgoing ledger;
- Outgoing rejects partial quantity and non-stock lots;
- derived `PARTIALLY_RECEIVED` and `FULLY_RECEIVED` PO states;
- receiving/outgoing permission and tenant isolation;
- Receiving and Outgoing PDF contracts;
- outbox creation exactly once for receiving and never for outgoing.

Live verification uses the existing Docker PostgreSQL application role, browser-facing API, and real generated PDFs.
