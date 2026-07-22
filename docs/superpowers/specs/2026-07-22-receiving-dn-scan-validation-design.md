# Receiving DN Scan Validation Design

## Goal

Replace the Delivery Note suggestion control with a scanner-first exact-match flow and prevent empty receiving sessions from crashing the Kanban scan page.

## User Flow

1. The operator opens **Create receiving**.
2. A single focused field labeled **Scan or Type DN Number** is shown. It has no dropdown, suggestion list, or preloaded DN options.
3. The operator scans or types the complete DN number and presses Enter.
4. The server validates the number and atomically creates the receiving session when valid.
5. A successful response immediately closes the modal and navigates to the Kanban scan session.

## Validation Outcomes

All operator-facing messages are in English:

- Unknown or ineligible DN: `Delivery Note is invalid.`
- DN with no outstanding Kanban: `Delivery Note has already been fully received.`
- DN locked by an active or paused session: `Delivery Note is currently being received in another session.`
- Unexpected error: `Receiving session could not be created.`

Matching is case-insensitive after trimming whitespace, but requires the complete DN number. Partial matches are rejected.

## Backend Contract

`POST /receiving-sessions` accepts `deliveryNoteNumber` instead of requiring a database UUID from the browser. Within one tenant-scoped transaction, the backend:

1. resolves the exact DN number;
2. confirms it belongs to an approved/receivable PO;
3. checks whether any Kanban remains in `ISSUED` state;
4. rejects an existing active or paused receiving session;
5. creates and returns the session.

Conflict responses include a stable machine-readable error code so the UI does not infer business states from free-form text.

## Empty Scan Safety

Receiving session API responses always serialize `scans` as an empty JSON array when no Kanban has been scanned. The frontend also treats a missing or null legacy value as an empty array. This removes the current `scans.length` runtime crash and keeps the scan input focused.

## Tests

- Backend tests cover exact/case-insensitive matching, invalid DN, fully received DN, and an existing session lock.
- Frontend tests prove there is no suggestion control, Enter submits the exact DN number, each validation code renders the correct English message, successful creation navigates to the session, and a null scan collection does not crash.
- Full backend, frontend, type-check, production build, and live Docker smoke checks run before completion.
