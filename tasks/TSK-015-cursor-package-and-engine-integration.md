# TSK-015: Cursor Package and Engine Integration

## Feature Design & Requirements
We need a clean abstraction for the terminal cursor that prevents cyclic dependencies between the `engine`, `dom`, and `render` packages.

1. **New Package `cursor`:**
   - Create `kite/cursor/cursor.go`.
   - Move `CursorShape` from `engine` to `cursor.Shape`.
   - Define `cursor.State`:
     ```go
     type State struct {
         Visible bool
         X, Y    int // Local coordinates relative to the component
         Shape   Shape
     }
     ```
   - Define `cursor.Provider` interface:
     ```go
     type Provider interface {
         CursorState() State
     }
     ```

2. **Style Package Updates:**
   - Add `CursorShape` and `CursorColor` (optional) to `style.Style` and `style.Computed`.
   - Ensure the CSS resolver cascades these properties.

3. **Engine Integration:**
   - In `engine.go` during the frame loop (after Paint), check the currently focused node (`e.focusManager.Current()`).
   - If the node has a `RenderObject()` that implements `cursor.Provider`, call `CursorState()`.
   - If `Visible` is true, translate the local `X, Y` to absolute screen coordinates using `layout.AbsoluteBounds` against the render object's `Fragment`.
   - Update the terminal hardware cursor via the backend.

## Tests Required
- Unit test in `cursor` package ensuring types are correct.
- Style resolver test verifying `CursorShape` inherits or defaults correctly.
- Engine test (using the mock backend) verifying that a focused custom render object correctly sets the hardware cursor position.

## Documentation Updates
- Update `engine/doc.go` to reflect the new cursor lifecycle.
