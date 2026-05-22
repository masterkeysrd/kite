# TSK-039: Implement Select (Dropdown) Component

## Description
Create an `element.Select` component that combines a trigger button with a dynamic `Overlay` containing a scrollable list of options, utilizing a temporary `focus.Scope`.

## Requirements

### 1. Select & Option Elements (`element/select.go`)
- Create `OptionElement` (just a data wrapper for `Value` and `Text`).
- Create `SelectElement`. It maintains `Value string`.
- **Trigger Button:** The Select's UA Shadow Subtree contains a single `ButtonElement` that displays the currently selected Option's text plus a dropdown arrow (e.g., `Option 1  ▼`).

### 2. Overlay Dropdown
- When the trigger button is clicked, the `SelectElement` instantiates an `element.Overlay`.
- Configuration: `Anchor` is the `SelectElement` itself, `Placement` is `PlacementBottom`, `Flip` is `true`.
- The Overlay's root should be a `Box` with `OverflowY(Auto)` and `ScrollbarY(true)`.
- Iterate through the `SelectElement`'s public `OptionElement` children. For each, create a focusable `Box`/`Button` inside the Overlay.
- **Mount:** Call `doc.ShowOverlay(overlay, zIndex)`.

### 3. Focus Trapping
- Once mounted, call `doc.FocusManager().PushScope(&focus.Scope{Root: overlay})`.
- This ensures Up/Down arrow keys strictly navigate the dropdown list and do not leak into the main document.

### 4. Selection & Teardown
- If an item in the overlay is clicked/Enter'ed:
  - Update the `Select` value.
  - Update the Trigger Button text.
  - Fire `EventChange`.
- **Teardown:** Whether an item was selected, or the user pressed `Escape` / clicked outside, the Select must:
  - Call `doc.HideOverlay(overlay)`.
  - Pop the focus scope `fm.PopScope()`.
  - Return focus to the `Select` trigger button.

## Tests
- Mount a Select with 3 Options. Trigger a click. Verify the overlay is added to the document.
- Verify Focus Manager scope is constrained to the overlay.
- Trigger `Enter` on an option. Verify overlay is removed, scope is popped, and value updates.