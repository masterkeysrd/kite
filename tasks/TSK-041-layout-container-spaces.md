# TSK-041: Introduce ContainingSpace and ContainerSpace into Layout

## 1. Objective
Refactor `ConstraintSpace` to carry two explicit parent-size references — `ContainingSpace` (parent border-box for percentage resolution) and `ContainerSpace` (parent content-box for child available space) — and extract a shared `BuildChildSpace` function that eliminates the duplicated child constraint generation across all layout algorithms.

## 2. Design & Requirements

- **Feature Design:** See ADR-018 for the full rationale.
- **Rules:**
  - `ContainingSpace` is always the parent's resolved **border-box** dimensions. All `KindPercent` resolutions MUST use this field. This aligns with ADR-017 (strict border-box model).
  - `ContainerSpace` is always `ContainingSpace - border - padding` (the parent's content-box). It represents the total available space for children before individual child margins are subtracted.
  - The existing `PercentageResolutionSize` field MUST be removed. All call sites that read it must switch to `ContainingSpace`.
  - `BuildChildSpace` is a standalone function (not a method on any algorithm). It handles only block-level child constraint generation. IFC wrapping (inline child grouping) remains in the algorithms.
  - `ConstraintSpaceBuilder` must be updated: remove `SetPercentageResolutionSize`, add `SetContainingSpace` and `SetContainerSpace`.
  - The root `ConstraintSpace` created in `render.LayoutPhase()` (`render/view.go:357`) must set `ContainingSpace` and `ContainerSpace` to the viewport size (since the viewport has no border/padding, they are equal).

## 3. Implementation Steps

### Step 1: Update `ConstraintSpace` struct (`layout/ng.go:53-68`)
Replace:
```go
type ConstraintSpace struct {
    AvailableSize            Size
    PercentageResolutionSize Size
    IsFixedInlineSize        bool
    IsFixedBlockSize         bool
    BreakToken               *BreakToken
}
```
With:
```go
type ConstraintSpace struct {
    AvailableSize   Size
    ContainingSpace Size        // Parent's border-box. Percentage resolution base.
    ContainerSpace  Size        // Parent's content-box. Available space for children.
    IsFixedInlineSize bool
    IsFixedBlockSize  bool
    BreakToken        *BreakToken
}
```

### Step 2: Update `ConstraintSpaceBuilder` (`layout/builders.go:1-40`)
- Remove `SetPercentageResolutionSize`.
- Add `SetContainingSpace(size Size)` and `SetContainerSpace(size Size)`.
- Update `NewConstraintSpaceBuilder` — the default initialization should no longer auto-set `PercentageResolutionSize = AvailableSize`. Instead, `ContainingSpace` and `ContainerSpace` default to zero and must be explicitly set by the caller.

### Step 3: Implement `BuildChildSpace` function (`layout/ng.go` or new file `layout/resolve.go`)
Signature:
```go
func BuildChildSpace(child Node, containerSpace Size, containingSpace Size, parentSpace ConstraintSpace) ConstraintSpace
```
This function consolidates the child constraint logic currently duplicated in `BlockAlgorithm.Layout()` (lines 148-189) and `ListAlgorithm.Layout()` (lines 136-167). It must:

1. Read `childStyle := child.Style()` and `childMargin := childStyle.Margin`.
2. Compute `childAvailWidth := max(0, containerSpace.Width - childMargin.Left - childMargin.Right)`.
3. Compute `childAvailHeight := max(0, containerSpace.Height - childMargin.Top - childMargin.Bottom)` (or use remaining block offset if provided).
4. Resolve inline size based on `childStyle.Width.Kind()`:
   - `KindCells` → `IsFixedInlineSize = true`, `AvailableSize.Width = childStyle.Width.CellsValue()`
   - `KindPercent` → `IsFixedInlineSize = true`, `AvailableSize.Width = int(float32(containingSpace.Width) * childStyle.Width.PercentValue() / 100.0)` — NOTE: resolves against `containingSpace` (border-box), not `containerSpace`.
   - `KindAuto` → `IsFixedInlineSize = true`, `AvailableSize.Width = childAvailWidth` (except for `DisplayTable` which stays unfixed).
   - `KindMaxContent` → `IsFixedInlineSize = false` (child calls `ComputeMinMaxSizes` itself).
5. Resolve block size based on `childStyle.Height.Kind()`:
   - `KindCells` → `IsFixedBlockSize = true`, `AvailableSize.Height = childStyle.Height.CellsValue()`
   - `KindPercent` → only if `parentSpace.IsFixedBlockSize`: `IsFixedBlockSize = true`, `AvailableSize.Height = int(float32(containingSpace.Height) * childStyle.Height.PercentValue() / 100.0)`.
   - Otherwise → `IsFixedBlockSize = false`.
6. Set `ContainingSpace = containingSpace` and `ContainerSpace = containerSpace` (these flow through for grandchildren — the child's own algorithm will compute its own container/containing space for its children).
7. Propagate `BreakToken` from `parentSpace` if applicable.

**Important:** The `childAvailHeight` computation in the current code also accounts for `builder.CurrentBlockOffset()` (remaining vertical space). `BuildChildSpace` should accept an optional `remainingBlockHeight int` parameter, or the caller can pre-adjust `containerSpace.Height` before calling. Recommend the latter to keep the function signature clean.

### Step 4: Update `render.LayoutPhase()` (`render/view.go:357-381`)
The viewport entry point must set both new fields:
```go
space := layout.NewConstraintSpaceBuilder(available).
    SetContainingSpace(available).  // Viewport border-box = full terminal size
    SetContainerSpace(available).   // Viewport has no border/padding
    SetIsFixedInlineSize(true).
    SetIsFixedBlockSize(true).
    ToConstraintSpace()
```

### Step 5: Refactor `BlockAlgorithm.Layout()` (`layout/block.go`)
1. After resolving `resolvedInlineSize`, compute once:
   ```go
   containingSpace := Size{Width: resolvedInlineSize, Height: a.Space.AvailableSize.Height}
   containerSpace := Size{Width: contentWidth, Height: max(0, a.Space.AvailableSize.Height - parentDecorY)}
   ```
2. Replace the child constraint building block (lines 148-189) with:
   ```go
   adjustedContainer := Size{Width: containerSpace.Width, Height: max(0, containerSpace.Height - builder.CurrentBlockOffset())}
   childSpace := BuildChildSpace(child, adjustedContainer, containingSpace, a.Space)
   ```
3. Remove the manual `childSpaceBuilder` calls, the `KindCells`/`KindPercent`/`KindAuto` switch, and the `SetPercentageResolutionSize` call.

### Step 6: Refactor `ListAlgorithm.Layout()` (`layout/list.go`)
Same pattern as Block, but use `column2Width` as the container width:
```go
containingSpace := Size{Width: resolvedInlineSize, Height: a.Space.AvailableSize.Height}
containerSpace := Size{Width: column2Width, Height: max(0, a.Space.AvailableSize.Height - parentDecorY)}
```
Replace lines 136-167 with `BuildChildSpace` call.

### Step 7: Refactor `FlexAlgorithm.Layout()` (`layout/flex.go`)
1. After resolving `resolvedWidth`/`resolvedHeight`, set:
   ```go
   containingSpace := Size{Width: resolvedWidth, Height: resolvedHeight}
   containerSpace := Size{Width: resolvedWidth - decorX, Height: resolvedHeight - decorY}
   ```
2. Use `containerSpace` instead of manually computing `contentMainSize`/`contentCrossSizeForItems` from decorations.
3. Update child `ConstraintSpace` building in the measurement pass (line 320-336) to pass `ContainingSpace` and `ContainerSpace`.
4. Note: Flex has specialized constraint logic (flex basis, grow/shrink) so `BuildChildSpace` may not apply directly. The key win here is using `ContainerSpace`/`ContainingSpace` instead of re-deriving from decorations, and passing them to children.

### Step 8: Refactor `TableAlgorithm` and sub-algorithms (`layout/table.go`)
1. `TableAlgorithm.Layout()`: After resolving `resolvedInlineSize`, set `ContainingSpace` and `ContainerSpace` and pass them to section children.
2. `TableSectionAlgorithm.Layout()`: Pass parent's containing/container space to row children.
3. `TableRowAlgorithm.Layout()`: When building cell constraint spaces (line 514-517), set `ContainingSpace` to the cell's border-box width and `ContainerSpace` to the cell's content-box width.

### Step 9: Update `OverlayAlgorithm` (`layout/overlay.go`)
Update the `ConstraintSpaceBuilder` call (line 16) to set `ContainingSpace` and `ContainerSpace` from the parent's space.

### Step 10: Update `IntrinsicMinMaxSizes` and `IntrinsicBlockSize` (`layout/ng.go:146-163`)
These functions create `ConstraintSpace` with empty/probe values. Set `ContainingSpace` and `ContainerSpace` to zero (intrinsic sizes should not depend on parent sizes). Verify this doesn't break any algorithm's `ComputeMinMaxSizes()`.

### Step 11: Update all `ComputeMinMaxSizes()` methods
Audit each algorithm's `ComputeMinMaxSizes()` method:
- `BlockAlgorithm.ComputeMinMaxSizes()` (`block.go:233-313`) — does not use `PercentageResolutionSize`, no change needed beyond field rename.
- `FlexAlgorithm.ComputeMinMaxSizes()` (`flex.go:94-151`) — same, no percentage usage.
- `ListAlgorithm.ComputeMinMaxSizes()` (`list.go:204-235`) — same.
- `TableAlgorithm.ComputeMinMaxSizes()` (`table.go:180-235`) — same.

### Step 12: Compile and fix remaining references
Search the entire codebase for `PercentageResolutionSize` and ensure zero references remain. Compile with `go build ./...` to catch any missed sites.

## 4. Testing Requirements

### 4.1. Unit Tests
Add a new test file `layout/container_space_test.go`:

- [ ] **`TestBuildChildSpace_KindCells`**: Child with `Width: 10` (cells). Verify `AvailableSize.Width == 10` and `IsFixedInlineSize == true`.
- [ ] **`TestBuildChildSpace_KindPercent`**: Parent `ContainingSpace.Width = 100` (border-box). Child with `Width: 50%`. Verify `AvailableSize.Width == 50` (resolved against border-box, NOT content-box).
- [ ] **`TestBuildChildSpace_KindAuto`**: Child with `Width: auto`, margin 5 each side, `ContainerSpace.Width = 80`. Verify `AvailableSize.Width == 70` and `IsFixedInlineSize == true`.
- [ ] **`TestBuildChildSpace_KindMaxContent`**: Verify `IsFixedInlineSize == false`.
- [ ] **`TestBuildChildSpace_HeightPercent_FixedParent`**: Parent `IsFixedBlockSize = true`, `ContainingSpace.Height = 40`. Child `Height: 50%`. Verify `AvailableSize.Height == 20`.
- [ ] **`TestBuildChildSpace_HeightPercent_AutoParent`**: Parent `IsFixedBlockSize = false`. Child `Height: 50%`. Verify `IsFixedBlockSize == false` (percentage height undefined when parent is auto).
- [ ] **`TestBuildChildSpace_PassthroughFields`**: Verify `ContainingSpace` and `ContainerSpace` are passed through to the returned `ConstraintSpace`.
- [ ] **`TestBuildChildSpace_DisplayTable_Auto`**: Child with `Display: table`, `Width: auto`. Verify `IsFixedInlineSize == false` (tables shrink-wrap).

### 4.2. Integration Tests
Add to `layout/container_space_test.go`:

- [ ] **`TestPercentResolvesAgainstBorderBox`**: Build a tree: parent with `Width: 20`, `Border: Single` (1 each side), `Padding: 2` each side. Child with `Width: 50%`. Run layout. Verify child fragment `Size.Width == 10` (50% of 20, the border-box). This tests the correctness fix — the old code would yield 50% of 14 (the content-box) = 7.
- [ ] **`TestContainerSpaceFlowsToGrandchild`**: Three-level tree: root (W:100) → parent (W:50, P:5) → child (W:50%). Verify child resolves 50% against parent's border-box (50), not root's.
- [ ] **`TestBlockChildUsesContainerSpace`**: Parent with `Width: 40`, `Border: 1` each, `Padding: 2` each. Child with `Width: auto`, `Margin: 3` each side. Verify child `AvailableSize.Width == 40 - 2 - 4 - 6 == 28` (border-box minus border minus padding minus margins).
- [ ] **`TestFlexChildReceivesContainingSpace`**: Flex parent with explicit width. Flex child with `Width: 50%`. Verify correct resolution.
- [ ] **`TestListChildReceivesContainerSpace`**: List item with marker. Block child after marker. Verify the child's available width accounts for the marker column correctly.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] **`TestRegression_PercentWidthBorderBox`**: A golden test with a parent (border + padding) containing a percentage-width child. Captures the visual output to ensure the percentage resolves against the border-box. Place in `tests/regressions/percent_width_border_box_test.go`.
- [ ] **`TestRegression_NestedPercentResolution`**: A three-level nesting scenario with percentage widths at each level. Ensures correct cascading. Place in `tests/regressions/nested_percent_resolution_test.go`.

### 4.4. Benchmarks
- [ ] **`BenchmarkBuildChildSpace`**: Benchmark the new function to ensure it adds no measurable overhead compared to the inline code it replaces. Should be sub-microsecond per call.
- [ ] **`BenchmarkLayoutWithContainerSpace`**: Run the full layout pass on a representative tree (50+ nodes, mixed Block/Flex/List) and compare against baseline to confirm no regression.

### 4.5. Documentation
- [ ] Update `layout/` package-level `doc.go` (if it exists) to document `ContainingSpace`, `ContainerSpace`, and `BuildChildSpace`.
- [ ] Update `AGENT.md` if it references `PercentageResolutionSize` or constraint space semantics.
- [ ] No `README.md` update needed — this is an internal layout engine change with no public API impact.
