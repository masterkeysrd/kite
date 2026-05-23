# Kite Architecture

This document serves as the high-level architectural overview for Kite (v2) based on our design sessions.

## 1. Core Principles
- **Terminal UI Framework:** A modern, DOM-like terminal UI framework for Go. It brings web-like development paradigms to the terminal environment.
- **In-memory operation:** No external database/storage requirements.
- **Clear Separation of Concerns:** Strict package isolation between DOM, Style, Layout, Paint, and Render layers to maintain an efficient rendering pipeline.
- **Performance-Oriented:** The pipeline targets 60FPS on the main thread, with expensive or asynchronous operations handled in a concurrent worker pool (Jobs) that dispatch results back to the main thread.

## 2. Rendering Pipeline Overview

The framework operates via a central nervous system called the **Engine (`/engine`)**. The engine runs a continuous frame loop that orchestrates a unified pipeline:

1. **Input Buffering & Coalescing:** Collect raw input events from the backend into a buffer. Just before the frame renders, drain the buffer and coalesce high-frequency events (e.g., aggregate wheel deltas, squash mouse moves) into semantic events, dispatching them through the DOM.
2. **Job Sync:** Collect completed job results from the concurrent worker pool into the microtask queue.
3. **Synchronize Phase (Pre-Layout):** Walk the logical DOM and project structural changes into the render tree. It flags dirty layout and style nodes.
4. **Task Draining:** Drain macrotasks (budget-capped) and microtasks (drained completely) to execute user events or lifecycle hooks.
5. **Style Phase:** Traverse the render tree to resolve inherited and explicit styles into `Computed` values.
6. **Layout Phase:** Traverse the dirty nodes, executing LayoutNG-inspired algorithms (Block, Flex, Inline) to produce immutable physical `Fragment` trees.
7. **Paint Phase:** Draw the resulting `Fragment` trees onto the framebuffer via absolute coordinates and clipping.
8. **Commit:** Push the framebuffer surface to the terminal via the decoupling backend (`/backend`).

## 3. Subsystems

### 3.1. DOM (Logical Tree)
- **Hardware Cursor:** The logical DOM does not handle physical terminal cursor calculations. The engine queries the `cursor.Provider` interface on the render object of the currently focused node to set the terminal cursor position automatically.
- **Responsibility:** Maintains the structural tree and interactivity states (`Focusable`, `Disabled`).
- **Core Entities:** `Document`, `Element`, `TextNode`.
- **Adoption & Identity:** Uses a self back-pointer (`outer`) set during the attach walk. Ensures `event.Target()` and `GetElementByID()` return the outermost user-visible wrapper (useful for custom widgets).
- **Events:** Responsible for the Capture -> Target -> Bubble event propagation model. Uses $O(1)$ checks for connectivity (`IsConnected()`).
- **UA Shadow Subtree:** Replaced and compound elements (e.g., `<input>`, `<textarea>`, future `<checkbox>`, `<radio>`, `<select>`) own a closed UA subtree via an internal `uaRoot` field on `dom.Element` (ADR-009). The subtree is invisible to public traversal (`Children()`, `GetElementByID()`), invisible to event dispatch, and never focusable; engine phases (Sync/Style/Layout/Paint) walk it as if it were a regular child. Identity retargeting reuses the existing `outer` back-pointer so `event.Target()` always resolves to the host.
- **Scroll Model:** Every `dom.Element` exposes `Scroll()`, `ScrollTo(x, y)`, `ScrollBy(dx, dy)` and fires `event.EventScroll` on mutation (ADR-012). Scroll state is held in a lazy `*scrollState` pointer allocated only when the element is observed to need scrolling. An element is a scroll container when its computed `OverflowX` or `OverflowY` is `Scroll` or `Auto`; the framework provides a default `Scrollable` wheel handler for those containers (overridable per-element). Programmatic scroll on non-containers stores state and fires events but has no visual effect, matching browser semantics.

### 3.2. Style Engine (`/style`)
- **Responsibility:** Parses and resolves CSS-like styling definitions.
- **Paradigm:** Uses an `Optional[T]` pattern for sparse definitions in `style.Style` (differentiating unset fields from zero-values).
- **Resolution:** The `Resolver` applies inheritance, default application, and merges everything into a raw `style.Computed` structure (no Optionals) that is consumed directly by the layout and render phases.
- **Cascade Origins:** Three layers contribute property values per element in increasing precedence: `DefaultStyle()` (UA defaults, author-overridable), `RawStyle()` (author input), and `IntrinsicStyle()` (UA-intrinsic, NOT author-overridable; ADR-010). Inherited values from the parent's `Computed` sit below all three. Internally the intrinsic layer is tagged `OriginUserAgent` for spec alignment.
- **Overflow Model:** `OverflowX` and `OverflowY` are per-axis properties accepting `Visible`, `Hidden`, `Clip`, `Scroll`, or `Auto`. A fluent `.Overflow(v)` shorthand on `style.Style` sets both axes at once for ergonomic cases while preserving asymmetric expressiveness (e.g., horizontal-only scroll). `Scroll` and `Auto` make the element a scroll container (ADR-012); `Hidden` and `Clip` clip without scrolling (ADR-011).
- **Isolation:** Has no dependencies on other Kitex packages.

### 3.3. Layout Engine (`/layout`)
- **Responsibility:** High-performance layout computations.
- **Design:** Inspired by Blink's LayoutNG. It computes layout in terms of logical geometry (agnostic of reading direction or physical coordinates initially) and returns immutable `Fragment` trees.
- **Constraint Propagation:** `ConstraintSpace` carries two spatial references that flow from parent to child (ADR-018):
  - **`ContainingSpace`:** The parent's border-box dimensions. All percentage-based sizes resolve against this (e.g., `width: 50%` → `ContainingSpace.Width * 50 / 100`), consistent with the strict border-box model (ADR-017).
  - **`ContainerSpace`:** The parent's content-box dimensions (border-box minus border and padding). This is the available space for children before individual child margins are subtracted.
  - A shared `BuildChildSpace` function centralizes child constraint generation, eliminating duplicated decoration math across formatting contexts.
- **Contexts:**
  - **Block Formatting Context (BFC):** Stacks elements vertically.
  - **Flex Formatting Context (FFC):** Lays out elements in one-dimensional rows or columns (supports growing, shrinking, alignment).
  - **Inline Formatting Context (IFC):** Lays out text and atomic inlines horizontally, wrapping them into line boxes. Uses a flat representation of `InlineItem`s.

### 3.4. Render Pipeline (`/render`)
- **Replaced & Compound Elements:** Form controls and other compound widgets compose their visuals as a closed UA Shadow Subtree on the logical element (ADR-009). They get a plain `render.Box` and rely on standard formatting contexts — no per-widget render object or layout algorithm. Text-based form controls (`<input>`, `<textarea>`) share a common `textControlBase`. Toggle controls (`<checkbox>`, `<radio>`) manage hidden text nodes for their glyphs. Complex composites like `<select>` combine a shadow trigger button with dynamic, out-of-flow `element.Overlay` popups and temporary `focus.Scope` trapping.
- **Responsibility:** The visual bridge between the logical DOM and physical layout.
- **Stateless Styling:** Render objects act as pure proxies for the three element-contributed style layers — `DefaultStyle()`, `RawStyle()`, and `IntrinsicStyle()` (ADR-010) — querying their underlying logical DOM node directly. They do not store sparse styles, avoiding state duplication.
- **Node Mirroring:** It strictly mirrors the DOM structure using a unified `render.Box` or `render.Text` (no explicit block/flex types here; the engine delegates algorithms at layout time based on `ComputedStyle.Display`).
- **Dirty Tracking:** Carries lifecycle synchronization flags (`NeedsSync`, `DirtyStyle`, `DirtyLayout`) without doing actual math calculations itself.

### 3.5. Event System (`/event`)
- **Responsibility:** Dispatching semantic interactions and input routing.
- **Event Coalescing:** The engine decouples raw input arrival from DOM dispatch. Incoming events are buffered and coalesced per-frame (e.g., squashing intermediate mouse movements, summing fast scroll wheel deltas) to guarantee UI resilience under high-frequency inputs (ADR-015).
- **Phases:** Advanced dispatcher supporting Capture, Target, and Bubble phases.
- **Synthesizer:** Translates raw terminal input (e.g., from Charmbracelet's `ultraviolet`) into semantic events (like key combinations or clicks).

### 3.6. Focus & Spatial Navigation (`/focus`)
- **Responsibility:** Managing interaction focus state.
- **Operation:** Focus state operates strictly on the logical `dom.Node` tree, utilizing `dom.Focusable` and `dom.Disableable` interfaces.
- **Spatial Navigation:** Queries physical geometry by accessing the physical `Fragment()` from the logical node's `RenderObject()`.

### 3.7. Paint & Backend (`/paint` & `/backend`)
- **Responsibility:** Terminal output decoupling and drawing.
- **Paint:** Interfaces with a logical framebuffer. Handles operations like clipping, filling cells, and applying formatted text.
- **Overflow Clipping:** During fragment recursion the paint engine composes `Surface.Clip()` calls to enforce `OverflowX/Y` in { `Hidden`, `Clip`, `Scroll`, `Auto` } (ADR-011). The fragment's own background and border paint unclipped at the parent's level; descendants paint onto a clipped sub-surface restricted to the content box. Nested overflow boxes compose via rect intersection.
- **Scroll Translation:** For scroll containers (computed overflow = `Scroll` or `Auto`), paint reads the element's raw scroll offset, clamps it on read to `[0, contentSize - viewportSize]`, and translates descendant origins by the negative clamped offset (ADR-012). Element state stores raw author intent and is never mutated by paint.
- **Scrollbar Rendering:** If an element is a scroll container and its `ComputedStyle.Scrollbar.X/Y` is true, the layout phase reserves a 1-column/row gutter. The paint phase automatically computes the viewport-to-content ratio and draws a customizable scrollbar track and thumb in that reserved gutter.
- **Border Post-Processing:** To automatically form correct Unicode junctions (e.g., `┼`, `├`) without manual coordinate math in layout, the `PaintEngine` runs a global $O(W \times H)$ post-processing pass over the framebuffer. Every cell explicitly tagged as a border is resolved against its cardinal neighbors. Junction merging for overlapping borders of varying weights is handled via a strict "Heaviest Style Wins" precedence rule (using an explicit `BorderStyle` enum stored per-cell). The resolver runs once on the root surface only; it must never be invoked on a clipped sub-surface.
- **Backend:** Decouples Kite from the actual terminal emulator. Implementations include an `ultraviolet` backend for real terminals and a `mock` backend for test environments.

### 3.8. Developer Tools (`/devtools`)
- **Responsibility:** Provide utilities to inspect, test, and debug Kite applications without bloating the core runtime.
- **Inspector (`/devtools/inspector`):** A lightweight HTTP server utilizing Server-Sent Events (SSE) to stream the live logical DOM tree, computed styles, and layout bounding boxes to a web browser interface.
- **X-Ray Mode:** An optional rendering flag built into the core `paint` engine but toggled via devtools. Overlays colored bounding boxes (margin, padding, content) for visual layout debugging.
- **Test Environment (`/devtools/testenv`):** A headless testing harness that wraps the existing `backend/mock`. Provides high-level APIs for structural DOM assertions (`GetNodeByID`), simulated input routing (`Type`, `Click`), layout verification, and golden/visual snapshot testing (producing HTML or ANSI dumps).

### 3.9. Overlay System
- **Responsibility:** Management and rendering of out-of-flow components like dropdowns, tooltips, and modal dialogs.
- **Document Integration:** The `dom.Document` maintains an explicit list of overlays via `ShowOverlay` and `HideOverlay`. Overlays are sorted by `zIndex`.
- **Anchored Positioning:** `element.Overlay` uses a custom layout algorithm that queries the physical bounds of an `Anchor` element (via `GetBoundingClientRect`) to position itself.
- **Smart Flipping:** If an overlay would overflow the viewport, the layout engine automatically flips it to the opposite side or chooses the "best fit" placement with the most available space.
- **Modal Dialogs:** `element.Dialog` provides a full-screen modal container that automatically traps focus using `focus.Scope`.
