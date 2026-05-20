# TSK-027: Paint Honors `overflow: clip` / `overflow: hidden`

## 1. Objective
Wire up the existing `Surface.Clip()` API inside `paint/engine.go::paintFragment` so that fragments whose computed `OverflowX` or `OverflowY` is `Hidden` or `Clip` actually clip their descendant paint output to their content box. Closes a longstanding correctness gap and is a hard prerequisite for the input/textarea refactors (TSK-024, TSK-025).

See **ADR-011**.

This task does **not** introduce scroll offsets, scroll containers, or wheel-event wiring. Scrolling is the concern of TSK-028.

## 2. Design & Requirements

### 2.1 Feature Design
- Extend the recursion in `paint/engine.go::paintFragment` so that, when entering a child whose ancestor (any node in the path) declared `OverflowX` or `OverflowY` in { `Hidden`, `Clip`, `Scroll`, `Auto` }, the descendants paint onto a `Surface` whose drawable area equals the **content box** of that ancestor.
- Use the existing `Surface.Clip(rect)` method. Composition for nested overflow boxes is automatic via `clippedSurface.Clip`'s intersection (`paint/framebuffer.go:135-137`).
- The fragment's own background and border paint **before** the clipped surface is created and pass the **uncliped** surface down to those helpers, so the fragment's own border-box decoration is never clipped by its own `overflow`.

### 2.2 Clip-Rect Computation
Given a fragment positioned at absolute `origin` with `frag.Size`, and a `ComputedStyle` carrying `Border.Widths()` and `Padding`:

```
contentBox.Origin.X = origin.X + border.Left + padding.Left
contentBox.Origin.Y = origin.Y + border.Top  + padding.Top
contentBox.Size.W   = frag.Size.Width  - border.Left - border.Right  - padding.Left - padding.Right
contentBox.Size.H   = frag.Size.Height - border.Top  - border.Bottom - padding.Top  - padding.Bottom
```

If `Width` or `Height` clamp to zero or negative, the clip rect is empty and the surface returned by `Clip` already drops all writes — no special case needed.

### 2.3 Behavioral Matrix
| `OverflowX` / `OverflowY` | Clip descendants? |
|---|---|
| `OverflowVisible` (default) | **No** |
| `OverflowHidden` | **Yes** (to content box) |
| `OverflowClip` | **Yes** (to content box) |
| `OverflowScroll` | **Yes** (so this also handles the clip half of TSK-028's scroll work) |
| `OverflowAuto` (new in TSK-028) | **Yes** (same) |

Mixed axes are supported: `OverflowX: Visible, OverflowY: Clip` clips only on the Y axis. Implement by computing an axis-asymmetric clip rect — the content box's width spans the full fragment when the X axis is `Visible`, and similarly for height.

### 2.4 Rules
- Paint never mutates `frag` or the render object. The clip rect is computed locally and used only for the duration of the recursion.
- The fragment's own background fill (`fillRect`) and border drawing (`drawBorder`) use the **unclipped** surface (`origin`-anchored, full border-box).
- Text owned by the fragment itself (`frag.Text`, painted at the parent's coordinate space) also uses the unclipped surface, because the text belongs to the fragment, not to its descendants. Today only block/inline-block fragments place text on themselves (line boxes), so this is consistent.
- Children, including their backgrounds, borders, and recursive descendants, paint onto the **clipped** surface.
- `resolveBorders` must continue to run on the **root** surface only. Document this invariant in code comments at both the call site (`paint/engine.go::Paint`) and the resolver function.

### 2.5 Out of Scope
- Scroll translation, scroll state, wheel routing, scrollbar UI — all TSK-028 / follow-up.
- Per-axis clip granularity beyond what the X/Y enum already expresses.
- `overflow-clip-margin` or any property that extends the clip rect beyond the content box.

## 3. Implementation Steps
1. In `paint/engine.go::paintFragment`, after painting the fragment's own background, border, and own text, compute the child clip rect from the fragment's computed style and `frag.Size`.
2. If the fragment establishes any clip (i.e., either axis has `Hidden`/`Clip`/`Scroll`/`Auto`), call `surface.Clip(childClipRect)` and pass the returned `Surface` to the child recursion. Otherwise pass `surface` unchanged.
3. For asymmetric overflow (X visible, Y clipped or vice versa), the unclipped axis extends to the fragment's full border-box extent on that axis; the clipped axis is content-box-inset.
4. Verify `resolveBorders` is called only from `Paint()` on the root surface and add a code comment recording the invariant.
5. Audit `paint/engine_test.go` and `paint/resolver_test.go` for assumptions that need updating now that clipping actually happens. Add new tests per §4.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] Fragment with `OverflowX/Y: Visible` and an oversized child paints the child fully (no clipping). Regression guard.
- [ ] Fragment with `OverflowX/Y: Hidden` and an oversized child clips writes that fall outside the content box. Verify cells outside the box are untouched.
- [ ] Fragment with `OverflowX/Y: Clip` behaves identically to `Hidden` for paint.
- [ ] Asymmetric overflow: `OverflowX: Hidden, OverflowY: Visible` clips left/right spill but allows top/bottom spill to paint.
- [ ] Border integrity: a fragment with `OverflowX/Y: Hidden` and a visible border still paints the full border (its own decoration is not clipped by its own `overflow`).
- [ ] Padding contributes correctly to the content-box clip rect; a child at `(border.Left + padding.Left + 1, ...)` paints inside the clip.
- [ ] Nested overflow boxes: a parent with `Hidden` containing a child with `Hidden` containing a grandchild that overflows both clips the grandchild at the intersection of both clip rects.
- [ ] Zero-sized content box (border + padding ≥ frag size) drops all descendant paint.

### 4.2 Integration Tests
- [ ] A box `{Width: 10, Height: 3, OverflowX: Hidden}` containing a text line of 30 characters paints only 10 cells of text; the framebuffer cells at X=10..29 (relative to the box) remain at their pre-paint values.
- [ ] The same box with `OverflowX: Visible` paints all 30 cells, possibly spilling over neighbors.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `overflow_clip_test.go` covering:
  - A box with hidden overflow containing a longer child — verify framebuffer outside the content box is untouched.
  - A box with hidden overflow inside a box with visible overflow — verify only the inner clip is applied.
  - A box with hidden overflow and a visible 1-cell border — verify the border's corners and edges are intact (not eaten by the clip).
  - Asymmetric: a horizontally-clipped row inside a vertically-clipped column.

### 4.4 Benchmarks
- [ ] `BenchmarkPaint_NoOverflow`: confirm < 3 % overhead for paint trees that have no clipping (the new branch must be cheap).
- [ ] `BenchmarkPaint_DeepNestedClips`: 5-deep `Hidden` nesting with 50 children at each level. Ensure the `clippedSurface` allocations do not regress paint by more than 10 %.

### 4.5 Documentation
- [ ] Update `paint/doc.go` describing the per-fragment clipping behavior with a reference to ADR-011.
- [ ] Update `AGENT.md` Paint section to record the clip invariant ("`resolveBorders` runs once on the root surface; never on a clipped sub-surface").
- [ ] Update `paint/README.md` (if it documents paint semantics) accordingly.
- [ ] No `README.md` change unless overflow is a documented top-level feature.
