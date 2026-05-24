# Task: CSS Grid Style API

## Description
Implement the Style definitions and API sugar for defining CSS Grids.

## Requirements
- In `style/enums.go`: Add `DisplayGrid` to the `Display` enum.
- In `style/types.go` or a new `style/grid.go`:
  - Define `GridTrackSize` struct (supporting Cells, Percentage, Fractional, Auto).
  - Define `GridPlacement` struct (representing column/row start and span).
- Add `GridTemplateColumns`, `GridTemplateRows`, `GridColumnGap`, `GridRowGap`, `GridColumn`, and `GridRow` to `style.Style` and `style.Computed`.
- **API Sugar**:
  - Implement a `Repeat(count int, sizes ...GridTrackSize)` function that returns a flattened slice of `GridTrackSize`.
  - Implement helpers like `Fr(float)`, `Cells(int)`, `Auto()`.

## Tests
- Write unit tests in `style_test.go` ensuring `Repeat(3, Fr(1))` outputs `[Fr(1), Fr(1), Fr(1)]`.
- Ensure the Style Resolver correctly propagates the new Grid properties from raw styles to computed styles.