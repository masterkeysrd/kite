# Task: Implement Terminal Capabilities Context

## Objective
Establish the `terminal` package as the sole gateway for host/OS-level capabilities (Clipboard, Frame scheduling) and remove these concerns from the logical `dom.Document`.

## Requirements
1. **Refine `terminal` Package (`terminal/terminal.go`):**
   - Retain the `Terminal` interface.
   - Retain the `Clipboard` interface.
   - Add a `Window` interface containing:
     ```go
     type Window interface {
         RequestFrame()
     }
     ```
   - Delete the `Layout` interface and the `Node` interface from `terminal/terminal.go` (this is handled by `dom.View` via TSK-073).

2. **Clean up `dom.Document` (`dom/interfaces.go` & `internal/dom/document.go`):**
   - Add `Terminal() terminal.Terminal` to the `dom.Document` interface.
   - Add `SetTerminal(terminal.Terminal)` to the internal document struct.
   - **Remove** `Clipboard()` and `SetClipboardProvider()` from `dom.Document`.
   - Leave Focus (`Focus`, `PushScope`, etc.) and Selection (`Selection`, `CreateRange`) exactly where they are on the Document.

3. **Engine Integration (`engine/engine.go`):**
   - Have the Engine (or a dedicated proxy struct inside the `engine` package) implement the `terminal.Terminal`, `terminal.Clipboard`, and `terminal.Window` interfaces.
   - For `Window.RequestFrame()`, proxy the call to the engine's internal frame scheduling mechanism.
   - For `Clipboard`, proxy the calls to the backend's clipboard bridge.
   - During Engine initialization, inject this terminal context into the document via `doc.SetTerminal(engineTerminal)`.

4. **Refactor Callers:**
   - Search the codebase for `OwnerDocument().Clipboard()` and update them to use `OwnerDocument().Terminal().Clipboard()`.
   - Update animations or components that previously requested frames to use `Terminal().Window().RequestFrame()`.

## Tests to Verify
- Run `go test ./internal/dom/...` to ensure document structural tests pass.
- Run `go test ./engine/...` to ensure the terminal context injection works.
- Run `go test ./element/...` to verify text controls successfully interact with the OS clipboard through the new terminal interface.

## Documentation Updates
- Update package documentation in `terminal/terminal.go` to clarify its role as the Host Environment Context.