# Task 4 Report: Supplier Order Index and Dedicated Editor

## Scope

Implemented Task 4 only:

- `frontend/components/supplier-orders/supplier-order-index.tsx`
- `frontend/components/supplier-orders/supplier-order-form.tsx`
- `frontend/components/supplier-orders/supplier-orders.test.tsx`
- `frontend/app/supplier-orders/page.tsx`
- `frontend/app/supplier-orders/new/page.tsx`
- `frontend/app/supplier-orders/[id]/page.tsx`
- `frontend/app/globals.css`

## TDD Evidence

### RED

Added the supplier-order UI tests before the components existed, then ran:

```powershell
npm test -- --run components/supplier-orders/supplier-orders.test.tsx
```

Observed the expected failure:

```text
Failed to resolve import "./supplier-order-form"
```

### GREEN

After implementation, ran:

```powershell
npm test -- --run components/supplier-orders/supplier-orders.test.tsx
npm run typecheck
npm test -- --run
npm run build
git diff --check
```

Observed:

```text
Supplier-order focused suite: 4 passed
Full frontend suite: 11 files, 22 tests passed
TypeScript: exit 0
Next.js production build: exit 0
git diff --check: exit 0
```

## Delivered Behavior

- Real compact PO list with API loading, empty/error/retry states, supplier-name resolution, and dedicated create/detail navigation.
- Create and detail/draft editor routes; submitted/cancelled orders render read-only.
- Native searchable supplier/material datalists; material lookup is active-only and supplier-filtered.
- `+ Raw Material` stays disabled before supplier selection; selected material choices are removed to prevent duplicates.
- Supplier change asks for confirmation and clears selected materials only after confirmation.
- Read-only master snapshots, positive integer Total Kanban control, and BigInt-based fixed-six-decimal calculation/display helpers without JS float arithmetic.
- Draft save, create-then-submit, update-then-submit, draft cancel, error/field-error rendering, and sticky compact action bar.

## Self-review

- Confirmed payload keys match the lower-camel purchase-order HTTP request contract: `supplierId`, `orderDate`, `expectedDeliveryDate`, `currency`, `notes`, `lines`, `rawMaterialId`, and `totalKanban`.
- Confirmed the UI contains no forbidden `Add Line`, `Line Items`, or `Total Lines` copy.
- Confirmed material/supplier fetches use the existing master-data endpoints and authenticated browser credentials.
- Confirmed index supplier labels are derived from the master-data supplier endpoint because the current PO response provides `supplierId`, not a supplier display name.

## Concerns

- None.

## Commit

`feat: activate supplier order UI`
