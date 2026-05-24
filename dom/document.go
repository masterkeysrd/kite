package dom

import (
	"iter"
	"slices"
	"unicode/utf8"

	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// idRegistrar is an unexported interface implemented by *document.
// element uses it (via a type assertion on ownerDocument) to maintain
// the O(1) ID map without creating an import cycle or exposing internals.
type idRegistrar interface {
	registerID(id string, el Element)
	unregisterID(id string)
}

type overlayRecord struct {
	el     Element
	zIndex int
	order  int
}

// document is the concrete, unexported implementation of Document.
type document struct {
	baseNode
	byID    map[string]Element
	anchors map[string]Element

	overlays  []overlayRecord
	nextOrder int

	focusManager any

	selection *selectionImpl

	selectionDragging bool
	anchorNode        Node
	anchorOffset      int

	// mutating is the reentrancy guard. It is true while an attach or detach
	// walk is executing. Ancestor-mutation inside a lifecycle callback is
	// detected by checking whether the node being mutated is outside the
	// subtree currently being walked.
	mutating bool
}

// Compile-time assertion.
var _ Document = (*document)(nil)

// NewDocument creates and returns a new, empty Document.
// The document itself is always connected.
func NewDocument() Document {
	d := &document{
		byID:    make(map[string]Element),
		anchors: make(map[string]Element),
	}
	d.self = d
	d.ownerDocument = d // the document owns itself
	d.connected = true  // the document is always connected
	d.kind = KindDocument
	d.name = "#document"
	d.needsSync = true
	d.selection = newSelection(d)

	d.AddEventListener(event.EventMouseDown, d.handleMouseDown)
	d.AddEventListener(event.EventMouseMove, d.handleMouseMove)
	d.AddEventListener(event.EventMouseUp, d.handleMouseUp)
	d.AddEventListener(event.EventCopy, d.handleCopy)
	d.AddEventListener(event.EventCut, d.handleCopy) // Cut also copies to clipboard
	d.AddEventListener(event.EventPaste, d.handlePaste)

	return d
}

// CreateElement returns a new, detached Element with the given tag name owned
// by this document.
func (d *document) CreateElement(tag string, self Node) Element {
	return newElement(tag, d, self)
}

// CreateTextNode returns a new, detached TextNode with the given data owned
// by this document.
func (d *document) CreateTextNode(data string, self Node) TextNode {
	return newTextNode(data, d, self)
}

// GetElementByID returns the Element whose ID equals id, or nil.
func (d *document) GetElementByID(id string) Element {
	return d.byID[id]
}

// FindAnchor returns the Element registered under name in the anchor registry,
// or nil.
func (d *document) FindAnchor(name string) Element {
	return d.anchors[name]
}

// RegisterAnchor adds el to the anchor registry under name.
func (d *document) RegisterAnchor(name string, el Element) {
	if name != "" {
		d.anchors[name] = el
	}
}

// UnregisterAnchor removes the entry for name from the anchor registry.
func (d *document) UnregisterAnchor(name string) {
	delete(d.anchors, name)
}

func (d *document) Body() Element {
	for child := range d.ChildNodes() {
		if el, ok := child.(Element); ok {
			return el
		}
	}
	return nil
}

func (d *document) ShowOverlay(el Element, zIndex int) {
	// If el is already an overlay, update its zIndex.
	for i, o := range d.overlays {
		if o.el == el {
			if o.zIndex == zIndex {
				return
			}
			d.overlays[i].zIndex = zIndex
			d.sortOverlays()
			d.MarkNeedsSync()
			return
		}
	}

	// Add new overlay.
	d.overlays = append(d.overlays, overlayRecord{
		el:     el,
		zIndex: zIndex,
		order:  d.nextOrder,
	})
	d.nextOrder++
	d.sortOverlays()
	d.MarkNeedsSync()
}

func (d *document) HideOverlay(el Element) {
	for i, o := range d.overlays {
		if o.el == el {
			d.overlays = append(d.overlays[:i], d.overlays[i+1:]...)
			d.MarkNeedsSync()
			return
		}
	}
}

func (d *document) Overlays() iter.Seq[Element] {
	return func(yield func(Element) bool) {
		for _, o := range d.overlays {
			if !yield(o.el) {
				return
			}
		}
	}
}

func (d *document) FocusManager() any {
	return d.focusManager
}

func (d *document) SetFocusManager(fm any) {
	d.focusManager = fm
}

func (d *document) Selection() Selection {
	return d.selection
}

func (d *document) CreateRange() Range {
	return &rangeImpl{doc: d}
}

func (d *document) sortOverlays() {
	slices.SortFunc(d.overlays, func(a, b overlayRecord) int {
		if a.zIndex != b.zIndex {
			return a.zIndex - b.zIndex
		}
		return a.order - b.order
	})
}

// --- Mouse Handlers ---------------------------------------------------------

func (d *document) handleMouseDown(ev event.Event) {
	mev, ok := ev.(*event.MouseEvent)
	if !ok || mev.Button != event.ButtonLeft {
		return
	}

	rootRO := d.RenderObject()
	if rootRO == nil {
		return
	}
	rootFrag := rootRO.Fragment()
	if rootFrag == nil {
		return
	}

	byteOffset := cursor.ByteOffsetAtPoint(rootFrag, mev.Screen.X, mev.Screen.Y)
	node, runeOffset := d.findNodeAtByteOffset(d, byteOffset)
	if node == nil {
		return
	}

	d.anchorNode = node
	d.anchorOffset = runeOffset
	d.selectionDragging = true

	// Clear existing selection and create a new collapsed range.
	d.selection.RemoveAllRanges()
	rng := d.CreateRange()
	rng.SetStart(node, runeOffset)
	rng.Collapse(true)
	d.selection.AddRange(rng)
}

func (d *document) handleMouseMove(ev event.Event) {
	if !d.selectionDragging {
		return
	}

	mev, ok := ev.(*event.MouseEvent)
	if !ok {
		return
	}

	rootRO := d.RenderObject()
	if rootRO == nil {
		return
	}
	rootFrag := rootRO.Fragment()
	if rootFrag == nil {
		return
	}

	byteOffset := cursor.ByteOffsetAtPoint(rootFrag, mev.Screen.X, mev.Screen.Y)
	currNode, currRuneOffset := d.findNodeAtByteOffset(d, byteOffset)
	if currNode == nil {
		return
	}

	// Update the selection range.
	if d.selection.RangeCount() > 0 {
		rng := d.selection.GetRangeAt(0)
		cmp := d.comparePositions(d.anchorNode, d.anchorOffset, currNode, currRuneOffset)
		if cmp <= 0 {
			rng.SetStart(d.anchorNode, d.anchorOffset)
			rng.SetEnd(currNode, currRuneOffset)
		} else {
			rng.SetStart(currNode, currRuneOffset)
			rng.SetEnd(d.anchorNode, d.anchorOffset)
		}
	}
}

func (d *document) handleMouseUp(ev event.Event) {
	if !d.selectionDragging {
		return
	}
	d.selectionDragging = false

	if d.selection.RangeCount() > 0 {
		rng := d.selection.GetRangeAt(0)
		if rng.IsCollapsed() {
			d.selection.RemoveAllRanges()
		}
	}
}

func (d *document) findBlockAncestor(n Node) Element {
	for curr := n; curr != nil; curr = curr.Parent() {
		if el, ok := curr.(Element); ok {
			ro := el.RenderObject()
			if ro == nil {
				continue
			}
			display := ro.ComputedStyle().Display
			if display == style.DisplayBlock || display == style.DisplayFlex ||
				display == style.DisplayListItem || display == style.DisplayTableCell {
				return el
			}
		}
	}
	return nil
}

func (d *document) findNodeAtByteOffset(root Node, targetOffset int) (Node, int) {
	currOffset := 0
	var walk func(Node) (Node, int, bool)
	walk = func(n Node) (Node, int, bool) {
		if t, ok := n.(TextNode); ok {
			data := t.Data()
			byteLen := len(data)
			if currOffset+byteLen >= targetOffset {
				remaining := targetOffset - currOffset
				runeOffset := 0
				byteCount := 0
				for _, r := range data {
					if byteCount >= remaining {
						break
					}
					byteCount += utf8.RuneLen(r)
					runeOffset++
				}
				return t, runeOffset, true
			}
			currOffset += byteLen
		}

		for child := range LayoutChildren(n) {
			if node, offset, found := walk(child); found {
				return node, offset, true
			}
		}
		return nil, 0, false
	}
	node, offset, found := walk(root)
	if found {
		return node, offset
	}
	// Fallback to the end of the last TextNode in the subtree
	var lastText TextNode
	var walkLast func(Node)
	walkLast = func(n Node) {
		if t, ok := n.(TextNode); ok {
			lastText = t
		}
		for child := range LayoutChildren(n) {
			walkLast(child)
		}
	}
	walkLast(root)
	if lastText != nil {
		return lastText, utf8.RuneCountInString(lastText.Data())
	}
	return nil, 0
}

func (d *document) comparePositions(nodeA Node, offsetA int, nodeB Node, offsetB int) int {
	if nodeA == nodeB {
		return offsetA - offsetB
	}

	var first Node
	var walk func(Node) bool
	walk = func(n Node) bool {
		if n == nodeA {
			first = nodeA
			return false
		}
		if n == nodeB {
			first = nodeB
			return false
		}
		for child := range LayoutChildren(n) {
			if !walk(child) {
				return false
			}
		}
		if n == d {
			for el := range d.Overlays() {
				if !walk(el) {
					return false
				}
			}
		}
		return true
	}
	walk(d)

	if first == nodeA {
		return -1
	}
	return 1
}

// --- attach / detach walks --------------------------------------------------

func (d *document) attachWalk(parent Node, child Node) {
	parentConnected := parent.IsConnected()
	d.walkAttach(child, parentConnected)
}

func (d *document) walkAttach(n Node, parentConnected bool) {
	b := asBase(n)
	if b == nil {
		return
	}

	// Steps 2 & 3: connection + registry + lifecycle.
	if parentConnected {
		b.connected = true
		d.registerNode(n)
		if lc, ok := n.(Lifecycle); ok {
			lc.OnConnected()
		}
	}

	// Recurse into children (pre-order: parent handled before children).
	for child := range n.ChildNodes() {
		d.walkAttach(child, parentConnected)
	}
}

func (d *document) detachWalk(parent Node, child Node) {
	parentConnected := parent.IsConnected()
	d.walkDetach(child, parentConnected)
}

func (d *document) walkDetach(n Node, parentConnected bool) {
	b := asBase(n)
	if b == nil {
		return
	}

	// Recurse into children first (post-order: children before parent).
	for child := range n.ChildNodes() {
		d.walkDetach(child, parentConnected)
	}

	if parentConnected {
		// Step 1: lifecycle callback fires while IsConnected() is still true.
		if lc, ok := n.(Lifecycle); ok {
			lc.OnDisconnected()
		}
		// Step 2: unregister then flip connected flag.
		d.unregisterNode(n)
		b.connected = false
	}
}

// --- registry helpers -------------------------------------------------------

func (d *document) registerNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			// Identity registry uses the current self pointer.
			d.byID[id] = el
		}
	}
}

func (d *document) unregisterNode(n Node) {
	if el, ok := n.(Element); ok {
		if id := el.ID(); id != "" {
			delete(d.byID, id)
		}
	}
}

func (d *document) registerID(id string, el Element) {
	if id != "" {
		d.byID[id] = el
	}
}

func (d *document) unregisterID(id string) {
	delete(d.byID, id)
}

func (d *document) asBase() *baseNode { return &d.baseNode }

// --- Clipboard Handlers -----------------------------------------------------

func (d *document) handleCopy(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	// If the event doesn't already have text (e.g. from a focused element's
	// local selection), use the global document selection.
	if _, hasText := ce.Items[event.MimeTextPlain]; !hasText {
		if text := d.selection.String(); text != "" {
			ce.Items[event.MimeTextPlain] = []byte(text)
		}
	}

	// Synchronize to the system clipboard if a bridge is available.
	if ce.Clipboard != nil {
		if text := ce.Text(); text != "" {
			ce.Clipboard.SetClipboard(text)
		}
	}
}

func (d *document) handlePaste(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	// If the items map is empty (e.g. a raw Ctrl+V that hasn't been handled
	// by a terminal extension), populate text/plain from the system clipboard
	// as a fallback.
	if len(ce.Items) == 0 && ce.Clipboard != nil {
		text := ce.Clipboard.GetClipboard()
		if text != "" {
			ce.Items[event.MimeTextPlain] = []byte(text)
		}
	}
}
