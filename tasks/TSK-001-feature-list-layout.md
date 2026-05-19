# Task: Implement List Layout Algorithm

## 1. Objective
Implement the `ListAlgorithm` in the layout engine to render list items with markers (bullets, numbers) using a virtual, layout-driven two-column structure, adhering to the principles defined in ADR 001.

## 2. Design & Requirements

### Feature Design
- **Style Additions:** 
  - Add `DisplayListItem` to `style.Display`.
  - Add `ListStyleType` enum to `style` package with values: `None`, `Disc` (`•`), `Circle` (`○`), `Square` (`■`), and `Decimal` (`1.`).
- **ListAlgorithm (`layout/list.go`):** 
  - Triggers when `node.Style().Display == style.DisplayListItem`.
  - Generates a transient text fragment for the marker based on `ListStyleType`.
  - For `Decimal`, calculates the ordinal by walking `node.PreviousSibling()` and counting consecutive `DisplayListItem` nodes.
  - Formats the layout as a two-column row:
    - **Column 1:** The synthesized marker fragment (measured to its intrinsic width).
    - **Column 2:** The actual child content, formatted via standard Block layout rules, constrained to the remaining available inline size.

### Rules
- **No New Render Objects:** Do not modify `/render` or `/engine` to inject phantom nodes.
- **Immutable Fragments:** The marker must be returned as a standard physical fragment in the tree so the Paint engine requires zero modifications.

## 3. Implementation Steps
1. **Update Styles:** Add `DisplayListItem` to `style/enums.go` and `ListStyleType` to `style/style.go` and `style/computed.go`.
2. **Algorithm Routing:** Update the main layout routing function (e.g., `layout.Compute` or `render.LayoutPhase`) to route `DisplayListItem` to a new `ListAlgorithm`.
3. **Implement ListAlgorithm:** Create `layout/list.go`. Mimic the setup of `BlockAlgorithm` but prepend the logical row generation for the marker.
4. **Ordinal Helper:** Implement a helper in `ListAlgorithm` to compute the ordinal number if `ListStyleType` is `Decimal`.
5. **Shaper Integration:** Use the text shaper to measure and create the inline item for the marker string.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: `ListStyleType: Disc` generates a marker fragment containing "• " and accurately reduces available width for block children.
- [ ] Test case 2: `ListStyleType: Decimal` computes correct ordinals (1., 2., 3.) for consecutive list items.
- [ ] Test case 3: Interrupted numbering. If a node is *not* a `DisplayListItem`, the ordinal count resets or behaves according to DOM structure logic.

### 4.2. Integration Tests
- [ ] Verify that a `DisplayListItem` with multi-line text content wraps the text strictly within Column 2 (it should not wrap *under* the bullet marker).

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test ensuring `DisplayListItem` without children does not crash when synthesizing a marker.

### 4.4. Benchmarks
- [ ] Benchmark `ListAlgorithm` with 100 consecutive `Decimal` list items to ensure the $O(N)$ sibling walk for ordinals does not significantly degrade the 60FPS layout budget.

### 4.5. Documentation
- [ ] Update `AGENT.md` (specifically the layout section) to note the `ListAlgorithm` virtual fragment generation strategy.
- [ ] Update `layout/doc.go` to mention the List Formatting Context.
