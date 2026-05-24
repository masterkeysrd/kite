# TSK-061: Automatic Component Memoization

## Objective
Implement an automatic, Just-In-Time component memoization system within the `extras/kitex` package. This prevents expensive cascading VDOM re-allocations and diffs for complex subtrees when their properties remain unchanged, utilizing a zero-overhead bottom-up complexity score to determine if reflection checks are worth the cost.

## Requirements

### 1. The `complexity()` Interface
In `extras/kitex/kitex.go`:
- Add a new unexported method to `nodeInternal` (or `Node`): `complexity() int`.
- **Text Nodes (`textNode`)**: `complexity()` always returns `1`.
- **Element Nodes (`elementNode[P]`)**: Add a private field `score int`. When an element is constructed in its factory function (e.g., `Box()`), compute its score as `1 + sum(child.complexity() for child in children)`.
- **Component Nodes (`ComponentNode[P]`)**: Add a private field `complexityScore int` and `shouldMemo bool`. In `Instantiate()`, right after executing `RenderFn`, read `c.rendered.complexity()` and assign it to `complexityScore`. If `complexityScore > 5` (a hardcoded constant), set `shouldMemo = true`.

### 2. Depth-Limited Reflection Check
- Write an internal generic/reflection helper `func deepEqualProps(oldProps, newProps any, maxDepth int) bool`.
- The function should use `reflect.Value` to recursively compare structs, slices, arrays, maps, and base types.
- If it encounters a `reflect.Func`, it MUST use the existing `funcEquals` helper to determine parity (preventing false negatives on stable closures).
- If the recursion depth hits `maxDepth` (e.g., 3), immediately return `false` to abort. This protects the render loop from spending an unbounded amount of time analyzing massive graph structures.

### 3. Modifying `Update()`
In `ComponentNode[P].Update()`:
- Before executing `c.rendered = c.RenderFn(c.PropsVal)`, check `c.shouldMemo`.
- If `c.shouldMemo` is true, call `deepEqualProps(oldComp.PropsVal, c.PropsVal, 3)`.
- If the props are equal, *skip* `c.RenderFn`. Instead, copy `c.rendered = oldComp.rendered` and instantly return, effectively freezing the diffing process for this entire subtree.
- If `c.shouldMemo` is false, or props have changed, or `maxDepth` was reached, proceed with normal execution.

### 4. `UseMemo` Hook
In `extras/kitex/hooks.go`:
- Implement `func UseMemo[T any](factory func() T, deps []any) T`.
- Store the `deps` slice and the resulting `T` value in a new internal `hookState`.
- On subsequent renders, compare the new `deps` slice to the old `deps` slice element-by-element (shallow compare using basic `!=` or `reflect.DeepEqual` on primitive values).
- If the slices are identical, return the cached `T`. Otherwise, run `factory()`, cache the new `T` and `deps`, and return the new value.

## Verification
- Add exhaustive benchmarks in `kitex_bench_test.go`:
  - Create a deeply nested component tree.
  - Benchmark `Update()` calls *with* memoization active (`shouldMemo=true` and identical props).
  - Benchmark `Update()` calls *without* memoization (simulating `shouldMemo=false` or forcing prop changes).
  - Document the performance delta in the PR/commit message to prove the reflection overhead is offset by the saved render/diff cycles.
- Write a unit test ensuring `deepEqualProps` respects the `maxDepth` limit.
- Write a unit test validating `UseMemo` only calls the factory function when the `deps` values change.
- Verify `complexity()` correctly calculates the node count bottom-up without panic on nil children.