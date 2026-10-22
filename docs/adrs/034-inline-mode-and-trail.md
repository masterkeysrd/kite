# ADR 034: Inline Mode and the Trail Package

## Status
Accepted

## Context
Traditional terminal UI (TUI) applications operate in full-screen mode by switching the terminal into the alternate screen buffer (`altscreen`). While this is ideal for full-screen dashboards, file managers, and text editors, command-line developer tools (such as build tools, package managers, deploy CLIs, and test runners like Jest or Vite) require an **inline** execution model.

In an inline model:
1. The application runs directly within the user's primary terminal scrollback buffer without taking over the full window.
2. The active viewport height dynamically resizes to match the rendered content height rather than forcing full terminal row dimensions.
3. Completed milestones, task logs, and progress steps can be permanently flushed into the native terminal scrollback above the live active viewport (the Ink `<Static>` model).
4. Terminal exit restores cursor visibility and cleanly places the shell prompt on the next line below the final rendered frame.

Kite's existing architecture assumed full-screen `altscreen` operation, fixed viewport block sizes in layout, and lacked a mechanism to inject rendered cell buffers into terminal scrollback above the live render region.

## Decision

We designed and implemented a three-tier solution across the backend, rendering engine, and a new `extras/trail` package:

### 1. Backend Inline Support (`SetInline`, `InsertAbove`)
We extended the `backend.Backend` interface with two methods:
- `SetInline(inline bool)`: Configures the backend to operate in inline mode. In `uv.Backend`, inline mode skips `EnterAltScreen` and `ExitAltScreen`, leaves mouse tracking disabled so native terminal text selection and mouse scrollback function normally, and writes `\r\n` on `Restore()` so the shell prompt starts on a fresh line below the final frame.
- `InsertAbove(buf *Buffer)`: Directly flushes a rendered cell buffer into the terminal scrollback history above the live frame. On `uv.Backend`, this coordinates with the render loop to move the relative cursor, write lines to native scrollback, and preserve the active inline frame below.

### 2. Engine and RenderView Auto-Height Layout
- Added `engine.Options{Inline: bool}`.
- Updated `RenderView` and `LayoutPhase`: When `Inline` is enabled, the root constraint space sets `SetIsFixedBlockSize(false)`. This instructs the layout engine to calculate the natural content height of the DOM tree (constrained only horizontally to the terminal width).
- In `StandardPipeline.Paint`: The active frame buffer is dynamically resized to `frag.Size.Height` (clamped to the terminal viewport height). The surface from `backend.BeginFrame()` is resized directly (`surface.Resize(width, height)`) without overwriting the backend's knowledge of the full terminal dimensions.
- Added `Engine.InsertAbove(node dom.Node)`: Creates a detached document, computes styling and layout constrained to the terminal width, executes fragment painting and border junction resolution, and forwards the resulting `*backend.Buffer` to `backend.InsertAbove`.

### 3. The `extras/trail` Package
We introduced `extras/trail`, an Ink-inspired inline CLI layer built on top of Kite and `extras/kitex`:
- **Lifecycle & Runners**: `trail.New` and `trail.Run` create and execute inline instances, automatically wiring `WithCtrlCExit(true)` and an `AppHandle` context (`UseApp()`).
- **Mid-Frame Exit Guarantee**: Registered an `OnFrameRendered` engine hook ensuring that if a handler calls `app.Exit(nil)`, the final updated frame (including final static logs and summary views) is rendered before the engine halts.
- **`<Static>` Scrollback Flushing**: `trail.Static[T]` tracks emitted items using a persistent `kitex.UseRef`. In `kitex.UseLayoutEffect`, newly appended items are rendered into temporary DOM containers and flushed above via `app.InsertAbove(node)`. Because `Static` returns `nil` in the active VDOM, it occupies zero height in the live inline frame.
- **Primitives and Hooks**: Re-exports core layout elements (`Box`, `Text`, `Span`, `Fragment`, `Newline`, `Spacer`) and provides inline hooks like `trail.UseApp()` and `trail.UseInput()`, fully interoperable with all standard `kitex` hooks (`UseState`, `UseEffect`, `UseInterval`, `UseRef`).

## Consequences

### Positive
- **Native Ink-like Experience in Go**: Developers can build dynamic inline CLI tools with reactive state, effects, and declarative component trees.
- **Zero Impact on Fullscreen TUIs**: Fullscreen Kite applications remain completely unaffected; inline mode is strictly opt-in via `engine.Options{Inline: true}`.
- **Accurate Subtree Styling & Borders**: Nodes inserted into scrollback via `Engine.InsertAbove` undergo full Kite styling, layout, and Unicode border junction resolution before being committed to scrollback.
- **Native Scrollback Preservation**: Native mouse selection, scrolling, and shell history work seamlessly during and after execution.

### Negative / Trade-offs
- **Height Clamping**: If an inline component's height exceeds the physical terminal window height, it clamps to the terminal height.
- **Disabled Mouse Mode**: By default, mouse event capture is disabled in inline mode to prevent intercepting native terminal scrolling and clipboard text selection.
