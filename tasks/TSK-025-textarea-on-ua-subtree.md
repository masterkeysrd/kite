# TSK-025: Refactor `<textarea>` onto UA Shadow Subtree

## 1. Objective
Implement `element.TextAreaElement` using the UA Shadow Subtree (TSK-018), the Intrinsic Style Layer (TSK-022), and `cursor.FromTextFragment` (TSK-023).  The host gets a plain `render.Box` and the standard IFC handles shaping, wrapping (via `white-space: pre-wrap`), mandatory breaks, soft wraps, and clipping.

Depends on: TSK-018, TSK-022, TSK-023, TSK-027, TSK-028.

## 2. Design & Requirements

### 2.1 Feature Design
- `TextAreaElement` becomes a thin host:
  - Embeds `*text.Buffer` (TSK-017) for 1D logical text editing.
  - In its constructor, creates a UA subtree consisting of a single internal text node whose content is bound to `Buffer.Value()`, attached via `Element.AttachUARoot()`.
  - On every mutation, propagates the new value to the UA text node and marks the render object dirty.
- UA-mandated styles via `IntrinsicStyle()`:
  - `Display: inline-block`
  - `OverflowX: OverflowClip`, `OverflowY: OverflowScroll` (vertical scroll container; clip on X because the textarea soft-wraps via `pre-wrap` so there is nothing to scroll horizontally).
  - `WhiteSpace: WhiteSpacePreWrap` — preserve `\n` and inter-word whitespace; allow soft wrap at break opportunities.
  - `OverflowWrap: OverflowWrapBreakWord` — emergency-break long unbreakable runs.
- Author-overridable defaults (e.g., the 20-column fallback width, padding) remain on `DefaultStyle()`. If the textarea has no explicit `Width` and the parent does not constrain it, fall back to the 20-column default.
- 2D navigation (Up/Down):
  - The host queries its render-box fragment.
  - Resolves the current cursor position via `cursor.FromTextFragment` to obtain `(curX, curY)`.
  - Computes the target line `curY ± 1`.
  - Within the target line, walks the line box's text clusters, summing `CellWidth` until it meets/exceeds `curX`, then accumulates bytes to derive the new byte offset.
  - Updates `Buffer.SetOffset()` with the result.
- Vertical scrolling uses the generic mechanism from TSK-028. The host's keystroke handler calls `host.ScrollTo(0, targetLineY)` to keep the cursor visible. Wheel-scroll is **enabled** (the default framework-provided `Scrollable` handles it). No bespoke `scrollX` / `scrollY` fields on the host.

### 2.2 Rules
- The UA subtree contains **only** one synthetic text node. The IFC's `white-space: pre-wrap; overflow-wrap: break-word` produces the desired multi-line layout from that single text node without any per-line synthesis on the element side.
- All UA-forced styles live in `IntrinsicStyle()`; the render object holds no hard-coded styles.
- No casts of the form `logicalNode.(interface { Value() ... })` exist anywhere.
- The element implements `cursor.Provider` directly.
- The Up/Down navigation logic lives on the host (it needs `Buffer` access) but operates entirely via the standard fragment tree — no `GetTargetOffset` interface on the render object.
- Overflow clipping is enforced by paint (TSK-027) via intrinsic `OverflowX: clip`.
- Vertical scroll translation is performed by paint (TSK-028) via intrinsic `OverflowY: scroll`; the host calls `ScrollTo` and never shifts fragments manually.

### 2.3 IFC Dependencies
- **`white-space: pre-wrap`** is fully supported by the IFC today (verified during design: `layout/inline.go:155, 267, 399, 447`).
- **`overflow-wrap: break-word`** is partially honored: the IFC's emergency-break path (`layout/inline.go:301-326`) always takes one cluster when no break opportunity exists, regardless of `OverflowWrap`. This matches the desired textarea behavior, so this task is **not blocked**. A separate, deferred cleanup task should later gate the emergency break on `OverflowWrap != Normal`.

### 2.4 Out of Scope
- Selection ranges; text highlighting; clipboard ops (could be future tasks).
- Hard-wrapping at column boundaries with explicit `cols="N"` attribute (defer; not needed for v1).
- Soft-wrap virtual line ↔ logical line distinction beyond what the IFC already provides.

## 3. Implementation Steps
1. Rewrite `element/input.go::TextAreaElement`:
   - Remove `CreateRenderObject`, `MarkDirty`, and `render.CustomObjectProvider` reference.
   - Add a private `uaText *dom.TextNode` field.
   - In `NewTextArea`, create the text node with `Buffer.Value()` and call `AttachUARoot(uaText)`.
   - After every mutation, call `uaText.SetData(buffer.Value())`.
   - Implement `IntrinsicStyle()` returning the four forced properties.
   - Implement `cursor.Provider` using `cursor.FromTextFragment`.
   - Move the Up/Down navigation logic onto the host, implementing it via fragment traversal.
2. Rewrite `scrollCursorIntoView` to derive line index from `cursor.FromTextFragment(...)` instead of re-walking line fragments by hand.
4. Create `tests/regressions/input_test.go`
5. Update `element/doc.go` to describe the textarea as a UA-shadow host with a single internal pre-wrap text node.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] After construction, `textarea.Children()` returns no public children.
- [ ] After inserting `"hello\nworld"`, the resulting layout has at least two line boxes containing `"hello"` and `"world"` respectively.
- [ ] Soft-wrap: a line of 30 spaces-separated words in a 10-cell-wide textarea wraps at word boundaries.
- [ ] Emergency-wrap: a 30-character unbreakable run in a 10-cell-wide textarea wraps mid-cluster.
- [ ] `IntrinsicStyle()` returns `Display=inline-block`, `OverflowX=clip`, `OverflowY=clip`, `WhiteSpace=PreWrap`, `OverflowWrap=BreakWord`.
- [ ] Setting `RawStyle().WhiteSpace(WhiteSpaceNormal)` does **not** change the resolved value (intrinsic wins).
- [ ] `CursorState()` returns correct `(X, Y)` at start, after typing on line 2, and at end of buffer.
- [ ] Up navigation from line 2 to line 1 preserves visual X column as closely as possible (target line is shorter ⇒ caret clamps to end-of-line).
- [ ] Down navigation past the last line is a no-op (cursor stays).

### 4.2 Integration Tests
- [ ] A textarea with `RawStyle().Width(20).Height(5)` correctly renders five visible lines, soft-wraps text exceeding column 20, and clips text beyond row 5.
- [ ] Focus lands on the textarea (host) and never on the UA text node.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `textarea_test.go` with the byte-offset edge cases previously handled by `GetTargetOffset`:
  - Cursor at end of a line terminated by mandatory break, Down key → start of next line.
  - Cursor at last character of last line, Down key → no movement.
  - Up navigation across a soft-wrapped boundary preserves visual column.
  - Inserting a newline at the very end of buffer increases line count by 1 and places cursor at column 0 of the new line.
- [ ] Verify that author overriding `RawStyle().WhiteSpace(Pre)` does not break wrap behavior (intrinsic still wins).

### 4.4 Benchmarks
- [ ] `BenchmarkTextAreaRender_SmallDoc`: 10 lines × 80 cols, single layout cycle. Must remain within ±10 % of pre-refactor numbers.
- [ ] `BenchmarkTextAreaRender_LargeDoc`: 500 lines × 80 cols. Profile to confirm the IFC walk does not regress when fed by a single text node of large size.
- [ ] `BenchmarkTextAreaType`: 200 sequential character inserts; total time should be dominated by IFC layout (constant per insert).

### 4.5 Documentation
- [ ] Update `element/doc.go`: `<textarea>` is a UA-shadow host with a single internal text node and intrinsic `inline-block; overflow: clip; white-space: pre-wrap; overflow-wrap: break-word`.
- [ ] Update `README.md` if textarea is mentioned in the feature list.
- [ ] Update `AGENT.md` if needed.
