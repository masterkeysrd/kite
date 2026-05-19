# ADR 004: Table Layout Builder Pattern

## Status
Accepted

## Context
As specified in `ADR-002`, our Table Layout Engine must handle several complex requirements simultaneously:
1. Two-pass layout (Grid Sizing followed by Fragment Generation).
2. Sorting children into semantic groups (`thead`, `tbody`, `tfoot`) and handling anonymous row wrappers.
3. Distributing intrinsic sizes across multiple columns via `ColSpan`.
4. Enforcing an implicit `-1` coordinate overlap between adjacent borders to support the global screen-space paint resolver (`TSK-010`).

Managing this mutable state directly inside `TableAlgorithm.Layout()` results in a massive, unreadable function that violates our clean architecture goals. The generic `BoxFragmentBuilder` provided in `layout/builders.go` is insufficient for grid-based state management.

## Decision
We will introduce a dedicated builder pattern specifically for tables: `TableFragmentBuilder` (and/or an internal `GridSizingBuilder`).

### Responsibilities of the Builder:
1. **Grouping State:** It will maintain the categorized slices of table sections (`headers`, `bodies`, `footers`) during the initial sorting traversal.
2. **Matrix Management:** It will encapsulate the `MinMaxSizes` array for column tracks and expose methods to distribute `ColSpan` constraints across these tracks.
3. **Coordinate Math:** It will encapsulate the implicit `-1` coordinate overlap logic. Instead of the algorithm doing raw math, the algorithm will call `builder.AddCell(frag, colIndex)`, and the builder will calculate the correct `layout.Point` utilizing the resolved column widths and the border collision state.
4. **Finalization:** It will yield a standard, immutable `*layout.Fragment` identical in shape to what `BoxFragmentBuilder` produces.

## Consequences

### Positive
- **Clean Algorithm:** `TableAlgorithm` becomes a clean coordinator that simply traverses nodes and orchestrates passes, rather than managing raw slice manipulation and coordinate math.
- **Testability:** The matrix math, spanning distribution, and overlap calculation can be unit-tested in isolation against the Builder, rather than requiring full layout integration tests.

### Negative / Trade-offs
- **Allocation Cost:** Introduces an additional stateful object allocation per table layout pass. However, given that tables are macro-layout containers (not typically nested thousands of deep), this allocation cost is negligible compared to the clarity gained.
