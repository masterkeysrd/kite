# Kite Layout Engine

The `layout` package implements Kite's high-performance, LayoutNG-inspired layout engine. Its primary responsibility is to take a tree of style-resolved nodes and compute their exact physical dimensions and positions, outputting an **Immutable Fragment Tree**.

## Intent and Philosophy

Unlike traditional layout engines that mutate the state of logical DOM nodes or render objects during measurement, this engine adheres to strict separation of constraints and immutability:
1. **No Mutations During Layout:** Layout algorithms never modify the incoming `Node` or its styles.
2. **Immutable Outputs:** The result of a layout operation is a `Fragment`. Once created, a `Fragment` is strictly read-only.
3. **Stateless Positioning:** A `Fragment` does not know its absolute position on the screen, nor does it contain a pointer to its parent. Its position (offset) is stored in the parent's `FragmentLink`. This allows identical fragments to be cached and reused in entirely different positions.
4. **Cache-Driven:** If a node's constraints (available space) and styles have not changed, the engine instantly returns the cached `Fragment`, skipping the subtree walk.

## Core Data Structures

*   **`Node`**: An interface representing the input to the layout engine (implemented by `render.Box` and `render.Text`). It provides access to computed styles and an iterator over its children.
*   **`ConstraintSpace`**: The input parameters for a layout operation. It defines the available size (width/height) and specifies whether those dimensions are fixed or flexible (e.g., shrink-to-fit).
*   **`Fragment`**: The physical, read-only output. It contains a `layout.Size` and a slice of `FragmentLink`s (children fragments and their X/Y offsets).
*   **`BoxFragmentBuilder`**: A mutable builder used by algorithms to accumulate inline size, block size, and child fragments. Calling `builder.ToFragment()` seals it into an immutable `Fragment`.
*   **`MinMaxSizes`**: Represents the intrinsic minimum (shrink-wrapped) and maximum (fully expanded) sizes of a node before flex or percentage rules are applied.

## The Layout Flow

Every layout operation broadly follows these steps:
1.  **Cache Check:** If the `Node` is clean and the `ConstraintSpace` exactly matches the previous run, return the cached `Fragment`.
2.  **Measurement Pass (Intrinsic Sizing):** The algorithm calculates the minimum and maximum sizes of its children to determine how to distribute space (crucial for Flexbox and Auto-sized blocks).
3.  **Constraint Resolution:** The algorithm resolves percentage sizes and auto margins against the `ConstraintSpace`.
4.  **Child Layout:** The algorithm loops over its children, creates a new `ConstraintSpace` for each, and calls `Layout()` on them.
5.  **Assembly:** Children fragments are placed at specific X/Y coordinates via `builder.AddChild()`.
6.  **Finalization:** The builder is sealed into a `Fragment`, cached on the `Node`, and returned.

---

## Layout Algorithms

The engine routes nodes to specific formatting context algorithms based on their `style.Display` property.

### Block Algorithm (`block.go`)

The Block Formatting Context (BFC) algorithm implements standard vertical stacking, the default behavior for `display: block` elements.

#### Detailed Flow
1. **Width Resolution**: The algorithm first resolves the block's inline size (width). If the width is `Auto` and the `ConstraintSpace` permits, the block stretches to fill the available inline space, minus any margins, borders, or padding.
2. **Child Traversal**: It iterates over its children in DOM sequence.
3. **Child Constraint Generation**: For each child, a new `ConstraintSpace` is built. The child's available width is locked to the parent's resolved content width.
4. **Layout & Positioning**: The child's `Layout()` method is invoked, yielding an immutable `Fragment`. The Block algorithm calculates the child's `X` offset (handling margins and text alignment) and `Y` offset (tracking the accumulated block height of all preceding siblings).
5. **Height Resolution**: After all children are laid out, the Block algorithm determines its own final block size (height). If the height is explicitly set, it uses that; otherwise, it shrink-wraps to the bottom edge of the last child (plus padding and borders).
6. **Margin Collapsing**: *(Note: Terminal UI margin collapsing rules are simpler than web CSS, often just resolving explicit cell gaps).*

### Flex Algorithm (`flex.go`)

The Flex Formatting Context handles 1D layouts with advanced alignment, wrapping, and distribution of space. It is designed to be agnostic of the `FlexDirection` (Row vs. Column) by operating on **Main** and **Cross** axes using a geometry abstraction (`flexGeometry`).

#### Detailed Flow
The flex algorithm is heavily optimized to avoid exponential ($O(n^2)$) layout times by strictly caching intermediate results across its two-pass system.

##### Pass 1: Intrinsic Measurement & Line Generation
1. **Item Collection**: All flex children are gathered. If `display: inline` children are found alongside blocks, they are bundled into an `AnonymousBlock` to participate as a single flex item.
2. **Order Sorting**: Items are reordered based on their CSS `order` property.
3. **Base Size Calculation**: The intrinsic `MinMaxSizes` of each item are measured. The item's `BaseSize` and `HypotheticalMainSize` (its size before any growing or shrinking) are calculated based on the available main-axis space.
4. **Line Breaking**: If `flex-wrap` is enabled, the algorithm accumulates items into logical `flexLine` groupings. When the sum of `HypotheticalMainSize`s exceeds the available main space, a new line is started.

##### Pass 2: Flexible Length Resolution
For each logical line, the engine must distribute the remaining free space (or shrink items if they overflow):
1. **Determine Free Space**: `Available Space - Sum(HypotheticalMainSize)`.
2. **Freeze Inflexible Items**: Items with `flex-grow: 0` (when there's extra space) or `flex-shrink: 0` (when overflowing) are "frozen" at their hypothetical sizes.
3. **Distribute Space**: The remaining space is distributed proportionally among the unfrozen items based on their `flex-grow` or `flex-shrink` factors.
4. **Min/Max Clamping Loop**: If a stretched/shrunk item hits its `min-width` or `max-width`, it is clamped, marked as "frozen", and the algorithm loops again to redistribute the leftover space among the remaining unfrozen items.

##### Final Layout & Alignment
1. **Forced Child Layout**: Every child is laid out again. This time, the parent builds a `ConstraintSpace` with strict, fixed dimensions based on the resolved flexible sizes. The child *must* obey these constraints, returning its final immutable `Fragment`.
2. **Main-Axis Alignment**: Using `justify-content` (Start, End, Center, Space-Between, etc.), the algorithm calculates the exact `X` (or `Y` for columns) coordinate for each item on the line.
3. **Cross-Axis Alignment**: Using `align-items` and `align-self`, the algorithm calculates the perpendicular coordinate for each item, shifting them up/down (or left/right) within the height/width of their specific `flexLine`.

### Inline Algorithm (`inline.go`)

The Inline Formatting Context (IFC) is highly specialized for text handling, operating very differently from Block and Flex contexts. It uses a flat-list architecture to maximize memory locality and line-breaking performance.

#### Detailed Flow
1. **Anonymous Wrapping**: The engine enforces that inline elements cannot be direct children of Flex or standard Block contexts without an intermediary. The parent engine creates an `AnonymousBlock`, which then delegates entirely to the Inline algorithm.
2. **Pre-Layout Collection**: Instead of walking a deep tree recursively during layout, the algorithm flattens all inline children (`<span>`, `<text>`, `<icon>`) into an array of `InlineItem`s.
3. **Whitespace Collapsing**: Based on `white-space` rules, consecutive spaces, tabs, and newlines are collapsed or preserved.
4. **Shaping & Measurement**: Text runs are sent to the `uniseg` shaper. The shaper determines exactly how many terminal cells each grapheme cluster requires. The results are heavily cached to prevent recalculating static text on every frame.
5. **Line Breaking**: The algorithm iterates through the `InlineItem`s, accumulating their cell widths. 
   - When the accumulated width exceeds the `ConstraintSpace.AvailableSize`, the algorithm looks backwards for the nearest valid "soft break" opportunity (usually a space or hyphen).
   - The line is split at that index.
6. **Line Box Construction**: For each broken line, a `LineBox` fragment is generated. The text elements inside are aligned vertically (handling varying font heights or inline-block vertical alignment, like `baseline` or `middle`). 
7. **Bidi Reordering (UAX#9)**: If RTL (Right-to-Left) text is present, the physical order of the fragments inside the `LineBox` is reversed or shuffled according to the Unicode Bidirectional Algorithm before final fragment assembly.

### List Algorithm (`list.go`)

The List Formatting Context (LFC) implements the layout for `display: list-item` elements. It specializes in rendering markers (bullets, numbers) using a virtual, layout-driven strategy that avoids phantom render objects.

#### Detailed Flow
1. **Virtual Marker Synthesis**: Unlike web engines that inject anonymous boxes for list markers, the `ListAlgorithm` synthesizes the marker as a transient, physical text fragment directly during the layout phase. This adheres to the "Unified Render Box" principle and prevents architectural leaks.
2. **Two-Column Row Layout**: The algorithm formats the list item as a logical two-column row:
   - **Column 1**: Contains the synthesized marker fragment (e.g., "• " or "1. "), measured to its exact cell width using the text shaper.
   - **Column 2**: Contains the actual child content of the list item, laid out using standard Block layout rules.
3. **Ordinal Calculation**: For numbered lists (`ListStyleDecimal`), the algorithm computes the ordinal by walking the previous logical siblings of the node to count consecutive `display: list-item` elements.
4. **Content Wrapping**: Multi-line content in Column 2 is constrained to the remaining available inline space, ensuring that text wraps cleanly adjacent to the marker rather than underneath it.

### Table Algorithm (`table.go`, `table_builder.go`)

The Table Formatting Context (`DisplayTable`) implements a rigorous two-pass layout algorithm to resolve intrinsic grid column dimensions and handle complex cell spanning (`ColSpan`, `RowSpan`). All mutable state for the two-pass process is managed by a dedicated `TableFragmentBuilder`.

#### `TableFragmentBuilder` (`table_builder.go`)

The `TableFragmentBuilder` is the central state-management object for a single table layout run. It is created by `TableAlgorithm` and threaded through to `TableSectionAlgorithm` and `TableRowAlgorithm`. Its responsibilities are:

1. **Section Grouping**: Collects `thead`, `tbody`, and `tfoot` children via `AddHeaderChild`, `AddBodyChild`, and `AddFooterChild`. Direct row or non-row children are wrapped into anonymous section and row nodes automatically.
2. **Grid Construction** (`BuildGrid`): Builds a logical 2D grid (`tableGrid`) by walking all rows and cells. The grid records each cell's `ColStart`, `ColSpan`, and `RowSpan`, resolving overlapping cells due to `RowSpan`. It also pre-computes per-column-junction border-overlap flags (see below).
3. **Column Sizing** (`DistributeSpan`): Populates `colMinMax []MinMaxSizes` — one entry per column. Single-span cells are measured first, then multi-span cells distribute any excess width proportionally across the spanned columns.
4. **Width Resolution** (`ResolveWidths`): Resolves the final per-column pixel widths (`colWidths`) given the table's total resolved inline size. The distributable budget is widened by the number of *actual* collapsed junctions (not all junctions unconditionally).
5. **Implicit Border-Collapse Math**: Provides `AdjustRowOffset` and `GetCellShift` to `TableSectionAlgorithm` and `TableRowAlgorithm` so they can apply the `-1` coordinate adjustments for overlapping borders.

#### Border-Collapse Coordinate Model

Kite uses **implicit border collapse** for table elements. The key rules are:

**Horizontal (X-axis) — cell-to-cell:**
- Cells are placed at `X = 0` within the row (no left-border inset). This ensures that a cell's left border physically overlaps with the row's own left border at the same terminal column, allowing the paint engine's junction resolver to merge them.
- When a cell at column `j` has a right border **and** the cell at column `j+1` has a left border, `GetCellShift` returns `1`. The calling row algorithm subtracts this from the next cell's X offset, causing the two borders to share the same terminal column.
- For **spanning cells** (`ColSpan > 1`), each internal collapsed junction within the span is subtracted from the cell's total width (`cellWidth--` for each `ColJunctionOverlap[j] == true` in the span range). The junction column does not exist inside a spanning cell.

**Vertical (Y-axis) — row-to-row:**
- Cells are placed at `Y = 0` within the row (the `BoxFragmentBuilder`'s automatic `border.Top` inset is explicitly reset to `0`). This ensures that a cell's top border shares the same terminal row as the row's own top border.
- The row's block size is set to `maxCellHeight` only (no extra `border.Bottom` added). The row's bottom border is therefore drawn at `Y = maxCellHeight - 1`, coinciding exactly with the cells' bottom borders.
- When the previous row has a bottom border **and** the next row has a top border, `AdjustRowOffset` returns `-1`. The section algorithm subtracts this from the next row's Y offset, making the two rows share that single border terminal row.
- `TableFragmentBuilder` is initialized with `lastRowBorderBottom = table.border.Edges.Top` so the very first row also collapses against the table's own top border when applicable.

**Table-edge overlaps (X-axis):**
- `tableGrid.ColJunctionOverlap[j]` is `true` when any row has both a right-bordered cell ending at column `j` and a left-bordered cell starting at column `j+1`.
- `tableGrid.LeftEdgeHasOverlap` / `RightEdgeHasOverlap` are `true` when the table's own left/right border intersects with the first/last column's cell borders.
- These flags drive two things:
  1. **Table width**: `tableMinMax` is reduced by 1 for each actual overlap (not unconditionally for every junction).
  2. **Section placement**: when `LeftEdgeHasOverlap` is true, sections are placed at `X = padding.Left` (instead of `X = border.Left + padding.Left`), and `childAvailWidth` is expanded by `border.Left` (and `border.Right` for `RightEdgeHasOverlap`), so that column 0's left border shares the table's left border column.

#### Detailed Flow

##### Pass 1: Grid Sizing & Measurement
1. **Section Grouping**: `TableAlgorithm` walks its children and calls the appropriate `builder.Add*Child` method to populate the three section buckets (header, bodies, footer). Anonymous wrappers are created for stray rows or cells.
2. **Grid Construction** (`BuildGrid`): Flattens all rows from all sections and builds `tableGrid`, including `ColJunctionOverlap`, `LeftEdgeHasOverlap`, and `RightEdgeHasOverlap`.
3. **Column Sizing**: Single-span cells are measured first (`IntrinsicMinMaxSizes`), then multi-span cells run `DistributeSpan` to push any excess width into the spanned columns.
4. **Table Width Resolution**: `tableMinMax` sums the per-column min/max values, then subtracts 1 for each genuinely collapsed border (junctions and table edges). `parentDecorX` (table borders + padding) is added back.
5. **Column Width Distribution** (`ResolveWidths`): The distributable pixel budget is `resolvedInlineSize - parentDecorX + overlaps_added_back`. This is distributed proportionally to max-content sizes, then evenly for any remainder.

##### Pass 2: Row & Cell Layout
1. **Section Invocation**: For each section (in header → body → footer order), `TableAlgorithm` builds a `ConstraintSpace` whose width is `childAvailWidthBase` (accounting for edge overlaps) and passes it to `TableSectionAlgorithm`, along with the relevant slice of `tableGrid.Rows` and the resolved `colWidths`.
2. **Row Invocation**: `TableSectionAlgorithm` iterates over its rows, calling `TableRowAlgorithm` for each. Before placing each row, it calls `builder.AdjustRowOffset` to apply the `-1` vertical collapse shift where row borders overlap.
3. **Cell Layout**: `TableRowAlgorithm` resets its builder's `currentBlockOffset` to `0` (overriding the automatic border inset), then iterates over the row's cells from `tableGrid`. For each cell it:
   a. Sums `colWidths` across the spanned columns, then subtracts 1 for each internal collapsed junction.
   b. Calls `builder.GetCellShift` to obtain the horizontal `-1` collapse shift.
   c. Lays the cell out as a standard `BlockAlgorithm` within a fixed-width `ConstraintSpace`.
   d. Places the cell at the adjusted `(X, 0)` offset.
4. **Row Height**: The row's height is set to `maxCellHeight + padding.Bottom` (no extra `border.Bottom`). The table aggregates row heights via `AdjustRowOffset` to arrive at its final block size.
