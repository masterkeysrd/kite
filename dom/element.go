package dom

import "iter"

// adopter is an unexported interface that allows the attach walk to set the
// outer back-pointer on a concrete element without exposing a public method.
// All *element values satisfy adopter.
type adopter interface {
	adopt(outer Element)
	outerSelf() Element
}

// coreElement is an unexported interface implemented by any type that wraps
// or embeds a *element (e.g. *document or widget types in the dom package).
// The attach/detach walks use it to reach the underlying *element for raw
// link operations when the parent is not itself a plain *element or *document.
type coreElement interface {
	coreEl() *element
}

// element is the concrete, unexported implementation of Element.
type element struct {
	baseNode
	// outer is the outermost wrapper that holds this element in the tree.
	// For a bare *element it equals itself; for a widget type that embeds
	// *element the DOM sets outer to the widget pointer during the attach
	// walk so that event.Target(), GetElementByID(), and RenderObject.Node()
	// all return the user-visible outermost value (see ADR-0036 §2).
	//
	// outer is intentionally never reset to nil on detach: the element value
	// did not change, only its connection state did. Keeping it stable avoids
	// a window where closure-captured pointers would see a stale outer.
	outer      Element
	tagName    string
	id         string
	firstChild Node
	lastChild  Node
}

// Compile-time assertion.
var _ Element = (*element)(nil)

// newElement allocates an element with the given tag name and owner document.
// The element's outer pointer defaults to itself.
func newElement(tag string, doc Document) *element {
	e := &element{tagName: tag}
	e.ownerDocument = doc
	e.outer = e
	return e
}

// adopt sets the outer back-pointer. Called by the attach walk; idempotent
// when outer is already set to the same value.
func (e *element) adopt(outer Element) { e.outer = outer }

// outerSelf returns the current outer back-pointer.
func (e *element) outerSelf() Element { return e.outer }

// coreEl returns the underlying *element. Satisfies the coreElement interface.
func (e *element) coreEl() *element { return e }

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
		r.registerID(id, e.outer)
	}
}

// FirstChild returns the first child Node, or nil.
func (e *element) FirstChild() Node { return e.firstChild }

// LastChild returns the last child Node, or nil.
func (e *element) LastChild() Node { return e.lastChild }

// Children returns an iterator over direct children in document order.
func (e *element) Children() iter.Seq[Node] {
	return func(yield func(Node) bool) {
		for n := e.firstChild; n != nil; n = n.NextSibling() {
			if !yield(n) {
				return
			}
		}
	}
}

// AppendChild adds child as the last child and returns child.
// It runs the detach walk on child if child already has a parent, then links
// child and runs the attach walk. Registry and lifecycle callbacks follow the
// walk order specified in ADR-0036 §4.
func (e *element) AppendChild(child Node) Node {
	return e.doAppendChild(child, e.outer)
}

// doAppendChild is the internal implementation used by both *element and
// *document so that the self-reference passed to the walk is always the
// outermost wrapper.
func (e *element) doAppendChild(child Node, selfEl Element) Node {
	if w, ok := e.ownerDocument.(walker); ok {
		w.runAppendChild(selfEl, child)
		return child
	}
	// Fallback: no walk driver available — link only (should not happen in
	// normal usage, but keeps the package functional in isolated tests that
	// construct elements without a full document).
	e.rawLink(child, selfEl)
	e.notifyStructureChange()
	return child
}

// InsertBefore inserts newChild immediately before ref and returns newChild.
// If ref is nil the call is equivalent to AppendChild.
func (e *element) InsertBefore(newChild, ref Node) Node {
	return e.doInsertBefore(newChild, ref, e.outer)
}

func (e *element) doInsertBefore(newChild, ref Node, selfEl Element) Node {
	if ref == nil {
		return e.doAppendChild(newChild, selfEl)
	}
	if w, ok := e.ownerDocument.(walker); ok {
		w.runInsertBefore(selfEl, newChild, ref)
		return newChild
	}
	e.rawLinkBefore(newChild, ref, selfEl)
	e.notifyStructureChange()
	return newChild
}

// RemoveChild removes child from this element and returns child.
func (e *element) RemoveChild(child Node) Node {
	return e.doRemoveChild(child, e.outer)
}

func (e *element) doRemoveChild(child Node, selfEl Element) Node {
	if w, ok := e.ownerDocument.(walker); ok {
		w.runRemoveChild(selfEl, child)
		return child
	}
	e.rawUnlink(child)
	e.notifyStructureChange()
	return child
}

// ReplaceChild inserts newChild in place of oldChild, removes oldChild, and
// returns oldChild.
func (e *element) ReplaceChild(newChild, oldChild Node) Node {
	return e.doReplaceChild(newChild, oldChild, e.outer)
}

func (e *element) doReplaceChild(newChild, oldChild Node, selfEl Element) Node {
	ref := oldChild.NextSibling()
	e.doRemoveChild(oldChild, selfEl)
	e.doInsertBefore(newChild, ref, selfEl)
	return oldChild
}

// --- raw link helpers (no walk, no lifecycle) --------------------------------

// rawLink appends child to this element without running any walk.
func (e *element) rawLink(child Node, selfEl Element) {
	l := asLinkable(child)
	l.setParent(selfEl)
	l.setOwnerDocument(e.ownerDocument)
	l.setNext(nil)
	l.setPrev(e.lastChild)

	if e.lastChild != nil {
		asLinkable(e.lastChild).setNext(child)
	} else {
		e.firstChild = child
	}
	e.lastChild = child
}

// rawLinkBefore inserts newChild immediately before ref without running any
// walk.
func (e *element) rawLinkBefore(newChild, ref Node, selfEl Element) {
	l := asLinkable(newChild)
	l.setParent(selfEl)
	l.setOwnerDocument(e.ownerDocument)

	prev := ref.PreviousSibling()
	l.setPrev(prev)
	l.setNext(ref)

	if prev != nil {
		asLinkable(prev).setNext(newChild)
	} else {
		e.firstChild = newChild
	}
	asLinkable(ref).setPrev(newChild)
}

// rawUnlink removes child from this element without running any walk.
func (e *element) rawUnlink(child Node) {
	prev := child.PreviousSibling()
	next := child.NextSibling()

	if prev != nil {
		asLinkable(prev).setNext(next)
	} else {
		e.firstChild = next
	}
	if next != nil {
		asLinkable(next).setPrev(prev)
	} else {
		e.lastChild = prev
	}

	l := asLinkable(child)
	l.setParent(nil)
	l.setPrev(nil)
	l.setNext(nil)
}

// notifyStructureChange calls MarkChildrenDirty on this element's render
// object when one is present.
func (e *element) notifyStructureChange() {
	if ro := e.renderObject; ro != nil {
		ro.MarkChildrenDirty()
	}
}
