# ADR 002: Table Layout and Fault Tolerance

## Status
Accepted

## Context
We need to implement a table layout engine to support structured data grids (`display: table`, `table-row`, `table-cell`). Tables introduce unique layout complexities:
1. **Grid Sizing:** Column widths depend on the intrinsic sizes of cells across *all* rows.
2. **Spanning:** Cells can span multiple columns (`colspan`) or rows (`rowspan`), requiring complex distribution of intrinsic sizes during measurement.
3. **Malformed Structures:** In HTML, users frequently author invalid tables (e.g., a `<td>` directly inside a `<table>` without a `<tr>`). If our engine panics on these structures, it creates a poor developer experience.

## Decision
We will implement a two-pass table layout algorithm within the bounds of our LayoutNG-inspired architecture and the Unified Render Box rule.

### 1. Unified Render Box
We will **not** create dedicated `render.Table` or `render.TableRow` objects. The render tree will continue to use `render.Box`. The layout engine will delegate to `TableAlgorithm`, `TableRowAlgorithm`, or `TableCellAlgorithm` purely based on `ComputedStyle().Display`.

### 2. Two-Pass Layout Algorithm (`layout/table.go`)
- **Pass 1 (Measurement/Grid Sizing):** The `TableAlgorithm` iterates through all rows and cells to compute intrinsic column widths. It handles `ColSpan` and `RowSpan` by distributing a cell's `MinMaxSizes` across the spanned columns/rows.
- **Pass 2 (Fragment Generation):** Once column widths are resolved, the table passes these fixed constraints down to the `TableRowAlgorithm`, which positions the cells horizontally.

### 3. Fault Tolerance via Anonymous Layout Rows
To handle malformed DOM structures without polluting the logical DOM or the core engine sync phase:
- We will rely on **Layout-Driven Fault Tolerance**, similar to how Inline Formatting Contexts (IFC) group contiguous inline elements into anonymous blocks.
- When the `TableAlgorithm` iterates its children, if it encounters a child that is *not* a `DisplayTableRow` (e.g., a direct `DisplayTableCell` or text), it groups that child (and any contiguous non-row siblings) into an internal, virtual **Anonymous Row**.
- It then processes this virtual row through the row measurement and layout logic.
- **Benefit:** The `engine.syncRenderTree` remains completely ignorant of CSS display types, preserving strict separation of concerns. The logical DOM and physical Render Tree remain true to the developer's code, while the resulting Fragment Tree is safely corrected.

## Consequences

### Positive
- **Architectural Consistency:** Adheres to the Unified Render Box rule.
- **Robustness:** Anonymous wrappers prevent engine panics from user error while keeping the logical DOM pristine.
- **Feature Completeness:** Explicit support for `colspan` and `rowspan` enables rich TUI data grids.

### Negative / Trade-offs
- **Complexity in TableAlgorithm:** The layout algorithm is slightly more complex because it must manage virtual row groupings on the fly, rather than iterating a guaranteed clean render tree.
- **Performance:** Two-pass layout is inherently slower than single-pass block layout. Strict caching of the measurement pass (Grid Sizing) will be crucial for maintaining 60FPS.
