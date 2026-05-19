# Kite Paint Package

The `paint` package is the final stage of the visual pipeline in Kite. It is responsible for taking the immutable physical geometry (the `layout.Fragment` tree) and drawing it into a 2D grid of terminal cells (the `FrameBuffer`).

## Intent and Philosophy

1. **Dumb and Fast:** The paint phase does absolutely zero layout math, size resolution, or styling inheritance. All of that is handled by the `layout` and `style` packages. The paint package simply reads coordinates and colors, and applies them to a grid.
2. **Absolute Coordinates:** Layout fragments store *relative* offsets (e.g., "I am 5 cells below my parent"). The paint engine maintains a running transform matrix (a global X/Y offset) as it walks the tree, converting these relative positions into absolute terminal coordinates.
3. **Clipping:** Terminals don't naturally support "hiding" text that overflows a box. The paint package implements software clipping regions (respecting `overflow: hidden`) to prevent child elements from drawing outside their parent's boundaries.

## Core Data Structures

*   **`PaintEngine`**: The orchestrator of the paint walk.
*   **`FrameBuffer`**: A 2D array representing the terminal screen. It implements the `Surface` interface.
*   **`Cell`**: Represents a single character on the screen. It holds the `rune`, foreground/background colors, and visual decorators (bold, italic, underline, reverse).
*   **`Surface`**: An interface abstracting the drawing target, allowing the engine to write strings, set cells, and define clipping boundaries.

## The Paint Flow

When the `engine` triggers the Paint Phase, it requests a new `FrameBuffer` from the backend and passes it, along with the root `layout.Fragment`, to the `PaintEngine`. The paint flow consists of two primary phases:

### Phase 1: Recursive Fragment Paint
The `PaintEngine` performs a depth-first traversal of the `Fragment` tree. Because Fragments are immutable and fully resolved, this walk is highly cache-friendly.

#### 1. Coordinate Translation
For every `FragmentLink` (which joins a parent to a child), the engine takes the parent's absolute `(X, Y)` position and adds the child's local offset. This absolute coordinate is passed down to the child's paint routine.

#### 2. Layered Drawing
For every fragment, the paint engine draws layers in a specific order (similar to the CSS stacking context):
1.  **Background:** Fills the fragment's bounding box with the `ComputedStyle.Background` color.
2.  **Borders:** Draws box-drawing characters (lines, rounded corners) around the perimeter based on `ComputedStyle.Border`. Every cell drawn as a border is marked with the `FlagIsBorder` attribute.
3.  **Content (Text):** If the fragment represents a text node, it writes the shaped runes into the buffer, applying text colors and decorators.

#### 3. Clipping & Overflow
Before drawing a subtree, the paint engine checks the `ComputedStyle.Overflow` properties of the current node. 
If overflow is hidden or scrollable, the paint engine pushes a new **Clip Rect** onto the `Surface`. Any child fragment that attempts to draw cells outside this clip rect will have those cells silently discarded, ensuring clean UI boundaries.

### Phase 2: Screen-Space Border Resolution
After the entire fragment tree has been painted, the `PaintEngine` performs a global post-processing pass over the `FrameBuffer`. 

This pass scans every cell marked with `FlagIsBorder`. For each border cell, it inspects its four cardinal neighbors (up, down, left, right). If a neighbor also has `FlagIsBorder`, they are considered connected. The engine then replaces the original border character with the correct Unicode box-drawing junction (e.g., `┬`, `┴`, `┼`, `╬`) that matches the style of the current cell.

This two-phase approach allows disparate elements—like a sidebar and a header, or adjacent table cells—to seamlessly "snap" together with perfect junctions without requiring manual configuration or awareness of their neighbors during the layout phase.

### 5. Overlays and Z-Index
Overlays (like dropdowns or modals) bypass normal DOM flow. The `engine` paints the main document root first, and then explicitly invokes the `PaintEngine` on the `Fragment` tree of each overlay. Because the `FrameBuffer` is shared, overlays simply overwrite the cells painted by the main document, achieving standard z-index stacking.