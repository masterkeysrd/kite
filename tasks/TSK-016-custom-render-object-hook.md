# TSK-016: Custom Render Object Hook

## Feature Design & Requirements
Currently, `engine.createRenderObject` hardcodes `render.Text` and `render.Box`. We need to allow logical DOM nodes to instantiate specialized replaced render objects (like `render.Input`).

1. **Interface in `render`:**
   - Define `CustomObjectProvider` in `render/object.go`:
     ```go
     type CustomObjectProvider interface {
         CreateRenderObject() Object
     }
     ```

2. **Engine Update:**
   - Modify `engine.createRenderObject(n dom.Node) render.Object`.
   - Before the `switch n.Kind()`, check if `n` implements `render.CustomObjectProvider`.
   - If it does, call `n.CreateRenderObject()` instead of `render.NewBox`.

## Tests Required
- Regression test in `engine/engine_test.go` verifying that a mock logical node implementing `CustomObjectProvider` successfully injects its custom render object into the render tree during the Sync phase.
