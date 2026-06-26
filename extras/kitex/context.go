package kitex

import "sync"

// Context is a typed context identity created via CreateContext.
type Context[T comparable] struct {
	defaultValue T
	stack        []*contextEntry[T]
	pool         *sync.Pool
}

// contextEntry holds a provider's current value and its subscriber set.
type contextEntry[T comparable] struct {
	value       T
	subscribers map[*componentRef]*contextSubscription[T]
}

// contextSubscription tracks a consumer's last seen value for dedup.
type contextSubscription[T comparable] struct {
	hookIndex int
	lastValue T
}

// contextHookState is stored in a consumer component's hooks slice.
type contextHookState[T comparable] struct {
	ctx       *Context[T]
	entry     *contextEntry[T]
	lastValue T
}

func (c *contextHookState[T]) getValue() any { return c.lastValue }

type contextUnsubscriber interface {
	unsubscribe(comp componentInstance)
}

func (c *contextHookState[T]) unsubscribe(comp componentInstance) {
	if c.entry != nil {
		delete(c.entry.subscribers, comp.getRef())
	}
}

// contextRegistry is the type-erased interface for capturing/restoring stacks of any Context.
type contextRegistry interface {
	captureStack() any
	restoreStack(any)
	hasStack() bool
}

func (c *Context[T]) hasStack() bool {
	return len(c.stack) > 0
}

func (c *Context[T]) captureStack() any {
	if len(c.stack) == 0 {
		return nil
	}
	s := make([]*contextEntry[T], len(c.stack))
	copy(s, c.stack)
	return s
}

func (c *Context[T]) restoreStack(val any) {
	if val == nil {
		if len(c.stack) == 0 {
			return
		}
		c.stack = nil
		return
	}
	c.stack = val.([]*contextEntry[T])
}

var (
	contextsMu  sync.RWMutex
	allContexts []contextRegistry
)

// contextSnapshot holds a copy of all context stacks at a specific point in time.
type contextSnapshot struct {
	states []any
}

func captureContexts() contextSnapshot {
	contextsMu.RLock()
	defer contextsMu.RUnlock()

	anyActive := false
	for _, ctx := range allContexts {
		if ctx.hasStack() {
			anyActive = true
			break
		}
	}
	if !anyActive {
		return contextSnapshot{}
	}

	states := make([]any, len(allContexts))
	for i, ctx := range allContexts {
		states[i] = ctx.captureStack()
	}
	return contextSnapshot{states: states}
}

func restoreContexts(snap contextSnapshot) {
	contextsMu.RLock()
	defer contextsMu.RUnlock()

	if len(snap.states) == 0 {
		for _, ctx := range allContexts {
			ctx.restoreStack(nil)
		}
		return
	}

	for i, ctx := range allContexts {
		if i < len(snap.states) {
			ctx.restoreStack(snap.states[i])
		} else {
			ctx.restoreStack(nil)
		}
	}
}

// CreateContext creates a Context[T] identity with a default value.
func CreateContext[T comparable](defaultValue T) *Context[T] {
	ctx := &Context[T]{
		defaultValue: defaultValue,
		pool: &sync.Pool{
			New: func() any { return &ProviderNode[T]{} },
		},
	}
	contextsMu.Lock()
	allContexts = append(allContexts, ctx)
	contextsMu.Unlock()
	return ctx
}

func (c *Context[T]) push(entry *contextEntry[T]) {
	c.stack = append(c.stack, entry)
}

func (c *Context[T]) pop() {
	if len(c.stack) > 0 {
		c.stack = c.stack[:len(c.stack)-1]
	}
}

func (c *Context[T]) top() *contextEntry[T] {
	if len(c.stack) == 0 {
		return nil
	}
	return c.stack[len(c.stack)-1]
}

// UseContext retrieves the current value of the context.
func UseContext[T comparable](ctx *Context[T]) T {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseContext must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		// First call: read top of stack
		entry := ctx.top()
		if entry == nil {
			// No provider: return default value
			hs := &contextHookState[T]{
				ctx:       ctx,
				entry:     nil,
				lastValue: ctx.defaultValue,
			}
			comp.setHookState(idx, hs)
			return ctx.defaultValue
		}

		// Subscribe to the top entry
		sub := &contextSubscription[T]{
			hookIndex: idx,
			lastValue: entry.value,
		}
		entry.subscribers[comp.getRef()] = sub

		hs := &contextHookState[T]{
			ctx:       ctx,
			entry:     entry,
			lastValue: entry.value,
		}
		comp.setHookState(idx, hs)
		return entry.value
	}

	hs := stateVal.(*contextHookState[T])
	currentEntry := ctx.top()

	if hs.entry != currentEntry {
		// Provider changed! Unsubscribe from old provider
		if hs.entry != nil {
			delete(hs.entry.subscribers, comp.getRef())
		}
		// Subscribe to new provider
		hs.entry = currentEntry
		if currentEntry != nil {
			sub := &contextSubscription[T]{
				hookIndex: idx,
				lastValue: currentEntry.value,
			}
			currentEntry.subscribers[comp.getRef()] = sub
			hs.lastValue = currentEntry.value
		} else {
			hs.lastValue = ctx.defaultValue
		}
	}

	if hs.entry == nil {
		return hs.lastValue
	}

	// Subsequent calls: read from entry.value
	val := hs.entry.value
	hs.lastValue = val

	if sub, ok := hs.entry.subscribers[comp.getRef()]; ok {
		sub.lastValue = val
	}
	return val
}
