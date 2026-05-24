# TSK-058: Kitex VDOM Reconciler (Diffing Engine)

## Description
Implement the core Virtual DOM Diffing algorithm (Reconciler) that translates `kitex` VDOM trees into imperative mutations on Kite's logical `dom` tree.

## Requirements
- **Mount Phase:** Implement the initial render logic that takes a root VDOM node (`react.Node`), creates the corresponding `dom.Node` instances, applies attributes/listeners, and calls `AppendChild` on the host document container.
- **Diffing Phase (Reconciliation):** Implement the tree traversal logic that compares `oldVNode` vs `newVNode`.
  - Handle Text Node diffs (update text content).
  - Handle Element Node diffs (add/remove children, update properties, attach/detach event listeners).
  - Handle Component Node diffs (re-execute the `RenderFn` if dirty, diff the resulting child tree).
- **List Keys:** Support a `Key` concept in the VDOM to efficiently reorder child elements without destroying and recreating underlying `dom.Element`s.
- **Engine Integration:** Expose a `react.Render(root react.Node, container dom.Element)` function. Ensure that all DOM mutations implicitly leverage Kite's existing `NeedsSync` / `ChildNeedsSync` flags.

## Testing
- **Integration Test:** Render a complex component tree (with nested `FC`s and mapped lists) to a mock `dom.Document`. Verify the resulting `dom.Node` tree matches expectations.
- **Update Test:** Trigger a state update using `UseState` and verify that the Reconciler generates minimal mutations (e.g., changes one `dom.TextNode` instead of replacing the entire container).
- **Regression Tests (`tests/regressions/`):** Add scenarios testing rapid state updates to ensure no memory leaks or orphaned `dom.Node` instances occur.
