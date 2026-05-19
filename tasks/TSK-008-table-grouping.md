# Task: Table Section Grouping (thead, tbody, tfoot)

## 1. Objective
Add support for table sections (`thead`, `tbody`, `tfoot`) to the logical DOM and Layout Engine, aligning with CSS standard behavior. This allows developers to create synchronized, sticky headers and semantic data groups within tables.

## 2. Design & Requirements
- **Feature Design:**
  - Standard HTML tables use `thead`, `tbody`, and `tfoot` to organize rows.
  - The layout engine must group rows by these sections, ensuring headers always appear first and footers always appear last, regardless of DOM insertion order.
  - Sizing constraints (column widths) must be synchronized across all sections.
  - **Fault Tolerance (Anonymous Groups):** If a `display: table` element contains direct `display: table-row` children (no body wrapper), the engine must wrap them in an anonymous table row group, preserving the single unified grid constraint context.
- **Rules:**
  - Add `DisplayTableHeaderGroup`, `DisplayTableRowGroup`, and `DisplayTableFooterGroup` to `style.Display`.
  - Create logical components: `element.TableHeader` (`thead`), `element.TableBody` (`tbody`), and `element.TableFooter` (`tfoot`) in `element/table.go`.
  - Update `TableAlgorithm` (from TSK-003 and TSK-004) to support sorting children into sections before measurement and layout.

## 3. Implementation Steps
1. **Enums:** Update `style/enums.go` to add the three new display types.
2. **DOM Components:** In `element/table.go`, add `TableHeader`, `TableBody`, and `TableFooter` with their appropriate default styles.
3. **Layout Logic (Sorting):** In `layout/table.go` (`TableAlgorithm`), add a pre-processing step to categorize children into `headers`, `bodies`, and `footers` lists.
4. **Layout Logic (Anonymous Groups):** While iterating children in `TableAlgorithm`, wrap direct `DisplayTableRow` children into an implicit `bodies` group.
5. **Layout Processing:** Update the two-pass layout to iterate through `headers` -> `bodies` -> `footers` sequentially.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test Case 1: Elements are laid out in order `thead` -> `tbody` -> `tfoot` even if inserted into the DOM as `tfoot` -> `tbody` -> `thead`.
- [ ] Test Case 2: Column widths are synchronized correctly when a wide cell is in the `tbody` and a narrow cell is in the `thead`.
- [ ] Test Case 3: A table with only `DisplayTableRow` children (no groups) is correctly measured and laid out via anonymous grouping.

### 4.2. Integration Tests
- [ ] Verify that a `TableHeader` retains its styles (e.g. `BorderBottom`) without breaking the global table layout.

### 4.5. Documentation
- [ ] Update `element/doc.go` to document the new `thead`, `tbody`, `tfoot` components.
