# ADR 015: Event Coalescing and Deferred Scroll Rendering

## Status
Accepted

## Context
When running Kite, scrolling a `TextArea` rapidly via the mouse wheel caused noticeable lag and stuttering.

Investigation into the `Engine.Run` event loop revealed that raw events (e.g., `event.WheelEvent`, `event.MouseEvent`) were processed immediately as they arrived from the backend channel, pushing `DirtyScroll` flags and occasionally triggering immediate state recalculations (like `ScrollCursorIntoView` in text controls).
Because the terminal can emit input events much faster than the 60FPS render budget (e.g., hundreds of mouse movements or scroll ticks per second), the application spent an unbalanced amount of time dispatching duplicate or obsolete input states, causing frame drops.

Additionally, text controls performed heavy layout tree lookups (e.g., `layout.ScrolledAbsoluteBounds`) synchronously within their event handlers to keep the cursor visible, compounding the lag.

## Decision
We will solve this performance issue through two complementary architectural patterns:

### 1. Engine-Level Event Coalescing
Instead of processing input events synchronously as they arrive:
1. The `Engine.Run` loop will push incoming raw events into an internal buffer.
2. The buffer will be drained *only* just before the `Engine.Frame()` execution begins (on the frame ticker).
3. While draining, the engine will coalesce high-frequency events:
   - **Wheel Events:** Multiple consecutive wheel events targeting the same `EventTarget` will have their `DeltaX` and `DeltaY` aggregated into a single, net event.
   - **Mouse Movements:** Multiple consecutive mouse moves will be squashed, dispatching only the final coordinate.

This completely decouples input arrival frequency from the DOM dispatch and rendering pipeline, ensuring that the DOM only processes the *net* state change once per frame.

### 2. Deferred Scroll State Rendering
Scrolling must remain purely an $O(1)$ mutation of state `(X, Y)` and a dirty flag assignment (`DirtyScroll`), avoiding any coordinate math during the event dispatch phase.
1. The `ScrollCursorIntoView` functionality inside `textControlBase` will be removed from synchronous event handlers (e.g., `OnWheel`, `OnKey`).
2. Heavy recalculations for scrolling or cursor positioning will be deferred. The text controls will set a simple flag (e.g. `needsScrollIntoView`).
3. The engine will introduce an explicitly deferred hook (or resolve it directly during the `Layout` or `Scroll` phases) to trigger these visibility adjustments *after* layout bounds are guaranteed to be stable and *before* the paint phase.

## Consequences

### Positive
- Massive reduction in redundant event dispatch overhead during high-frequency user input (mouse moves, rapid scrolling).
- Smooth 60FPS rendering of scroll views and textareas.
- Strict enforcement of separation between State Mutation (Event Loop) and State Evaluation (Render Pipeline).

### Negative
- Slightly increased complexity in the `engine` package to manage the input buffer and coalescing rules.
- Potential edge cases if an author's custom widget strictly relied on capturing every micro-tick of a mouse movement (rare for TUI applications).
