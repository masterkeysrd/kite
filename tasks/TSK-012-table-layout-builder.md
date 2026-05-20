# Task: Table Layout Builder Pattern

## 1. Objective
Refactor the mutable state management of the Table Layout engine out of `TableAlgorithm` and into a dedicated `TableFragmentBuilder` (and/or `GridSizingBuilder`). This builder will handle section grouping, column matrix math, and implicit border overlapping calculations.

## 2. Design & Requirements

### Builder Capabilities (`layout/table_builder.go` or within `table.go`)
- **Grouping:** The builder must expose methods like `AddHeaderChild`, `AddBodyChild`, `AddFooterChild` (and handle anonymous row grouping) to collect the nodes during the initial walk.
- **Grid Sizing State:** It must manage an internal slice of `MinMaxSizes` representing the columns. It needs a method `DistributeSpan(cell Node, colIndex int, colSpan int)` to handle the complex math of stretching minimum widths across multiple columns.
- **Overlap Math:** To fulfill the implicit border collapse requirement (`TSK-011`), the builder must evaluate the `Border.Edges` of adjacent cells/rows when they are added to the layout. It adjusts the `Point.X` and `Point.Y` coordinates by `-1` where borders intersect, ensuring the final fragment tree overlaps correctly for the paint engine's post-processor.
- **Output:** It must have a `ToFragment()` method that compiles all this internal state into the standard, immutable `*layout.Fragment`.

### `TableAlgorithm` Cleanup
- The core algorithm should instantiate this builder.
- Pass 1 (Sizing) iterates over the builder's grouped rows to populate the builder's column constraints.
- Pass 2 (Layout) iterates over the groups, calling `builder.AddRowFragment(...)`.

## 3. Implementation Steps
1. Create the `TableFragmentBuilder` struct in the `layout` package.
2. Move the matrix distribution logic (for spans) into the builder.
3. Move the coordinate tracking logic (including the `-1` border overlap from TSK-011) into the builder's `AddCell` or `AddRow` methods.
4. Refactor `TableAlgorithm.Layout()` to rely entirely on this builder.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test the builder in isolation: Verify `DistributeSpan` correctly modifies the internal column widths array.
- [ ] Test that adding two cells side-by-side where both have `Border.Edges.Right/Left == true` results in the second cell's `X` coordinate being offset by `-1`.

### 4.2. Integration Tests
- [ ] Verify that `TableAlgorithm` still correctly outputs an immutable fragment tree that matches the previous raw implementation.

### 4.5. Documentation
- [ ] Add `doc.go` comments to the new builder struct explaining its role in the two-pass algorithm.
