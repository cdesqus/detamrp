# Enterprise Dropdown Controls Design

## Scope

Refine the Supplier Orders row controls (`Action` and `Docs`) and the application header user-menu trigger so their boxes, spacing, and chevrons feel precise and consistent with the existing monochrome interface. Menu contents and business behavior remain unchanged.

## Visual Direction

- Preserve the current white, charcoal, and neutral-gray palette.
- Use one reusable SVG chevron rather than font glyphs so stroke weight, baseline, and alignment are deterministic.
- Give each control a clear label area and a consistently aligned chevron at the right edge.
- Keep compact dimensions appropriate for a dense enterprise table and header.
- Use restrained neutral hover and focus treatments; introduce no navy or other new accent color.

## Controls

### Supplier Order Row Triggers

- `Action` and `Docs` use the same height, border, radius, typography, and internal spacing.
- The text remains left aligned and the chevron remains centered in a fixed-width trailing area.
- The chevron rotates 180 degrees while its corresponding menu is open.
- Open state uses a slightly stronger neutral border and background.
- Hover uses a subtle neutral-gray background.
- Keyboard focus uses a visible charcoal/gray focus ring without shifting layout.

### Administrator Trigger

- Replace the current text arrow with the same SVG chevron used by the row triggers.
- Preserve the header-specific height and truncation behavior for long display names.
- Align the chevron vertically and horizontally independent of the name length.
- Apply the same neutral hover, focus, open-state, and rotation language as the row triggers.

## Component Boundaries

- Add a small presentational chevron icon to the existing shared icon module.
- Keep menu state and event handling in the current Supplier Orders and App Shell components.
- Apply dedicated class names or state attributes to the existing trigger buttons; do not create a new dropdown framework for this visual-only change.

## Accessibility

- SVG chevrons are decorative and hidden from assistive technology.
- Existing accessible button names, `aria-haspopup`, and `aria-expanded` semantics remain intact.
- Open state is represented by `aria-expanded`, allowing styling and tests to follow the semantic state.
- Focus-visible treatment must remain clearly distinguishable from hover.

## Testing and Verification

- Component tests assert that each trigger renders the SVG chevron instead of a text glyph.
- Tests assert `aria-expanded` changes when menus open and close, covering the state used for rotation styling.
- Existing menu interaction tests continue to cover opening, Escape dismissal, outside-click dismissal, and focus restoration.
- Run the targeted Supplier Orders and App Shell tests, then the frontend typecheck and lint.
- Inspect the rendered Supplier Orders page at desktop width to confirm box alignment, chevron centering, hover/focus/open states, and consistency with the supplied reference screenshot.

## Out of Scope

- Changing dropdown menu contents or actions.
- Altering table data, status colors, navigation, authentication, or logout behavior.
- Introducing a new color accent or redesigning unrelated buttons.
