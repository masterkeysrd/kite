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
