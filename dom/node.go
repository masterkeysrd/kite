package dom

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/render"
)

// linkable is an unexported interface used internally to update the link
// fields of a Node without exposing mutation methods on the public interface.
// All concrete node types in this package satisfy linkable.
type linkable interface {
	setParent(p Element)
	setNext(n Node)
	setPrev(n Node)
	setOwnerDocument(d Document)
	setConnected(v bool)
}

// asLinkable asserts that n satisfies linkable. All concrete node types
// produced by this package do; a panic here indicates that a foreign Node
// implementation was passed to a mutation method.
func asLinkable(n Node) linkable {
	return n.(linkable)
}

// baseNode holds the common link fields shared by element and textNode.
// It is embedded by value in both concrete types.
type baseNode struct {
	event.Target

	parent        Element
	next          Node
	prev          Node
	ownerDocument Document
	renderObject  render.Object
	// connected is true when this node is reachable from the Document root.
	// It is toggled by the attach/detach walks run inside the mutation
	// methods (see ADR-0036 §3).
	connected bool
}

// Parent returns the parent Element, or nil.
func (b *baseNode) Parent() Element { return b.parent }

// NextSibling returns the next sibling Node, or nil.
func (b *baseNode) NextSibling() Node { return b.next }

// PreviousSibling returns the previous sibling Node, or nil.
func (b *baseNode) PreviousSibling() Node { return b.prev }

// OwnerDocument returns the Document that owns this node.
func (b *baseNode) OwnerDocument() Document { return b.ownerDocument }

// IsConnected reports whether this node is reachable from the Document root.
func (b *baseNode) IsConnected() bool { return b.connected }

// RenderObject returns the associated render.Object, or nil.
func (b *baseNode) RenderObject() render.Object { return b.renderObject }

// SetRenderObject attaches or detaches the render object for this node.
func (b *baseNode) SetRenderObject(ro render.Object) { b.renderObject = ro }

// linkable implementation (unexported setters).
func (b *baseNode) setParent(p Element)         { b.parent = p }
func (b *baseNode) setNext(n Node)              { b.next = n }
func (b *baseNode) setPrev(n Node)              { b.prev = n }
func (b *baseNode) setOwnerDocument(d Document) { b.ownerDocument = d }
func (b *baseNode) setConnected(v bool)         { b.connected = v }

// --- Node interface implementation (defaults for non-container nodes) --------

func (b *baseNode) AppendChild(child Node) Node {
	panic("dom: node does not support children")
}

func (b *baseNode) InsertBefore(newChild, ref Node) Node {
	panic("dom: node does not support children")
}

func (b *baseNode) RemoveChild(child Node) Node {
	panic("dom: node does not support children")
}

func (b *baseNode) ReplaceChild(newChild, oldChild Node) Node {
	panic("dom: node does not support children")
}

func (b *baseNode) FirstChild() Node { return nil }
func (b *baseNode) LastChild() Node  { return nil }

func (b *baseNode) Children() iter.Seq[Node] {
	return func(yield func(Node) bool) {}
}
