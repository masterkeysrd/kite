# ADR 028: DOM and Render Engine Decoupling

## Status
Accepted

## Context
Currently, the logical DOM tree and the physical Render tree are tightly coupled. `internal/dom.BaseNode` holds a direct reference to a `render.Object` (`RenderObject()` / `SetRenderObject()`). Additionally, the `render.Object` interface has grown into a massive monolith (over 20 methods) that attempts to bridge Tree operations, Style resolution, Layout algorithms, and Event targeting. 

This violates the separation of concerns and prevents the DOM from being a pure, mathematically isolated logical structure. Furthermore, relying on the `render.Object` to proxy style definitions back to the DOM creates unnecessary interface bloat.

To prepare for the beta release and ensure the engine acts strictly as a coordinator, we need to achieve a complete disconnection: the DOM must know nothing about the Render Engine, and the Render Engine must hold strongly-typed references back to the DOM.

## Decision
1. **Remove Render Pointers from DOM:** We will remove `renderObject`, `RenderObject()`, and `SetRenderObject()` from `internal/dom.BaseNode` and the `dom.Node` interface. The `dom` package will have zero imports or knowledge of the `render` package.
2. **Engine-Level Mapping:** The `engine.Engine` will maintain an internal `map[dom.Node]render.Object`. This map will be used during the Sync Phase to O(1) lookup physical nodes from logical nodes without polluting the DOM structure.
3. **Strongly-Typed Back-Pointers:** The `render.Object` will update its `LogicalNode()` method to return a strongly-typed `dom.Node` (imported from the `dom` package) rather than an `any` interface.
4. **Interface Segregation (Style Proxying):** The `render.Object` interface will be simplified. It will no longer implement the declarative style methods (`RawStyle()`, `DefaultStyle()`, `IntrinsicStyle()`). The Style Engine (`style.Resolver`) will pull these declarative styles directly from the strongly-typed `dom.Node`. The Render Object will only be responsible for holding the resulting `*style.Computed` state.
5. **Engine as Coordinator:** This shift delegates specific styling and layout data logic to their respective domains, positioning the `engine.Engine` as a pure pipeline coordinator.

## Consequences
### Positive
* **Pure DOM:** The logical DOM becomes completely isolated from physical rendering, making it easier to test and reason about.
* **Simplified Render Object:** Removing style proxy methods from `render.Object` shrinks the interface significantly, making it easier to implement custom render objects.
* **Clearer Engine Role:** The engine's role as a coordinator is solidified by owning the map that binds the two independent trees.

### Negative
* **Map Overhead:** Maintaining a `map[dom.Node]render.Object` in the engine adds slight memory overhead and requires careful lifecycle management during DOM node attachment/detachment to prevent memory leaks (the map must be cleared when a node is removed).
