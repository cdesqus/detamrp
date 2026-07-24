# PO Dropdown Dismiss Design

Supplier Order `Action` and `Docs` menus close when the user clicks anywhere outside the open menu, presses Escape, opens the other menu, or completes a menu action. Clicking inside the menu must not dismiss it before the selected action runs. The change is UI-only and adds no dependency.

Success criteria:

- Outside pointer interaction closes either menu.
- Escape closes either menu.
- Opening Action closes Docs and opening Docs closes Action.
- Existing menu actions and document links continue to work.

