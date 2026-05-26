# ADR 026: Kitex Hooks Expansion

## Status

Accepted

## Context

The `kitex` reactive framework currently provides four hooks: `UseState`, `UseRef`, `UseMemo`, and `CreateRef`. While sufficient for simple demos, this set is inadequate for building real applications. Developers cannot run side effects (data fetching, timers, subscriptions), share state across distant components without prop-drilling, or express complex state machines.

This ADR introduces five new core hooks and a `Destroy()` component lifecycle, along with the engine integration strategy for effect scheduling.

## Decision

### 1. Effect Hooks — Dual-Variant API

Two pairs of effect hooks are introduced, each with a "simple" (no cleanup) and "cleanup" variant:

```go
// Post-commit effects (fire after terminal Commit, via deferred macrotask)
func UseEffect(effect func(), deps []any)
func UseEffectCleanup(effect func() func(), deps []any)

// Layout effects (fire synchronously after reconciliation, before Style/Layout/Paint)
func UseLayoutEffect(effect func(), deps []any)
func UseLayoutEffectCleanup(effect func() func(), deps []any)
```

**Rationale:** Go does not support optional return values. Requiring every effect to return a cleanup function (even when unused) adds noise to the common case. Following the `FC`/`SimpleFC` precedent, the simple variant gets the clean name; the cleanup variant gets an explicit suffix.

**Dependency semantics** (matching React):
- `nil` deps → run after every render.
- `[]any{}` (empty slice) → run once on mount, cleanup on unmount.
- `[]any{a, b}` → run when any dep changes (compared via `reflect.DeepEqual`, reusing the existing `depsEqual` helper from `UseMemo`).

**Hook state:** Both variants store an `effectHookState` struct in the component's `hooks []any` slice at the current `hookIndex`:

```go
type effectHookState struct {
    deps       []any
    cleanup    func()       // cleanup from previous execution
    isLayout   bool         // true for UseLayoutEffect variants
    pending    bool         // true when deps changed, awaiting drain
    // One of these is set (not both):
    simpleFn   func()       // for UseEffect / UseLayoutEffect
    cleanupFn  func() func() // for UseEffectCleanup / UseLayoutEffectCleanup
}
```

During render, each effect hook compares deps. If changed (or first mount), the effect is marked `pending = true` and appended to a package-level pending queue (`pendingLayoutEffects` or `pendingEffects`).

### 2. `Destroy()` Lifecycle

A new `Destroy()` method is added to the `componentInstance` interface and implemented on `ComponentNode[P]`:

```go
func (c *ComponentNode[P]) Destroy()
```

`Destroy()` performs:
1. **Effect cleanup:** Iterates `c.hooks`, finds all `effectHookState` entries, and invokes their stored `cleanup` function.
2. **Context unsubscription:** Finds all `contextHookState` entries and removes the component from each provider's subscriber set.
3. **Recursive teardown:** Walks `c.rendered` subtree and calls `Destroy()` on any nested `ComponentNode`.

The reconciler calls `Destroy()` **before** `parent.RemoveChild()` at all unmount sites (lines 94, 104, and 314 of `reconciler.go`), alongside the existing `ClearAllSubscriptions` call.

### 3. Effect Scheduling — Approach C (Zero Engine Changes)

Effect scheduling follows React's model faithfully, requiring no modifications to the engine package:

**Layout effects** are drained synchronously within the `OnComponentDirty` callback, immediately after `reconcile()` returns:

```
OnComponentDirty:
  1. flushPendingEffects()          // guarantee: frame N-1 effects complete first
  2. reconcile(parent, old, new, realNode)
  3. drainLayoutEffects()           // synchronous, may trigger re-reconciliation
  4. if state changed in step 3 → repeat from 2 (capped at N iterations)
```

**Regular effects** are deferred: after reconciliation, kitex enqueues `flushPendingEffects` as an `engine.PostMacro(fn)` macrotask. This macrotask executes during the **next frame's** `drainMacroTasks` phase (line 805 of `engine.go`), after the terminal has committed the previous frame.

**Flush-before-render guarantee:** `flushPendingEffects()` is also called at the top of `OnComponentDirty` (step 1 above), ensuring effects from frame N complete before frame N+1's reconciliation begins. This matches React's `flushPassiveEffects()` call at the top of `performConcurrentWorkOnRoot`.

**Re-entrancy cap:** Layout effect draining that triggers state changes (and thus re-reconciliation) is capped at a configurable iteration limit (e.g., 10) to prevent infinite loops, matching React's behavior.

### 4. Context System — Stack + Subscription

A React-style context system using O(1) stack-based provider lookup and subscription-based dirty propagation:

```go
func CreateContext[T any](defaultValue T) *Context[T]
func UseContext[T any](ctx *Context[T]) T
```

**`Context[T]`** is a typed identity holding:
- `defaultValue T` — returned when no Provider is found.
- `stack []*contextEntry[T]` — active provider entries, pushed/popped during render traversal.

**`ProviderNode[T]`** is a specialized VDOM node (like `ComponentNode`) that:
- Pushes a `contextEntry{value, subscribers}` onto `ctx.stack` before rendering children.
- Pops after children render.
- On value change during reconciliation, iterates its subscriber set and marks consumers dirty.

**`UseContext[T]`** stores a `contextHookState[T]` in the component's `hooks` slice:

```go
type contextHookState[T comparable] struct {
    provider  *contextEntry[T]  // direct reference, stable across re-renders
    lastValue T                 // value at last render, for dedup
}
```

- **First mount:** Reads top of `ctx.stack` (O(1)), subscribes to the entry, stores reference in hook state.
- **Re-renders:** Reads value directly from stored `provider` reference (O(1)). Stack is not consulted.

**Memoization bypass:** When a Provider's value changes, it marks all subscribers dirty. The `Update()` method's memoization check is amended to respect the dirty flag:

```go
// Current:
if c.shouldMemo && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) { ... }
// Updated:
if c.shouldMemo && !oldComp.IsDirty() && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) { ... }
```

**Deduplication:** Before marking a subscriber dirty, the Provider compares its current value against the subscriber's `lastSeenValue`. If equal (subscriber already rendered with the new value during normal reconciliation), the dirty mark is skipped.

**Performance vs. VDOM parent pointers:**

| Metric | Parent Pointers (rejected) | Stack + Subscription (chosen) |
|---|---|---|
| First `UseContext` | O(depth) walk | **O(1)** stack read |
| Re-render lookup | O(1) | O(1) |
| Per-node memory | Pointer on every node | Only on active providers |

### 5. `UseReducer` and `UseCallback`

Thin wrappers with zero architectural impact:

```go
func UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A))
```
Internally calls `UseState[S](initial)`. The `dispatch` function applies the reducer to the current state and calls the setter.

```go
func UseCallback[T any](callback T, deps []any) T
```
Delegates to `UseMemo[T](func() T { return callback }, deps)`.

### 6. Terminal-Specific Convenience Hooks

Userland hooks built on the core primitives, shipped in a separate `kitex/hooks` sub-package:

```go
func UseFocus(ref Ref[dom.Element]) bool
func UseKeyboard(handler func(event.KeyEvent), deps []any)
```

These demonstrate composability and are not part of the core `kitex` package.

## Consequences

- Developers can build real applications: data fetching, subscriptions, timers, shared state, complex forms.
- Zero modifications to the `engine` package — effects integrate via existing `PostMacro` macrotask API.
- The `componentInstance` interface gains a `Destroy()` method — a breaking change internal to the `kitex` package.
- The reconciler must call `Destroy()` at all unmount sites alongside `ClearAllSubscriptions`.
- The `Update()` memoization guard must include a dirty-flag check to support context propagation through memoized subtrees.
- The `Context[T]` stack is a package-level structure protected by the existing `renderMutex`, safe for single-threaded rendering but requiring review if concurrent rendering is ever introduced.
