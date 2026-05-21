# ADR 014: Kite DevTools Architecture

## Status
Accepted

## Context
As Kite matures, building complex TUI applications requires robust debugging and testing capabilities. Direct terminal debugging is notoriously difficult due to layout squashing, raw ANSI escape complexities, and limited screen real estate. Furthermore, preventing layout regressions requires a way to run deterministic headless tests against the rendering pipeline.

Currently, we have `backend/mock` which captures frames but provides no high-level testing ergonomics, and we lack any visual inspection tooling for developers building applications.

## Decision
We will introduce a new top-level package `kite/devtools` to provide external utilities that don't bloat the core production runtime. The architecture encompasses three primary pillars:

### 1. Web-Based DOM & Layout Inspector (`devtools/inspector`)
Instead of an in-terminal split-pane inspector (which disrupts the application layout), we will build an out-of-band inspector. 
- A lightweight HTTP server is launched by the devtools package.
- It uses **Server-Sent Events (SSE)** to stream a one-way pipeline of the logical DOM tree structure, computed styles, and physical layout bounds (X, Y, Width, Height) to a web browser.
- This provides infinite screen space for deep tree inspection without altering the TUI state.

### 2. In-Terminal X-Ray Mode
To solve immediate layout bugs visually, the core `paint` engine will add an optional rendering flag.
- When toggled (via a devtools integration hotkey), the engine will draw explicit bounding boxes over components.
- Coloring semantics: Content Box (Blue), Padding Box (Green), Margin Box (Red).

### 3. Headless Test Environment (`devtools/testenv`)
We will build a high-level wrapper around the existing `backend/mock`. 
- **DOM Queries:** Methods like `GetNodeByID` to traverse the logical DOM directly in tests.
- **Event Simulation:** Methods like `Type()` and `Click()` that dispatch through Kite's standard synthetic event pipeline.
- **Golden Testing:** It will snapshot the `paint.FrameBuffer` into `.golden` files to catch unintended visual regressions in automated tests.
- **Visual Dumps:** A utility to output the framebuffer state as colored ANSI text or HTML for detailed CI/CD error reporting.

## Consequences
**Positive:**
- Zero performance or binary size impact on the core `kite` production runtime.
- SSE provides extremely low-latency DOM synchronization compared to constant polling.
- `testenv` combined with Golden Testing provides strict layout regression guards.

**Negative:**
- Requires maintaining a minimal HTML/JS dashboard alongside the Go codebase.
- "X-Ray Mode" requires a slight intrusion into the core `paint` engine to support the dev flag.