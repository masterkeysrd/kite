# AI Agent Rules & Context for Kite (v2)

This document provides guidelines and architectural context for AI assistants and coding agents operating within the Kite repository.

## 🧠 System Context

*   **Project Purpose:** A web-like Terminal UI framework that uses a DOM, Flexbox layout, and standard event propagation to render rich TUIs.
*   **Tech Stack:** Go 1.26.1.
*   **Key Dependencies:** Charmbracelet ecosystem (`github.com/charmbracelet/ultraviolet`, `github.com/charmbracelet/colorprofile`), and `github.com/rivo/uniseg` for text shaping.
*   **Database/Storage:** None. The project operates purely in-memory.

## 🏛 Architectural Rules

1.  **Strict Package Isolation:**
    *   The `/dom` package models the logical tree only. It **must not** contain layout algorithms, computed styles, or drawing logic. Interactivity state (`Focusable()`, `Disabled()`) belongs strictly to the DOM.
    *   The `/style` package has **no dependencies** on other Kitex packages. Keep it isolated.
    *   State bridging happens via `/render` objects. DOM nodes point to a `render.Object`, but they do not own the rendering lifecycle; they only carry structural synchronization flags (`NeedsSync`, `ChildNeedsSync`).
2.  **Synchronize Phase (Pre-Layout):**
    *   The engine performs an explicit Sync Phase at the start of every frame.
    *   It walks the logical DOM tree and projects it into the render tree.
    *   Structural changes (insertions/removals) in the DOM trigger synchronization flags that propagate to the document root.
    *   The engine creates/removes `render.Box` or `render.Text` objects on the fly during this phase. There is no `render.Block` or `render.Flex`—the engine uses a unified `render.Box` which delegates to the appropriate algorithm at layout time based on `ComputedStyle.Display`.
    *   **Custom Render Objects:** Logical nodes can implement `render.CustomObjectProvider` to override the default creation logic and provide specialized render objects (e.g., for replaced elements like inputs, or the specialized `render.Overlay`).
3.  **Element Identity & Adoption:**
    *   Every `dom.Element` carries an `outer` back-pointer. This pointer ensures that when widgets wrap standard elements, functions like `event.Target()`, `GetElementByID()`, and `RenderObject.Node()` always return the outermost, user-visible wrapper.
    *   Do not reset the `outer` pointer to `nil` on detach. The identity must remain stable.
4.  **Styling Paradigm:**
    *   Always use the `Optional[T]` wrapper (e.g., `style.Some(val)`) when defining properties in `style.Style`. This distinguishes between a field that is explicitly unset versus a zero-value.
    *   The bridge to the rendering layer is `style.Computed`, which contains raw values (no optionals) after the resolver applies inheritance.
    *   The `style.Resolver` applies four cascade layers in order (weakest → strongest): inherited values, element-type defaults (`DefaultStyle()`), author styles (`RawStyle()`), UA-intrinsic styles (`IntrinsicStyle()`). See ADR-010.
    *   **UA-mandated styles must live on the element via `IntrinsicStyle()`. They must not be hard-coded in render objects.** Replaced and compound elements (e.g. `<input>`, `<textarea>`) override `IntrinsicStyle()` to return a sparse `style.Style` with properties such as `Display: InlineBlock` or `OverflowX: Clip` that authors cannot override.
5.  **Event Bubbling:**
    *   Events must strictly follow the Capture -> Target -> Bubble sequence. 
    *   Avoid introducing "IntentEvents" (a deprecated concept from v1). Rely on the `Synthesizer` to convert raw inputs into semantic events.
6.  **Inline Layout (LayoutNG):**
    *   Inline formatting contexts (IFC) must use a flat representation of `InlineItem`s rather than a recursive tree walk during line breaking.
    *   Text nodes must be collapsed and shaped before layout.
    *   `inline-block` elements are treated as atomic inlines that run their own block layout internally.
7.  **Flex Layout (LayoutNG):**
    *   Flex layout utilizes a two-pass approach (`FlexAlgorithm`): a measure pass to determine flex base sizes and a layout pass to resolve flexible lengths and alignment.
    *   All mutable state for the algorithm lives in `FlexLineBuilder` (`flex_builder.go`), which handles item collection, line chunking, flexible length resolution (freeze and restart), and axis alignment.
    *   The algorithm must use logical geometry (`MainSize`, `CrossSize`) to remain agnostic of the `flex-direction`.
    *   To maintain performance, the resolution loop must utilize the "freeze and restart" strategy for items hitting their min/max constraints, ensuring $O(N)$ or near-$O(N)$ complexity.
8.  **Focus Management:**
    *   Focus state and navigation logic operate strictly on the logical `dom.Node` tree.
    *   The `focus.Manager` uses the `dom.Focusable` and `dom.Disableable` interfaces to determine interactivity.
    *   Spatial navigation queries physical geometry by accessing the `RenderObject().Fragment()` of the logical nodes, rather than using the render tree as the primary source of truth for focus.
9.  **List Layout (Virtual Markers):**
    *   `ListAlgorithm` implements the `DisplayListItem` layout using a specialized two-column row layout.
    *   The marker is synthesized as a virtual, transient text fragment directly during layout (based on `ListStyleType`) to avoid creating phantom nodes in the render tree.
    *   Ordinal calculation for numbered lists uses an $O(N)$ sibling walk of the logical tree.
10. **Table Layout:**
    *   Table layout utilizes a two-pass approach (`TableAlgorithm`): a measurement pass to determine intrinsic grid column sizing (accounting for `ColSpan` and `RowSpan`), followed by a layout pass to resolve rows and place cells.
    *   All mutable state for the two passes lives in `TableFragmentBuilder` (`table_builder.go`), which handles section grouping, the `tableGrid` (including per-junction border-overlap flags), column min/max sizing, `DistributeSpan`, `ResolveWidths`, `AdjustRowOffset`, and `GetCellShift`.
    *   Cells act as independent block formatting contexts constrained by the rigid widths dictated by the parent table.
    *   Table routing is strictly driven by `style.DisplayTable`, `style.DisplayTableRow`, and `style.DisplayTableCell`. No specialized render nodes exist.
    *   **Implicit Border Collapse** — Kite tables always use border-collapse semantics. The coordinate rules are:
        *   Cells are placed at `(X=0, Y=0)` within the row (the `BoxFragmentBuilder`'s automatic `border.Top` inset is reset to `0`). This makes cell borders physically overlap with row borders at the same terminal pixel, allowing the paint engine's junction resolver to merge them.
        *   `GetCellShift` returns `1` when a cell's left border and the previous cell's right border both exist, shifting the new cell left by 1 so both borders share one terminal column.
        *   For **spanning cells** (`ColSpan > 1`), each `ColJunctionOverlap[j] == true` inside the span reduces `cellWidth` by 1 (the junction column does not exist inside a spanning cell).
        *   `AdjustRowOffset` returns `-1` when consecutive rows both have touching borders, collapsing them onto a single shared terminal row.
        *   The table width and distributable column budget are adjusted only for *actual* overlapping junctions (tracked in `tableGrid.ColJunctionOverlap`, `LeftEdgeHasOverlap`, `RightEdgeHasOverlap`), never unconditionally.
        *   When `LeftEdgeHasOverlap` is true, sections are placed at `X = padding.Left` (no left-border gap) and `childAvailWidth` is expanded so column 0 can share the table's left border column.
11. **UA Shadow Subtree (ADR-009):**
    *   Host elements (replaced or compound widgets) attach a closed UA shadow subtree via `Element.AttachUARoot(root)` in their constructor.
    *   **Public traversal APIs must never expose UA-subtree nodes.** `ChildNodes()`, `FirstChild()`, `LastChild()`, `Children()`, and `GetElementByID()` are always UA-invisible; author code has no path to reach UA internals.
    *   **Engine phases use `dom.LayoutChildren(n)`** (not `n.ChildNodes()`) to walk the union of public children and UA root's children. This iterator is the single authoritative walker for Sync, Style, Layout, and Paint; it must never be used in author-facing code.
    *   Every node in the UA subtree has its `self` back-pointer set to the host element (`setOuterRecursive`) so `event.Target()` and identity queries always collapse to the host (reuses ADR-0036).
    *   UA nodes must never implement `dom.Focusable`. `focus.Manager` uses the public `Children()` iterator and therefore inherently skips UA nodes — no focus-engine changes are needed.
    *   `dom.IsUANode(n)` reports `true` for any node stamped by `AttachUARoot`. O(1) via the `inUASubtree` flag on `baseNode`.
12. **Paint — Overflow Clipping (ADR-011):**
    *   `paint/engine.go::paintFragment` checks `ComputedStyle.OverflowX` / `OverflowY` after painting a fragment's own background and border. If either axis is non-`Visible`, it calls `surface.Clip(contentBoxRect)` and passes the returned sub-surface to the child recursion.
    *   A fragment's own background fill and border decoration are always painted onto the **unclipped** parent surface — a node's overflow property never clips its own border-box.
    *   For asymmetric overflow (one axis `Visible`, the other clipped), the clip rect spans the full surface extent on the `Visible` axis so that axis remains truly unconstrained.
    *   **`resolveBorders` invariant:** `PaintEngine.resolveBorders` is called exactly once, on the **root** `Surface`, after the full fragment tree has been painted. It must never be called on a clipped sub-surface, because the junction resolver must see the complete set of border cells across the entire viewport.
13. **Scroll Model (ADR-012):**
    *   Every `dom.Element` exposes `Scroll()`, `ScrollTo(x, y)`, and `ScrollBy(dx, dy)`.
    *   Scroll state is held in a lazy `*scrollState` pointer on the element, allocated only when needed.
    *   Programmatic scroll is valid on any element; however, paint only applies translation if the computed style indicates the element is a scroll container (`overflow: scroll` or `overflow: auto`).
    *   Paint **clamps on read**: the stored scroll offset is the raw author intent, which paint clamps to the actual content extent at render time.
    *   Mutating scroll marks the render object `DirtyScroll`. Paint clears this flag.
14. **Overlay System (ADR-008):**
    *   The `dom.Document` maintains an explicit list of overlays via `ShowOverlay` and `HideOverlay`. Overlays are sorted by `zIndex`.
    *   `element.Overlay` uses a custom layout algorithm (`OverlayAlgorithm`) that positions itself relative to an `Anchor` element using `GetBoundingClientRect`.
    *   **Smart Flipping:** Overlays automatically flip to the opposite side if they overflow the viewport. If they overflow both sides, they use **Best Fit** logic, choosing the side with the most available space.
    *   Overlays must default to `Display: InlineBlock` to ensure they shrink-wrap to their content, preventing false-positive horizontal overflows.
    *   `element.Dialog` provides a full-screen modal that uses the overlay system and traps keyboard focus via `focus.Scope`.
    *   The engine respects the calculated physical offsets of overlays during the paint phase. Standard elements used as overlays (without a custom positioner) fall back to margin-based absolute positioning relative to the viewport.
15. **Animation System (ADR-021):**
    *   The `/animation` package provides imperative property interpolation completely decoupled from the style engine.
    *   The `engine.Engine` tracks and ticks active animations at the start of every frame loop.
    *   Animations are self-scheduling: if any animations are active, the engine calls `RequestFrame()` to keep the loop running at 60FPS.
    *   `Tween[T]` is the primary animator, combining a `Duration`, `EasingFunction`, and `Interpolator[T]` with an `OnUpdate` callback.
    *   Property updates within an `OnUpdate` callback should typically use `element.RawStyle()` as a base, mutate it, and call `element.Style(s)` to trigger a style-dirty flag.

## 🗺️ Concern → File Map

Use this table as the first lookup before grepping. It maps the most common engineering concerns to the authoritative source file(s).

| Concern | Primary File(s) |
|---|---|
| **DOM tree structure & node lifecycle** | `dom/node.go`, `dom/element.go`, `dom/interfaces.go` |
| **DOM document & factory methods** | `dom/document.go` |
| **UA shadow subtree primitives** | `dom/ua.go` (ADR-009) |
| **Element scroll state (Scroll/ScrollTo/ScrollBy)** | `dom/scroll_controller.go` (ADR-012) |
| **`outer` back-pointer / identity (ADR-0036)** | `dom/outer.go` |
| **Style Optional[T] wrapper** | `style/optional.go` |
| **Style property declarations** | `style/style.go` |
| **Computed style (post-resolver values)** | `style/computed.go` |
| **Style cascade & resolver** | `style/resolver.go`, `style/cascade.go` (ADR-010) |
| **Border fluent API & metadata** | `style/border.go` |
| **Grid style properties and aliases** | `style/grid.go` |
| **Render object interfaces & dirty flags** | `render/object.go`, `render/dirty.go` |
| **Render box / text nodes** | `render/box.go`, `render/text.go` |
| **Render view (root container)** | `render/view.go` |
| **Custom render object hook** | `render/block.go` (CustomObjectProvider) |
| **Layout fragment geometry** | `layout/geometry.go` |
| **Block layout algorithm** | `layout/block.go` |
| **Inline layout / IFC** | `layout/inline.go` |
| **Flex layout algorithm** | `layout/flex.go`, `layout/flex_builder.go` |
| **Grid layout algorithm** | `layout/grid.go` |
| **Grid layout builder** | `layout/grid_builder.go` |
| **List layout (virtual markers)** | `layout/list.go` |
| **Table layout algorithm** | `layout/table.go`, `layout/table_builder.go` |
| **Layout entry-point (NG dispatcher)** | `layout/ng.go` |
| **Layout child constraint space builder** | `layout/resolve.go` (BuildChildSpace, ADR-018) |
| **Paint engine & overflow clipping** | `paint/engine.go` (ADR-011) |
| **Paint framebuffer & surface** | `paint/framebuffer.go`, `paint/types.go` |
| **Border intersection resolver** | `paint/engine.go` (`resolveBorders`) |
| **Event types & interfaces** | `event/events.go` |
| **Event dispatcher (capture/bubble)** | `event/dispatcher.go` |
| **Raw-input → semantic event synthesis** | `event/synthesizer.go` |
| **Focus manager & tab navigation** | `focus/focus.go` |
| **Spatial (arrow-key) navigation** | `focus/spatial/spatial.go` |
| **Hardware cursor state & Provider** | `cursor/cursor.go` |
| **Cursor from IFC fragment** | `cursor/from_text_fragment.go` |
| **Cursor byte-offset hit-test** | `cursor/offset_at_point.go` |
| **Editor buffer (text model)** | `editor/buffer.go` |
| **Overlay system & algorithms** | `element/overlay.go`, `layout/overlay.go` (ADR-008) |
| **Modal Dialogs** | `element/dialog.go` |
| **Engine frame loop** | `engine/engine.go` |
| **Animation system & Tweens** | `animation/animation.go` |
| **Engine cursor wiring** | `engine/cursor.go` |
| **Engine job / microtask queue** | `engine/job.go` |
| **Backend interface** | `backend/backend.go` |
| **Mock backend (for tests)** | `backend/mock/mock.go` |
| **Element base & fluent API** | `element/element.go` |
| **Shared text-control mechanics** | `element/text_control.go` (ADR-013) |
| **`<input>` element** | `element/input.go` |
| `<textarea>` element | `element/textarea.go` |
| `<select>` element | `element/select.go` |
| `<box>` / `<span>` elements | `element/box.go`, `element/span.go` |
| **`<br>` element** | `element/br.go` |
| **List elements (ul/ol/li)** | `element/list.go` |
| **Table elements (table/tr/td/…)** | `element/table.go` |
| **Text element** | `element/text.go` |
| **Text shaping (grapheme clusters)** | `text/shape.go`, `text/cluster.go` |
| **Key codes & modifiers** | `key/key.go`, `key/mod.go` |
| **Regression test suite** | `tests/regressions/` |
| **ADR documents** | `docs/adrs/` |
| **DevTools Server, endpoints & profiling** | `devtools/server.go`, `devtools/server_test.go` |
| **VDOM primitive wrappers & API** | `extras/kitex/kitex.go` |
| **Kitex hooks (UseState, UseRef, UseMemo, UseEffect, UseEffectCleanup, UseLayoutEffect, UseLayoutEffectCleanup, UseReducer, UseCallback)** | `extras/kitex/hooks.go` |
| **Kitex Context System (CreateContext, UseContext, Provider)** | `extras/kitex/context.go`, `extras/kitex/provider.go` |
| **Kitex automatic memoization (complexity, deepEqualProps)** | `extras/kitex/kitex.go` (memoization helpers section) |
| **Global state store & subscriber API** | `extras/kites/store.go` |
| **Kites global state integration hook** | `extras/kites/hooks.go` |
| **Stack Navigation component & interfaces** | `extras/flight/flight.go`, `extras/flight/stack.go` |
| **Navigation context & hooks** | `extras/flight/hooks.go` |
| **Wind query cache client & context** | `extras/wind/client.go` |
| **Wind query hook (`Use`)** | `extras/wind/query.go` |
| **Wind mutation hook (`UseMutation`)** | `extras/wind/mutation.go` |

## 📋 Task Workflow

When the agent is assigned a task `./tasks/task_list.md`, it **must** update that file's status row to **`In Progress`** as the very first action — before reading the task's Markdown file, before exploring the codebase, and before writing any code. Failing to do this is a workflow violation regardless of how well the implementation turns out.

At the **end** of every task the agent must also:

1. **Update the Concern → File Map** in this file if the task introduced a new file, deleted a file, renamed a file, or moved a concern to a different file. Every row in the map must remain accurate after the task completes.
2. **Keep `README.md` consistent** with any user-visible API or package changes.

## 🧑‍💻 Coding Conventions

1.  **Declarative UI API:**
    *   **Always** use the declarative functional constructors in the `element` package (e.g., `element.Box()`, `element.UL()`, `element.Span()`) when constructing UI trees.
    *   Avoid using manual `NewBox(doc)` or `AppendChild` calls unless you are implementing a new custom element type or low-level DOM logic.
    *   Leverage variadic children and automatic string boxing for concise UI code.
2.  **Continuous Documentation Maintenance:**
    *   **Always** keep `README.md` and `AGENT.md` up-to-date. If you introduce new packages, modify core architectural patterns, or change significant dependencies, you must update these files to reflect the new state of the project.
3.  **Modern Go Features:** 
    *   Utilize Go 1.24+ standard library features.
    *   Use iterators (`iter.Seq[T]`) for traversing collections, such as `Node.Children()`.
4.  **Interfaces and Embedding:** 
    *   Favor small, composable interfaces (e.g., `dom.Node`, `dom.Element`, `dom.TextNode`).
    *   When creating internal implementations, use unexported structs (e.g., `element`) and assert compile-time interface compliance (`var _ Element = (*element)(nil)`).
    *   **Stable interface assertions are mandatory for every public interface implementor.** Every concrete type that satisfies a public interface must carry a `var _ InterfaceName = (*ConcreteType)(nil)` guard at package scope — not just the ones that seem tricky. This ensures the compiler catches broken contracts immediately rather than at the call site.
5.  **Documentation:** 
    *   All packages must contain a `doc.go` file summarizing the package's responsibility.
    *   Reference ADRs (Architecture Decision Records) in docstrings when touching core mechanics (e.g., `ADR-0036` for DOM adoption).

## 🧪 Testing Strategy

1.  **Table-Driven Tests:** Prefer table-driven structures using the standard `testing` package.
2.  **Mocking:** Use the `backend/mock` package for testing the Render and Paint pipelines without requiring a physical TTY or terminal emulator.
3.  **Benchmarks:** Any changes to `/layout`, `/style` resolving, or `/paint` logic must be accompanied by `testing.B` benchmarks, as performance is critical in a 60FPS UI loop.
4.  **No Panics:** Ensure test assertions do not result in raw panics. Handled disconnected/nil states gracefully in DOM manipulation tests.
5.  **Always run tests with a timeout:** Use `go test -timeout 30s ./...` (or a per-package equivalent) so that a deadlock or hung goroutine causes a clean failure instead of blocking the terminal indefinitely. Never invoke `go test` without `-timeout`.
6.  **Regression file headers:** Every file under `tests/regressions/` must begin with a package-level comment that states which component(s) it covers and the originating task or bug. Use this format:
    ```go
    // Regression tests for <ComponentName> — covers <TSK-XXX / brief description>.
    ```
This makes it immediately clear which source files are relevant without reading the test bodies.



## Source Map

This source map summarises the repository packages, their responsibilities, and key files. It is generated from the repository scan and mirrors `SOURCE_MAP.md`.

### Packages

- **animation** — Path: `animation/`
    - Description: Imperative property interpolation, easing functions, and tweening utilities. (See `animation/doc.go`)
    - Key files: `animation/doc.go`, `animation.go`, tests

- **backend** — Path: `backend/`
    - Description: Defines the `Backend` interface and frame lifecycle hooks; supplies `Surface` for paint engine. (See `backend/doc.go`)
    - Key files: `backend/doc.go`, `backend.go`, `mock/`, `uv/`

- **cursor** — Path: `cursor/`
    - Description: Unified hardware cursor abstraction and helpers (`FromTextFragment`) for translating byte offsets to cell coordinates. (See `cursor/doc.go`)
    - Key files: `cursor/doc.go`, `cursor.go`, `offset_at_point.go`, `from_text_fragment.go`, tests

- **dom** — Path: `dom/`
    - Description: Logical DOM node tree, lifecycle, adoption, UA shadow subtree, and scroll semantics. (See `dom/doc.go`)
    - Key files: `dom/doc.go`, `document.go`, `node.go`, `element.go`, `text_node.go`, `outer.go`, `scroll_controller.go`

- **editor** — Path: `editor/`
    - Description: Text editing and `Buffer` utilities; Unicode-safe mutations and navigation. (See `editor/doc.go`)
    - Key files: `editor/doc.go`, `buffer.go`, tests

- **element** — Path: `element/`
    - Description: High-level UI components (Box, Span, Input, TextArea, Table, List, Overlay, Dialog, Button) and declarative builders. (See `element/doc.go`)
    - Key files: `element/doc.go`, `element.go`, `input.go`, `textarea.go`, `list.go`, `table.go`, `overlay.go`, `dialog.go`, `button.go`, tests

- **engine** — Path: `engine/`
    - Description: Main event loop, frame pipeline (Tasks → Style → Layout → Paint → Sync), task queues, and worker pool. Coordinates other packages. (See `engine/doc.go`)
    - Key files: `engine/doc.go`, `engine.go`, `clock.go`, `cursor.go`, `job.go`, tests

- **event** — Path: `event/`
    - Description: Event types, Dispatcher, Synthesizer and key/wheel/scroll semantics. (See `event/doc.go`)
    - Key files: `event/doc.go`, `dispatcher.go`, `events.go`, `synthesizer.go`, tests

- **examples** — Path: `examples/`
    - Description: Example applications and usage demos. Subpackages: `app1`, `button`, `flex`, `input`, `list`, `table`, `textarea`.

- **extras/kitex** — Path: `extras/kitex/`
    - Description: Lightweight, fully-typed Virtual DOM (VDOM) primitive wrappers that map 1:1 to real element package instances. (See `extras/kitex/doc.go`)
    - Key files: `extras/kitex/doc.go`, `kitex.go`, `hooks.go`, tests

- **extras/kites** — Path: `extras/kites/`
    - Description: Lightweight, thread-safe external state store and kitex integration hook for global state management. (See `extras/kites/doc.go`)
    - Key files: `extras/kites/doc.go`, `store.go`, `hooks.go`, tests

- **extras/flight** — Path: `extras/flight/`
    - Description: Stack-based type-safe navigation and hooks (push/pop) with automated focus isolation. (See `extras/flight/doc.go`)
    - Key files: `extras/flight/doc.go`, `flight.go`, `hooks.go`, `stack.go`, tests

- **extras/wind** — Path: `extras/wind/`
    - Description: Async data fetching, caching, query invalidation, and mutation state management. (See `extras/wind/doc.go`)
    - Key files: `extras/wind/doc.go`, `client.go`, `query.go`, `mutation.go`, tests


- **focus** — Path: `focus/`
    - Description: Focus management, `focus.Manager`, reasons, scope stack, and spatial navigation. (See `focus/doc.go`)
    - Key files: `focus/doc.go`, `focus.go`, `spatial/`, tests

- **key** — Path: `key/`
    - Description: Key event representation and helpers (Key struct, matching helpers). (See `key/key.go`)
    - Key files: `key/key.go`, `mod.go`

- **layout** — Path: `layout/`
    - Description: Layout algorithms and formatting contexts (Block, Flex, Inline/IFC, List, Table, Overlay). Produces fragment trees consumed by paint. (See `layout/doc.go`)
    - Key files: `layout/doc.go`, `flex.go`, `block.go`, `inline.go`, `table.go`, `overlay.go`, `builders.go`, tests

- **paint** — Path: `paint/`
    - Description: Paint phase: rasterises layout fragments into terminal cells, clipping and border resolution invariants. (See `paint/doc.go`)
    - Key files: `paint/doc.go`, `framebuffer.go`, `engine.go` (tests reference), `resolver_test.go`

- **render** — Path: `render/`
    - Description: Render-object layer bridging DOM with layout/style/paint, tracking dirty state and computed styles. (See `render/doc.go`)
    - Key files: `render/doc.go`

- **style** — Path: `style/`
    - Description: Style value types, computed resolution, four-layer cascade and fluent helpers. (See `style/doc.go`)
    - Key files: `style/doc.go`, resolver and sheet implementations, tests

- **text** — Path: `text/`
    - Description: Grapheme cluster segmentation, shaping, cell-width measurement, and line-break classification. (See `text/cluster.go`)
    - Key files: `text/cluster.go`, `text/shaper.go`, `shape.go`, tests


### Docs and Project Files

- `docs/`: design docs, ADRs, roadmap and INSTRUCTIONS. Useful for architecture context.
- `logs/`: runtime logs and request dumps (not source code).
- `tasks/`: task tracking and design task templates.
- `README.md`, `WORKSPACE.md`, `go.mod`, `go.sum`: project-level metadata and module configuration.

---
