# ADR 001: Virtual Markers for List Items in Layout Phase

## Status
Accepted

## Context
We need to implement a layout algorithm for list items (`DisplayListItem`) to support rendering markers like bullets or numbers, similar to HTML's `<li>` elements. In Chromium's LayoutNG (which inspires our layout engine), list markers often generate dedicated render objects or anonymous boxes (e.g., `LayoutOutsideListMarker`). 

However, Kite operates within a terminal environment using a discrete character grid and adheres to strict architectural rules:
1. **Unified Render Box:** We only use `render.Box` and `render.Text`. We do not create specialized render objects per display type.
2. **Terminal Grid Constraints:** Floating elements with negative coordinates (like `list-style-position: outside` without explicit margins) risk clipping or breaking visual alignment in a TUI.
3. **No Phantom Nodes:** Injecting synthetic render objects during the DOM-to-Render Sync phase complicates tree diffing, hit testing, and focus management because these nodes have no logical DOM equivalent.

## Decision
We will implement list layout using a **Virtual Marker in Layout** approach.

1. **No New Render Objects:** We will not create a `render.ListItem` or `render.Marker`. Nodes with `DisplayListItem` will continue to use `render.Box`.
2. **Virtual Fragment Generation:** The `layout.ListAlgorithm` will dynamically synthesize the marker as an atomic layout item (e.g., a shaped text fragment like `"• "`) directly during the layout phase based on the `style.ListStyleType`.
3. **Terminal-Safe Row Layout:** We disregard the complex CSS `list-style-position` (inside vs outside) mechanics. Instead, the `ListAlgorithm` will format the list item as a specialized two-column row layout. The first column holds the virtual marker fragment, and the remaining available inline space is given to the block content. This ensures multi-line text wraps cleanly adjacent to the bullet and prevents coordinate clipping.
4. **Ordinal Computation:** For numbered lists (`Decimal`), the layout algorithm will compute its ordinal by walking its previous logical siblings (`node.PreviousSibling()`) during the measure/layout pass to count preceding `DisplayListItem` nodes.

## Consequences

### Positive
- **Architectural Purity:** Adheres strictly to the Unified Render Box rule and prevents leaking layout/style logic into the Engine's Sync phase.
- **Simplicity:** No changes required to `engine/engine.go` or `paint/engine.go`. The Paint engine naturally draws the physical fragments produced by the layout algorithm.
- **Terminal Reliability:** The two-column row approach guarantees no clipping and perfect text wrapping in a grid environment.

### Negative / Trade-offs
- **$O(N)$ Ordinal Lookups:** Numbered lists require traversing previous siblings during layout. This is technically $O(N^2)$ for an entire list, but acceptable since terminal TUIs rarely have massive lists (e.g., 10,000+ items). If this becomes a bottleneck, we can introduce a caching layer for list ordinals in the future.
- **No Direct Styling of Markers:** Because the marker does not exist in the DOM or Render tree, it cannot be targeted independently by events or isolated styles (it inherits the exact style of the list item).
