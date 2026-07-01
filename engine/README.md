# Kite Engine Package

The `engine` package acts as the central nervous system of the Kite framework. It orchestrates the entire application lifecycle, binding the logical DOM, event system, and the rendering pipelines (`style`, `layout`, `paint`) into a cohesive, high-performance frame loop.

## Intent and Philosophy

1. **Single-Threaded Render Tree:** To maximize performance and eliminate lock contention, all DOM mutations, style resolutions, and layout measurements occur strictly on the main thread.
2. **Concurrent User Space:** User-defined heavy operations (like database queries or network requests) run concurrently in a worker pool. They communicate with the main thread via asynchronous task queues.
3. **Phase-Driven Pipeline:** The engine never updates the screen randomly. It strictly batches updates into a discrete frame pipeline (inspired by browser internals) targeting a smooth 60FPS.
4. **Backend Agnosticism:** The engine communicates with terminals via a `backend.Backend` interface, making it trivial to swap between real terminal rendering (Ultraviolet) and mock backends for headless testing.

## Core Data Structures

*   **`Engine`**: The primary orchestrator. It holds the root `dom.Document`, the `render.RenderView`, the `style.Resolver`, the worker pool, and the event dispatcher.
*   **`Job`**: An interface for user-defined asynchronous work. Jobs execute off the main thread but return their completion callbacks safely back to the main thread.
*   **Task Queues**: The engine maintains separate queues for Microtasks (immediate, fast callbacks) and Macrotasks (budget-capped, lower priority work).

## The Frame Pipeline

When `engine.Run()` is called, it enters a continuous event loop. If the UI is marked dirty or a frame is explicitly requested, it executes the **Frame Pipeline**:

### 1. Input Coalescing
Immediately before the frame pipeline begins, the engine drains and coalesces the raw event buffer from the backend:
*   **High-Frequency Filtering**: Multiple `MouseMove` events in a single frame are discarded, keeping only the latest coordinate to reduce redundant layout work.
*   **Delta Aggregation**: Multiple `Wheel` events targeting the same element are summed into a single aggregate event with combined deltas.
*   **Synthesis**: Coalesced events are converted into structured DOM events and dispatched through the logical tree.

### 2. Task Draining
Before touching the UI, the engine processes pending state changes:
*   **Worker Results**: Collects finished `Job` callbacks and queues them as microtasks.
*   **Macrotasks**: Executes pending macrotasks (e.g., timers or deferred UI updates) up to a strict wall-clock/count budget to prevent frame drops.
*   **Microtasks**: Exhausts all microtasks (promises/callbacks) immediately.

### 3. Sync Phase (Pre-Layout)
Gated by the `NeedsSync` flag on the logical DOM. Before styling or layout begins, the engine ensures the visual tree matches the logical tree.
*   **Diffing & On-the-Fly Attachment**: The engine walks the DOM following `ChildNeedsSync` relay flags. It creates missing `render.Box`es for new elements, and destroys/unlinks `render.Box`es for elements removed from the DOM or set to `display: none`.
*   **Result**: A perfectly clean, 1:1 render tree is passed to the downstream algorithms.

### 4. Style Phase
Gated by `DirtyStyle` or `ChildNeedsStyle`. The engine calls `style.ResolveTree`. 
*   It applies CSS inheritance and user styles, pulling the sparse `RawStyle` from the logical DOM and outputting a fully resolved `style.Computed` object for each node in the render tree.

### 5. Layout Phase
Gated by `DirtyLayout`. The engine delegates to `render.LayoutPhase` (which utilizes the `layout` package's LayoutNG algorithms).
*   The engine provides the viewport `ConstraintSpace`.
*   The layout algorithms calculate intrinsic sizes, apply Flex/Block rules, and generate an immutable `Fragment` tree.

### 6. Paint Phase
Gated by `DirtyPaint`. The engine creates a new `paint.Surface` (a FrameBuffer) from the backend.
*   It walks the immutable `Fragment` tree, calculating absolute coordinates and pushing colors, borders, and text into the 2D grid.
*   It handles overlays (e.g., modals, dropdowns) by painting them strictly after the main document tree.

### 7. Backend Sync (Commit)
The engine calls `backend.EndFrame()`. The backend diffs the newly painted FrameBuffer against the previous frame and flushes the minimal set of ANSI escape sequences to the terminal.

## Event Handling

The engine reads raw input from the backend and pushes it into an internal buffer. Immediately before each frame, it coalesces high-frequency inputs (mouse moves and wheel deltas) to reduce redundant DOM processing. The resulting coalesced raw events are then translated via an `event.Synthesizer` and routed through an `event.Dispatcher`.
*   **Mouse Events:** Hit-tested against the immutable Layout `Fragment` tree to find the exact logical DOM target, then dispatched with standard Capture/Target/Bubble phases.
*   **Keyboard Events:** Routed via the `focus.Manager` directly to the currently active DOM node, bubbling up to the document root.

## Caret & Spatial Focus Navigation

The engine coordinates character-level text selection carets and spatial focus transitions:
*   **Automatic Routing:** The engine routes arrow keys to the focused element's `MoveCaret()` method if it implements `dom.SpatialCaret`.
*   **Boundary Crossing:** If caret movement hits the text boundary, the engine automatically triggers a spatial focus transition to the nearest element in that direction.
*   **Programmatic API:** Developers can trigger these transitions programmatically using the high-level `Engine.MoveCaret(dir)` and `Engine.NavigateFocus(dir)` methods, or using the logical DOM `dom.Document` methods.