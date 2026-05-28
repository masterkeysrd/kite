# ADR 031: Terminal Cursor Management Strategy

## Status
Accepted

## Context
Historically, Kite managed the hardware terminal cursor by polling a `cursor.Provider` interface on the currently focused element's `render.Object`. 
With the decoupling of the DOM and Render engines (ADR-028) and the introduction of the `terminal` host context (ADR-030), this polling mechanism is out of place. 

We had to decide between two standard paradigms:
1. **Selection-Driven (Declarative):** Like HTML, the cursor (caret) is simply a collapsed `dom.Selection`. The engine automatically plots the hardware cursor at the selection's physical layout coordinates.
2. **Terminal API (Imperative):** Expose a `terminal.Cursor` API, allowing components to manually declare where the hardware cursor should be.

While the Selection-Driven approach is ideal for standard elements (`<input>`, `<textarea>`), it severely hinders custom components (like code editors) which manage complex internal virtual grids and need direct access to the hardware cursor without faking DOM text nodes.

## Decision
We will adopt a **Hybrid Approach**:

1. **Delete `cursor.Provider`:** The interface and the engine's polling mechanism will be removed completely.
2. **Selection as Default:** By default, the Engine will resolve the `dom.Document`'s active `Selection`. If the selection is collapsed (start == end), the engine maps the logical text offset to physical coordinates and automatically plots the hardware cursor.
3. **Imperative Override (`terminal.Cursor`):** We will expand the `terminal.Terminal` interface to include a `Cursor()` manager. 
   ```go
   type Cursor interface {
       SetPosition(x, y int)
       SetShape(shape style.CursorShape)
       Hide()
   }
   ```
   Custom components (like editors) can call this API during their event handling or lifecycle hooks. 
4. **Resolution Precedence:** At the end of the frame, the Engine will resolve the cursor state:
   - If `terminal.Cursor` was imperatively invoked during the frame, that state wins.
   - Else if a collapsed `dom.Selection` exists, it resolves and wins.
   - Else, the hardware cursor is hidden.

## Consequences
### Positive
* Standard text inputs behave exactly like browsers—they just manipulate the logical `dom.Selection`.
* Custom editors get unhindered, ergonomic access to the real hardware cursor.
* We remove the ugly `cursor.Provider` type-assertion polling from the core Engine render loop.

### Negative
* The Engine must reset the imperative `terminal.Cursor` state at the beginning of every frame to ensure old positions don't persist if a custom component stops calling `SetPosition`.