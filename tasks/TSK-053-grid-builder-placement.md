# Task: Grid Builder and Auto-Placement

## Description
Implement the core mathematical model and auto-placement algorithm for the Grid layout in the `layout` package.

## Requirements
- Create `layout/grid_builder.go`.
- Define a structure to hold the 2D matrix of grid cells and track dimension state.
- **Track Sizing**: Implement the logic to resolve fixed and percentage track sizes against the `ContainerSpace`.
- **Auto-Placement**:
  - Implement a method that accepts a list of layout nodes.
  - Place items with explicit `GridColumn`/`GridRow` first.
  - Implement a cursor algorithm `(x, y)` that advances through the matrix to place remaining items into the first available empty space that fits their required span.

## Tests
- Write pure unit tests for the builder (without invoking actual UI elements).
- Provide a matrix of items with mixed explicit and implicit placements, and assert their final `(x, y)` coordinates in the builder state.