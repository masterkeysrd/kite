# Task: TSK-062 — Kitex Effect Hooks and Destroy Lifecycle

**ADR:** 026-kitex-hooks-expansion
**Depends on:** TSK-057 (hooks context), TSK-058 (reconciler)

### Summary
Implement `UseEffect`, `UseEffectCleanup`, `UseLayoutEffect`, `UseLayoutEffectCleanup` hooks and the `Destroy()` component lifecycle in the `kitex` package. This includes the effect scheduling mechanism (Approach C — zero engine changes).

### Key Design Decisions (from ADR-026)
- **Dual-variant API:** `UseEffect(fn func(), deps []any)` (no cleanup) and `UseEffectCleanup(fn func() func(), deps []any)` (with cleanup). Same pattern for `UseLayoutEffect`.
- **Dep semantics:** `nil` → every render; `[]any{}` → mount only; `[]any{a,b}` → on change (via existing `depsEqual`).
- **Layout effects** fire synchronously after `reconcile()` within `OnComponentDirty`.
- **Regular effects** are deferred via `engine.PostMacro` macrotask, flushed at start of next frame's Task Draining.
- **Flush-before-render guarantee:** `flushPendingEffects()` called at top of `OnComponentDirty` before reconciliation.
- **Re-entrancy cap:** Layout effect draining that triggers state changes is capped at 10 iterations.
- **`Destroy()` lifecycle:** New method on `componentInstance` interface. Runs effect cleanups, unsubscribes from context providers, recursively destroys child components.

### Files to Create/Modify

#### [MODIFY] `extras/kitex/hooks.go`
- Add `effectHookState` struct:
  ```go
  type effectHookState struct {
      deps       []any
      cleanup    func()
      isLayout   bool
      pending    bool
      simpleFn   func()
      cleanupFn  func() func()
  }
  ```
- Add package-level pending queues:
  ```go
  var (
      pendingLayoutEffects []*pendingEffect
      pendingEffects       []*pendingEffect
      effectsMutex         sync.Mutex
  )
  type pendingEffect struct {
      state *effectHookState
  }
  ```
- Implement `UseEffect(effect func(), deps []any)` — gets current component via `getCurrentComponent()`, increments hook index, on first call creates `effectHookState{simpleFn: effect, deps: deps, pending: true}` and appends to `pendingEffects`. On subsequent calls, compares deps via `depsEqual`; if changed, marks `pending = true` and appends to queue.
- Implement `UseEffectCleanup(effect func() func(), deps []any)` — same as above but sets `cleanupFn` instead of `simpleFn`.
- Implement `UseLayoutEffect(effect func(), deps []any)` — same as `UseEffect` but sets `isLayout = true` and appends to `pendingLayoutEffects`.
- Implement `UseLayoutEffectCleanup(effect func() func(), deps []any)` — same pattern.
- Implement `drainLayoutEffects()` — iterates `pendingLayoutEffects`, for each: run `cleanup` if exists, run effect (either `simpleFn` or `cleanupFn`), store new cleanup, clear `pending`. Clear the queue.
- Implement `flushPendingEffects()` — same as above but for `pendingEffects` queue.
- Implement `scheduleEffectFlush()` — uses a package-level `var postMacroFn func(func())` to enqueue `flushPendingEffects` as a macrotask. The `postMacroFn` is set during kitex initialization by the application (bridges to `engine.PostMacro`).

#### [MODIFY] `extras/kitex/kitex.go`
- Add `Destroy()` method to `ComponentNode[P]`:
  ```go
  func (c *ComponentNode[P]) Destroy() {
      for _, h := range c.hooks {
          if eff, ok := h.(*effectHookState); ok {
              if eff.cleanup != nil {
                  eff.cleanup()
              }
          }
          // Context cleanup will be handled by TSK-064
      }
      if c.rendered != nil {
          destroyNode(c.rendered)
      }
  }
  ```
- Add `destroyNode(n Node)` helper that walks the VDOM subtree calling `Destroy()` on any `componentInstance` nodes, recursing into children.
- Add `Destroy()` to the `componentInstance` interface.

#### [MODIFY] `extras/kitex/reconciler.go`
- In `OnComponentDirty` callback (line 15): add `flushPendingEffects()` call before `reconcile()`, and `drainLayoutEffects()` call after `reconcile()` with re-entrancy cap.
- At all unmount sites (lines 94, 104, 314): call `destroyNode(oldNode)` before `ClearAllSubscriptions(realNode)`.
- In `Render()` when unmounting root (line 56): call `destroyNode(oldRoot)` before removing.

#### [NEW] `extras/kitex/effects_test.go`
- Tests for all four effect variants.

### Required Unit Tests
1. `TestUseEffect_RunsAfterFlush` — verify effect runs when `flushPendingEffects()` is called, not during render.
2. `TestUseEffect_DepsNil_RunsEveryRender` — verify effect with nil deps runs after every re-render.
3. `TestUseEffect_DepsEmpty_RunsOnce` — verify effect with `[]any{}` runs only on mount.
4. `TestUseEffect_DepsChanged_Reruns` — verify effect re-runs when deps change.
5. `TestUseEffectCleanup_CleansUpBeforeRerun` — verify cleanup from previous run is called before new effect runs.
6. `TestUseEffectCleanup_CleansUpOnUnmount` — verify cleanup runs when component is destroyed.
7. `TestUseLayoutEffect_RunsSynchronouslyAfterReconcile` — verify layout effects drain within `OnComponentDirty`.
8. `TestUseLayoutEffect_CanTriggerReRender` — verify layout effect that calls setState triggers re-reconciliation.
9. `TestUseLayoutEffect_ReentrancyCap` — verify infinite loop is capped.
10. `TestDestroy_RunsAllCleanups` — verify `Destroy()` calls cleanup for all effect hooks.
11. `TestDestroy_RecursiveChildren` — verify `Destroy()` recurses into nested component subtrees.
12. `TestFlushBeforeRender_Guarantee` — verify pending effects from previous frame are flushed before new reconciliation.

### Benchmarks
- `BenchmarkUseEffect` — measure time taken to run effects with varying numbers of pending effects.
- `BenchmarkDestroy` — measure time taken to destroy a component with varying numbers of effect hooks and nested children.
- `BenchmarkLayoutEffectReRender` — measure time taken for a layout effect that triggers a re-render, including the re-entrant effect drain.
- `BenchmarkFlushBeforeRender` — measure time taken for `flushPendingEffects()` when called at the start of `OnComponentDirty`.
- `BenchmarkDepsChange` — measure time taken to compare deps and schedule effects on change.
- `BenchmarkNoDepsChange` — measure time taken when deps do not change (should skip effect).

### Test Cases
- Effect with `nil` deps and component re-renders 3 times → effect callback invoked 3 times.
- Effect with `[]any{count}` where count changes from 1→2→2→3 → effect runs 3 times (mount, 1→2, 2→3), skips 2→2.
- Cleanup effect returns `func() { canceled = true }` → on deps change, `canceled` becomes true before new effect runs.
- Component unmounts → all effect cleanups run, including nested component cleanups.
- Layout effect calls `setState` → triggers re-reconciliation within same `OnComponentDirty` call.
- Layout effect infinite loop → capped at 10 iterations, does not hang.

### Acceptance Criteria
- All four effect hooks work correctly with the hook index cursor mechanism.
- Layout effects fire synchronously after reconcile; regular effects fire deferred via macrotask.
- `flushPendingEffects` is called before reconciliation (flush-before-render guarantee).
- `Destroy()` is called at all reconciler unmount sites.
- All cleanup functions run in the correct order.
- No modifications to the `engine` package.

### Documentation Updates
- Update `AGENT.md` to list the new hooks.
- Add an example in `examples/` demonstrating `UseEffect` for a timer and `UseEffectCleanup` for a subscription.
