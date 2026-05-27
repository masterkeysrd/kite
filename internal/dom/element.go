package dom

import (
	"iter"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

// Element is the concrete, exported implementation of Element.
type Element struct {
	BaseNode
	tagName string
	id      string
	class   string
	uaRoot  dom.Node // closed UA shadow subtree root; nil by default (ADR-009)
	scroll  *ScrollState
}

type ScrollState struct {
	X, Y int
}

// Compile-time assertion.
var _ dom.Element = (*Element)(nil)

// NewElement allocates an element with the given tag name and owner document.
// If self is nil, the element's self pointer defaults to itself.
func NewElement(doc dom.Document, tag string, self dom.Node) *Element {
	return newElement(tag, doc, self)
}

// newElement allocates an element with the given tag name and owner document.
func newElement(tag string, doc dom.Document, self dom.Node) *Element {
	e := &Element{tagName: tag}
	e.ownerDocument = doc
	e.self = e // self is always the raw *Element for DOM dispatch
	if self != nil {
		e.outer = self // outer is the user-visible wrapper (for identity resolution)
	} else {
		e.outer = e // outer defaults to self when no wrapper
	}
	e.kind = dom.KindElement
	e.name = tag
	return e
}

// asBase returns the underlying *BaseNode.
func (e *Element) asBase() *BaseNode { return &e.BaseNode }

// TagName returns the tag name.
func (e *Element) TagName() string { return e.tagName }

// ID returns the element's identifier.
func (e *Element) ID() string { return e.id }

// Class returns the element's classification.
func (e *Element) Class() string { return e.class }

// SetClass sets the element's classification.
func (e *Element) SetClass(class string) { e.class = class }

// SetID sets the element's identifier. Registry maintenance
// when the node is disconnected only the stored id field is
// updated; the identity registry is touched only when the node is connected.
func (e *Element) SetID(id string) {
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
		if el, ok := e.self.(dom.Element); ok {
			r.registerID(id, el)
		}
	}
}

// ReplaceWith replaces this element with the given nodes.
func (e *Element) ReplaceWith(nodes ...dom.Node) dom.Element {
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

// ChildNodes (from dom.Node) is implemented by BaseNode.

// Children is a helper that returns only Element children.
// Deprecated: use ChildNodes() and filter by kind.
func (e *Element) Children() iter.Seq[dom.Node] {
	return e.ChildNodes()
}

// IntrinsicStyle returns the UA-mandated sparse style for this element.
// The base element implementation returns an empty style.Style{}, meaning no
// UA properties are forced. Replaced and compound elements override this
// method to enforce UA-mandatory properties (ADR-010).
func (e *Element) IntrinsicStyle() style.Style { return style.Style{} }

// AttachUARoot attaches root as the closed UA shadow subtree for this host.
// See dom.Element.AttachUARoot for the full contract.
func (e *Element) AttachUARoot(root dom.Node) {
	if e.uaRoot != nil {
		panic("dom: AttachUARoot called more than once on the same element")
	}
	if root == nil {
		return
	}
	e.uaRoot = root
	// Propagate the host's outer pointer (user-visible wrapper) onto every
	// node in the UA subtree. This ensures event.Target() and identity
	// queries collapse to the host (ADR-0036).
	host := e.outer
	setOuterRecursive(root, host)
	// Mark the host so the engine syncs the new subtree on the next frame.
	e.self.MarkNeedsSync()
}

// UARoot returns the closed UA shadow subtree root, or nil.
func (e *Element) UARoot() dom.Node { return e.uaRoot }

func (e *Element) Scroll() (x, y int) {
	if e.scroll == nil {
		return 0, 0
	}
	return e.scroll.X, e.scroll.Y
}

func (e *Element) ScrollTo(x, y int) {
	if e.scroll == nil {
		e.scroll = &ScrollState{}
	}
	dx := x - e.scroll.X
	dy := y - e.scroll.Y
	if dx == 0 && dy == 0 {
		return
	}
	e.scroll.X = x
	e.scroll.Y = y

	if ro := e.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyScroll)
	}

	// Dispatch event.EventScroll.
	ev := event.NewScrollEvent(x, y, dx, dy)

	// Build the ancestor path for dispatch (root -> target).
	var path []event.EventTarget
	for p := e.self; p != nil; p = p.Parent() {
		path = append(path, p)
	}
	// Reverse the path.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	dispatcher := event.NewDispatcher()
	dispatcher.Dispatch(ev, path)
}

func (e *Element) ScrollBy(dx, dy int) {
	x, y := e.Scroll()

	// If the element has a render object, we must base the relative scroll
	// on the current clamped visual position to avoid accumulation at boundaries.
	if ro := e.RenderObject(); ro != nil {
		maxSX, maxSY := ro.MaxScroll()
		x = max(0, min(x, maxSX))
		y = max(0, min(y, maxSY))
	}

	e.ScrollTo(x+dx, y+dy)
}

func (e *Element) ScrollCursorIntoView() {
	// Base implementation is a no-op. Elements with a cursor (input, textarea)
	// override this to ensure the caret remains visible after layout.
}

func (e *Element) ProvidesCursor() bool {
	// Base implementation returns false.
	return false
}

func (e *Element) GetBoundingClientRect() (geom.Rect, bool) {
	if !e.connected {
		return geom.Rect{}, false
	}

	// Traverse up to find the root node (usually the dom.Document).
	var root = e.self
	for p := root.Parent(); p != nil; {
		root = p
		p = root.Parent()
	}

	// Grab the root fragment from its render object.
	ro := root.RenderObject()
	if ro == nil {
		return geom.Rect{}, false
	}
	rootFragment := ro.Fragment()
	if rootFragment == nil {
		return geom.Rect{}, false
	}

	// Target the render object of this element.
	targetRO := e.RenderObject()
	if targetRO == nil {
		return geom.Rect{}, false
	}

	rect, _, found := layout.ScrolledAbsoluteBounds(rootFragment, targetRO)
	return rect, found
}

// setOuterRecursive walks the subtree rooted at n and sets the self/outer
// back-pointer of every node to outer. This implements the ADR-0036 identity
// propagation required for UA shadow subtrees (ADR-009).
func setOuterRecursive(n dom.Node, outer dom.Node) {
	if n == nil {
		return
	}
	if b := asBase(n); b != nil {
		b.outer = outer
		b.inUASubtree = true
	}
	for child := range n.ChildNodes() {
		setOuterRecursive(child, outer)
	}
}
