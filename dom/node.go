package dom

import (
	"iter"
	"strings"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/render"
)

var _ Node = (*baseNode)(nil)

type baseNode struct {
	event.Target

	name           string
	kind           Kind
	self           Node
	parent         Node
	next           Node
	prev           Node
	firstChild     Node
	lastChild      Node
	ownerDocument  Document
	renderObject   render.Object
	connected      bool
	needsSync      bool
	childNeedsSync bool
}

// asBase returns the underlying *baseNode for any Node produced by this package.
func asBase(n Node) *baseNode {
	if n == nil {
		return nil
	}

	// If the node provides the baseNode directly, we are done.
	if b, ok := n.(interface{ asBase() *baseNode }); ok {
		return b.asBase()
	}

	// Otherwise, it's a wrapper, so unwrap it and try again.
	return asBase(n.Unwrap())
}

func (b *baseNode) adopt(newDoc Document) {
	if b.ownerDocument == newDoc {
		return
	}
	b.ownerDocument = newDoc
	for n := b.firstChild; n != nil; n = n.NextSibling() {
		asBase(n).adopt(newDoc)
	}
}

func (b *baseNode) Kind() Kind { return b.kind }

func (b *baseNode) NodeName() string { return b.name }

func (b *baseNode) Parent() Node { return b.parent }

func (b *baseNode) ParentElement() Element {
	if el, ok := b.parent.(Element); ok {
		return el
	}
	return nil
}

func (b *baseNode) NextSibling() Node                { return b.next }
func (b *baseNode) PreviousSibling() Node            { return b.prev }
func (b *baseNode) OwnerDocument() Document          { return b.ownerDocument }
func (b *baseNode) IsConnected() bool                { return b.connected }
func (b *baseNode) RenderObject() render.Object      { return b.renderObject }
func (b *baseNode) SetRenderObject(ro render.Object) { b.renderObject = ro }

func (b *baseNode) NeedsSync() bool      { return b.needsSync }
func (b *baseNode) ChildNeedsSync() bool { return b.childNeedsSync }

func (b *baseNode) MarkNeedsSync() {
	if b.needsSync {
		return
	}
	b.needsSync = true
	// Propagate ChildNeedsSync up.
	for p := b.parent; p != nil; p = p.Parent() {
		pb := asBase(p)
		if pb.childNeedsSync {
			break
		}
		pb.childNeedsSync = true
	}
}

func (b *baseNode) ClearSyncFlags() {
	b.needsSync = false
	b.childNeedsSync = false
}

func (b *baseNode) Unwrap() Node { return nil }

func (b *baseNode) FirstChild() Node { return b.firstChild }
func (b *baseNode) LastChild() Node  { return b.lastChild }

func (b *baseNode) HasChildNodes() bool { return b.firstChild != nil }

func (b *baseNode) ChildNodes() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for n := b.firstChild; n != nil; {
			next := n.NextSibling()
			if !yield(n) {
				return
			}
			n = next
		}
	}
}

func (b *baseNode) AppendChild(child Node) Node {
	return b.self.InsertBefore(child, nil)
}

func (b *baseNode) InsertBefore(newChild, ref Node) Node {
	if b.kind == KindText {
		panic("dom: text node does not support children")
	}

	if newChild == b.self {
		panic("dom: cannot insert a node into itself")
	}

	// Cross-document check.
	if newChild.OwnerDocument() != b.ownerDocument {
		if newChild.IsConnected() {
			panic("dom: cross-document insertion of connected nodes is forbidden")
		}
		asBase(newChild).adopt(b.ownerDocument)
	}

	// Remove from old parent if any.
	if p := newChild.Parent(); p != nil {
		p.RemoveChild(newChild)
	}

	newBase := asBase(newChild)
	newBase.parent = b.self

	if ref == nil {
		// Link as last child.
		newBase.prev = b.lastChild
		newBase.next = nil
		if b.lastChild != nil {
			asBase(b.lastChild).next = newChild
		} else {
			b.firstChild = newChild
		}
		b.lastChild = newChild
	} else {
		// Link before ref.
		if ref.Parent() != b.self {
			panic("dom: reference node is not a child of this node")
		}
		refBase := asBase(ref)
		newBase.prev = refBase.prev
		newBase.next = ref
		if refBase.prev != nil {
			asBase(refBase.prev).next = newChild
		} else {
			b.firstChild = newChild
		}
		refBase.prev = newChild
	}

	// Trigger attach walk if connected.
	if b.connected {
		if w, ok := b.ownerDocument.(interface {
			attachWalk(parent Node, child Node)
		}); ok {
			w.attachWalk(b.self, newChild)
		}
	}

	b.notifyStructureChange()
	return newChild
}

func (b *baseNode) RemoveChild(child Node) Node {
	if child.Parent() != b.self {
		panic("dom: node to be removed is not a child of this node")
	}

	cBase := asBase(child)

	// Trigger detach walk if connected.
	if b.connected {
		if w, ok := b.ownerDocument.(interface {
			detachWalk(parent Node, child Node)
		}); ok {
			w.detachWalk(b.self, child)
		}
	}

	// Unlink.
	if cBase.prev != nil {
		asBase(cBase.prev).next = cBase.next
	} else {
		b.firstChild = cBase.next
	}
	if cBase.next != nil {
		asBase(cBase.next).prev = cBase.prev
	} else {
		b.lastChild = cBase.prev
	}

	cBase.parent = nil
	cBase.next = nil
	cBase.prev = nil
	cBase.connected = false

	b.notifyStructureChange()
	return child
}

func (b *baseNode) ReplaceChild(newChild, oldChild Node) Node {
	ref := oldChild.NextSibling()
	b.RemoveChild(oldChild)
	b.InsertBefore(newChild, ref)
	return oldChild
}

func (b *baseNode) Contains(descendant Node) bool {
	for n := descendant; n != nil; n = n.Parent() {
		if n == b.self {
			return true
		}
	}
	return false
}

func (b *baseNode) TextContent() string {
	if b.kind == KindText {
		if tn, ok := b.self.(TextNode); ok {
			return tn.Data()
		}
	}
	var sb strings.Builder
	for child := range b.ChildNodes() {
		sb.WriteString(child.TextContent())
	}
	return sb.String()
}

func (b *baseNode) CloneNode(deep bool) Node {
	var clone Node
	switch b.kind {
	case KindDocument:
		// Cloning a document is usually not supported or returns a special copy.
		// Kite v2 interfaces suggest Document is special.
		panic("dom: cloning document is not supported")
	case KindElement:
		el := b.self.(Element)
		clone = b.ownerDocument.CreateElement(el.TagName(), nil)
		if el.ID() != "" {
			clone.(Element).SetID(el.ID())
		}
	case KindText:
		tn := b.self.(TextNode)
		clone = b.ownerDocument.CreateTextNode(tn.Data(), nil)
	}

	if deep {
		for child := range b.ChildNodes() {
			clone.AppendChild(child.CloneNode(true))
		}
	}
	return clone
}

func (b *baseNode) notifyStructureChange() {
	b.self.MarkNeedsSync()
}
