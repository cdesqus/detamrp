# Master Data CRUD and Collapsible Navigation Design

## Objective

Deliver the first functional CRUD slice for Measurements, Suppliers, and Raw Materials in dependency order, while making every sidebar group collapsible. The implementation must preserve tenant isolation, RBAC, audit ownership, compact UX, and the distinction between base units and Kanban grouping.

## Scope and Dependency Order

Implementation order is:

1. Collapsible sidebar groups.
2. Measurement CRUD.
3. Supplier CRUD.
4. Raw Material CRUD.
5. Docker and browser-path verification.

Raw Materials are implemented after Measurements and Suppliers because each material requires an active base unit and primary supplier.

## Collapsible Navigation

Data Master, Procurement, Logistics, and Settings are expandable/collapsible groups.

- The group containing the active route opens automatically.
- Users may manually open or close each group.
- Group state is persisted in browser local storage.
- Dashboard and Reports remain direct links.
- When the entire sidebar is collapsed, compact group icons remain visible and submenu labels are hidden.
- Mobile uses the existing sidebar drawer behavior.

## Common CRUD Behavior

All three masters support:

- compact searchable index;
- create in a right-side drawer;
- edit in the same drawer with existing values;
- activate/deactivate rather than hard delete;
- active/status filtering where relevant;
- explicit field-level validation messages;
- list refresh and drawer close after a successful save;
- Created By, Updated By, Created At, and Updated At columns available through column visibility;
- tenant-safe API access through PostgreSQL RLS;
- permission checks using `master_data.view` and `master_data.manage`.

No record is hard-deleted in the MVP. Deactivation requires confirmation. Normal create and edit saves do not require confirmation.

## Measurement CRUD

### Fields

- Unit Code
- Unit Name
- Decimal Allowed
- Status

### Rules

- Code is trimmed and normalized to uppercase.
- Code is unique within a tenant.
- `KANBAN` is prohibited because Kanban is a physical lot grouping, not a base measurement.
- Code and name are required.
- A measurement referenced by Raw Materials remains stored and may only be deactivated.
- Only active measurements can be assigned to newly created or edited Raw Materials.

## Supplier CRUD

### Fields

- Supplier ID
- Sage Supplier Code
- Supplier Name
- Email
- Phone
- Address
- Contact Person
- Currency
- Status

### Rules

- Supplier ID is normalized to uppercase and unique within a tenant.
- Sage Supplier Code is required and unique within a tenant.
- Supplier Name and Email are required.
- Email must use a valid address shape.
- Currency is selected from the supported ISO 4217 set initially containing IDR, USD, EUR, JPY, and SGD.
- A referenced supplier remains stored and may only be deactivated.
- Only active suppliers can be assigned to newly created or edited Raw Materials.

## Raw Material CRUD

### Fields

- Item Code
- Sage Item Code
- Raw Material Name
- Primary Supplier
- Base Unit
- Qty per Kanban
- Minimum Stock
- Standard Unit Price
- Currency
- Description
- Status

### Rules

- Item Code and Sage Item Code are normalized to uppercase and unique within a tenant.
- Name, supplier, and base unit are required.
- Supplier and base unit selectors support both dropdown selection and typed search.
- Qty per Kanban must be greater than zero.
- Minimum Stock must be zero or greater.
- Standard Unit Price must be zero or greater.
- Currency is derived from the selected primary supplier and is read-only in the material form.
- The backend verifies material currency equals the supplier currency; the client value is not trusted.
- Raw Material stock remains tracked in Base Unit. Kanban only represents the physical lot quantity stored in Qty per Kanban.
- The standard price becomes the default commercial unit-price snapshot when a future PO is created. PO records will preserve their own snapshot so later master-price changes do not rewrite approved commitments.

## Database Changes

A new additive migration updates `raw_materials` with:

- `standard_unit_price numeric(20,6) NOT NULL DEFAULT 0 CHECK (standard_unit_price >= 0)`;
- `currency char(3) NOT NULL` populated from each existing material's primary supplier before enforcing non-null.

The migration preserves tenant-scoped composite foreign keys, existing RLS, and existing records. Raw Material create/update operations derive currency inside the same transaction that validates the supplier.

## Backend Architecture

The modular-monolith backend adds a master-data HTTP module divided by responsibility:

- route handlers decode requests and map errors;
- service methods normalize and validate domain input;
- PostgreSQL stores execute tenant-scoped queries inside `database.WithTenant`;
- response DTOs include audit display names without exposing password or session fields.

### Endpoints

- `GET /master-data/measurements`
- `POST /master-data/measurements`
- `GET /master-data/measurements/:id`
- `PATCH /master-data/measurements/:id`
- corresponding routes for `/suppliers` and `/raw-materials`.

List endpoints accept `search`, `active`, `limit`, and `offset`. Raw Materials additionally accept `supplier_id`. Default limit is 50 and maximum limit is 200.

### Authorization

- Read endpoints require `master_data.view`.
- Create and update endpoints require `master_data.manage`.
- The authenticated user ID supplies created/updated audit fields.
- Every database transaction sets tenant and user context before accessing RLS tables.

## API Errors

Errors use a stable JSON shape:

```json
{
  "error": "validation_failed",
  "message": "Please correct the highlighted fields",
  "fields": {
    "code": "Code is already in use"
  }
}
```

- malformed input returns HTTP 400;
- missing records return HTTP 404;
- uniqueness or referenced-state conflicts return HTTP 409;
- permission denial returns HTTP 403;
- unexpected database failures return HTTP 500 without exposing SQL details.

## Frontend Architecture and UX

- Each master route loads real API data after session validation.
- A reusable compact `DataTable` owns search, filters, column visibility, loading, error, and empty states.
- A reusable `FormDrawer` owns focus trapping, Escape/close behavior, unsaved-state confirmation, and mobile full-width presentation.
- Each domain owns a focused form component and typed API adapter.
- Search requests are debounced and stale responses cannot overwrite newer results.
- Create/update success closes the drawer and reloads the current filtered list.
- Failed save keeps the drawer open and maps server field errors to the matching controls.
- Disabled future features are not mixed into active CRUD controls.

## Data Administration Views

### Measurement columns

Code, Name, Decimal Allowed, Status, Created By, Updated By, Updated At, Actions.

### Supplier columns

Supplier ID, Sage Supplier Code, Name, Email, Phone, Currency, Status, Created By, Updated By, Updated At, Actions.

### Raw Material columns

Item Code, Sage Item Code, Name, Primary Supplier, Base Unit, Qty per Kanban, Minimum Stock, Standard Unit Price, Currency, Status, Created By, Updated By, Updated At, Actions.

## Verification

- Navigation tests cover independently collapsible groups, active-route expansion, persistence, and collapsed-sidebar behavior.
- Backend domain tests cover normalization, required fields, numeric boundaries, `KANBAN` rejection, currency derivation, and inactive references.
- Store integration tests cover tenant isolation, pagination, search, unique conflicts, audit ownership, and create/update behavior against PostgreSQL.
- API tests cover authentication, permission denial, validation shapes, CRUD success, not found, and conflict responses.
- Frontend tests cover list loading, create/edit drawers, field errors, searchable supplier/unit selectors, deactivation confirmation, and refresh-after-save.
- Migration tests verify standard price and currency constraints.
- Frontend test, lint, typecheck, production build, backend test, vet, Docker rebuild, authentication, and browser CRUD smoke tests must pass.

## Deferred Work

- Hard deletion.
- Bulk import/export of master records.
- Sage inbound master synchronization.
- Warehouse and Warehouse Location CRUD.
- PO creation and commercial price snapshot implementation.
