# ADR 018: Layout ContainingSpace and ContainerSpace

## Status
Accepted

## Context
Kite's LayoutNG-inspired layout engine passes constraints down the tree via `ConstraintSpace`. Currently, each formatting context algorithm (Block, Flex, List, Table) independently computes the parent's decoration sizes (border + padding), derives the content-box width, and manually builds child `ConstraintSpace` values. This leads to two problems:

1. **Duplication:** The decoration subtraction and child constraint building logic is copy-pasted across `BlockAlgorithm`, `ListAlgorithm`, `TableSectionAlgorithm`, and partially in `FlexAlgorithm`. Block and List share ~40 lines of nearly identical child constraint code.

2. **Inconsistency:** `PercentageResolutionSize` is set to different values across algorithms — sometimes the parent content-box width, sometimes the available width, sometimes the section width. Under Kite's strict border-box model (ADR-017), percentage dimensions must always resolve against the parent's border-box, but the current code resolves against the content-box (inner size). This is a correctness bug.

Children also have no direct access to their parent's resolved dimensions — they only see `AvailableSize` (which has their own margins subtracted) and `PercentageResolutionSize` (which is inconsistently set).

## Decision

We will introduce two explicit spatial concepts into `ConstraintSpace` and a shared helper function:

### 1. `ContainingSpace` (Size)
The parent's **border-box** dimensions (the full resolved size of the parent element). This is the reference for **percentage resolution**: a child with `width: 50%` resolves as `ContainingSpace.Width * 50 / 100`.

### 2. `ContainerSpace` (Size)
The parent's **content-box** dimensions (border-box minus border minus padding). This is the **available space for children** before individual child margins are subtracted. Algorithms use this to compute per-child `AvailableSize`.

The relationship is always: `ContainerSpace = ContainingSpace - Border - Padding`.

### 3. `PercentageResolutionSize` Removal
The existing `PercentageResolutionSize` field is removed from `ConstraintSpace`. Its role is fully replaced by `ContainingSpace`.

### 4. Updated `ConstraintSpace`
```go
type ConstraintSpace struct {
    AvailableSize     Size        // Per-child: ContainerSpace - child margins (or explicit size)
    ContainingSpace   Size        // Parent's border-box. Percentage resolution base.
    ContainerSpace    Size        // Parent's content-box. Available space for children.
    IsFixedInlineSize bool
    IsFixedBlockSize  bool
    BreakToken        *BreakToken
}
```

### 5. Shared `BuildChildSpace` Function
A standalone function centralizes the child constraint space generation that is currently duplicated across algorithms:

```go
func BuildChildSpace(child Node, container Size, containingSpace Size, parentSpace ConstraintSpace) ConstraintSpace
```

This function:
- Reads the child's computed style (Width/Height kind)
- Computes `AvailableSize` from `container` minus child margins
- Resolves `KindPercent` against `containingSpace` (the parent's border-box)
- Resolves `KindCells` to explicit values
- Sets `IsFixedInlineSize` / `IsFixedBlockSize` flags appropriately
- Passes `ContainingSpace` and `ContainerSpace` through for grandchildren
- Handles `KindAuto` (stretch for block, shrink-wrap check for inline-block/table)

### 6. IFC Wrapping Exclusion
Inline Formatting Context grouping (the iterator-based inline child collection pattern) remains in individual algorithms. `BuildChildSpace` handles only the constraint-space math for block-level children.

## Consequences

### Positive
- **Correctness:** Percentage resolution is now consistently against the parent's border-box, aligned with ADR-017 (strict border-box model).
- **Deduplication:** ~40 lines of child constraint building removed from each of Block, List, and partially Table/Flex algorithms, replaced by a single `BuildChildSpace` call.
- **Parent Visibility:** Children have direct access to parent dimensions via `ContainingSpace` and `ContainerSpace`, enabling better intrinsic sizing decisions.
- **Consistency:** All algorithms use the same percentage resolution and available space semantics — no more ad-hoc `SetPercentageResolutionSize` calls with varying values.
- **TSK-040 Subsumption:** The strict border-box audit task becomes largely addressed by design, since decoration math is centralized rather than scattered across algorithms.

### Negative / Trade-offs
- **Migration Scope:** Every algorithm that creates child `ConstraintSpace` values must be updated. The entry point (`LayoutPhase`) must set the initial `ContainingSpace` and `ContainerSpace`.
- **Two Fields Instead of One:** `ContainingSpace` and `ContainerSpace` are always related (`ContainerSpace = ContainingSpace - decorations`), so there is mild redundancy. However, pre-computing both avoids repeated subtraction in hot paths and makes the semantic intent explicit at each usage site.
