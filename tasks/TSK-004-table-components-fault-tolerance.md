# Task: Implement Table DOM Components and Layout Fault Tolerance

## 1. Objective
Create the logical DOM components for Tables (`Table`, `TableRow`, `TableCell`) and update the `TableAlgorithm` to handle malformed table structures by generating anonymous layout rows (analogous to IFC anonymous blocks).

## 2. Design & Requirements

### Feature Design
- **DOM Components (`element/table.go`):**
  - `element.Table` (`table`): Defaults to `DisplayTable`.
  - `element.TableRow` (`tr`): Defaults to `DisplayTableRow`.
  - `element.TableCell` (`td`): Defaults to `DisplayTableCell`. Must include `SetColSpan(int)` and `SetRowSpan(int)` methods (store these as properties on the logical node, which layout will read via an interface).
- **Layout Fault Tolerance (`layout/table.go`):**
  - Do **not** modify `engine/engine.go` or the render tree generation.
  - When the `TableAlgorithm` iterates its children during measurement and layout passes, it must check `child.Style().Display`.
  - If a child is a `DisplayTableRow`, process normally.
  - If a child is a `DisplayTableCell` (or any non-row element), group it and any contiguous non-row siblings into a virtual "Anonymous Row". Route this virtual group through the `TableRowAlgorithm` logic.

### Rules
- **Logical Purity:** Do not modify the logical `dom.Node` tree or the `render.Box` tree to fix the structure. 
- **Engine Ignorance:** The core engine (`syncRenderTree`) must remain unaware of table semantics.

## 3. Implementation Steps
1. **DOM Components:** Create `element/table.go` and implement the 3 structs, embedding `elementBase[Self]`.
2. **Col/Row Span Interfaces:** Define an interface (e.g., `layout.TableSpanner`) that `TableCell` implements so the layout engine can read the span values without importing the `element` package.
3. **TableAlgorithm Update:** Modify `TableAlgorithm` (from TSK-003) to group non-row children into anonymous rows during its iteration loops.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: `element.TableCell` correctly stores and returns ColSpan and RowSpan values.
- [ ] Test case 2: A `DisplayTable` layout node containing only `DisplayTableCell` nodes correctly groups them into a single virtual row during layout measurement without panicking.

### 4.2. Integration Tests
- [ ] Build a malformed table in Go (`Table.AppendChild(TableCell)`), run a full frame tick, and assert the layout engine successfully renders the cell within the table structure.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test ensuring that updating the text inside a malformed table cell successfully invalidates the table layout and recalculates intrinsic bounds.

### 4.4. Benchmarks
- [ ] N/A.

### 4.5. Documentation
- [ ] Update `README.md` with an example of building a table.
