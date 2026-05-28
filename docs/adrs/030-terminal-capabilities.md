# ADR 030: Terminal Capabilities Context

## Status
Accepted

## Context
As the framework has grown, `dom.Document` has started accumulating methods that interact with the physical host environment (such as `Clipboard()` and `SetClipboardProvider()`). Additionally, components (like animations or async fetchers) need a way to communicate with the Engine's render loop (e.g., `RequestFrame()`) without having direct access to the `engine` package.

To keep the `dom` package purely logical and structural, we need to draw a strict boundary between "Logical UI State" and "Host Environment Capabilities".

## Decision
We establish a new `terminal` package to act as the Host Environment Context (conceptually similar to the `window` object in browsers, but tailored for TUIs). 

### Boundary Definitions
1. **`dom.Document` retains Logical UI State:**
   - **Node Tree:** `CreateElement`, `GetElementByID`
   - **Focus Management:** `CurrentFocus`, `Focus`, `PushScope` (Focus dictates interaction with logical nodes).
   - **Selection State:** `Selection`, `CreateRange` (Selection references logical text offsets).
   - **Layout Queries:** `DefaultView()` (via ADR-029).

2. **`terminal.Terminal` assumes Host Environment Capabilities:**
   - **Clipboard (`terminal.Clipboard`):** Synchronizes with the OS/TTY clipboard (OSC 52). 
   - **Window (`terminal.Window`):** Interfaces with the engine's frame loop, exposing methods like `RequestFrame()` for animations.

### Implementation
- The `dom.Document` interface will expose a `Terminal() terminal.Terminal` method.
- The `engine.Engine` will implement the `terminal.Terminal` interface and inject itself into the Document during initialization.
- We will remove clipboard-related methods from `dom.Document` and `dom.View`.
- The draft `Layout` interface currently sitting in the `terminal` package will be deleted, as layout queries are properly handled by `dom.View` (ADR-029).

## Consequences
### Positive
* **Purity:** `dom.Document` is stripped of host-level concerns, making it a pure structural and semantic root.
* **Separation of Concerns:** External hardware or OS interactions are strictly routed through the `terminal` package.
* **Capability Expansion:** If we need to add window resizing events, terminal bell ringing, or title setting in the future, the `terminal.Window` interface provides an obvious, isolated home for those capabilities.

### Negative
* Components must perform a slightly longer chain to access the clipboard (`el.OwnerDocument().Terminal().Clipboard().WriteText(...)`), though this matches web standards (`window.navigator.clipboard`).