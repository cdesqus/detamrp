# Centered Master Data Modal Design

## Scope

Replace the right-side master-data drawer with a centered modal for both create and edit. Improve sidebar hierarchy by indenting child module links under collapsible group headers.

## Modal

- Keep the existing shared `MasterDataCrud` behavior and API flow.
- Render create and edit in the same centered modal.
- Use a compact width for short forms and a wide width for Supplier and Raw Material forms.
- Constrain height to the viewport; scroll the form body while keeping header and footer visible.
- On mobile, use the available viewport width with small outer margins.
- Preserve backdrop click, close, cancel, validation, and save behavior.

## Sidebar hierarchy

- Indent links belonging to collapsible groups.
- Add a subtle vertical hierarchy guide.
- Keep Dashboard and Reports at root alignment.
- Remove indentation when the entire sidebar is icon-only.

## Verification

- Component tests cover create and edit using a centered modal dialog.
- Navigation tests cover nested link classes.
- Run the full frontend test suite and production build.
