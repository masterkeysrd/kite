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

// CreateContext creates a Context[T] identity with a default value.
func CreateContext[T comparable](defaultValue T) *Context[T] {
	return &Context[T]{
		defaultValue: defaultValue,
		pool: &sync.Pool{
			New: func() any { return &ProviderNode[T]{} },
		},
	}
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
