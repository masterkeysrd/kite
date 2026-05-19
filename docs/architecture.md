# Kite Architecture

This document serves as the high-level architectural overview for Kite (v2) based on our design sessions.

## 1. Core Principles
- **Terminal UI Framework:** A modern, DOM-like terminal UI framework for Go. It brings web-like development paradigms to the terminal environment.
- **In-memory operation:** No external database/storage requirements.
- **Clear Separation of Concerns:** Strict package isolation between DOM, Style, Layout, Paint, and Render layers to maintain an efficient rendering pipeline.
- **Performance-Oriented:** The pipeline targets 60FPS on the main thread, with expensive or asynchronous operations handled in a concurrent worker pool (Jobs) that dispatch results back to the main thread.

## 2. Rendering Pipeline Overview

The framework operates via a central nervous system called the **Engine (`/engine`)**. The engine runs a continuous frame loop that orchestrates a unified pipeline:

1. **Job Sync:** Collect completed job results from the concurrent worker pool into the microtask queue.
2. **Synchronize Phase (Pre-Layout):** Walk the logical DOM and project structural changes into the render tree. It flags dirty layout and style nodes.
3. **Task Draining:** Drain macrotasks (budget-capped) and microtasks (drained completely) to execute user events or lifecycle hooks.
4. **Style Phase:** Traverse the render tree to resolve inherited and explicit styles into `Computed` values.
5. **Layout Phase:** Traverse the dirty nodes, executing LayoutNG-inspired algorithms (Block, Flex, Inline) to produce immutable physical `Fragment` trees.
6. **Paint Phase:** Draw the resulting `Fragment` trees onto the framebuffer via absolute coordinates and clipping.
7. **Commit:** Push the framebuffer surface to the terminal via the decoupling backend (`/backend`).

## 3. Subsystems

### 3.1. DOM (Logical Tree)
- **Responsibility:** Maintains the structural tree and interactivity states (`Focusable`, `Disabled`).
- **Core Entities:** `Document`, `Element`, `TextNode`.
- **Adoption & Identity:** Uses a self back-pointer (`outer`) set during the attach walk. Ensures `event.Target()` and `GetElementByID()` return the outermost user-visible wrapper (useful for custom widgets).
- **Events:** Responsible for the Capture -> Target -> Bubble event propagation model. Uses $O(1)$ checks for connectivity (`IsConnected()`).

### 3.2. Style Engine (`/style`)
- **Responsibility:** Parses and resolves CSS-like styling definitions.
- **Paradigm:** Uses an `Optional[T]` pattern for sparse definitions in `style.Style` (differentiating unset fields from zero-values).
- **Resolution:** The `Resolver` applies inheritance, default application, and merges everything into a raw `style.Computed` structure (no Optionals) that is consumed directly by the layout and render phases.
- **Isolation:** Has no dependencies on other Kitex packages.

### 3.3. Layout Engine (`/layout`)
- **Responsibility:** High-performance layout computations.
- **Design:** Inspired by Blink's LayoutNG. It computes layout in terms of logical geometry (agnostic of reading direction or physical coordinates initially) and returns immutable `Fragment` trees.
- **Contexts:**
  - **Block Formatting Context (BFC):** Stacks elements vertically.
  - **Flex Formatting Context (FFC):** Lays out elements in one-dimensional rows or columns (supports growing, shrinking, alignment).
  - **Inline Formatting Context (IFC):** Lays out text and atomic inlines horizontally, wrapping them into line boxes. Uses a flat representation of `InlineItem`s.

### 3.4. Render Pipeline (`/render`)
- **Responsibility:** The visual bridge between the logical DOM and physical layout.
- **Stateless Styling:** Render objects act as pure proxies for author styles (`RawStyle()`) and element defaults (`DefaultStyle()`), querying their underlying logical DOM node directly. They do not store sparse styles, avoiding state duplication.
- **Node Mirroring:** It strictly mirrors the DOM structure using a unified `render.Box` or `render.Text` (no explicit block/flex types here; the engine delegates algorithms at layout time based on `ComputedStyle.Display`).
- **Dirty Tracking:** Carries lifecycle synchronization flags (`NeedsSync`, `DirtyStyle`, `DirtyLayout`) without doing actual math calculations itself.

### 3.5. Event System (`/event`)
- **Responsibility:** Dispatching semantic interactions and input routing.
- **Phases:** Advanced dispatcher supporting Capture, Target, and Bubble phases.
- **Synthesizer:** Translates raw terminal input (e.g., from Charmbracelet's `ultraviolet`) into semantic events (like key combinations or clicks).

### 3.6. Focus & Spatial Navigation (`/focus`)
- **Responsibility:** Managing interaction focus state.
- **Operation:** Focus state operates strictly on the logical `dom.Node` tree, utilizing `dom.Focusable` and `dom.Disableable` interfaces.
- **Spatial Navigation:** Queries physical geometry by accessing the physical `Fragment()` from the logical node's `RenderObject()`.

### 3.7. Paint & Backend (`/paint` & `/backend`)
- **Responsibility:** Terminal output decoupling and drawing.
- **Paint:** Interfaces with a logical framebuffer. Handles operations like clipping, filling cells, and applying formatted text.
- **Backend:** Decouples Kite from the actual terminal emulator. Implementations include an `ultraviolet` backend for real terminals and a `mock` backend for test environments.
