# DETA MRP Master Data, Documents, Branding, and Activity Log Design

Date: 2026-07-30

## Objective

Extend the application with Unit, Category, Packing, and Plant master data; make the destination Plant part of the purchase-order flow; expose important master information consistently in lists, PDFs, and email; rebrand the product as DETA MRP; support company-controlled images; and add an immutable Activity Log.

Implementation will use staged vertical slices so each stage can be reviewed and verified before the next one starts.

## 1. Master Data and Navigation

The Data Master navigation hierarchy will be:

- Measurements
  - Unit
  - Category
  - Packing
- Plants
- Suppliers
- Raw Materials

The existing Measurements data represents units. It will be renamed throughout the database, backend, API, routes, UI, and tests:

- `measurements` table becomes `units`.
- `/measurements` becomes `/units`.
- User-facing labels become Unit or Units.
- Existing `base_unit_id` fields remain unchanged.
- The change is delivered through a new forward migration. Existing migrations are not rewritten.

Category and Packing are independent master-data modules with:

- Code
- Name
- Description
- Active status
- Existing audit fields

Plant is an independent master-data module with:

- Code
- Name
- Address
- Active status
- Existing audit fields

All records are tenant-isolated and use deactivate/reactivate instead of hard deletion.

## 2. Raw Material and Purchase Order Rules

Each Raw Material selects exactly one Category and one Packing. These are master-derived values, not editable PO fields.

Legacy Raw Materials may remain visible without these relations after migration. However:

- Creating or editing a Raw Material requires both values.
- A PO containing an incomplete legacy Raw Material cannot be submitted.
- The validation response identifies the affected material.

One PO selects exactly one destination Plant at header level. Every line and generated document in that PO uses the same Plant. Plant can be selected while the PO is a draft and is locked when the PO enters approval.

At transaction time, stable snapshots are stored for:

- Category code and name on each PO line
- Packing code and name on each PO line
- Plant code, name, and address on the PO header

Master changes or deactivation must not alter historical PO, DN, Kanban, email, or list data.

## 3. Lists and Tables

All operational and master-data lists will use one consistent table system:

- Search and relevant filters above the table
- Clear column headers and compact rows
- Consistent status badges
- Compact actions on the right
- Standard loading, empty, and error states
- Horizontal scrolling on narrow screens
- Category, Packing, and Plant columns only where relevant

The shared presentation pattern must avoid page-specific visual drift while allowing each list to choose its own columns and filters.

## 4. PDF Documents

All PDFs use a clean, ink-efficient white design with lines as information separators. Solid black and grey blocks are not used.

### Purchase Order

- Open, uncluttered header
- Company identity and PO number are prominent
- Supplier and destination Plant information use a clear two-column information area
- Materials use one primary table
- Summary and approval areas use restrained borders
- Category and Packing appear per material

### Delivery Note

- Professional open header without boxed header cells
- DN number is visually prominent
- Supplier and destination Plant are easy to locate
- QR area is separate and clear
- Material and totals use aligned tables
- Remarks and signatures use minimal necessary borders

### Kanban Card

- Three cards per page
- Structured grid inspired by the approved reference
- Company logo/name and destination Plant in the header
- Large part number and part name
- Supplier, Category, Packing, quantity, PO/DN number, and order date
- Primary QR code with `Card 1/N` directly below it on the left
- Card numbering restarts for each material
- Lines separate information without dark fill blocks

## 5. Product and Company Branding

The permanent product name is **DETA MRP**.

All user-visible occurrences of Order Stock and the OS brand mark are replaced across:

- Login and application shell
- Page metadata and UI copy
- PDFs
- Emails
- SMTP defaults and test messages

Internal Go module names and package identifiers are not renamed solely for branding unless they are user-visible.

Company Settings controls:

- Company Name
- Company Logo
- Login Background

The deployment serves one company, so the public login branding can use the single installation's company settings without tenant selection.

Uploads are stored in the database and included in normal backups:

- Logo: PNG, JPEG, or WebP; maximum 2 MB
- Login background: PNG, JPEG, or WebP; maximum 5 MB

The server validates the decoded file type and size rather than trusting filename extensions. The UI provides preview, replace, and reset actions. Missing images use a clean DETA MRP fallback.

A narrow public branding endpoint exposes only the company name and login-safe image content. It exposes no private settings or internal metadata.

## 6. Professional Email Templates

All generated emails use one responsive, light visual system:

- Inline company logo that works without authentication
- Company identity and DETA MRP header
- Clear title and purpose
- Table-based summary for PO/DN number, Supplier, Plant, dates, status, and totals
- Compact material table with part, Category, Packing, quantity/Kanban, and monetary values where relevant
- Consistent footer

The template applies to:

- PO approval request
- Approval and rejection result
- Supplier document delivery
- SMTP test

Approval emails provide clear Approve, Reject, and View PO actions. Supplier emails explain the PO, DN, and Kanban attachments and printing/shipping instructions. Rejection emails include actor, time, and reason. If a logo is unavailable, the text header remains complete and aligned.

## 7. Activity Log

Activity Log is a Settings module protected by `activity_log.view`.

It records state-changing business actions, including:

- Create and edit
- Activate and deactivate
- Submit, approve, reject, and cancel
- DN issue and cancellation
- Receiving and inventory movement
- Company, role, user, and other settings changes

Page views are not logged.

Each immutable event stores:

- Timestamp
- Actor
- Module
- Action
- Target type and stable record identifier/code
- Safe before/after change summary

The list supports date, user, module, and action filters. A row can open structured change details.

Passwords, tokens, raw logo/background bytes, and other secrets are never logged. Media changes record only a safe statement such as `Company Logo updated`.

The activity event is written in the same database transaction as the business mutation. If audit persistence fails, the mutation rolls back.

## 8. Error Handling and Compatibility

- Inactive master records remain readable in history but cannot be selected for new work.
- PO submission fails clearly when Plant, Category, or Packing is missing.
- Invalid, corrupt, disguised, or oversized image files are rejected.
- PDFs and emails fall back to text branding if an image cannot be decoded.
- Existing historical records continue to render using snapshots or explicit safe fallback values.
- Frontend and backend route renames for Unit ship together; a legacy `/measurements` compatibility API is not retained.

## 9. Verification

Automated coverage will include:

- Forward migrations and tenant isolation
- Unit rename and route/navigation behavior
- Category, Packing, and Plant CRUD
- Raw Material relationships and PO submission validation
- Snapshot stability across PO, DN, Kanban, lists, and email
- One-Plant-per-PO enforcement
- `Card 1/N` sequencing per material and three cards per page
- Image upload, reset, decoding, size validation, and public endpoint sanitization
- Activity Log persistence, permissions, filters, rollback behavior, and secret redaction
- PDF and email rendering regression tests
- Existing backend and frontend suites

## 10. Delivery Order

1. Unit rename plus Category, Packing, and Plant master-data foundation
2. Raw Material relations, destination Plant, snapshots, validations, and transaction lists
3. PO, DN, and Kanban PDF redesign
4. DETA MRP branding, Company Settings media, and login treatment
5. Professional email templates
6. Activity Log

Each stage ends with focused automated tests, the full relevant test suite, and a reviewable commit before the next stage begins.
