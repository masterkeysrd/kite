# TSK-023: cursor.FromTextFragment Helper

## 1. Objective
Add a shared cursor-positioning helper in the `cursor` package that translates a byte offset into a `(X, Y)` cell coordinate by walking a standard IFC fragment tree. This replaces the bespoke per-widget cursor math currently duplicated in `render.Input` and `render.TextArea`.

## 2. Design & Requirements

### 2.1 Feature Design
- Add `cursor.FromTextFragment(root *layout.Fragment, byteOffset int) (x, y int, ok bool)`.
- The function walks the fragment tree:
  - **Line dimension (Y):** iterate `root.Children` (one `FragmentLink` per line box from the IFC). Sum the byte length of each line by walking the line box's text-fragment descendants and accumulating `len(cluster.Bytes)`. The first line whose accumulated bytes encloses `byteOffset` yields `y`.
  - **Column dimension (X):** within the matching line box, walk text-fragment children in order; for each `text.Cluster`, accumulate `CellWidth` until reaching `byteOffset`. The accumulated width is `x`.
- Return `(0, 0, false)` if the offset cannot be located (empty fragment, offset out of range).

### 2.2 Rules
- **Pure function:** no state, no side effects, no dependency on the host element.
- **Package isolation:** `cursor` depends on `layout` (for `*layout.Fragment`) and `text` (for `Cluster.Bytes` length and `CellWidth`). Nothing else.
- **Handles trailing offsets:** when `byteOffset` equals the total byte length, return the position immediately after the last cluster on the last line (`x` past the last glyph, `y` at the last line index).
- **Mandatory-break clusters (`\n`):** the IFC currently emits these inside line text but the line breaks already account for them — the helper does not need newline-aware logic; it just sums `len(c.Bytes)` uniformly.
- **Multi-fragment lines:** a line box may contain multiple text fragments (e.g., styled spans). The helper walks them in order.
- **Atomic inlines:** atomic-inline children inside a line box (non-text fragments) contribute 0 bytes to the byte count and their `Size.Width` to the visual X. Document this explicitly even if the input/textarea UA subtrees produce only text fragments.

### 2.3 Out of Scope
- Reverse mapping `(x, y) → byteOffset` (useful for click-to-position; could be a follow-up `cursor.OffsetFromCell`).
- Bi-directional text.
- Multi-fragment selection ranges.

## 3. Implementation Steps
1. Add `cursor/from_text_fragment.go` exposing `FromTextFragment(root *layout.Fragment, byteOffset int) (x, y int, ok bool)`.
2. Implement the two-level walk described in 2.1.
3. Handle edge cases: nil root, empty `Children`, offset past the end, offset equal to a line's terminal byte.
4. Document the function in `cursor/doc.go` and reference TSK-023.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] Single-line fragment, ASCII text: offsets `0`, mid-string, end-of-string return expected `(x, 0)`.
- [ ] Two-line fragment from a `\n` split: offsets in line 1 return `y == 0`; offsets in line 2 return `y == 1`.
- [ ] Soft-wrapped two-line fragment (no `\n`): boundary offset at the wrap point returns `y == 1, x == 0`.
- [ ] Wide CJK clusters: each character contributes `CellWidth == 2`; the helper accumulates correctly.
- [ ] Multi-cluster grapheme (ZWJ emoji): a single grapheme cluster of multiple bytes contributes its single `CellWidth` once.
- [ ] Empty root fragment returns `(0, 0, false)`.
- [ ] `byteOffset` exceeding the total byte count returns `(0, 0, false)`.
- [ ] `byteOffset == total` (end of buffer) returns the cell immediately past the last glyph.

### 4.2 Integration Tests
- [ ] Build a small IFC fragment tree from real text (`"hello\nworld"`) and verify the helper agrees with the X/Y observed by the existing TextArea cursor logic (regression-equivalence test).

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `cursor_from_text_fragment_test.go` reproducing the byte-arithmetic edge cases found during TextArea development (mandatory break at end of last line, scroll-shifted lines, leading-space-collapsed lines).

### 4.4 Benchmarks
- [ ] `BenchmarkFromTextFragment_ShortLine`: ASCII line of 40 chars, offset 20.
- [ ] `BenchmarkFromTextFragment_LongWrapped`: 200-character soft-wrapped text across 5 line boxes, offset in last line. Must complete in under 1 µs to stay within the 60 FPS frame budget when called once per render.

### 4.5 Documentation
- [ ] Document the helper in `cursor/doc.go`.
- [ ] No `AGENT.md` change required (this is a leaf helper, not an architectural rule).
