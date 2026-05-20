# TSK-028: Generic Scroll Offset on DOM Elements

## 1. Objective
Introduce a uniform, browser-style scroll mechanism: every `dom.Element` exposes `Scroll()`, `ScrollTo(x, y)`, `ScrollBy(dx, dy)`; scroll containers (per computed `overflow`) translate their children at paint time; wheel events automatically route to the nearest scroll container; an `event.EventScroll` fires on mutation. Replaces the bespoke `scrollX` / `scrollY` machinery in `<input>` and `<textarea>`.

See **ADR-012**.

Depends on TSK-027 (paint must clip the content box before scroll translation can hide off-box content).

## 2. Design & Requirements

### 2.1 Feature Design

#### Scroll State Storage
- Add a lazy `*scrollState` pointer on the unexported element implementation (`dom/element.go`):
  ```go
  type scrollState struct { X, Y int }
  ```
- The pointer is `nil` by default. Allocated the first time `ScrollTo` / `ScrollBy` is called on the element (or whenever the element is observed to need scroll, e.g., default wheel handler invoking `ScrollBy`).
- The pointer is **never** freed (memory remains until the element is GC'd); the `(0, 0)` state is a valid scrolled state.

#### Public Element API
Every `dom.Element` exposes:

| Method | Behavior |
|---|---|
| `Scroll() (x, y int)` | Returns `(0, 0)` when `scrollState == nil`; otherwise `(state.X, state.Y)`. |
| `ScrollTo(x, y int)` | Allocates `scrollState` lazily, assigns the raw values, marks `DirtyScroll`, dispatches `event.EventScroll`. No clamping at the element level. |
| `ScrollBy(dx, dy int)` | Sugar over `ScrollTo(Scroll() + delta)`. |

The values stored are raw author intent. Clamping happens at paint time (see §2.1 Paint Translation below).

#### `style.Overflow` Enum Addition
Add `OverflowAuto` to `style/enums.go`:
```go
// OverflowAuto enables scrolling when content overflows; the scrollbar is
// hidden when content fits. (Scrollbar UI is a separate task; until then,
// OverflowAuto behaves like OverflowScroll.)
OverflowAuto
```

#### Fluent Shorthand on `style.Style`
Add a fluent helper `.Overflow(v Overflow)` that sets both `OverflowX` and `OverflowY` to the same value. The two underlying axis-specific properties remain so authors can still express asymmetric overflow (`overflow-x: scroll; overflow-y: hidden`).

```go
// Equivalent to .OverflowX(v).OverflowY(v):
style.Style{}.Overflow(style.OverflowScroll)
```

#### Scroll-Container Definition
An element is a **scroll container** when its resolved `ComputedStyle.OverflowX` ∈ { `OverflowScroll`, `OverflowAuto` } OR `ComputedStyle.OverflowY` ∈ { `OverflowScroll`, `OverflowAuto` }. The check happens against the computed style — not the raw style — so cascade and intrinsic-style rules apply.

#### Paint Translation
Building on TSK-027:
1. Paint draws the fragment's own background and border at the parent's level (unclipped, unscrolled).
2. Paint draws the fragment's own text at the parent's level (no scroll applied to the fragment's own text).
3. Paint computes the **content-box clip rect** (TSK-027) and produces a clipped sub-surface.
4. If the fragment's node is a scroll container, paint reads its raw scroll offset and **clamps on read**:
   - `scrollExtent.X = max(0, contentSize.X - viewportSize.X)` where `contentSize` is the union of child bounding boxes and `viewportSize` is the content-box size.
   - `scrollExtent.Y` analogously.
   - `clamped = (clamp(raw.X, 0, scrollExtent.X), clamp(raw.Y, 0, scrollExtent.Y))`.
5. The child recursion uses `childOrigin = parentOrigin + childLink.Offset - clamped` so descendants paint shifted by the negative scroll offset, with anything outside the clip rect dropped by the clipped surface.

Clamping is on read only — `scrollState` is never mutated by paint. This preserves author intent across viewport resizes (e.g., set `scrollY = 9999`, viewport later grows so clamping returns a smaller value, then shrinks again and the same `9999` clamps back to the new max).

#### Default `Scrollable` Wheel Handler
The framework provides an internal default `Scrollable` implementation. The default `OnWheel`:
1. Reads `WheelEvent.DeltaX`, `DeltaY`.
2. Calls `host.ScrollBy(dx, dy)`.
3. Calls `e.StopPropagation()` so the event does not bubble past the first scroll container.

The synthesizer's `ScrollableResolver` (currently declared but ignored — `event/synthesizer.go:33-35`) is finally wired:
- The engine constructs a resolver that, given an `EventTarget`, returns:
  1. An author-registered `Scrollable` (if any — future API for opting in/out without changing computed style); else
  2. The framework default if the target's element is a scroll container per its computed style; else
  3. `nil`.
- The synthesizer stores the resolver and passes the map produced by walking the path through it when calling `Dispatcher.DispatchWheel`.

#### `event.EventScroll`
- New `EventType` constant: `EventScroll`.
- New event struct `ScrollEvent` carrying `(X, Y int)` (new offset) and `(DeltaX, DeltaY int)` (change). Bubbles. Cancelable: false (matches DOM).
- Fired by `ScrollTo` after mutating state (so listeners observe the new value). `ScrollBy` fires once with the consolidated delta.

#### Programmatic Scroll on Non-Containers
`ScrollTo(x, y)` is valid on any element. For non-scroll-containers the state is stored and the event fires, but paint applies no translation. This matches browser DOM behavior and lets authors prepare scroll state before toggling overflow.

### 2.2 Rules
- **DOM owns the state.** `scrollState` lives on the element implementation; render objects and paint read it via the existing logical-node back-pointer.
- **Paint clamps on read.** Element stores raw intent.
- **`DirtyScroll` is the right flag.** Mutating scroll marks the render object `DirtyScroll`. Style and layout caches are preserved. Paint clears the flag.
- **No new fields on `render.Object`.** Scroll extent is computed at paint time from the fragment's child bounds and content-box size.
- **Strict package isolation.** `event/synthesizer.go` already declares the resolver hook; this task wires it. The `paint` package depends on `dom` only via the logical-node back-pointer on `*layout.Fragment.Node`, which already exists.
- **The wheel default may be opted out** by registering an author-supplied `Scrollable` for the element that does nothing (a no-op `OnWheel`). This is the mechanism `<input>` uses to disable wheel scrolling.

### 2.3 Out of Scope
- **Visible scrollbar glyphs (track + thumb).** Deferred. `OverflowAuto`'s hide-when-fits behavior also belongs there.
- **`element.scrollIntoView()`** helper. Useful future addition, separate task.
- **Click-and-drag on scrollbar to scroll.** Belongs to the scrollbar UI task.
- **Smooth scrolling, animations, momentum.** Out of scope.
- **Computed `scrollWidth` / `scrollHeight` getters.** Add if/when needed; the scroll extent computation happens at paint and is not currently exposed.

## 3. Implementation Steps

1. Add `OverflowAuto` to `style/enums.go::Overflow`. Update `style/computed.go`, `style/style.go`, `style/resolver.go` if any switch statements enumerate values.
2. Add the fluent `.Overflow(v Overflow)` shorthand to `style.Style`. Implementation sets `OverflowX` and `OverflowY` to the same `Some(v)`.
3. In the element implementation (`dom/element.go` or equivalent):
   - Add `scroll *scrollState`.
   - Implement `Scroll() (int, int)`, `ScrollTo(x, y int)`, `ScrollBy(dx, dy int)`.
   - `ScrollTo` marks the render object `DirtyScroll` (using the existing flag from `render/dirty.go`) and dispatches `event.EventScroll`.
4. Promote `Scroll`, `ScrollTo`, `ScrollBy` to the `dom.Element` interface.
5. In `event/events.go`:
   - Add `EventScroll EventType`.
   - Add `ScrollEvent` struct + constructor + getters.
6. In `event/synthesizer.go`:
   - Persist `ScrollableResolver` from `SynthesizerOptions` into the synthesizer struct (currently dropped on the floor).
   - When processing raw wheel events, walk the hit-tested path and build the `scrollables map[EventTarget]Scrollable` via the resolver.
   - Call `Dispatcher.DispatchWheel(e, path, scrollables)` (already exists).
7. Add an internal default `Scrollable` (e.g., `dom/scroll_controller.go::defaultScroller`) whose `OnWheel` calls `host.ScrollBy(e.DeltaX, e.DeltaY)` then `e.StopPropagation()`.
8. In `engine/engine.go`, install a `ScrollableResolver` in the synthesizer options that returns the default scroller for any element whose computed style indicates scroll containerness. Allow author override via a future hook (not implemented now but the resolver structure must permit it).
9. In `paint/engine.go::paintFragment`, after the TSK-027 clip computation:
   - Detect whether the fragment's node is a scroll container.
   - If so, compute `scrollExtent` from `frag.Children` union bounds vs. content-box size, clamp the raw scroll, and shift child origins.
10. Update `<input>` and `<textarea>` in TSK-024 / TSK-025 to remove bespoke `scrollX` / `scrollY` fields. (Performed in those tasks; this task only enables the move.)

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] `ScrollTo(5, 10)` on a non-container element stores the state, returns `(5, 10)` from `Scroll()`, and fires `EventScroll`.
- [ ] `ScrollTo` on a non-container does not translate any paint output.
- [ ] `ScrollBy(1, 2)` after `ScrollTo(10, 20)` produces `Scroll() == (11, 22)`.
- [ ] `ScrollTo(0, 9999)` on a scroll container clamps at paint time to the scroll extent; `Scroll()` still returns `(0, 9999)`.
- [ ] Setting `IntrinsicStyle().OverflowY(OverflowScroll)` makes the element a scroll container; setting it back to `OverflowVisible` removes the translation while preserving the raw scroll value.
- [ ] `OverflowAuto` behaves identically to `OverflowScroll` for paint and wheel handling (until scrollbar UI lands).
- [ ] `style.Style{}.Overflow(OverflowClip)` sets both `OverflowX` and `OverflowY` to `Clip` after merge.

### 4.2 Wheel Routing
- [ ] A wheel event whose hit target is a non-scroll-container descendant bubbles to the first scroll-container ancestor and mutates its scroll.
- [ ] When the target itself is a scroll container, its own scroll is mutated; the event does not bubble.
- [ ] An author who registers a no-op `Scrollable` for an element prevents wheel-scroll on that element even though its computed style declares a scroll container (used by `<input>`).
- [ ] When no ancestor is a scroll container, the wheel event reaches the document root without effect.

### 4.3 Paint Translation
- [ ] A scroll container of `{Width: 10, Height: 5}` whose content extends to `{Width: 10, Height: 20}` with `ScrollTo(0, 5)` paints content rows 5..9.
- [ ] `ScrollTo(0, 999)` clamps to row 15 (so rows 15..19 are visible).
- [ ] `ScrollTo(-3, 0)` clamps to `(0, 0)`.
- [ ] Background and border paint at the unscrolled position; only descendants shift.

### 4.4 Integration Tests
- [ ] A `<div>` styled with `Overflow(OverflowScroll)`, fixed size, and 30 lines of text scrolls correctly on wheel input.
- [ ] An `EventScroll` listener attached to the container fires on every `ScrollTo` / `ScrollBy` and on wheel events.
- [ ] Refactored `<input>` (TSK-024) scrolls horizontally as the cursor moves past the right edge; the wheel does not horizontally scroll (no-op `Scrollable` override).
- [ ] Refactored `<textarea>` (TSK-025) scrolls vertically on wheel and on programmatic `ScrollTo`.

### 4.5 Regression Tests (at `./tests/regressions/`)
- [ ] Add `scroll_offset_test.go` covering:
  - Scroll preserved across DOM moves (detach + re-attach).
  - Scroll preserved across overflow-mode toggles.
  - Nested scroll containers: wheel hits the innermost.
  - `EventScroll` bubbles through ancestor listeners.
  - Clamping reacts correctly when content shrinks below the current scroll offset (offset stays the same value, paint clamps to the new max).

### 4.6 Benchmarks
- [ ] `BenchmarkPaint_NoScroll`: confirm scroll-container check adds < 3 % overhead in trees with no scroll containers.
- [ ] `BenchmarkPaint_ManyScrollContainers`: 100 scroll containers, each with 20 children. Verify the per-container clamp + translate is O(1) per container.
- [ ] `BenchmarkScrollBy_DirtyScroll`: invoke `ScrollBy` 1000 times; verify only `DirtyScroll` (not `DirtyLayout` / `DirtyStyle`) is set so the layout cache is preserved.

### 4.7 Documentation
- [ ] Update `dom/doc.go` to describe `Scroll`, `ScrollTo`, `ScrollBy`, and the lazy state.
- [ ] Update `style/doc.go` to record the `Overflow` shorthand and the addition of `OverflowAuto`.
- [ ] Update `paint/doc.go` to describe paint-side scroll translation (referencing ADR-012).
- [ ] Update `event/doc.go` to record `EventScroll` and the wired `ScrollableResolver`.
- [ ] Update `AGENT.md` "Architectural Rules" section: add a rule that scroll state is DOM-owned and clamping is paint-side.
- [ ] Update `README.md` if scrolling is a documented top-level feature.
