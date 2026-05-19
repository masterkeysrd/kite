# Package focus

Package `focus` implements focus management and keyboard navigation for Kite.

## 🧠 Focus Management

Focus state is managed by the `focus.Manager`, which tracks:
* **Current Focus:** The `dom.Node` that currently has focus.
* **Reason:** How focus was acquired (`ReasonProgrammatic`, `ReasonPointer`, `ReasonKeyboard`, `ReasonRestore`).
* **Scopes:** A stack of `focus.Scope` objects that restrict navigation to specific subtrees (e.g., for modals).

## 🎯 Focusable Criteria

A node is considered focusable if it meets all of the following conditions:
1. **Semantic Capability:** The logical node must implement `dom.Focusable` and `IsFocusable()` must return `true`.
2. **Enabled State:** If the node implements `dom.Disableable`, `IsDisabled()` must return `false`.
3. **Visibility:** Its associated `render.Object` must exist, and its `ComputedStyle().Display` must not be `DisplayNone`.
4. **Scope:** The node must be within the currently active focus scope.

## ⌨️ Navigation

### Tab Navigation
The `focus.Manager` provides `Next()` and `Previous()` methods to navigate through focusable elements in DOM tree order.

### Spatial Navigation
The `focus/spatial` sub-package provides directional navigation (Up, Down, Left, Right). It uses the physical coordinates of the elements (retrieved via `RenderObject().Fragment()`) to determine the best candidate in the requested direction.

## 💍 Focus Rings

Painters use `Manager.IsFocusVisible(node)` to decide whether to draw a focus indicator. This follows the web's `:focus-visible` heuristic, returning `true` only if focus was acquired via the keyboard.
