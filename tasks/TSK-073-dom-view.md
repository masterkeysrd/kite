# Task: Implement DOM View for Layout Queries

## Objective
Implement the `dom.View` pattern so that logical DOM nodes can query their physical layout bounds (`GetBoundingClientRect`) and `ComputedStyle` without depending on the Render Engine.

## Requirements
1. **Define `dom.View`:**
   - In `dom/interfaces.go` (and mirrored in `internal/dom`), define the `View` interface:
     ```go
     type View interface {
         GetBoundingClientRect(Node) (geom.Rect, bool)
         GetComputedStyle(Node) *style.Computed
     }
     ```
2. **Document Association:**
   - Add `DefaultView() View` to `dom.Document`.
   - Add `SetDefaultView(View)` to `internal/dom.Document` (or the internal struct implementation) so the engine can inject it.

3. **Engine Implementation:**
   - In the `engine` package, create a dedicated, lightweight struct (e.g., `domViewProxy`) that implements `dom.View`.
   - This struct should hold a reference to the `map[dom.Node]render.Object` created in TSK-071.
   - The implementation must use the map to look up the node, and then return `layout.AbsoluteBounds(ro.Fragment())` or `ro.ComputedStyle()`.
   - Ensure the Engine instantiates this proxy and calls `document.SetDefaultView(proxy)` during initialization.

4. **Refactor Element Proxies:**
   - Modify `internal/dom.Element.GetBoundingClientRect()` to proxy the request:
     ```go
     if view := e.OwnerDocument().DefaultView(); view != nil {
         return view.GetBoundingClientRect(e)
     }
     return geom.Rect{}, false
     ```
   - Do the same for any `ComputedStyle()` helpers that currently exist on the DOM elements.

## Tests to Verify
- Update `internal/dom/bounding_rect_test.go` to inject a mock `dom.View` instead of relying on an attached `render.Object`.
- Run `go test ./engine/...` and `go test ./element/...` (specifically overlay and dropdown tests) to ensure layout queries still return correct coordinates via the Engine proxy.

## Documentation Updates
- None required.