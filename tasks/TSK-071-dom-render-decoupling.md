# Task: DOM and Render Decoupling

## Objective
Sever the dependency of the `dom` package on the `render` package to achieve architectural purity, making the logical DOM strictly structural. Delegate the binding of the DOM tree and the Render tree to the `engine` package using an internal mapping structure.

## Requirements
1. **Remove `render.Object` from DOM:**
   - Remove the `renderObject` field from `internal/dom.BaseNode`.
   - Remove the `RenderObject()` and `SetRenderObject(render.Object)` methods from `internal/dom.BaseNode` and the `dom.Node` interface.
   - Remove any `internal/render` imports from the `dom` and `internal/dom` packages.

2. **Strongly-Type `render.Object` Back-pointer:**
   - Update `render.Object.LogicalNode()` to return `dom.Node` (imported from `github.com/masterkeysrd/kite/dom`) instead of `any`.
   - Ensure the `render.Box` and `render.Text` implementations store this as a typed pointer.

3. **Engine State Mapping:**
   - Inside the `engine.Engine` (or its immediate sync phase context), introduce a mapping to track the physical projection of logical nodes:
     `renderMap map[dom.Node]render.Object`
   - Update the Synchronize Phase logic in `engine/pipeline.go` (or wherever tree construction occurs) to populate and query this map instead of reading/writing `dom.Node.RenderObject()`.
   - **Crucial:** Implement cleanup logic in the sync phase to delete map entries when `dom.Node`s are detached/garbage collected, avoiding memory leaks.

## Tests to Verify
- Run `go test ./internal/dom/...` to ensure all structural logic remains sound.
- Run `go test ./engine/...` to ensure the Synchronize Phase correctly builds the render tree and maps nodes.
- Verify through benchmarks (if applicable) that the O(1) map lookup in the engine doesn't severely impact the 60FPS sync phase budget.

## Documentation Updates
- No user-facing API changes requiring `README.md` updates.
- Ensure the `internal/dom` doc string reflects its new decoupled purity.