# TSK-019: Document Overlay API and Render Root

## Feature Design & Requirements
We need to modify the logical Document and the Render View to support a top layer (z-indexed list of overlays) that paints above the normal layout flow.

1. **Update `dom.Document`:**
   - In `dom/interfaces.go`, add to `Document`:
     - `ShowOverlay(el Element, zIndex int)`
     - `HideOverlay(el Element)`
     - `Overlays() iter.Seq[Element]`
   - In `dom/document.go`, implement a stable sorted list (or map with sorted extraction) based on `zIndex` and insertion order.

2. **Update `engine.Engine` (Sync Phase):**
   - In `syncRenderTree`, ensure the engine iterates over `document.Overlays()` and appends their render objects to the `render.RenderView` in a dedicated `OverlayChildren` list (or interweaved at the end of standard children).

3. **Update `render.RenderView` (Layout & Paint):**
   - The Root Layout Algorithm must process standard children, then overlay children.
   - Overlay children are given the full Viewport size as their `ConstraintSpace.AvailableSize`.

## Tests Required
- Unit test in `dom` verifying `ShowOverlay` sorts correctly by zIndex and preserves insertion order for identical zIndices.
- Engine Sync test verifying overlays are added to the RenderView.
