package dom

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
)

// Implementation defines the set of factory functions and utilities provided
// by the internal/dom package.
type Implementation struct {
	NewDocument     func() Document
	NewElement      func(doc Document, tag string, self Node) Element
	NewTextNode     func(doc Document, data string, self Node) TextNode
	LayoutChildren  func(n Node) iter.Seq[Node]
	Outer           func(n Node) Node
	IsUANode        func(n Node) bool
	UARoot          func(el Element) Node
	DefaultScroller func(host Element) event.Scrollable
}

var impl Implementation

// RegisterImplementation registers the concrete implementation of the DOM.
// This is called by internal/dom's init() function.
func RegisterImplementation(i Implementation) {
	impl = i
}

// NewDocument creates and returns a new, empty Document.
func NewDocument() Document {
	if impl.NewDocument == nil {
		panic("dom: implementation not registered. Did you forget to import internal/dom?")
	}
	return impl.NewDocument()
}

// NewElement allocates an element with the given tag name and owner document.
func NewElement(doc Document, tag string, self Node) Element {
	return impl.NewElement(doc, tag, self)
}

// NewTextNode allocates a TextNode with the given data and owner document.
func NewTextNode(doc Document, data string, self Node) TextNode {
	return impl.NewTextNode(doc, data, self)
}

// LayoutChildren returns an iterator that yields the engine-visible children of n.
func LayoutChildren(n Node) iter.Seq[Node] {
	return impl.LayoutChildren(n)
}

// Outer returns the self wrapper for el when one has been adopted.
func Outer(n Node) Node {
	return impl.Outer(n)
}

// IsUANode reports whether n is part of a UA shadow subtree.
func IsUANode(n Node) bool {
	return impl.IsUANode(n)
}

// UARoot returns the UA shadow subtree root attached to el, or nil.
func UARoot(el Element) Node {
	return impl.UARoot(el)
}

// DefaultScroller returns a new Scrollable implementation for the given host.
func DefaultScroller(host Element) event.Scrollable {
	return impl.DefaultScroller(host)
}
