# Task: Implement Table Layout Algorithm

## 1. Objective
Implement the layout algorithms (`TableAlgorithm`, `TableRowAlgorithm`, `TableCellAlgorithm`) to support grid-based table layouts, including `ColSpan` and `RowSpan` support, based on the design in ADR 002.

## 2. Design & Requirements

### Feature Design
- **Style Additions (`style/enums.go`):** 
  - Add `DisplayTable`, `DisplayTableRow`, `DisplayTableCell` to the `Display` enum.
- **Layout Routing (`layout/compute.go` or equivalent):**
  - Route `DisplayTable` -> `TableAlgorithm`.
  - Route `DisplayTableRow` -> `TableRowAlgorithm`.
  - Route `DisplayTableCell` -> `BlockAlgorithm` (cells just act as BFCs with rigid constraints passed by the row).
- **`TableAlgorithm` (Two-Pass):**
  - **Pass 1 (Grid Sizing):** Walk all rows and cells. Calculate the `MinMaxSizes` of each column. Handle `ColSpan` by distributing the spanned cell's minimum width across the target columns.
  - **Pass 2 (Layout):** Generate the `BoxFragmentBuilder`. Call layout on rows, passing down the resolved column widths.
- **`TableRowAlgorithm`:**
  - Lays out child cells horizontally. Forces the cell's `AvailableSize.Width` to match the column width provided by the parent Table.

### Rules
- **No New Render Objects:** Do not create `render.Table`. Use `render.Box`.
- **Immutability:** The output must be an Immutable Fragment Tree.

## 3. Implementation Steps
1. **Enums:** Add table display types to `style`.
2. **TableAlgorithm:** Create `layout/table.go`. Implement the measurement pass to calculate column widths.
3. **TableRowAlgorithm:** Implement the row layout pass.
4. **Layout Router:** Update the layout switch statement to instantiate these algorithms.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: A 2x2 table correctly aligns cell widths so columns are uniform.
- [ ] Test case 2: A cell with `ColSpan(2)` correctly forces the combined width of the two columns to encompass its text.

### 4.2. Integration Tests
- [ ] Verify that a `DisplayTable` correctly shrinks to fit its content if `Width` is Auto, or stretches if `Width` is Percent.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] N/A for this specific algorithm task (fault tolerance handled in a separate task).

### 4.4. Benchmarks
- [ ] Benchmark a 50x10 table layout to ensure the two-pass algorithm does not severely impact the 60FPS budget.

### 4.5. Documentation
- [ ] Update `AGENT.md` layout section to mention the two-pass table layout.
- [ ] Document `TableAlgorithm` in `layout/doc.go`.
