# Kite Architecture

This document serves as the high-level architectural overview for Kite (v2) based on our design sessions.

## 1. Core Principles
- **Terminal UI Framework:** A modern, DOM-like terminal UI framework for Go. It brings web-like development paradigms to the terminal environment.
- **In-memory operation:** No external database/storage requirements.
- **Clear Separation of Concerns:** Strict package isolation between DOM, Style, Layout, Paint, and Render layers to maintain an efficient rendering pipeline.
- **Performance-Oriented:** The pipeline targets 60FPS on the main thread, with expensive or asynchronous operations handled in a concurrent worker pool managed by a global `terminal.Scheduler`. Promises seamlessly bridge background work back to the main thread via microtasks.

## 2. Rendering Pipeline Overview

The framework operates via a central nervous system called the **Engine (`/engine`)**. The engine acts purely as a coordinator, running a continuous frame loop that orchestrates a unified pipeline:

1. **Input Buffering & Coalescing:** Collect raw input events from the backend into a buffer. Just before the frame renders, drain the buffer and coalesce high-frequency events (e.g., aggregate wheel deltas, squash mouse moves) into semantic events, dispatching them through the DOM.
2. **Task Draining:** The engine coordinates with the `terminal.Scheduler` to drain macrotasks (budget-capped) and microtasks (drained completely) on the main thread. This executes user events, effect hooks, and resolved Promise callbacks.
3. **Synchronize Phase (Pre-Layout):** Walk the logical DOM and project structural changes into the render tree. The Engine uses an internal map (`map[dom.Node]render.Object`) to link the independent logical and physical trees. It flags dirty layout and style nodes.
4. **Style Phase:** Traverse the render tree to resolve inherited and explicit styles into `Computed` values.
5. **Layout Phase:** Traverse the dirty nodes, executing LayoutNG-inspired algorithms (Block, Flex, Inline) to produce immutable physical `Fragment` trees.
6. **Paint Phase:** Draw the resulting `Fragment` trees onto the framebuffer via absolute coordinates and clipping.
7. **Commit:** Push the framebuffer surface to the terminal via the decoupling backend (`/backend`).

## 3. Subsystems

### 3.1. DOM (Logical Tree)
- **Strict Isolation:** The logical DOM has zero knowledge of the physical rendering pipeline. It does not contain any references to `render.Object`.
- **View Proxy:** To query physical properties like `GetBoundingClientRect` or `ComputedStyle` without coupling, the DOM defines a `dom.View` interface. The `engine.Engine` implements this interface and injects itself into the `dom.Document`. Elements proxy layout queries up to this View.
- **Terminal Context:** The DOM has no knowledge of OS clipboards or render loops. It exposes a `Terminal()` accessor that returns a `terminal.Terminal` interface, which is implemented by the Engine to bridge OS capabilities (Clipboard, Window frames, Hardware Cursor, Scheduler).
- **Hardware Cursor:** The hardware cursor is managed via a hybrid, style-driven model. Elements declare their cursor preferences (shape, color, blink) via a `Cursor` struct in their `style.Style`. By default, the Engine automatically translates a collapsed `dom.Selection` into physical coordinates and applies the focused element's cursor style. The legacy `cursor.Provider` interface has been removed.
- **Responsibility:** Maintains the structural tree and interactivity states (`Focusable`, `Disabled`).
- **Core Entities:** `Document`, `Element`, `TextNode`.
- **Adoption & Identity:** Uses a self back-pointer (`outer`) set during the attach walk. Ensures `event.Target()` and `GetElementByID()` return the outermost user-visible wrapper (useful for custom widgets).
- **Events:** Responsible for the Capture -> Target -> Bubble event propagation model. Uses $O(1)$ checks for connectivity (`IsConnected()`).
- **Selection API:** The Document maintains a global `dom.Selection` state containing logical `dom.Range` boundaries across `dom.Text` nodes. This acts as the single source of truth for user text highlights.
- **UA Shadow Subtree:** Replaced and compound elements (e.g., `<input>`, `<textarea>`, future `<checkbox>`, `<radio>`, `<select>`) own a closed UA subtree via an internal `uaRoot` field on `dom.Element` (ADR-009). The subtree is invisible to public traversal (`Children()`, `GetElementByID()`), invisible to event dispatch, and never focusable; engine phases (Sync/Style/Layout/Paint) walk it as if it were a regular child. Identity retargeting reuses the existing `outer` back-pointer so `event.Target()` always resolves to the host. Text-based controls locally manage their own `SelectionStart` and `SelectionEnd`, projecting this to the document's global selection only for visual rendering, protecting the shadow tree.
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
  - **Grid Formatting Context (GFC):** Lays out elements in a two-dimensional grid using a 3-phase matrix builder (Track Sizing, Auto-Placement, Layout). Supports explicit sizing, fractional (`fr`) distributions, and `gap`s.
  - **Inline Formatting Context (IFC):** Lays out text and atomic inlines horizontally, wrapping them into line boxes. Uses a flat representation of `InlineItem`s.

### 3.4. Render Pipeline (`/render`)
- **Engine Mapping:** The physical Render Tree is completely detached from the logical DOM Tree. The `engine.Engine` maintains the mapping between them using a dictionary structure. `render.Object` maintains a strongly-typed back-pointer to its source `dom.Node`.
- **Replaced & Compound Elements:** Form controls and other compound widgets compose their visuals as a closed UA Shadow Subtree on the logical element (ADR-009). They get a plain `render.Box` and rely on standard formatting contexts — no per-widget render object or layout algorithm. Text-based form controls (`<input>`, `<textarea>`) share a common `textControlBase`. Toggle controls (`<checkbox>`, `<radio>`) manage hidden text nodes for their glyphs. Buttons (`<button>`) handle click semantics and provide visual feedback for active/pressed states. Complex composites like `<select>` combine a shadow trigger button with dynamic, out-of-flow `element.Overlay` popups and temporary `focus.Scope` trapping.
- **Selection Resolution:** Implements a decoupled "Push Model" for text selection. The active `dom.Selection` is resolved against the layout fragment tree to produce an independent list of physical `paint.Rect`s *before* the paint phase begins. This avoids $O(N)$ DOM lookups during text rasterization and ensures immutable layout fragments remain perfectly cacheable.
- **Responsibility:** The visual bridge between the logical DOM and physical layout.
- **Stateless Styling:** The Render Engine stores the resolved `style.Computed` state, but delegates the reading of declarative input styles (`RawStyle`, `DefaultStyle`, `IntrinsicStyle`) directly to the associated `dom.Node`.
- **Node Mirroring:** It strictly mirrors the DOM structure using a unified `render.Box` or `render.Text` (no explicit block/flex types here; the engine delegates algorithms at layout time based on `ComputedStyle.Display`).
- **Dirty Tracking:** Carries lifecycle synchronization flags (`NeedsSync`, `DirtyStyle`, `DirtyLayout`) without doing actual math calculations itself.

### 3.5. Event System (`/event`)
- **Responsibility:** Dispatching semantic interactions and input routing.
- **Event Coalescing:** The engine decouples raw input arrival from DOM dispatch. Incoming events are buffered and coalesced per-frame (e.g., squashing intermediate mouse movements, summing fast scroll wheel deltas) to guarantee UI resilience under high-frequency inputs (ADR-015).
- **Phases:** Advanced dispatcher supporting Capture, Target, and Bubble phases.
- **Synthesizer:** Translates raw terminal input (e.g., from Charmbracelet's `ultraviolet`) into semantic events (like key combinations or clicks). Supports multi-MIME rich data (e.g., images) by delegating complex handshakes to terminal extensions.

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
- **Selection Masking:** The paint context accepts a pre-calculated list of selection bounding rectangles. When rendering cells, it acts as an overlay mask, inverting or restyling colors for any cell that intersects a selection rect.
- **Border Post-Processing:** To automatically form correct Unicode junctions (e.g., `┼`, `├`) without manual coordinate math in layout, the `PaintEngine` runs a global $O(W \times H)$ post-processing pass over the framebuffer. Every cell explicitly tagged as a border is resolved against its cardinal neighbors. Junction merging for overlapping borders of varying weights is handled via a strict "Heaviest Style Wins" precedence rule (using an explicit `BorderStyle` enum stored per-cell). The resolver runs once on the root surface only; it must never be invoked on a clipped sub-surface.
- **Backend:** Decouples Kite from the actual terminal emulator. The `backend` package is completely agnostic of internal types (like `paint.Cell` or `style.Cursor`). Implementations like `ultraviolet` and `mock` consume a generic `backend.Buffer` populated by the `Engine` at the end of each frame. This strict isolation allows backends to focus purely on I/O and protocol translation.

### 3.8. Developer Tools (`/devtools`)
- **Responsibility:** Provide utilities to inspect, test, and debug Kite applications without bloating the core runtime.
- **Inspector (`/devtools/inspector`):** A lightweight HTTP server utilizing Server-Sent Events (SSE) to stream the live logical DOM tree and computed styles. The frontend is built with **Preact and Vite** (ADR-020), compiled into a single embedded HTML file to provide a rich, component-based UI without requiring Node.js for end-users. It supports a generic `Extension` interface allowing external packages (like `kitex`) to inject additional inspection tabs and payloads out-of-band.
- **Profiler:** A built-in performance tracing system using a **Hybrid Architecture** (ADR-019). The core engine delegates phases to a `Pipeline` interface wrapped in a Decorator to capture high-level phase times cleanly. For deep-tree granular timings (like specific layout algorithms), an inlineable `TraceContext` struct is injected into contexts (e.g., `LayoutContext`), ensuring zero allocations when disabled. DevTools features an internal Flamechart/Waterfall view and exports data in the Chrome Trace Event Format (JSON). Startup tracing can be activated via `engine.WithProfiler(true)`.
- **X-Ray Mode:** An optional rendering flag built into the core `paint` engine but toggled via devtools. Overlays colored bounding boxes (margin, padding, content) for visual layout debugging.
- **Test Environment (`/devtools/testenv`):** A headless testing harness that wraps the existing `backend/mock`. Provides high-level APIs for structural DOM assertions (`GetNodeByID`), simulated input routing (`Type`, `Click`), layout verification, and golden/visual snapshot testing (producing HTML or ANSI dumps).

### 3.9. Overlay System
- **Responsibility:** Management and rendering of out-of-flow components like dropdowns, tooltips, and modal dialogs.
- **Document Integration:** The `dom.Document` maintains an explicit list of overlays via `ShowOverlay` and `HideOverlay`. Overlays are sorted by `zIndex`.
- **Anchored Positioning:** `element.Overlay` uses a custom layout algorithm that queries the physical bounds of an `Anchor` element (via `GetBoundingClientRect`) to position itself.
- **Smart Flipping:** If an overlay would overflow the viewport, the layout engine automatically flips it to the opposite side or chooses the "best fit" placement with the most available space.
- **Modal Dialogs:** `element.Dialog` provides a full-screen modal container that automatically traps focus using `focus.Scope`.

### 3.10. Animation System (`/animation`)
- **Responsibility:** Providing smooth, time-based property interpolation.
- **Paradigm:** The framework uses an explicit, imperative animation architecture (ADR-021) rather than declarative CSS transitions. This keeps the Style and Layout engines pure and stateless.
- **Engine Integration:** Developers register generic `Tween[T]` objects with the `Engine`. The engine ticks active animations at the start of its frame loop. If animations are active, the engine automatically invokes `RequestFrame()` to maintain a continuous 60FPS loop, returning to an idle sleep state once all animations complete.

### 3.11. Reactive UI Framework (`/extras/kitex`)
- **Responsibility:** Provide a high-level, React-like Virtual DOM architecture for complex application state management.
- **Components:** Custom components are defined as pure functions strongly typed with Go Generics (`react.FC<P>`). The framework wraps them in lightweight VDOM boundary nodes.
- **State Management:** Components declare reactive state using hook functions (e.g., `kitex.UseState`). The engine manages an internal execution stack to persist hook context implicitly, removing the need to pass context variables through rendering closures.
- **Refs:** The framework provides `kitex.UseRef` and `kitex.CreateRef` hooks, backed by a generic `Ref[T]` type, to safely retain mutable references (such as real DOM nodes) across renders without triggering state updates.
- **Memoization:** Like the React Compiler, `kitex` utilizes an automatic, Just-In-Time (JIT) memoization architecture. Components calculate their structural complexity bottom-up during allocation. When a component's complexity exceeds a safe threshold, the engine automatically memoizes it, bypassing future `RenderFn` executions and reconciliation passes by performing a depth-limited reflection check (`deepEqualProps`) on properties. An explicit `UseMemo` hook is also available.
- **Side Effects (ADR-026):** The framework provides dual-variant effect hooks — `UseEffect`/`UseEffectCleanup` for post-commit effects and `UseLayoutEffect`/`UseLayoutEffectCleanup` for synchronous post-reconciliation effects. Layout effects fire synchronously after `reconcile()` within the `OnComponentDirty` callback. Regular effects are deferred via `engine.PostMacro` and flushed at the start of the next frame's Task Draining phase, matching React's `flushPassiveEffects` guarantee. A `flushPendingEffects()` call at the top of `OnComponentDirty` ensures effects from frame N complete before frame N+1's reconciliation.
- **Reducers:** `UseReducer[S, A](reducer, initial)` provides a `(state, action) → state` pattern for complex state machines, wrapping `UseState` internally with a dispatch function that applies the reducer.
- **Callback Memoization:** `UseCallback[T](callback, deps)` stabilizes function references across re-renders, delegating to `UseMemo` internally.
- **Context System (ADR-026):** A React-style context API for sharing state without prop-drilling. `CreateContext[T](defaultValue)` produces a typed `Context[T]` identity. `ProviderNode[T]` pushes a `contextEntry` onto the context's internal stack during render; `UseContext[T]` reads from the stack on first mount (O(1)), then from a stored provider reference on re-renders (O(1)). Providers maintain subscriber sets and mark consumers dirty on value change, bypassing automatic memoization via a `!oldComp.IsDirty()` guard in `Update()`. The `lastSeenValue` on hook state prevents duplicate re-renders for consumers already processed during normal reconciliation.
- **Component Lifecycle — `Destroy()`:** When the reconciler removes a component (unmount, tag mismatch), it calls `Destroy()` on the `ComponentNode` before DOM removal. `Destroy()` runs all effect cleanups, unsubscribes from context providers, and recursively destroys nested component subtrees.
- **Reconciliation:** On state change, the component re-executes. A VDOM diffing engine compares the new lightweight tree against the previous frame and generates a minimal set of imperative patches (mutations) to the underlying logical `dom.Node` tree.
- **VDOM Primitives:** The `kitex` package defines lightweight, fully-typed VDOM representations (e.g., `kitex.Button(kitex.ButtonProps{...})`) that map 1:1 to the real, heavy DOM nodes in the root `element` package. The reconciler translates these VDOM nodes into imperative updates on the underlying `element` nodes.
- **Convenience Hooks (`/extras/kitex/hooks`):** Terminal-specific hooks built on the core primitives: `UseFocus(ref)` subscribes to focus/blur events and returns a reactive boolean; `UseKeyboard(handler, deps)` registers scoped keyboard handlers. These demonstrate composability and are shipped as a separate sub-package.

### 3.12. Global State Management (`/extras/kites`)
- **Responsibility:** Provide a thread-safe, external state management solution ("Kite Store") decoupled from the VDOM.
- **Architecture:** Follows a Zustand/Redux-style "External Store" model. Global state is instantiated via `kites.Create[T](initialState)`, wrapping the state in a `sync.RWMutex`. This allows concurrent mutations from background goroutines or network handlers without data races.
- **Kitex Integration:** Components subscribe to state slices using the `kites.Use(store, selector)` hook. The selector function extracts a specific slice of the global state.
- **Render Optimization:** The `kites.Use` hook leverages Go's `comparable` constraint on the selector's return type. When the global store updates, the hook evaluates the new slice; it only triggers a component re-render if the new slice differs from the previous one, providing automatic performance bailouts for unaffected components.

### 3.13. Navigation & Routing (`/extras/flight`)
- **Responsibility:** Provide a stack-based navigation system for switching between full-screen views in `kitex` applications.
- **Architecture:** Bypasses web-style URL/Path routing in favor of a "Stack Navigator" paradigm, which is more natural for TUIs (drill-down and pop). The `flight.Stack` component maintains a history slice of pushed routes.
- **Type Safety:** Routes are defined as an empty interface (`flight.Route`). Developers define routes using standard Go structs (e.g., `type ProfileRoute struct { ID string }`), ensuring strict type safety for route parameters without resorting to maps or generic dictionaries. The `RenderRoute` function resolves the component via a standard Go type switch.
- **Explicit Interaction:** The framework does not magically hijack keys (like `Esc`) for navigation. Developers explicitly bind navigation actions using `kitex` keyboard hooks.
- **Focus Isolation:** The `flight.Stack` automatically wraps the currently active route in an `element.FocusScope`. This guarantees that when a new screen is pushed, keyboard navigation (Tabbing) cannot accidentally interact with buttons on the hidden screens beneath it.

### 3.14. Async Data Fetching & Caching (`/extras/wind`)
- **Responsibility:** Provide robust async data fetching, caching, and background refetching for Kitex components.
- **Architecture:** Inspired by React Query. Uses a `QueryClient` provided via Kitex Context.
- **Type-Safe Keys:** Uses Go's generic `comparable` constraint for cache keys (`K comparable`), ensuring strict type safety and zero-reflection performance when matching cache entries.
- **Hooks:** 
  - `UseQuery`: Manages the async state machine (`IsLoading`, `Data`, `Error`) and dedupes concurrent requests for the same key.
  - `UseMutation`: Executes side-effects (e.g., POST/DELETE) and injects a `MutationContext` into its `OnSuccess`/`OnError` callbacks. This context provides direct access to the `Client` so developers can call `Client.InvalidateQueries(exactKey)` without requiring additional hook lookups.

### 3.15. Form Architecture & Validation (`/extras/form`)
- **Responsibility:** Provide a unified pipeline from raw terminal input up to strongly-typed, validated Go structs.
- **Low-Level DOM (`element.Form`):** The core engine implements a `<form>` primitive. Form controls (`<input>`, `<checkbox>`) implement `dom.FormControl` to expose `Name()` and `Value()`. The `element.Form` intercepts "Enter" keystrokes and `type="submit"` button clicks to implicitly gather all control values and dispatch a single `event.SubmitEvent` carrying a `map[string]any` payload.
- **Kitex Integration:** Developers use `kitex.Form(kitex.FormProps{...})` to wrap their controls declaratively.
- **High-Level Validation (`extras/form`):** Inspired by React Hook Form, this package provides the `form.Use[T]` API. It takes the raw `map[string]any` from the `kitex.Form`, securely maps it into the user's defined generic struct `T`, runs synchronous validation logic, and manages the meta-state (`IsSubmitting`, `Errors`, `IsValid`) during async submission.

### 3.16. Clipboard System
- **Responsibility:** Managing global text selection and synchronizing data with the system clipboard.
- **Multi-MIME Data:** The `event.ClipboardEvent` follows a `DataTransfer`-like model, carrying a map of payloads keyed by MIME type (e.g., `text/plain`, `image/png`). This allows components to negotiate their preferred data format during a paste.
- **Global Integration:** The `dom.Document` acts as a central coordinator. If a focused element does not handle a copy/paste event, the Document falls back to the global `dom.Selection` to ensure seamless system-wide synchronization.
- **System Sync:** Uses the **OSC 52** escape sequence in supported backends to securely read from and write to the system clipboard across local and remote (SSH) sessions without external dependencies.

### 3.13. Terminal Extensions (`/backend` & `/internal/term`)
- **Responsibility:** Providing a pluggable architecture for terminal-specific protocols (e.g., advanced graphics, secure transfers).
- **Architecture:** The `backend.TerminalExtension` interface allows packages to intercept raw terminal sequences before they reach the main event loop. Extensions can perform multi-step handshakes (like Kitty's OSC 5522) and emit high-level framework events.
- **Initialization:** The `Engine` initializes registered extensions immediately after the backend starts, providing them direct access to the terminal's raw output writer for protocol negotiation.
- **Kitty Integration:** Includes a first-class extension for the **Kitty Secure Clipboard Transfer** protocol, enabling rich image and multi-format text pasting through a secure password-protected handshake.
