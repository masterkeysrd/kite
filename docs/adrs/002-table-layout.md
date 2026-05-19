# ADR 002: Table Layout and Fault Tolerance

## Status
Accepted

## Context
We need to implement a table layout engine to support structured data grids. Tables introduce unique layout complexities:
1. **Grid Sizing:** Column widths depend on the intrinsic sizes of cells across *all* rows.
2. **Spanning:** Cells can span multiple columns (`colspan`) or rows (`rowspan`), requiring complex distribution of intrinsic sizes during measurement.
3. **Grouping:** Tables require structural sections (`thead`, `tbody`, `tfoot`) to support features like sticky headers, cohesive styling, and layout ordering, while keeping column sizing synchronized across all sections.
4. **Malformed Structures:** In HTML, users frequently author invalid tables (e.g., a `<td>` directly inside a `<table>` without a `<tr>`). If our engine panics on these structures, it creates a poor developer experience.

## Decision
We will implement a two-pass table layout algorithm within the bounds of our LayoutNG-inspired architecture and the Unified Render Box rule, closely matching CSS table specifications.

### 1. Unified Render Box
We will **not** create dedicated `render.Table` or `render.TableRow` objects. The render tree will continue to use `render.Box`. The layout engine will delegate to specific algorithms purely based on `ComputedStyle().Display` (e.g., `DisplayTable`, `DisplayTableRowGroup`, `DisplayTableRow`, `DisplayTableCell`).

### 2. Table Section Grouping
The framework will explicitly support `display: table-header-group` (`thead`), `display: table-row-group` (`tbody`), and `display: table-footer-group` (`tfoot`). 
- **Layout Ordering:** Regardless of DOM order, the `TableAlgorithm` will always position header groups first, followed by row groups, and finally footer groups.
- **Synchronized Sizing:** The grid sizing pass evaluates all cells across all sections simultaneously to calculate global column constraints.

### 3. Two-Pass Layout Algorithm (`layout/table.go`)
- **Pass 1 (Measurement/Grid Sizing):** The `TableAlgorithm` sorts sections, then iterates through all rows and cells to compute intrinsic column widths. It handles `ColSpan` and `RowSpan` by distributing a cell's `MinMaxSizes` across the spanned columns/rows.
- **Pass 2 (Fragment Generation):** Once column widths are resolved, the table passes these fixed constraints down through the sections to the `TableRowAlgorithm`, which positions the cells horizontally.

### 4. Fault Tolerance via Anonymous Wrappers
To handle malformed DOM structures without polluting the logical DOM or the core engine sync phase, we will rely on **Layout-Driven Fault Tolerance**:
- **Anonymous Row Groups:** If the table algorithm encounters a `DisplayTableRow` child that is not wrapped in a group, it generates a virtual `Anonymous Table Row Group` to hold it.
- **Anonymous Rows:** If the algorithm encounters a child that is *not* a `DisplayTableRow` (e.g., a direct `DisplayTableCell` or text), it groups that child into a virtual **Anonymous Row** inside the current section.
- **Benefit:** The `engine.syncRenderTree` remains completely ignorant of CSS display types, preserving strict separation of concerns. The logical DOM remains true to the developer's code, while the Fragment Tree safely corrects the layout.

## Consequences

### Positive
- **Architectural Consistency:** Adheres to the Unified Render Box rule.
- **Robustness:** Anonymous wrappers prevent engine panics from user error while keeping the logical DOM pristine.
- **Web Parity:** Full support for `thead`, `tbody`, `tfoot` simplifies data grids and sticky header requirements.

### Negative / Trade-offs
- **Complexity in TableAlgorithm:** The layout algorithm must manage sorting and virtual groupings on the fly, complicating iteration over children.
- **Performance:** Two-pass layout is inherently slower than single-pass block layout. Strict caching of the measurement pass (Grid Sizing) will be crucial for maintaining 60FPS.
