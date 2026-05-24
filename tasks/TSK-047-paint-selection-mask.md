# Task: Paint Masking for Text Selection

## Description
Implement the "Push Model" for text selection by translating logical DOM bounds into physical rectangles, and applying them as a mask during the paint phase to maintain 60FPS caching.

## Requirements
- **Selection Resolution Phase**:
  - In `render.View` (or a `selection` manager), immediately before paint, map the active `dom.Selection` to physical screen bounds.
  - Utilize `layout.ScrolledAbsoluteBounds` to find the rectangles encompassing the selected text fragments.
  - Generate a `[]paint.SelectionRect` slice representing these bounds.
- **Paint Context Update**:
  - Inject the `[]paint.SelectionRect` slice into the `paint.Context` or the paint engine state for the frame.
- **Paint Overlay Masking**:
  - Update `paint/engine.go` so that when a terminal cell is drawn (or as a final pass), if its `(x,y)` coordinate intersects a `SelectionRect`, its background and foreground colors are inverted (or replaced by specific selection theme colors).
- **Styling Hooks**:
  - (Optional for this task, but prepare) Consider reading `SelectionBackgroundColor` from the `computed.Style` if we want customizable highlight colors.

## Tests
- Headless test rendering a document with a pre-set selection range and validating the output ANSI snapshot shows inverted/highlighted characters.
- Ensure layout fragments remain cached (dirty flags untouched) when the selection bounds change.