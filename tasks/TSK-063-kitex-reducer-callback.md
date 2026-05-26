# Task: TSK-063 — Kitex UseReducer and UseCallback Hooks

**ADR:** 026-kitex-hooks-expansion
**Depends on:** TSK-057 (hooks context)

### Summary
Implement `UseReducer` and `UseCallback` hooks in the `kitex` package. These are thin wrappers over existing primitives (`UseState` and `UseMemo` respectively) with zero architectural impact.

### Files to Create/Modify

#### [MODIFY] `extras/kitex/hooks.go`

**`UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A))`**
- Internally calls `UseState[S](initial)` to get `(getState, setState)`.
- Creates a `dispatch` function that:
  1. Calls `getState()` to get current state.
  2. Applies `reducer(currentState, action)` to get new state.
  3. Calls `setState(newState)`.
- Returns `(getState, dispatch)`.
- The dispatch closure must capture the `componentRef` indirection (inherited from `UseState`) so it always operates on the current active component instance.

**`UseCallback[T any](callback T, deps []any) T`**
- Delegates to `UseMemo[T](func() T { return callback }, deps)`.
- Returns the memoized callback reference.
- Note: In Go, `T` is constrained to `any` not `func(...)`. The type safety for function types relies on the caller passing the correct type. This is acceptable because Go generics don't support function-type constraints.

### Required Unit Tests

#### File: `extras/kitex/hooks_test.go` (append to existing)

1. `TestUseReducer_InitialState` — verify initial state is returned on first render.
2. `TestUseReducer_Dispatch` — verify dispatching an action applies the reducer and returns new state.
3. `TestUseReducer_MultipleDispatches` — verify sequential dispatches accumulate correctly (e.g., counter reducer with increment/decrement actions).
4. `TestUseReducer_DispatchTriggersReRender` — verify dispatch marks the component dirty and triggers `OnComponentDirty`.
5. `TestUseReducer_StableGetterAcrossRenders` — verify `getState` function identity is stable across re-renders (same closure).
6. `TestUseCallback_ReturnsSameRef` — verify same deps returns the same function reference.
7. `TestUseCallback_UpdatesOnDepsChange` — verify changed deps returns a new function reference.
8. `TestUseCallback_NilDeps` — verify nil deps returns a new reference every render.

### Test Cases
- Counter reducer: `func(state int, action string) int` with "increment"/"decrement" actions.
- Dispatch 3 increments → state is 3.
- Dispatch from outside render (via stored dispatch reference) → works correctly via `componentRef` indirection.
- `UseCallback` with `deps = []any{x}`: render with x=1 → ref A; render with x=1 → ref A (same); render with x=2 → ref B (new).

### Acceptance Criteria
- `UseReducer` correctly wraps `UseState` and the dispatch function applies the reducer.
- `UseCallback` correctly delegates to `UseMemo`.
- Both hooks integrate with the existing hook index cursor (incrementing `hookIndex` exactly once per call).
- Dispatching actions from `UseReducer` triggers component re-render through the existing `MarkDirty` → `OnComponentDirty` pipeline.
- All tests pass.

### Documentation Updates
- Update `AGENT.md` to list the new hooks.
- Add a `UseReducer` example to `examples/` showing a form with multiple fields managed by a reducer.
