# TSK-024: Refactor `<input>` onto UA Shadow Subtree

## 1. Objective
Implement `element.InputElement` using the UA Shadow Subtree (TSK-018), the Intrinsic Style Layer (TSK-022), and `cursor.FromTextFragment` (TSK-023). The host gets a plain `render.Box` and the standard IFC handles text shaping, scrolling, and clipping.

Depends on: TSK-018, TSK-022, TSK-023, TSK-027, TSK-028.

## 2. Design & Requirements

### 2.1 Feature Design
- `InputElement` becomes a thin host:
  - Embeds `*text.Buffer` (TSK-017) for 1D logical text editing.
  - In its constructor, creates a UA subtree consisting of a single internal text node whose content is bound to `Buffer.Value()`, and attaches it via `Element.AttachUARoot()`.
  - On every keystroke handler, after mutating the buffer, updates the UA text node's value and calls `MarkDirty(DirtyLayout | DirtyPaint)`.
- The input no longer implements `render.CustomObjectProvider`. The engine creates a plain `render.Box` for it. The IFC produces a single line box containing the shaped text.
- UA-mandated styles move to `IntrinsicStyle()`:
  - `Display: inline-block`
  - `OverflowX: OverflowScroll`, `OverflowY: OverflowClip` (horizontal scroll container; clip on Y because single-line)
  - `WhiteSpace: WhiteSpaceNoWrap` (single-line: never wrap)
- Author-overridable defaults (e.g., `Width`, padding) remain on `DefaultStyle()`.
- Cursor positioning: `InputElement.CursorState()` calls `cursor.FromTextFragment(host.RenderObject().Fragment(), buffer.ByteOffset())`, then offsets the result by the host's content-box origin. Paint-side scroll translation (TSK-028) handles the shift; the host does not subtract `scrollX` manually.
- Horizontal scrolling uses the generic mechanism from TSK-028. The host's keystroke handler calls `host.ScrollTo(targetX, 0)` to keep the cursor visible. Wheel-scroll is disabled by registering a no-op `Scrollable` for the input (typing-on-wheel is the wrong UX for a single-line field). Bespoke `scrollX` field is removed.

### 2.2 Rules
- The UA subtree contains **only** one synthetic text node. No box wrappers, no inline spans. Keeps the IFC walk trivial.
- The element implements `cursor.Provider` directly (the host, not a render object).
- All UA-forced styles live in `IntrinsicStyle()`. No styles are set on the render object.
- No casts of the form `logicalNode.(interface { Value() string ... })` exist anywhere. The standard render-object/element contract is sufficient.
- The element implements `dom.Focusable` (returns `true`) and keeps its existing key-binding behavior.
- Overflow clipping is enforced by the paint phase (TSK-027) via the intrinsic `OverflowY: clip`; horizontal scroll translation by the paint phase (TSK-028) via the intrinsic `OverflowX: scroll`. No manual clipping or offset arithmetic in the host or render object.
- The host registers a no-op `Scrollable` to disable wheel-scrolling on the input.

### 2.3 Out of Scope
- Selection ranges, multi-line input, password masking. Future tasks.
- IME / dead-key composition handling. Future task.
- Click-to-position the caret (reverse cursor mapping). Future task.

## 3. Implementation Steps
1. Rewrite `element/input.go::InputElement`:
   - Remove `CreateRenderObject`, `MarkDirty` (use `RenderObject().MarkDirty(...)` directly), and any `render.CustomObjectProvider` reference.
   - Add a private `uaText *dom.TextNode` field holding the synthetic text node.
   - In `NewInput`, create the text node with `Buffer.Value()` and call `AttachUARoot(uaText)`.
   - In each keystroke handler, after mutating the buffer, call `uaText.SetData(buffer.Value())`.
   - Implement `IntrinsicStyle()` returning the three forced properties.
   - Implement `cursor.Provider` on the element itself using `cursor.FromTextFragment`.
2. Update `element/input.go::scrollCursorIntoView` to use `cursor.FromTextFragment` instead of re-shaping clusters.
3. Delete:
   - `render/input.go::Input`, `render/input.go::InputAlgorithm`.
   - Any `NewInput` factory in the `render` package.
4. Update `tests/regressions/input_test.go` to remove the old `provider.CreateRenderObject()` cast and use the new path. Where possible, replace bespoke fragment walks with assertions on the standard render-box fragment.
5. Update `element/doc.go` to describe the input as a UA-shadow-host with a one-line text subtree.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] After construction, `input.Children()` returns no public children (the UA text node is invisible).
- [ ] After typing, the UA-internal text node's value matches `Buffer.Value()`.
- [ ] `IntrinsicStyle()` returns `Display=inline-block`, `OverflowX=clip`, `OverflowY=clip`, `WhiteSpace=NoWrap`.
- [ ] Setting `RawStyle().Display(Block)` does not change the resolved `Display` (intrinsic wins).
- [ ] `CursorState()` returns the correct `(X, Y)` for offsets at start, mid, and end of buffer, including with `scrollX > 0`.
- [ ] Inserting text wider than the box keeps the cursor in view (scrollX adjusts).

### 4.2 Integration Tests
- [ ] An input inside a flex row with `RawStyle().Width(10)` renders exactly 10 cells wide, scrolls horizontally when text exceeds 10, and clips painted output beyond the content box.
- [ ] Focus navigation lands on the input (host) and not on the UA text node.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Replace the existing `input_test.go` regression with an updated version that:
  - Verifies typing produces the same painted output as before this refactor (golden-file comparison if available, otherwise cluster-level assertions).
  - Verifies cursor coordinates match the prior implementation for the same buffer contents and scroll states.
  - Verifies that author setting `RawStyle().OverflowX(Visible)` does not cause text to paint beyond the input's content box.

### 4.4 Benchmarks
- [ ] `BenchmarkInputRender_ShortText`: 10-character input, repeated layout cycle. Must remain within ±10 % of the pre-refactor numbers (the IFC pipeline replaces the bespoke algorithm).
- [ ] `BenchmarkInputType`: simulate inserting 50 characters one-by-one, measuring layout + paint per keystroke.

### 4.5 Documentation
- [ ] Update `element/doc.go`: `<input>` is a UA-shadow host with a single internal text node and intrinsic `inline-block; overflow: clip; white-space: nowrap`.
- [ ] Update `README.md` if input is mentioned in the feature list.
- [ ] Update `AGENT.md` if the previous "Replaced Elements" rule is referenced.
