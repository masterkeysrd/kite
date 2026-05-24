# Task: Logical Text Selection (DOM)

## Description
Implement the logical `Selection` and `Range` APIs on the `dom.Document` to represent text highlight state, mirroring standard browser APIs.

## Requirements
- Introduce `dom.Range` representing a text selection segment:
  - `StartContainer *dom.Node` (should be restricted or typically a `dom.Text`)
  - `StartOffset int` (Rune index)
  - `EndContainer *dom.Node`
  - `EndOffset int`
- Implement `dom.Selection`:
  - Maintained by `dom.Document` via `doc.Selection()`.
  - Can hold at least one `dom.Range`.
  - Methods: `AddRange()`, `RemoveAllRanges()`, `String()` (extracts combined text from the ranges).
- Dispatch `event.TypeSelectionChange` on the document whenever the active range changes.
- Ensure offsets are strictly validated against `utf8.RuneCountInString` of the underlying text nodes.

## Tests
- Unit tests verifying range boundaries, expanding ranges, and retrieving the string value correctly from across multiple sibling `dom.Text` nodes.
- Test that `SelectionChange` events fire correctly.