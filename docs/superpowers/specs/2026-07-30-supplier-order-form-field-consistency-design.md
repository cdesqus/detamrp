# Supplier Order Form Field Consistency Design

Date: 2026-07-30

## Objective

Make the Create/Edit Supplier Order header fields visually consistent and replace the Plant native dropdown with the same searchable selection pattern used by Supplier.

## Scope

This change affects only the Supplier Order header form:

- Supplier
- Plant
- Order Date
- Expected Delivery Date
- Currency
- Sage PO Number when present
- Notes

The Raw Materials toolbar and table are explicitly out of scope.

## Plant Selection

Editable Plant selection uses an input with a datalist, matching Supplier.

- Options display `CODE — Name`.
- Selecting an exact option commits its stable `plantId` and `plantName`.
- Incremental typing does not clear the last committed Plant.
- An unmatched value is restored to the last committed Plant on blur.
- Clearing an uncommitted field leaves the Plant validation rule unchanged.
- Read-only orders continue to display the stored Plant snapshot.

The backend request and database model remain unchanged because the form continues submitting `plantId`.

## Field Presentation

All header controls share one form-control visual system:

- Equal control height where the control is single-line
- Identical border, radius, horizontal padding, font, foreground, and background
- Consistent focus and disabled/read-only treatment
- Labels, validation messages, and control edges align within the grid

Notes remains a multiline textarea and spans two columns. Its border, radius, font, and focus treatment match the single-line controls. Responsive layouts continue collapsing the existing grid without horizontal overflow.

## Error Handling

- Plant lookup failure retains the current retry message and action.
- Required Plant validation remains attached to the Plant field.
- Invalid free text is never submitted as a Plant identifier.
- Existing Supplier, date, and currency behavior is unchanged.

## Verification

Automated tests cover:

- Plant renders as a searchable combobox rather than a native select.
- Exact Plant selection commits the expected `plantId`.
- Incremental or unmatched Plant text does not corrupt the committed value.
- Blur restores the committed Plant label.
- Header controls use the shared control styling hook.
- Existing Supplier Order tests, lint, typecheck, and production build remain green.

