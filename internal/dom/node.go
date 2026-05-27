package dom

import (
	"iter"
	"strings"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/render"
)

var _ dom.Node = (*BaseNode)(nil)

type BaseNode struct {
	event.Target

	name           string
	kind           dom.Kind
	self           dom.Node
	outer          dom.Node // identity wrapper set by setOuterRecursive for UA subtrees (ADR-0036)
	parent         dom.Node
	next           dom.Node
	prev           dom.Node
	firstChild     dom.Node
	lastChild      dom.Node
	ownerDocument  dom.Document
	renderObject   render.Object
	connected      bool
	needsSync      bool
	childNeedsSync bool
	inUASubtree    bool // true when this node is part of a UA shadow subtree (ADR-009)
}

// asBase returns the underlying *BaseNode for any dom.Node produced by this package.
func asBase(n dom.Node) *BaseNode {
	if n == nil {
		return nil
	}

	// If the node provides the BaseNode directly, we are done.
	if b, ok := n.(interface{ asBase() *BaseNode }); ok {
		return b.asBase()
	}

	// Otherwise, it's a wrapper, so unwrap it and try again.
	if unwrapped := n.Unwrap(); unwrapped != nil {
		return asBase(unwrapped)
	}
	return nil
}

func (b *BaseNode) adopt(newDoc dom.Document) {
	if b.ownerDocument == newDoc {
		return
	}
	b.ownerDocument = newDoc

	// Standard children.
	for n := b.firstChild; n != nil; n = n.NextSibling() {
		asBase(n).adopt(newDoc)
	}

	// ADR-009: If this is an element with a UA shadow subtree, adopt it too.
	if b.kind == dom.KindElement {
		// Use a type assertion to check for element-specific uaRoot.
		// We use the same pattern as asBase() to pierce wrappers.
		if el, ok := b.self.(interface{ UARoot() dom.Node }); ok {
			if uaRoot := el.UARoot(); uaRoot != nil {
				asBase(uaRoot).adopt(newDoc)
			}
		}
	}
}

func (b *BaseNode) Kind() dom.Kind { return b.kind }

func (b *BaseNode) NodeName() string { return b.name }

func (b *BaseNode) Parent() dom.Node { return b.parent }

func (b *BaseNode) ParentElement() dom.Element {
	if el, ok := b.parent.(dom.Element); ok {
		return el
	}
	return nil
}

func (b *BaseNode) NextSibling() dom.Node            { return b.next }
func (b *BaseNode) PreviousSibling() dom.Node        { return b.prev }
func (b *BaseNode) OwnerDocument() dom.Document      { return b.ownerDocument }
func (b *BaseNode) IsConnected() bool                { return b.connected }
func (b *BaseNode) RenderObject() render.Object      { return b.renderObject }
func (b *BaseNode) SetRenderObject(ro render.Object) { b.renderObject = ro }

func (b *BaseNode) NeedsSync() bool      { return b.needsSync }
func (b *BaseNode) ChildNeedsSync() bool { return b.childNeedsSync }

func (b *BaseNode) MarkNeedsSync() {
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

	// ADR-009: If this node is in a UA subtree, propagate ChildNeedsSync
	// to the host element via the outer pointer.
	if b.inUASubtree && b.outer != nil {
		asBase(b.outer).MarkNeedsSync()
	}
}

func (b *BaseNode) ClearSyncFlags() {
	b.needsSync = false
	b.childNeedsSync = false
}

func (b *BaseNode) EventTarget() event.EventTarget {
	if b.outer != nil {
		return b.outer
	}
	return b.self
}

func (b *BaseNode) Unwrap() dom.Node { return nil }

func (b *BaseNode) FirstLayoutChild() dom.Node {
	if b.firstChild != nil {
		return b.firstChild
	}
	if el, ok := b.self.(dom.Element); ok {
		if ua := UARoot(el); ua != nil {
			return ua.FirstChild()
		}
	}
	return nil
}

func (b *BaseNode) NextLayoutSibling(child dom.Node) dom.Node {
	if next := child.NextSibling(); next != nil {
		return next
	}
	// If child is the last public child, jump to first UA child.
	if child.Parent() == b.self {
		if el, ok := b.self.(dom.Element); ok {
			if ua := UARoot(el); ua != nil {
				return ua.FirstChild()
			}
		}
	}
	return nil
}

func (b *BaseNode) FirstChild() dom.Node { return b.firstChild }
func (b *BaseNode) LastChild() dom.Node  { return b.lastChild }

func (b *BaseNode) HasChildNodes() bool { return b.firstChild != nil }

func (b *BaseNode) ChildNodes() iter.Seq[dom.Node] {
	return func(yield func(dom.Node) bool) {
		for n := b.firstChild; n != nil; {
			next := n.NextSibling()
			if !yield(n) {
				return
			}
			n = next
		}
	}
}

func (b *BaseNode) AppendChild(child dom.Node) dom.Node {
	return b.self.InsertBefore(child, nil)
}

func (b *BaseNode) InsertBefore(newChild, ref dom.Node) dom.Node {
	if b.kind == dom.KindText {
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

	if b.inUASubtree {
		setOuterRecursive(newChild, b.outer)
	}

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
			attachWalk(parent dom.Node, child dom.Node)
		}); ok {
			w.attachWalk(b.self, newChild)
		}
	}

	b.notifyStructureChange()
	return newChild
}

func (b *BaseNode) RemoveChild(child dom.Node) dom.Node {
	if child.Parent() != b.self {
		panic("dom: node to be removed is not a child of this node")
	}

	cBase := asBase(child)

	// Trigger detach walk if connected.
	if b.connected {
		if w, ok := b.ownerDocument.(interface {
			detachWalk(parent dom.Node, child dom.Node)
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

func (b *BaseNode) ReplaceChild(newChild, oldChild dom.Node) dom.Node {
	ref := oldChild.NextSibling()
	b.RemoveChild(oldChild)
	b.InsertBefore(newChild, ref)
	return oldChild
}

func (b *BaseNode) Contains(descendant dom.Node) bool {
	for n := descendant; n != nil; n = n.Parent() {
		if n == b.self {
			return true
		}
	}
	return false
}

func (b *BaseNode) TextContent() string {
	if b.kind == dom.KindText {
		if tn, ok := b.self.(dom.TextNode); ok {
			return tn.Data()
		}
	}
	var sb strings.Builder
	for child := range b.ChildNodes() {
		sb.WriteString(child.TextContent())
	}
	return sb.String()
}

func (b *BaseNode) CloneNode(deep bool) dom.Node {
	var clone dom.Node
	switch b.kind {
	case dom.KindDocument:
		// Cloning a document is usually not supported or returns a special copy.
		// Kite v2 interfaces suggest Document is special.
		panic("dom: cloning document is not supported")
	case dom.KindElement:
		el := b.self.(dom.Element)
		clone = b.ownerDocument.CreateElement(el.TagName(), nil)
		if el.ID() != "" {
			clone.(dom.Element).SetID(el.ID())
		}
	case dom.KindText:
		tn := b.self.(dom.TextNode)
		clone = b.ownerDocument.CreateTextNode(tn.Data(), nil)
	}

	if deep {
		for child := range b.ChildNodes() {
			clone.AppendChild(child.CloneNode(true))
		}
	}
	return clone
}

func (b *BaseNode) notifyStructureChange() {
	b.self.MarkNeedsSync()
}
