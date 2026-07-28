# Supplier Order Row Menu Portal Design

## Goal

Make the Supplier Orders `Action` and `Docs` menus usable without vertical table scrolling when the result set contains only a few rows.

## Root Cause

The menus are absolutely positioned inside `.table-frame`. That element must use horizontal overflow for the wide transaction table. CSS overflow rules therefore clip vertical descendants and create a vertical scrollbar when a menu extends below a short table.

## Design

- Render the open row menu into `document.body` with a React portal.
- Keep the trigger buttons inside the table.
- Position the portal with `position: fixed` from the active trigger's `getBoundingClientRect()`.
- Match the current menu widths and compact styling.
- Prefer opening below the trigger. Open above when the available viewport space below is smaller than the menu height and the space above is larger.
- Clamp the menu horizontally inside an 8-pixel viewport margin.
- Recalculate position when the menu opens, the window resizes, or any scroll container/window scrolls.
- Close the menu on outside pointer interaction, Escape, route-triggering action, or when the other menu is opened.
- Preserve existing disabled items, links, permissions, email actions, and cancellation behavior.
- Preserve keyboard focus behavior and return focus to the trigger when Escape closes the menu.
- Remove row z-index workarounds that are no longer required. Keep horizontal table scrolling unchanged.

## Scope

The change applies to both `Action` and `Docs` menus on the Supplier Orders index. It does not redesign other tables or change backend behavior.

## Testing

- A one-row table opens a full Action menu without adding vertical table overflow.
- Docs and Action remain mutually exclusive.
- Outside click and Escape close the portal.
- Escape restores focus to the active trigger.
- Resize and scroll reposition the portal.
- The menu opens upward near the viewport bottom.
- Existing Supplier Order action/permission tests remain successful.

