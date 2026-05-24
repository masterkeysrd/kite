# ADR 022: Text Selection & Clipboard Architecture

## Status
Accepted

## Context
Kite needs a way to support text selection (highlighting) and clipboard integration (Copy/Paste) akin to standard browser behaviors. The challenge lies in mapping logical DOM text nodes to their physical terminal coordinates efficiently, without violating our strict caching and immutability guarantees.

Specifically:
- Layout relies on Immutable Fragments (`layout.Fragment`).
- Re-measuring or mutating fragments for state changes like "highlighted text" breaks layout caching.
- Querying logical DOM selection state inside the 60FPS Paint loop causes severe performance degradation ($O(N)$ type assertions and DOM lookups per character).

## Decision

We will implement a **Push Model (Independent List) Architecture** for text selection.

### 1. Logical State (DOM Layer)
We introduce standard browser-like models in the `dom` package:
- `dom.Range`: Represents a start and end offset within text nodes.
- `dom.Selection`: Maintained by the `dom.Document`, manages the active ranges.

### 2. Pre-Paint Resolution Phase (Render Layer)
Instead of the paint engine pulling data from the DOM, we resolve the selection into physical bounds *between* layout and paint.
- The `render.View` (or a dedicated manager) will observe `dom.Selection`.
- Before paint, it maps the logical ranges to physical screen-space rectangles by querying the cached `layout.Fragment` tree using absolute bounds utilities.
- It produces a flat slice of `paint.SelectionRect` objects.

### 3. Mask-Based Painting (Paint Layer)
- The paint engine accepts the `[]paint.SelectionRect` as an independent input to its context.
- The layout engine and its fragments remain 100% ignorant of selection state, preserving all cache hits.
- During rasterization, if a rendered cell intersects with any of these rectangles, the paint engine applies the selection background/foreground colors (acting as a visual mask/overlay).

### 4. Clipboard API
We will leverage the existing `ClipboardBridge` within `event.Synthesizer`. 
- The terminal backend (Ultraviolet) provides the OS integration.
- `dom.Document` sets up global event listeners for `event.TypeCopy` and `event.TypePaste` to interact with `dom.Selection.String()` and the underlying event's `ClipboardData`.

### 5. Input and TextArea Controls (UA Shadow Subtrees)
Because `<input>` and `<textarea>` encapsulate their text inside a closed UA Shadow Subtree (ADR-009), they cannot be natively selected by the global DOM selection API.
- They maintain `SelectionStart` and `SelectionEnd` as local rune indices.
- During layout/sync, if `Start != End`, they explicitly push a temporary `dom.Range` mapping to their hidden shadow text nodes into the global `dom.Selection` to trigger the high-performance Paint Masking.
- They listen to `Copy`, `Cut`, and `Paste` events directly during the capture/target phase to manipulate their internal value string and call `event.PreventDefault()`.

## Consequences
- **Positive:** Maximum rendering performance. Text selection visually updates without invalidating any layout algorithms or fragment caches.
- **Positive:** Clean decoupling. The paint package doesn't need to know what a `dom.Node` is.
- **Negative:** Requires mapping terminal `(X,Y)` back to logical rune indices during mouse drag (hit-testing), which involves reverse-calculating width clusters.