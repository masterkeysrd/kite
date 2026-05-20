# ADR-012 - Generic Scroll Offset on DOM Elements

## Status
Accepted.

## Context
Today only `<input>` and `<textarea>` carry scroll offsets, stored as bespoke `scrollX` / `scrollY` fields on the element and applied at layout time by placing text fragments at negative coordinates. The terminal viewport (and, after ADR-011, the parent overflow clip) is what makes the off-box pixels invisible. The approach has three problems:

1. **It does not generalize.** Any other element that wants scrolling has to invent the same per-host machinery.
2. **It violates DOM-owned state.** Browser-style scroll is observable behavior (`element.scrollTop`, `element.scrollLeft`, `scroll` event). With state hidden inside specific element types, authors cannot drive scroll programmatically through a uniform API.
3. **It tangles scroll with layout.** Placing text at negative offsets means the layout fragment for a scrolled view is not the same as the layout for an unscrolled view — re-shaping or re-running layout when only the scroll offset changes wastes the `DirtyScroll` optimization that already exists (`render/dirty.go:26-29`).

The framework already contains half of the plumbing for a generic scroll model:

- `WheelEvent` (`event/events.go:214+`).
- `Scrollable` interface (`event/events.go:343-348`).
- `Dispatcher.DispatchWheel` with bubble-to-first-scrollable semantics (`event/dispatcher.go:141-193`).
- `DirtyScroll` flag (`render/dirty.go:26-29`).
- `SynthesizerOptions.ScrollableResolver` (declared but never wired — `event/synthesizer.go:33-35`).

What is missing is: scroll state on the DOM, paint-side translation of children, a default wheel handler, and a `scroll` event. ADR-011 (paint clipping) is a hard prerequisite — without overflow clipping, a scrolled box would paint outside its content area.

We considered:
- **Scroll state on `render.Object` instead of `dom.Element`.** Rejected: scroll is observable state that survives DOM tree mutations and is naturally part of the logical model. Render objects already store transient frame state; mixing in scroll position fights the "DOM owns interactivity state" rule (TSK-007).
- **Always-present `int` scroll fields on every element.** Rejected: 8 bytes per element times thousands of elements is non-trivial, and 99 % of elements never scroll. A lazy pointer is cleaner.
- **A marker interface (`dom.ScrollContainer`) opt-in.** Rejected: scroll-containerness is determined by computed style (`overflow`), not by the element type. Authors changing styles dynamically should change scroll behavior dynamically.
- **A single `Overflow` property replacing `OverflowX`/`OverflowY`.** Rejected: loses the asymmetric case (`overflow-x: scroll; overflow-y: hidden`). We instead add a fluent shorthand `.Overflow(...)` that sets both at once (D3b — see design session notes).
- **Always rendering a visible scrollbar.** Rejected: scrollbar glyphs are a UI feature; the state + paint + wheel mechanism is independent of them. Visible scrollbars defer to a follow-up task.

## Decision
We introduce a generic scroll model owned by `dom.Element`, applied by paint, and routed through the existing wheel dispatch.

### 1. Scroll State on `dom.Element` (Lazy)
- The element implementation gains an unexported `scroll *scrollState` pointer that is **nil** by default.
- `scrollState` carries `X int` and `Y int` — raw author intent, unclamped.
- The pointer is allocated lazily the first time the element either (a) is observed to be a scroll container by the computed-style cascade *and* needs an offset, or (b) has `ScrollTo` / `ScrollBy` called on it. Elements that never scroll never allocate.
- Mutating `scrollState` marks the render object with `DirtyScroll` (already defined in `render/dirty.go`). The paint phase clears `DirtyScroll` together with `DirtyPaint` on completion.

### 2. Public Element API
Every `dom.Element` exposes:

| Method | Behavior |
|---|---|
| `Scroll() (x, y int)` | Returns the raw scroll offset (0, 0 if never scrolled). |
| `ScrollTo(x, y int)` | Sets the raw offset. Marks `DirtyScroll`. Dispatches an `event.EventScroll` (bubbles). |
| `ScrollBy(dx, dy int)` | Sugar over `ScrollTo(Scroll() + delta)`. |

These methods exist on every element regardless of whether it is a scroll container (matching browser DOM, where `scrollTop` exists on every `Element`). For non-scroll-containers the value is observable but has no visual effect.

### 3. Scroll-Container Definition
An element is a **scroll container** when its resolved `ComputedStyle.OverflowX` or `ComputedStyle.OverflowY` ∈ { `OverflowScroll`, `OverflowAuto` }.

We add `OverflowAuto` to the `style.Overflow` enum (`style/enums.go`) with this ADR. Until a follow-up adds scrollbar UI, `Auto` is treated identically to `Scroll` in paint and wheel routing — the difference is purely UI/scrollbar-visibility, which is deferred.

### 4. Fluent Shorthand on `style.Style`
We add a fluent shorthand `.Overflow(v Overflow)` on `style.Style` that sets both `OverflowX` and `OverflowY` to the same value. The underlying axis-specific properties remain to preserve the asymmetric case. This is the "single knob" ergonomic improvement (D3b).

### 5. Paint Translation
Building on ADR-011, when `paintFragment` recurses into a scroll-container fragment:
1. The fragment's own background and border paint at the parent's level (unaffected by scroll).
2. A clipped surface is created for the **content box** (ADR-011).
3. The scroll offset is read from the element and **clamped on read** to `[0, scrollExtent]` where `scrollExtent = max(0, contentSize - viewportSize)` per axis. The element's raw value is *not* mutated by paint.
4. Each child's origin is translated by `(-clampedX, -clampedY)` before recursion.

### 6. Default `Scrollable` (Framework-Provided)
The engine vends a default `Scrollable` implementation for any element that is a scroll container per its computed style. The default `OnWheel` translates `WheelEvent.DeltaX/DeltaY` into `ScrollBy(dx, dy)` on the host element. Authors can override by registering their own `Scrollable` (the synthesizer's `ScrollableResolver` returns the override first, falling back to the default).

The `event.Synthesizer` stores `ScrollableResolver` (currently declared but ignored) and calls it during wheel-event processing. The engine wires the resolver so it consults: (a) explicit author override; (b) framework default for computed scroll containers; (c) nil.

### 7. `event.EventScroll`
New event type, bubbles, fired by `ScrollTo` / `ScrollBy` and by the default wheel handler after it mutates state. Carries the new `(x, y)` and a delta `(dx, dy)`.

### 8. Programmatic Scroll Always Works
`ScrollTo` / `ScrollBy` work even on elements whose computed style is *not* a scroll container — the state is stored, the event fires, but paint applies no translation (because the box does not clip). This matches the browser: setting `scrollTop` on a `<div>` with `overflow: visible` is a no-op visually but the state and event still exist.

### 9. Out of Scope
- **Visible scrollbar glyphs / track + thumb rendering.** Deferred to a follow-up task. `OverflowAuto`'s "hide when fits" behavior also belongs there.
- **Smooth scrolling / animation.** Out of scope.
- **`element.scrollIntoView()`** helper. Useful future addition, separate task.
- **Reverse scroll mapping (click-on-scrollbar to scroll).** Belongs to the scrollbar UI task.

### 10. Relation to Input / TextArea (ADR-009)
- `<input>`: intrinsic `OverflowX(OverflowClip)` is replaced by `OverflowX(OverflowScroll)` so the host becomes a generic horizontal scroll container. The host's keystroke handler calls `host.ScrollTo(...)` to keep the cursor visible. Wheel-scroll is disabled by overriding the `Scrollable` with a no-op (typing inside an `<input>` should not horizontally scroll the field via the wheel — that's a deliberate UX choice).
- `<textarea>`: intrinsic `OverflowY(OverflowScroll)` (and `OverflowX(OverflowClip)` to disallow horizontal scroll; the textarea soft-wraps via `white-space: pre-wrap`). Keystroke handler drives `ScrollTo`; wheel handler defaults to active (textareas scroll on wheel).
- TSK-024 (input refactor) and TSK-025 (textarea refactor) take a hard dependency on ADR-012.

## Consequences

### Positive
- A uniform browser-like scroll API on every element.
- DOM-owned, observable, survives sync. Authors can read `element.Scroll()` and listen for `event.EventScroll`.
- Reuses every piece of existing scroll plumbing (`Scrollable`, `DirtyScroll`, `WheelEvent`, `DispatchWheel`, `ScrollableResolver`); no parallel infrastructure.
- The `DirtyScroll` fast path becomes meaningful — pure scroll changes skip style and layout, only paint runs.
- Input / TextArea lose ~50 lines of bespoke scroll-into-view code each; a single helper drives both.

### Negative / Trade-offs
- One lazy pointer field on every `dom.Element` (8 bytes on 64-bit). Zero allocation cost until first scroll.
- Paint gains a small per-frame branch and clamp-on-read computation per scroll container.
- `ScrollTo` on a non-container is observable state with no visual effect, which some authors may find surprising. This matches the browser and is documented.
- Computed-style changes that toggle scroll-container status do **not** clear the scroll state; the offset is preserved across the toggle, again matching the browser.
