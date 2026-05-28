# Task: Implement Hybrid Terminal Cursor Management

## Objective
Remove the deprecated `cursor.Provider` interface and implement the hybrid cursor strategy: declarative fallback based on `dom.Selection` and imperative override via `terminal.Cursor`.

## Requirements
1. **Extend `terminal` Package (`terminal/terminal.go`):**
   - Add the `Cursor` capability to the `terminal.Terminal` interface.
   - Define the `Cursor` interface:
     ```go
     type Cursor interface {
         SetPosition(x, y int)
         SetShape(shape style.CursorShape)
         Hide()
     }
     ```
2. **Remove `cursor.Provider`:**
   - Delete the `cursor.Provider` interface from `cursor/cursor.go`.
   - Remove `cursor.Provider` implementation blocks from `element/input.go`, `element/textarea.go`, and `element/text_control.go`.
   - Ensure the internal `textControlBase` correctly maps its local caret state to the global `dom.Selection` when focused (this may already be done via TSK-050).

3. **Engine Implementation (`engine/engine.go`):**
   - Implement the `terminal.Cursor` interface on the Engine's terminal context proxy (from TSK-074).
   - The engine must track an internal `imperativeCursorState` struct that resets at the beginning of every frame loop.
   - At the end of the `Paint` phase (or just before commit):
     - **Check 1:** If the imperative `terminal.Cursor` was used this frame, use those physical coordinates and shape.
     - **Check 2:** If not, query the `dom.Document`'s Selection. If it is collapsed, use the layout fragment tree to map the text node offset to physical absolute coordinates (using existing helpers like `cursor.FromTextFragment`) and use those.
     - **Check 3:** Otherwise, instruct the terminal backend to hide the hardware cursor.

## Tests to Verify
- Update `engine/cursor_test.go` to remove mock `cursor.Provider` objects.
- Write tests in `engine/...` to verify the precedence logic: calling `Terminal().Cursor().SetPosition(...)` must override a collapsed DOM selection for that frame.
- Verify standard `<input>` elements still display the blinking cursor correctly via the selection fallback.

## Documentation Updates
- Update `cursor/doc.go` to explain the hybrid strategy, noting that standard widgets use Selection while custom editors should use `Terminal().Cursor()`.