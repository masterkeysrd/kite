package dom

import (
	"iter"
)

// element is the concrete, unexported implementation of Element.
type element struct {
	baseNode
	tagName string
	id      string
}

// Compile-time assertion.
var _ Element = (*element)(nil)

// NewElement allocates an element with the given tag name and owner document.
// If self is nil, the element's self pointer defaults to itself.
func NewElement(doc Document, tag string, self Node) Element {
	return newElement(tag, doc, self)
}

// newElement allocates an element with the given tag name and owner document.
func newElement(tag string, doc Document, self Node) *element {
	e := &element{tagName: tag}
	e.ownerDocument = doc
	if self == nil {
		e.self = e
	} else {
		e.self = self
	}
	e.kind = KindElement
	e.name = tag
	return e
}

// asBase returns the underlying *baseNode.
func (e *element) asBase() *baseNode { return &e.baseNode }

// TagName returns the tag name.
func (e *element) TagName() string { return e.tagName }

// ID returns the element's identifier.
func (e *element) ID() string { return e.id }

// SetID sets the element's identifier. Registry maintenance follows
// ADR-0036 §10: when the node is disconnected only the stored id field is
// updated; the identity registry is touched only when the node is connected.
func (e *element) SetID(id string) {
	if e.id == id {
		return
	}
	r, hasReg := e.ownerDocument.(idRegistrar)
	if hasReg && e.connected && e.id != "" {
		r.unregisterID(e.id)
	}
	e.id = id
	if hasReg && e.connected && id != "" {
		// Use self for registration so wrappers are registered.
		if el, ok := e.self.(Element); ok {
			r.registerID(id, el)
		}
	}
}

// ReplaceWith replaces this element with the given nodes.
func (e *element) ReplaceWith(nodes ...Node) Element {
	parent := e.parent
	if parent == nil {
		return e
	}
	ref := e.next
	parent.RemoveChild(e.self)
	for _, n := range nodes {
		parent.InsertBefore(n, ref)
	}
	return e
}

// ChildNodes (from Node) is implemented by baseNode.

// Children is a helper that returns only Element children.
// Deprecated: use ChildNodes() and filter by kind.
func (e *element) Children() iter.Seq[Node] {
	return e.ChildNodes()
}
