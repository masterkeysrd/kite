# TSK-020: Element Bounding Client Rect

## Feature Design & Requirements
Elements must be able to report their physical terminal bounds so that anchored overlays (like dropdowns) know where to draw themselves.

1. **Update `dom.Element` Interface:**
   - Add `GetBoundingClientRect() (layout.Rect, bool)` to `dom/interfaces.go`.

2. **Implement in `dom.element`:**
   - The logical element must retrieve its `RenderObject()`.
   - Call a new global utility `layout.AbsoluteBounds(engineFragmentRoot, targetNode)`.
   - Note: Since the logical DOM does not have direct access to the engine's `RenderView` root fragment, we should define `GetBoundingClientRect` to traverse *up* the logical tree to find the root, grab its fragment, and then compute the absolute bounds using `layout.AbsoluteBounds`.
   - *Alternative:* Add an `AbsoluteBounds()` method to the `render.Object` interface that walks up the `FragmentLink` tree to sum offsets.

## Tests Required
- Build a mock fragment tree with offsets and verify `GetBoundingClientRect` correctly sums the absolute X,Y coordinates.
