# Task: Text Control Clipboard Mechanics (Copy/Cut/Paste)

## Description
Implement local interceptors for clipboard events within `textControlBase` to allow copy, cut, and paste functionality in inputs and textareas.

## Requirements
- Attach listeners for `event.TypeCopy`, `event.TypeCut`, and `event.TypePaste` on the `textControlBase` host element.
- **Copy**:
  - If `SelectionStart != SelectionEnd`, extract the selected substring from `.Value()`.
  - Write it to `event.ClipboardData`.
  - Call `event.PreventDefault()`.
- **Cut**:
  - Same as Copy, but additionally delete the substring from the internal value.
  - Collapse the selection start/end to the deletion point.
  - Dispatch a `TypeInput` event.
- **Paste**:
  - Read `event.ClipboardData`.
  - Delete any actively selected text.
  - Insert the pasted string at the caret/SelectionStart index.
  - Move the caret to the end of the pasted text.
  - Dispatch a `TypeInput` event.
  - Call `event.PreventDefault()`.

## Tests
- Simulate a `TypeCopy` event on a focused input with selected text; verify the event data is populated.
- Simulate a `TypePaste` event on an input; verify the value string updates correctly and the caret moves.
- Verify `TypeCut` modifies both the clipboard data and the element's value.