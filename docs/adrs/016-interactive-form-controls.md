# ADR 016: Interactive Form Controls Architecture

## Status
Accepted

## Context
Kite currently has advanced text inputs (`Input` and `TextArea`) built on the UA Shadow Subtree architecture (ADR-009). However, a standard UI framework requires a comprehensive suite of interactive controls: Buttons, Checkboxes, Radio Buttons, and Selects (Dropdowns).

We need a standardized architectural approach to building these elements that respects terminal coordinate systems, keeps the public DOM clean of complex ASCII drawing nodes, and integrates with Kite's existing focus and event systems.

## Decision
We will implement the standard form controls using the following paradigms:

### 1. Button
A dedicated `element.Button` type, implementing `Focusable()`.
Unlike form controls that hide internal text nodes inside a UA Shadow Subtree, the `Button` acts as a standard layout container (BFC or FFC). Authors append standard `Text` or `Box` nodes to it.
The button translates `Enter` and `Space` `KeyEvent`s into semantic `EventClick` events, acting as a normalized interaction target.

### 2. Checkbox
A compound widget built on the **UA Shadow Subtree** pattern (ADR-009).
- **State:** Maintains a `Checked` boolean.
- **Visuals:** Uses a hidden inner subtree containing a text node (e.g., `[ ]` or `[X]`). The structural DOM tree remains unpolluted.
- **Customization:** Exposes properties to customize the checked/unchecked glyphs.

### 3. RadioGroup & Radio
To avoid slow `O(N)` DOM tree walks to resolve radio grouping by `name` attributes (HTML style), we enforce a strict parent-child structural relationship.
- **`element.RadioGroup`:** Holds the active `Value` state and coordinates its children. Fires `EventChange`.
- **`element.Radio`:** Similar to a Checkbox (using a UA Shadow Subtree for `( )` / `(•)` visuals), but it is stateless regarding deselection. Clicking it dispatches an internal event to the parent `RadioGroup`, which updates the active value and forces sibling radios to re-render.

### 4. Select (Dropdown)
A highly composite widget utilizing the UA Shadow Subtree, the `Overlay` layout system, and Focus Scopes.
- **Host Component:** `element.Select` acts as the data holder. Its Shadow Subtree contains a UI Button displaying the current value and a dropdown indicator (e.g., `Value  ▼`).
- **Options:** Authors provide `element.Option` children to the `Select`. These are pure data nodes and are not rendered directly in the standard flow.
- **The Dropdown Overlay:** Upon activation (Click/Enter), the `Select` mounts an `element.Overlay` dynamically anchored to itself (using `PlacementBottom` and `Flip: true`).
- **Focus Trapping:** When the overlay opens, the `Select` pushes a new `focus.Scope` targeting the overlay's root to trap keyboard navigation inside the list. When an option is chosen or the overlay is dismissed (Escape), the scope is popped, the value updates, and the overlay is unmounted.

## Consequences

### Positive
- **Clean Public DOM:** Form control visuals (like `[X]` strings) do not clutter the logical DOM tree.
- **Robust Dropdowns:** Reuses the existing robust Overlay and Focus Scope engines, preventing the need for complex z-index hacking.
- **Strict Architecture:** Radios are O(1) to coordinate due to strict parent-child enforcement, bypassing the historical pitfalls of HTML's radio `name` grouping.

### Negative
- **Component Complexity:** The `Select` component will be quite complex internally, requiring careful lifecycle management (mounting/unmounting overlays and managing focus scopes correctly on destruction).
