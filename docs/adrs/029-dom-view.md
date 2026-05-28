# ADR 029: DOM View and Layout Queries

## Status
Accepted

## Context
With the decision (ADR-028) to sever the hard link between `dom.Node` and `render.Object`, the DOM package lost its ability to directly answer layout and style queries like `GetBoundingClientRect()` and `ComputedStyle()`. These methods are critical for developers building overlays, custom focus management, and responsive layouts.

However, adding a direct back-channel from the DOM to the Engine would re-introduce the coupling we just eliminated. We need a way for DOM elements to query their physical properties without knowing anything about the rendering pipeline or `render.Object`.

## Decision
1. **The `dom.View` Interface:** We will introduce a new, read-only `dom.View` interface in the `dom` package. This interface defines the contract for layout and style queries.
   ```go
   type View interface {
       GetBoundingClientRect(Node) (geom.Rect, bool)
       GetComputedStyle(Node) *style.Computed
   }
   ```
2. **Document Association:** The `dom.Document` will hold an optional reference to this `View`, accessible via `document.DefaultView()`. This mirrors the `document.defaultView` API in standard browsers.
3. **Engine Implementation:** The `engine` package will implement this interface via a dedicated, lightweight struct (e.g., `domViewProxy`). This proxy struct will hold a reference to the Engine's internal `map[dom.Node]render.Object`. When the Engine initializes and attaches to a Document, it injects this proxy as the `DefaultView`. We chose an independent struct over modifying `render.RenderView` to maintain strict boundaries and encapsulate the mapping logic within the Engine.
4. **Proxy Methods:** Existing methods on `dom.Element` (like `GetBoundingClientRect`) will remain, but they will be refactored to simply proxy the request up to the Document's View.
   ```go
   func (e *Element) GetBoundingClientRect() (geom.Rect, bool) {
       if view := e.OwnerDocument().DefaultView(); view != nil {
           return view.GetBoundingClientRect(e)
       }
       return geom.Rect{}, false
   }
   ```

## Consequences
### Positive
* **Maintains Purity:** The `dom` package remains completely ignorant of `render.Object`, Layout algorithms, and the Engine.
* **O(1) Performance:** The Engine's implementation of `dom.View` will utilize the `map[dom.Node]render.Object` introduced in ADR-028, ensuring these queries remain $O(1)$ fast.
* **Ergonomics Maintained:** Developers can continue to call `myElement.GetBoundingClientRect()` without changing their existing application code.
* **Testability:** In unit tests (like `internal/dom/...`), a mock `dom.View` can be injected to test layout-dependent logic without booting the entire physical rendering engine.

### Negative
* **Lifecycle Nuance:** If an element asks for its bounds before the first frame has rendered (or if it is disconnected), the View will return false/empty. This is standard browser behavior but requires developers to understand the render lifecycle.
