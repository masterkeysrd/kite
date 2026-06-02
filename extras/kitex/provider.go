package kitex

import (
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
	refs     []dom.Node
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
	t := reflect.TypeOf((*T)(nil)).Elem()
	return "provider-" + t.String()
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

// realNodes returns the slice of DOM nodes representing the provider's children.
func (p *ProviderNode[T]) realNodes() []dom.Node {
	return p.refs
}

func (p *ProviderNode[T]) setRefs(els []dom.Node) {
	p.refs = els
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
	p.refs = nil
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

// Instantiate instantiates all children of the provider and returns them as a slice of DOM nodes.
func (p *ProviderNode[T]) Instantiate(doc dom.Document) []dom.Node {
	p.initEntry()
	p.pushEntry()
	defer p.popEntry()

	var reals []dom.Node
	for _, child := range p.children {
		if child != nil {
			reals = append(reals, child.Instantiate(doc)...)
		}
	}
	p.refs = reals
	return reals
}

// Update is called when updating a provider node.
func (p *ProviderNode[T]) Update(els []dom.Node, old Node) {
	p.refs = els
	p.initEntry()
	p.pushEntry()
	defer p.popEntry()

	var oldProv *ProviderNode[T]
	if old != nil {
		if op, ok := old.(*ProviderNode[T]); ok {
			oldProv = op
			p.updateFrom(oldProv)
		}
	}

	if len(p.children) == 1 && oldProv != nil && len(oldProv.children) == 1 {
		p.children[0].Update(els, oldProv.children[0])
	}
}
