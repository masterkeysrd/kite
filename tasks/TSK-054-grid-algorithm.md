# Task: Grid Layout Algorithm

## Description
Implement the `Algorithm` interface for CSS Grid, coordinating the `GridBuilder` and producing the final `ImmutableFragment` tree.

## Requirements
- Create `layout/grid.go` implementing `layout.Algorithm`.
- When `DisplayGrid` is encountered, route the node to `GridAlgorithm`.
- **Algorithm Flow**:
  - Initialize the `GridBuilder`.
  - Push all children into the builder and run Auto-Placement.
  - Execute the Measure Pass: for `auto` tracks, ask the children for their `ComputeMinMaxSizes()` and expand the track widths/heights accordingly.
  - Execute the Fractional Pass: Distribute remaining `ContainerSpace` to `fr` tracks.
  - Execute Layout Pass: Iterate through the placed items, build a specific `ConstraintSpace` for each cell (accounting for `gap`), and call `layout.Compute()`.
  - Pack the resulting child fragments into a `BoxFragmentBuilder` and return.

## Tests
- Create regression tests in `layout/regression_test.go` or `grid_test.go`.
- Verify a 2x2 grid with `1fr` tracks equally divides the space.
- Verify that an `auto` track correctly expands to the size of its largest child fragment.