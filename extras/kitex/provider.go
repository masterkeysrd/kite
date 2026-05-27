package kitex

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/masterkeysrd/kite/dom"
)

// ProviderNode is a virtual/passthrough context provider node.
type ProviderNode[T comparable] struct {
	ctx      *Context[T]
	value    T
	children []Node
	entry    *contextEntry[T]
	pool     *sync.Pool
}

// Ensure compile-time interface compliance.
var _ Node = (*ProviderNode[int])(nil)
var _ nodeInternal = (*ProviderNode[int])(nil)
var _ providerInstance = (*ProviderNode[int])(nil)

// Provider constructs a ProviderNode for the given context and value.
func (c *Context[T]) Provider(value T, children ...Node) *ProviderNode[T] {
	p := c.pool.Get().(*ProviderNode[T])
	p.ctx = c
	p.value = value
	p.children = children
	p.entry = nil
	p.pool = c.pool
	return p
}

// TagName returns a type-specific tag for the provider.
func (p *ProviderNode[T]) TagName() string {
	var zero T
	return "provider-" + reflect.TypeOf(zero).String()
}

// Props returns the provider's current value.
func (p *ProviderNode[T]) Props() any {
	return p.value
}

// Children returns the provider's virtual children.
func (p *ProviderNode[T]) Children() []Node {
	return p.children
}

// Key returns the provider's key (always empty).
func (p *ProviderNode[T]) Key() string {
	return ""
}

// realNode returns nil because the ProviderNode has no DOM element of its own.
func (p *ProviderNode[T]) realNode() dom.Node {
	return nil
}

func (p *ProviderNode[T]) containsProvider() bool {
	return true
}

func (p *ProviderNode[T]) isProvider() bool {
	return true
}

func (p *ProviderNode[T]) hasDirectProvider() bool {
	return false // A provider itself is not its own direct provider in this context
}

// complexity returns the pre-computed node-count of the subtree.
func (p *ProviderNode[T]) complexity() int {
	score := 1
	for _, child := range p.children {
		if child != nil {
			if n, ok := child.(nodeInternal); ok {
				score += n.complexity()
			}
		}
	}
	return score
}

// Release releases resources back to the pool (or nils them).
func (p *ProviderNode[T]) Release() {
	if p.pool == nil {
		return
	}
	p.children = nil
	p.entry = nil
	pool := p.pool
	p.pool = nil
	pool.Put(p)
}

// initEntry initializes the context entry for this provider.
func (p *ProviderNode[T]) initEntry() {
	if p.entry == nil {
		p.entry = &contextEntry[T]{
			value:       p.value,
			subscribers: make(map[*componentRef]*contextSubscription[T]),
		}
	}
}

// pushEntry pushes the entry onto the context stack.
func (p *ProviderNode[T]) pushEntry() {
	p.initEntry()
	p.ctx.push(p.entry)
}

// popEntry pops the entry from the context stack.
func (p *ProviderNode[T]) popEntry() {
	p.ctx.pop()
}

// updateFrom transfers context entry and notifies subscribers of value changes.
func (p *ProviderNode[T]) updateFrom(old providerInstance) {
	if oldProv, ok := old.(*ProviderNode[T]); ok {
		p.entry = oldProv.entry
		if p.value != oldProv.value {
			p.entry.value = p.value
			p.entry.notifySubscribers(p.value)
		}
	}
}

// notifySubscribers notifies all subscribed consumers when the context value changes.
func (e *contextEntry[T]) notifySubscribers(newValue T) {
	for ref, sub := range e.subscribers {
		if sub.lastValue != newValue {
			ref.mu.Lock()
			node := ref.node
			ref.mu.Unlock()
			if node != nil {
				node.MarkDirty()
			}
		}
	}
}

// Instantiate is called when the ProviderNode is treated as a single root.
func (p *ProviderNode[T]) Instantiate(doc dom.Document) dom.Node {
	p.initEntry()
	p.pushEntry()
	defer p.popEntry()

	if len(p.children) == 1 {
		return p.children[0].Instantiate(doc)
	}
	panic(fmt.Sprintf("ProviderNode (%s) used as a single VDOM node root must have exactly one child", p.TagName()))
}

// Update is called when the ProviderNode is treated as a single root.
func (p *ProviderNode[T]) Update(el dom.Node, old Node) {
	p.initEntry()
	p.pushEntry()
	defer p.popEntry()

	if len(p.children) == 1 {
		var oldChild Node
		if oldProv, ok := old.(*ProviderNode[T]); ok && len(oldProv.children) == 1 {
			oldChild = oldProv.children[0]
			p.updateFrom(oldProv)
		}
		p.children[0].Update(el, oldChild)
		return
	}
	panic(fmt.Sprintf("ProviderNode (%s) used as a single VDOM node root must have exactly one child", p.TagName()))
}
