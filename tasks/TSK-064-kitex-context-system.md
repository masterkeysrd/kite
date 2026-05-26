# Task: TSK-064 — Kitex Context System (CreateContext, Provider, UseContext)

**ADR:** 026-kitex-hooks-expansion
**Depends on:** TSK-057 (hooks context), TSK-058 (reconciler), TSK-062 (Destroy lifecycle for context unsubscription)

### Summary
Implement the React-style Context system in the `kitex` package: `CreateContext[T]`, `ProviderNode[T]`, and `UseContext[T]`. Uses a stack-based provider lookup with O(1) reads and subscription-based dirty propagation that bypasses automatic memoization.

### Key Design Decisions (from ADR-026)
- **Stack-based lookup:** `Context[T]` maintains an internal `stack []*contextEntry[T]`. Providers push/pop during render. `UseContext` reads top of stack on first mount (O(1)).
- **Subscription model:** Providers maintain a subscriber set. On value change, they mark consumers dirty.
- **Stored provider reference:** After first mount, `UseContext` stores a direct `*contextEntry[T]` reference in hook state. Re-renders read from this reference (O(1)), never consulting the stack.
- **Memoization bypass:** `ComponentNode.Update()` memoization check amended: `if c.shouldMemo && !oldComp.IsDirty() && deepEqualProps(...)`. The dirty flag from context subscription bypasses prop-equality memoization.
- **Deduplication:** Before marking a subscriber dirty, provider compares current value against subscriber's `lastSeenValue`. If equal, skip.

### Files to Create/Modify

#### [NEW] `extras/kitex/context.go`

**Types:**
```go
// Context[T] is a typed context identity created via CreateContext.
type Context[T comparable] struct {
    defaultValue T
    stack        []*contextEntry[T]
}

// contextEntry holds a provider's current value and its subscriber set.
type contextEntry[T comparable] struct {
    value       T
    subscribers map[componentInstance]*contextSubscription[T]
}

// contextSubscription tracks a consumer's last seen value for dedup.
type contextSubscription[T comparable] struct {
    hookIndex int
    lastValue T
}

// contextHookState is stored in a consumer component's hooks slice.
type contextHookState[T comparable] struct {
    ctx      *Context[T]
    entry    *contextEntry[T]
    lastValue T
}

func (c *contextHookState[T]) getValue() any { return c.lastValue }
```

Note: `T` is constrained to `comparable` so that value deduplication (`lastSeenValue != currentValue`) works without reflection.

**Functions:**
```go
func CreateContext[T comparable](defaultValue T) *Context[T]
func UseContext[T comparable](ctx *Context[T]) T
```

**`CreateContext`:** Returns a `&Context[T]{defaultValue: defaultValue}`.

**`UseContext`:**
- Gets current component via `getCurrentComponent()`, increments hook index.
- First call: reads top of `ctx.stack`. If empty, returns `ctx.defaultValue` (no subscription). If non-empty, subscribes to the top entry, stores `contextHookState{ctx, entry, entry.value}` in hook state.
- Subsequent calls: reads `entry.value` from stored hook state. Updates `lastValue`.
- Returns the value.

#### [NEW] `extras/kitex/provider.go`

**`ProviderNode[T comparable]`** — a VDOM node similar to `ComponentNode`:

```go
type ProviderNode[T comparable] struct {
    ctx      *Context[T]
    value    T
    children []Node
    entry    *contextEntry[T]
    rendered []Node // mirrored children for reconciliation
    ref      dom.Node
}
```

Implements the `Node` interface (`TagName()`, `Children()`, `Instantiate()`, `Update()`, `Key()`).

**`Instantiate(doc dom.Document) dom.Node`:**
1. Creates a `contextEntry{value: p.value, subscribers: map...}`.
2. Pushes entry onto `p.ctx.stack`.
3. Instantiates all children into a container (or uses a passthrough DOM node).
4. Pops entry from stack.
5. Stores entry as `p.entry`.

**`Update(el dom.Node, old Node) `:**
1. Transfers entry from old provider.
2. Pushes entry onto stack.
3. If `p.value != oldProvider.value`: updates `entry.value`, notifies subscribers (mark dirty with dedup).
4. Reconciles children.
5. Pops entry from stack.

**Subscriber notification:**
```go
func (e *contextEntry[T]) notifySubscribers(newValue T) {
    for comp, sub := range e.subscribers {
        if sub.lastValue != newValue {
            comp.MarkDirty()
        }
    }
}
```

**Provider factory on Context:**
```go
func (c *Context[T]) Provider(value T, children ...Node) *ProviderNode[T]
```

#### [MODIFY] `extras/kitex/kitex.go`
- Amend `ComponentNode[P].Update()` memoization check (line ~1730 area):
  ```go
  // Before:
  if c.shouldMemo && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) {
  // After:
  if c.shouldMemo && !oldComp.IsDirty() && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) {
  ```

#### [MODIFY] `extras/kitex/kitex.go` — `Destroy()` method
- Add context unsubscription logic to `Destroy()` (from TSK-062):
  ```go
  if ctxHook, ok := h.(*contextHookState[???]); ok {
      // Need type-erased unsubscription
  }
  ```
  Since `contextHookState` is generic, use a `contextUnsubscriber` interface:
  ```go
  type contextUnsubscriber interface {
      unsubscribe(comp componentInstance)
  }
  func (c *contextHookState[T]) unsubscribe(comp componentInstance) {
      delete(c.entry.subscribers, comp)
  }
  ```
  In `Destroy()`: check `if unsub, ok := h.(contextUnsubscriber); ok { unsub.unsubscribe(c) }`

#### [MODIFY] `extras/kitex/reconciler.go`
- Add `ProviderNode` handling in `reconcile()` — treat it similarly to `componentInstance` for child reconciliation.

### Required Unit Tests

#### File: `extras/kitex/context_test.go`

1. `TestCreateContext_DefaultValue` — verify `UseContext` returns default when no Provider.
2. `TestProvider_ProvidesValue` — verify consumer receives Provider's value.
3. `TestProvider_NestedProviders` — verify inner Provider shadows outer for its subtree.
4. `TestProvider_ValueChange_TriggersConsumerReRender` — verify consumer re-renders when Provider value changes.
5. `TestProvider_ValueChange_BypassesMemoization` — verify memoized intermediate component doesn't block consumer re-render.
6. `TestProvider_Deduplication` — verify consumer that already rendered with new value is NOT re-rendered again by subscription notification.
7. `TestProvider_MultipleConsumers` — verify all consumers of the same Provider re-render on value change.
8. `TestProvider_Unmount_Unsubscribes` — verify `Destroy()` removes consumer from subscriber set.
9. `TestUseContext_StableAcrossReRenders` — verify stored provider reference is used (not stack lookup) on re-renders.
10. `TestProvider_DifferentContextTypes` — verify two different `Context[T]` types don't interfere.

### Test Cases
- Theme context: `CreateContext[string]("light")`. Provider sets "dark". Consumer reads "dark".
- Nested: outer Provider sets "dark", inner Provider sets "blue". Consumer under inner reads "blue". Consumer under outer (but not inner) reads "dark".
- Memoization bypass: Provider → MemoizedWrapper → Consumer. Provider value changes. Consumer re-renders even though MemoizedWrapper's props didn't change.
- Unmount consumer → subscriber set shrinks. Provider value change no longer touches the removed consumer.

### Acceptance Criteria
- `CreateContext`, `UseContext`, and Provider work correctly for sharing state.
- O(1) lookup on both first mount and re-renders.
- Memoization bypass works: context changes propagate through memoized subtrees.
- Subscriber deduplication prevents unnecessary re-renders.
- `Destroy()` properly unsubscribes consumers from provider entries.
- No modifications to the `engine` package.

### Documentation Updates
- Update `AGENT.md` to document the context system.
- Add a `UseContext` example to `examples/` demonstrating a theme context with nested providers.
