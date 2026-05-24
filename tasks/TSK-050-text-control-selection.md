# Task: Text Control Local Selection

## Description
Implement local selection mechanics for `textControlBase` (shared by `<input>` and `<textarea>`) and bridge it to the global selection for rendering.

## Requirements
- **State Management**:
  - Add `SelectionStart` and `SelectionEnd` to `textControlBase`.
  - Add API `SetSelectionRange(start, end int)`.
  - Ensure the existing `caretIndex` logic is tied to `SelectionEnd` when a selection is active.
- **Visual Bridging**:
  - When `textControlBase` performs its Sync/Render phase, if `SelectionStart != SelectionEnd`, programmatically create a `dom.Range` targeting the hidden `dom.Text` nodes inside its UA Shadow Subtree.
  - Push this range to the `dom.Document`'s `Selection` to reuse the global paint masking system.
  - If `SelectionStart == SelectionEnd`, clear the document selection if this element owns it.
- **Keyboard Interaction**:
  - Modify the existing arrow key handlers. If `Shift` is held, update `SelectionEnd` while keeping `SelectionStart` anchored.
- **Mouse Interaction**:
  - `MouseDown`: Set both start and end to the hit-tested rune index.
  - `MouseMove`: If dragging, update `SelectionEnd`.

## Tests
- Simulate Shift+Arrow keys in `testenv` and assert `SelectionStart` and `SelectionEnd` values.
- Verify that a `dom.Range` is correctly populated on the document when an input has an active selection.