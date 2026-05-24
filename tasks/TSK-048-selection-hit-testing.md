# Task: User Interaction and Hit-Testing for Selection

## Description
Enable mouse dragging to select text within the terminal, mapping physical coordinates back to logical text offsets.

## Requirements
- Attach a default internal listener for `MouseDown`, `MouseMove`, and `MouseUp` on the `document` (or the highest level window surface) to track dragging intent.
- When dragging starts on a `dom.Text` node (or its wrapper element):
  - Find the corresponding `layout.Fragment`.
  - Map the `event.X, event.Y` back to the exact rune offset.
  - *Math requirement*: Iterate over the fragment's `Text []text.Cluster`, summing `CellWidth` to find which rune index the mouse is hovering over.
- Continuously update the `dom.Selection`'s `EndOffset` during `MouseMove`.
- Clear the selection if the user clicks cleanly without dragging.

## Tests
- Simulate mouse down, drag, and mouse up events in the headless `testenv`.
- Assert that `doc.Selection()` correctly reflects the bounds of the text dragged over.