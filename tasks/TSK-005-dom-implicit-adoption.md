# Task: Implement Implicit DOM Adoption

## 1. Objective
Update the `dom` package to support implicit document adoption when appending or inserting detached nodes, enabling the construction of UI trees without threading a single `Document` reference (ADR 003).

## 2. Design & Requirements

### Feature Design
- **`dom.Node` modifications:**
  - Introduce an internal method `adopt(newDoc Document)` that recursively walks a node and its children, updating their `ownerDocument` pointers.
- **`dom.baseNode` (`AppendChild`, `InsertBefore`):**
  - Before appending, check if `child.OwnerDocument() != parent.OwnerDocument()`.
  - If they differ, check if the child is connected (`child.IsConnected()`).
  - If connected, **panic** (cross-document append of live nodes is forbidden).
  - If detached, call `child.adopt(parent.OwnerDocument())` to align the ownership before running the attach walk.

### Rules
- Ensure `adopt` walks the entire subtree of the appended node.
- Do not remove the developer panic for *connected* nodes.

## 3. Implementation Steps
1. Add `setOwnerDocument(Document)` or `adopt(Document)` helper to `baseNode`.
2. Recursively apply the new document to `firstChild` through `lastChild`.
3. Update `AppendChild` and `InsertBefore` (or wherever the core tree mutation logic resides in `dom/node.go` or `dom/element.go`) to trigger adoption for detached foreign nodes.
4. Update `TestElement_CrossDocumentAppend_PanicsInDev` in `dom/lifecycle_test.go` to only panic if the node is connected, and add a new test for successful adoption of detached nodes.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: Appending a detached node created by `doc2` into `doc1` successfully updates the `ownerDocument` of the node and all its children to `doc1`.
- [ ] Test case 2: Attempting to append a node from `doc2` that is *already connected* to `doc2`'s tree into `doc1` still panics.

### 4.2. Integration Tests
- [ ] N/A (Handled entirely within `dom` package tests).

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test to ensure `GetElementByID` registration works correctly for adopted nodes when they finally connect to the new document.

### 4.4. Benchmarks
- [ ] N/A.

### 4.5. Documentation
- [ ] Update `dom/doc.go` to explicitly explain the new implicit adoption rules for detached subtrees.
