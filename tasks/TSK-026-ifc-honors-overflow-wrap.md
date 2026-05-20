# TSK-026: IFC Honors `overflow-wrap`

## 1. Objective
Make the Inline Formatting Context (IFC) emergency-break path conditional on the computed `overflow-wrap` (a.k.a. `word-break`) property, instead of always taking one cluster when no break opportunity is found. Without this fix, the IFC silently behaves as if every node has `overflow-wrap: anywhere`, which masks the spec-default `overflow-wrap: normal` (no emergency break, content overflows the line).

This is a deferred IFC cleanup spun off from the input/textarea design session. It is **not a prerequisite** for TSK-018, TSK-022, TSK-023, TSK-024 or TSK-025 — `<input>` (whose intrinsic style is `white-space: nowrap`) and `<textarea>` (whose intrinsic style sets `overflow-wrap: break-word`) get the wrap behavior they want regardless. Other elements that rely on default `overflow-wrap: normal` may currently exhibit the wrong behavior.

## 2. Design & Requirements

### 2.1 Feature Design
The IFC line-breaking algorithm currently has two emergency-break sites in `layout/inline.go` that ignore `ComputedStyle.OverflowWrap`:

1. **`findFittingClusters` (≈ `layout/inline.go:416-423`)** — when `canWrap` is true and the next cluster would overflow with no recorded break opportunity, it returns `count=0, width=0` to signal "let the caller emergency-break."
2. **Caller emergency-break (≈ `layout/inline.go:301-326`)** — on `count==0` and `currentX==0`, the caller unconditionally consumes one cluster from the head of the remaining slice to make forward progress.

After this task, both sites must honor `comp.OverflowWrap`:

- `OverflowWrapNormal` (spec default): **never** emergency-break. The unbreakable run is emitted in full on the current line and is allowed to overflow the available width. The line ends naturally at the next break opportunity (mandatory or container boundary).
- `OverflowWrapBreakWord`: emergency-break **only** when the run would otherwise cause the line to overflow, and only after exhausting normal break opportunities. This is the existing behavior of `OverflowWrapAnywhere` — both values share the same physical break decision; they differ only in how they interact with min-content sizing (see 2.4).
- `OverflowWrapAnywhere`: same emergency-break behavior as `BreakWord`. The distinction lives in `ComputeInlineMinMaxSizes`.

### 2.2 Rules
- **Default behavior changes.** Any element that previously relied on emergency breaking with `OverflowWrap` unset (i.e., implicit default `Normal`) will see different output. This is a **correctness fix**, but consumers should be audited (see 2.5).
- **`WhiteSpaceNoWrap`** disables wrapping entirely — `OverflowWrap` is irrelevant in that mode; content overflows.
- **`WhiteSpacePre`** also disables wrapping; only mandatory breaks. `OverflowWrap` is irrelevant.
- **`WhiteSpaceNormal` and `WhiteSpacePreWrap`** are the only modes where `OverflowWrap` takes effect.
- The min-content contribution (`ComputeInlineMinMaxSizes`) must reflect emergency-break eligibility: `OverflowWrapAnywhere` lowers min-content to the widest single cluster; `OverflowWrapBreakWord` does **not** (matches CSS spec). See 2.4.

### 2.3 Implementation Plan (high level)
1. Thread `OverflowWrap` into `findFittingClusters`. When the property is `Normal`, do **not** return `(0, 0, false)` to request an emergency break; instead, continue accumulating clusters past `availableWidth` until a break opportunity is found or the run ends. Return that accumulated `(count, width, forceBreak)`.
2. In the caller emergency-break site (`layout/inline.go:301-326`), only execute the one-cluster fallback when `comp.OverflowWrap == OverflowWrapBreakWord || OverflowWrapAnywhere`. Otherwise, do not synthesize a break — the cluster sequence overflows the line and the next iteration handles it.
3. Update `ComputeInlineMinMaxSizes` (`layout/inline.go:435+`) so the min-content size lowers to the widest single cluster **only** when `OverflowWrap == OverflowWrapAnywhere`. `BreakWord` keeps the spec-defined min-content of "longest unbreakable run."

### 2.4 Min-Content vs Max-Content Subtlety (informative)
Per CSS Text 3:

| Value | Affects line breaking? | Affects min-content sizing? |
|---|---|---|
| `normal` | No | No |
| `break-word` | **Yes** — break inside unbreakable runs to avoid overflow | **No** — min-content remains the longest unbreakable run |
| `anywhere` | **Yes** — same as `break-word` for line breaking | **Yes** — min-content shrinks to longest single cluster |

The IFC must respect both columns of this table.

### 2.5 Audit & Migration
- Audit existing call-sites and tests that depend on text wrapping (block elements containing text, list items, table cells, flex children) for behavioral changes. Inputs (`<input>`) and textareas (`<textarea>`) are covered by their own tests in TSK-024 / TSK-025.
- Where author intent was emergency-break (which "worked by accident"), update the element's `RawStyle()` or `DefaultStyle()` to set `OverflowWrap: BreakWord` explicitly. **Do not** add a global UA default — `normal` is the spec default and must remain so.

### 2.6 Out of Scope
- Implementing `word-break: break-all` / `word-break: keep-all` semantics beyond what the IFC already does (the `WordBreak` enum exists but is partially honored — separate task if needed).
- Hyphenation, language-specific break tables.

## 3. Implementation Steps
1. Modify `findFittingClusters` in `layout/inline.go` to accept the `OverflowWrap` value (either as an extra parameter or by reading it inline from `comp`). When the property is `Normal` and no break opportunity is found, switch from the "return `(0, 0, false)` and let the caller emergency-break" path to "continue past `availableWidth` and emit the full unbreakable run."
2. Modify the caller emergency-break path (`layout/inline.go:301-326`) to gate the one-cluster fallback on `comp.OverflowWrap == BreakWord || Anywhere`. When `Normal`, skip the fallback entirely; the loop continues with the full run on the current line, which will exceed `currentX > l.width` and trigger the natural line-end logic.
3. Update `ComputeInlineMinMaxSizes` to set the min-content to `max(cluster.CellWidth)` only when `OverflowWrap == OverflowWrapAnywhere`. For `Normal` and `BreakWord`, keep min-content as the longest unbreakable run (existing behavior).
4. Add or update inline doc comments at both code sites to reference CSS Text 3 and this task.
5. Sweep the test suite for places where the implicit emergency break was relied on; either update those tests to expect overflow (preferred, matches the new correct behavior) or set `OverflowWrap: BreakWord` on the test fixture.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] `WhiteSpaceNormal, OverflowWrapNormal`: a 30-character unbreakable run inside a 10-cell-wide container overflows the line (line width > container width, no break inserted). Verify the line fragment's `Size.Width` reflects the overflow.
- [ ] `WhiteSpaceNormal, OverflowWrapBreakWord`: the same 30-character run inside a 10-cell-wide container breaks at column 10 (and again at 20, …). No overflow.
- [ ] `WhiteSpaceNormal, OverflowWrapAnywhere`: same line-break behavior as `BreakWord`. Verified separately because min-content differs.
- [ ] `WhiteSpacePreWrap, OverflowWrapNormal`: text with `\n` plus an unbreakable run preserves newlines as mandatory breaks but does NOT emergency-break the long run. The run overflows its line.
- [ ] `WhiteSpacePreWrap, OverflowWrapBreakWord`: newlines preserved AND the long run wraps at the container edge.
- [ ] `WhiteSpaceNoWrap, OverflowWrapBreakWord`: no wrap occurs regardless of `OverflowWrap` (no-wrap overrides).
- [ ] Mixed content: a soft-break opportunity followed by a long unbreakable run with `Normal` wraps at the soft break, then the run overflows. With `BreakWord`, the run wraps mid-cluster at the container edge.

### 4.2 Integration Tests
- [ ] A block element containing the test strings above renders the expected line counts.
- [ ] A flex item that does not set `OverflowWrap` (implicit `Normal`) overflows its flex line when given an unbreakable run — verify the flex container does NOT mistakenly wrap.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `inline_overflow_wrap_test.go` covering:
  - URLs in body text (`https://example.com/very/long/path/...`) with `Normal` (overflows) vs `BreakWord` (wraps).
  - Multi-byte CJK runs (no spaces) with `Normal` — verify natural per-cluster soft break still works (CJK clusters carry `BreakAnywhere`, so they wrap even under `Normal`).
- [ ] Min-content regression: a paragraph containing one long unbreakable run reports its min-content as the run width when `Normal` or `BreakWord`, but as the widest single cluster when `Anywhere`.

### 4.4 Benchmarks
- [ ] `BenchmarkIFC_OverflowWrapNormal`: baseline, should be unchanged from current performance.
- [ ] `BenchmarkIFC_OverflowWrapBreakWord`: should be within ±5 % of `Normal` (one extra branch per cluster).
- [ ] `BenchmarkComputeInlineMinMaxSizes_Anywhere`: verify the cluster-scan does not regress block layout sizing significantly (< 10 % overhead vs `Normal`).

### 4.5 Documentation
- [ ] Update `layout/doc.go` with a brief note on how `WhiteSpace` and `OverflowWrap` interact, mirroring the table in section 2.4.
- [ ] Update `style/doc.go` (if it documents the wrap properties) to clarify that the spec-default `Normal` does not emergency-break.
- [ ] Update `AGENT.md` "Inline Layout (LayoutNG)" rule (or add a sub-bullet) noting that `OverflowWrap` gates emergency breaks.
- [ ] No `README.md` change unless wrap behavior is documented there.
