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

When the `engine` triggers the Paint Phase, it requests a new `FrameBuffer` from the backend and passes it, along with the root `layout.Fragment`, to the `PaintEngine`.

### 1. Tree Traversal
The `PaintEngine` performs a depth-first traversal of the `Fragment` tree. Because Fragments are immutable and fully resolved, this walk is highly cache-friendly.

### 2. Coordinate Translation
For every `FragmentLink` (which joins a parent to a child), the engine takes the parent's absolute `(X, Y)` position and adds the child's local offset. This absolute coordinate is passed down to the child's paint routine.

### 3. Layered Drawing
For every fragment, the paint engine draws layers in a specific order (similar to the CSS stacking context):
1.  **Background:** Fills the fragment's bounding box with the `ComputedStyle.Background` color.
2.  **Borders:** Draws box-drawing characters (lines, rounded corners) around the perimeter based on `ComputedStyle.Border`.
3.  **Content (Text):** If the fragment represents a text node, it writes the shaped runes into the buffer, applying text colors and decorators.

### 4. Clipping & Overflow
Before drawing a subtree, the paint engine checks the `ComputedStyle.Overflow` properties of the current node. 
If overflow is hidden or scrollable, the paint engine pushes a new **Clip Rect** onto the `Surface`. Any child fragment that attempts to draw cells outside this clip rect will have those cells silently discarded, ensuring clean UI boundaries.

### 5. Overlays and Z-Index
Overlays (like dropdowns or modals) bypass normal DOM flow. The `engine` paints the main document root first, and then explicitly invokes the `PaintEngine` on the `Fragment` tree of each overlay. Because the `FrameBuffer` is shared, overlays simply overwrite the cells painted by the main document, achieving standard z-index stacking.