package dom

import "fmt"

// idRegistrar is an unexported interface implemented by *document.
// element uses it (via a type assertion on ownerDocument) to maintain
// the O(1) ID map without creating an import cycle or exposing internals.
type idRegistrar interface {
	registerID(id string, el Element)
	unregisterID(id string)
}

// anchorRegistrar is an unexported interface for the landmark registry.
// element uses it to register/unregister anchors when connected/disconnected.
type anchorRegistrar interface {
	registerAnchor(name string, el Element)
	unregisterAnchor(name string)
}

// walker is an unexported interface implemented by *document.
// element delegates all structural mutations to the document so the
// attach/detach walks, reentrancy guard, and cross-document check are
// all applied in one place (see ADR-0036).
type walker interface {
	runAppendChild(parent Element, child Node)
	runInsertBefore(parent Element, newChild, ref Node)
	runRemoveChild(parent Element, child Node)
}

// document is the concrete, unexported implementation of Document.
// It embeds element and overrides outer so that parent pointers installed on
// appended children point to the *document value, not the embedded *element.
type document struct {
	element
	byID    map[string]Element
	anchors map[string]Element
	// mutating is the reentrancy guard. It is true while an attach or detach
	// walk is executing. Ancestor-mutation inside a lifecycle callback is
	// detected by checking whether the node being mutated is outside the
	// subtree currently being walked (see ADR-0036 §8).
	mutating bool
}

// Compile-time assertion.
var _ Document = (*document)(nil)

// NewDocument creates and returns a new, empty Document.
// The document itself is always connected (ADR-0036 §3).
func NewDocument() Document {
	d := &document{
		byID:    make(map[string]Element),
		anchors: make(map[string]Element),
	}
	d.tagName = "#document"
	d.outer = d         // children's Parent() returns the document
	d.ownerDocument = d // the document owns itself
	d.connected = true  // the document is always connected
	return d
}

// CreateElement returns a new, detached Element with the given tag name owned
// by this document.
func (d *document) CreateElement(tag string) Element {
	return newElement(tag, d)
}

// CreateTextNode returns a new, detached TextNode with the given data owned
// by this document.
func (d *document) CreateTextNode(data string) TextNode {
	return newTextNode(data, d)
}

// GetElementByID returns the Element whose ID equals id, or nil.
// The registry is maintained by the attach/detach walks per ADR-0036 §10.
func (d *document) GetElementByID(id string) Element {
	return d.byID[id]
}

// FindAnchor returns the Element registered under name in the anchor registry,
// or nil. The anchor registry is separate from the ID registry.
func (d *document) FindAnchor(name string) Element {
	return d.anchors[name]
}

// RegisterAnchor adds el to the anchor registry under name.
// Called by Anchor elements when their Name property is set while
// the node is connected.
func (d *document) RegisterAnchor(name string, el Element) {
	if name != "" {
		d.anchors[name] = el
	}
}

// UnregisterAnchor removes the entry for name from the anchor registry.
func (d *document) UnregisterAnchor(name string) {
	delete(d.anchors, name)
}

// --- walker implementation --------------------------------------------------

// runAppendChild implements the authoritative AppendChild sequence from
// ADR-0036: validate → detach old parent → link → attach walk → dirty.
func (d *document) runAppendChild(parent Element, child Node) {
	d.validateCrossDocument(child)
	if child.Parent() != nil {
		d.runRemoveChild(child.Parent(), child)
	}
	parentElem := asElement(parent)
	parentElem.rawLink(child, parent)
	d.attachWalk(parent, child)
	parentElem.notifyStructureChange()
}

// runInsertBefore implements the authoritative InsertBefore sequence.
func (d *document) runInsertBefore(parent Element, newChild, ref Node) {
	if ref == nil {
		d.runAppendChild(parent, newChild)
		return
	}
	d.validateCrossDocument(newChild)
	if newChild.Parent() != nil {
		d.runRemoveChild(newChild.Parent(), newChild)
	}
	parentElem := asElement(parent)
	parentElem.rawLinkBefore(newChild, ref, parent)
	d.attachWalk(parent, newChild)
	parentElem.notifyStructureChange()
}

// runRemoveChild implements the authoritative RemoveChild sequence.
func (d *document) runRemoveChild(parent Element, child Node) {
	d.detachWalk(parent, child)
	asElement(parent).rawUnlink(child)
	asElement(parent).notifyStructureChange()
}

// --- attach / detach walks --------------------------------------------------

// attachWalk runs a pre-order walk over the subtree rooted at child,
// performing adoption, connection, and lifecycle dispatch (ADR-0036 §4).
func (d *document) attachWalk(parent Element, child Node) {
	parentConnected := parent.IsConnected()
	d.walkAttach(child, parentConnected)
}

func (d *document) walkAttach(n Node, parentConnected bool) {
	// Step 1: adoption — set outer back-pointer.
	if a, ok := n.(adopter); ok {
		a.adopt(n.(Element))
	}

	// Steps 2 & 3: connection + registry + lifecycle.
	if parentConnected {
		asLinkable(n).setConnected(true)
		d.registerNode(n)
		if lc, ok := n.(Lifecycle); ok {
			lc.OnConnected()
		}
	}

	// Recurse into children (pre-order: parent handled before children).
	for child := range n.Children() {
		d.walkAttach(child, parentConnected)
	}
}

// detachWalk runs a post-order walk over the subtree rooted at child,
// performing lifecycle dispatch, unregistration, and disconnection
// (ADR-0036 §5).
func (d *document) detachWalk(parent Element, child Node) {
	parentConnected := parent.IsConnected()
	d.walkDetach(child, parentConnected)
}

func (d *document) walkDetach(n Node, parentConnected bool) {
	// Recurse into children first (post-order: children before parent).
	for child := range n.Children() {
		d.walkDetach(child, parentConnected)
	}

	if parentConnected {
		// Step 1: lifecycle callback fires while IsConnected() is still true.
		if lc, ok := n.(Lifecycle); ok {
			lc.OnDisconnected()
		}
		// Step 2: unregister then flip connected flag.
		d.unregisterNode(n)
		asLinkable(n).setConnected(false)
	}
	// Step 3: outer stays set (ADR-0036 §5).
}

// --- registry helpers -------------------------------------------------------

// registerNode adds n to the appropriate registries if it carries an ID or
// is an anchor. Called during the attach walk only when parentConnected.
func (d *document) registerNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			outer := el
			if a, ok := n.(adopter); ok {
				outer = a.outerSelf()
			}
			d.byID[id] = outer
		}
	}
}

// unregisterNode removes n from the registries. Called during the detach walk
// only when parentConnected.
func (d *document) unregisterNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			delete(d.byID, id)
		}
	}
}

// --- cross-document validation ----------------------------------------------

// validateCrossDocument panics in dev builds if child belongs to a different
// document than this one (ADR-0036 §9).
func (d *document) validateCrossDocument(child Node) {
	if child.OwnerDocument() != Document(d) {
		panic(fmt.Sprintf(
			"dom: cross-document append: source document %p → dest document %p",
			child.OwnerDocument(), Document(d),
		))
	}
}

// --- idRegistrar implementation (unexported) --------------------------------

func (d *document) registerID(id string, el Element) {
	if id != "" {
		d.byID[id] = el
	}
}

func (d *document) unregisterID(id string) {
	delete(d.byID, id)
}

// --- anchorRegistrar implementation (unexported) ----------------------------

func (d *document) registerAnchor(name string, el Element) {
	if name != "" {
		d.anchors[name] = el
	}
}

func (d *document) unregisterAnchor(name string) {
	delete(d.anchors, name)
}

// --- document-level overrides of mutation methods ---------------------------
// The embedded *element's AppendChild / InsertBefore / RemoveChild /
// ReplaceChild methods call e.doXxx(child, e.outer). For *document, outer is
// already set to d (the *document value), so those calls route through the
// walker interface correctly. No override is needed.

// --- helpers ----------------------------------------------------------------

// asElement returns the underlying *element for any Element produced by this
// package. It handles *element, *document, and any wrapper type that embeds
// *element and satisfies the coreElement interface.
func asElement(el Element) *element {
	switch v := el.(type) {
	case *document:
		return &v.element
	case *element:
		return v
	case coreElement:
		return v.coreEl()
	case interface{ DOMElement() Element }:
		return asElement(v.DOMElement())
	default:
		panic(fmt.Sprintf("dom: asElement: unexpected type %T", el))
	}
}
