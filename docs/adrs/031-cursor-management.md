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
2. **Style-Driven Configuration:** Every `style.Style` now includes a `Cursor` field of type `style.Cursor`. This struct allows elements to declare their desired cursor shape, color, and blinking behavior when focused.
   ```go
   type Cursor struct {
       Shape Optional[CursorShape]
       Blink Optional[bool]
       Color Optional[color.Color]
   }
   ```
3. **Selection as Default:** Standard text elements (`InputElement`, `TextAreaElement`) configure their `Cursor` style to match their internal insertion point. By default, the Engine resolves the `dom.Document`'s active `Selection`. If the selection is collapsed, the engine translates the logical offset to physical coordinates and applies the `Cursor` style from the focused element.
4. **Backend Agnosticism:** The `backend` package remains agnostic of internal `style` or `cursor` packages. The `Engine` is responsible for mapping internal `style.CursorShape` values to `backend.CursorShape` and calling the appropriate backend methods (`SetCursorPos`, `SetCursorShape`, `SetCursorColor`).

## Consequences
### Positive
* Standard text inputs behave exactly like browsers—they just manipulate the logical `dom.Selection`.
* Custom widgets can precisely control the hardware cursor via their `IntrinsicStyle` or `RawStyle` without faking DOM text nodes.
* The `backend` package is fully decoupled from the internal type system.
* We remove the ugly `cursor.Provider` type-assertion polling from the core Engine render loop.

### Negative
* The Engine must resolve and reconcile the physical cursor state against the backend state on every frame to ensure synchronization.