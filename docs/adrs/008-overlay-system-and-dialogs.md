# ADR-008 - Overlay System and Dialogs

## Context
Kite requires a robust mechanism to render out-of-flow elements such as dropdown menus, tooltips, and modal dialogs. Standard layout algorithms operate within formatting contexts (Block, Flex, Table) and do not support arbitrary absolute positioning (`z-index`, `position: absolute`). We must support top-layer rendering without polluting the styling engine with physical coordinates. Furthermore, anchoring elements (like a dropdown attached to a button) requires querying physical bounds.

## Decision
We will implement an **Overlay System** managed directly by the logical Document and Render Root, utilizing a fixed-viewport model without global scrolling.

1. **Document Overlay Management:**
   - The `dom.Document` will maintain an explicit top layer via `ShowOverlay(el, zIndex)` and `HideOverlay(el)`.
   - Overlays are sorted by `zIndex` (ascending) and insertion order.

2. **Render Root Segregation:**
   - During the Sync Phase, the Engine diffs the main `Body()` and the `Overlays()`.
   - The `RenderView` processes the main flow first. Then, it layouts and paints the overlays sequentially. Overlays inherently paint on top of the main layout, respecting their explicit `zIndex`.

3. **Absolute Positioning via `GetBoundingClientRect`:**
   - The `dom.Element` interface will expose `GetBoundingClientRect() (layout.Rect, bool)`, mirroring the browser API. This queries the cached layout fragment tree to return absolute terminal screen coordinates.
   - We will not implement global "page scroll"; the viewport is strictly the hardware terminal size, meaning client coordinates equal screen coordinates.

4. **Anchored Overlays with Smart Flipping (`element.Overlay`):**
   - A new generic component that acts as a custom render object (`render.Overlay`).
   - It accepts an `Anchor` element and a `Placement` enum (Top, Bottom, Left, Right).
   - During layout, it calculates its intrinsic size, queries `Anchor.GetBoundingClientRect()`, and calculates physical `X,Y` offsets based on the placement.
   - If `Flip` is enabled, the render object performs collision detection against the viewport bounds. If the overlay overflows the requested side (e.g., Bottom), it automatically recalculates its position for the opposite side (e.g., Top). 
   - **Best Fit Logic:** If the overlay overflows both the primary and opposite placements, it defaults to the side with the most available space (e.g., if Top has 5 cells and Bottom has 10, it picks Bottom).

5. **Dialog Component (`element.Dialog`):**
   - A modal component that attaches itself via `ShowOverlay`.
   - It spans 100% of the viewport and centers its content using standard Flexbox constraints.
   - It traps keyboard navigation by pushing a `focus.Scope` upon connection.

## Consequences
- **Positive:** No `z-index` or `position: absolute` complexity in the CSS styling engine.
- **Positive:** Keeps math simple by avoiding global page scroll offsets.
- **Positive:** Enables robust anchoring for complex UI widgets (Dropdowns, Tooltips).
- **Negative:** Elements that need to act as overlays must be explicitly managed via the overlay API rather than just applying a CSS class.
