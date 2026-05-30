package dom

import (
	"iter"
	"slices"
	"unicode/utf8"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/terminal"
)

// idRegistrar is an unexported interface implemented by *Document.
// Element uses it (via a type assertion on ownerDocument) to maintain
// the O(1) ID map without creating an import cycle or exposing internals.
type idRegistrar interface {
	registerID(id string, el dom.Element)
	unregisterID(id string)
}

type OverlayRecord struct {
	el     dom.Element
	zIndex int
	order  int
}

// Document is the concrete, exported implementation of Document.
type Document struct {
	BaseNode
	byID    map[string]dom.Element
	anchors map[string]dom.Element

	overlays  []OverlayRecord
	nextOrder int

	focusHandle dom.FocusHandle
	clipboard   terminal.Clipboard
	terminal    terminal.Terminal
	defaultView dom.View

	selection *Selection

	selectionDragging bool
	anchorNode        dom.Node
	anchorOffset      int

	// mutating is the reentrancy guard. It is true while an attach or detach
	// walk is executing. Ancestor-mutation inside a lifecycle callback is
	// detected by checking whether the node being mutated is outside the
	// subtree currently being walked.
	mutating bool
}

// Compile-time assertion.
var _ dom.Document = (*Document)(nil)

// NewDocument creates and returns a new, empty Document.
// The document itself is always connected.
func NewDocument() *Document {
	d := &Document{
		byID:    make(map[string]dom.Element),
		anchors: make(map[string]dom.Element),
	}
	d.self = d
	d.ownerDocument = d // the document owns itself
	d.connected = true  // the document is always connected
	d.kind = dom.KindDocument
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
func (d *Document) CreateElement(tag string, self dom.Node) dom.Element {
	return newElement(tag, d, self)
}

// CreateTextNode returns a new, detached TextNode with the given data owned
// by this document.
func (d *Document) CreateTextNode(data string, self dom.Node) dom.TextNode {
	return newTextNode(data, d, self)
}

// GetElementByID returns the Element whose ID equals id, or nil.
func (d *Document) GetElementByID(id string) dom.Element {
	return d.byID[id]
}

// FindAnchor returns the Element registered under name in the anchor registry,
// or nil.
func (d *Document) FindAnchor(name string) dom.Element {
	return d.anchors[name]
}

// RegisterAnchor adds el to the anchor registry under name.
func (d *Document) RegisterAnchor(name string, el dom.Element) {
	if name != "" {
		d.anchors[name] = el
	}
}

// UnregisterAnchor removes the entry for name from the anchor registry.
func (d *Document) UnregisterAnchor(name string) {
	delete(d.anchors, name)
}

func (d *Document) Body() dom.Element {
	for child := range d.ChildNodes() {
		if el, ok := child.(dom.Element); ok {
			return el
		}
	}
	return nil
}

func (d *Document) ShowOverlay(el dom.Element, zIndex int) {
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
	d.overlays = append(d.overlays, OverlayRecord{
		el:     el,
		zIndex: zIndex,
		order:  d.nextOrder,
	})
	d.nextOrder++
	d.sortOverlays()
	d.MarkNeedsSync()
}

func (d *Document) HideOverlay(el dom.Element) {
	for i, o := range d.overlays {
		if o.el == el {
			d.overlays = append(d.overlays[:i], d.overlays[i+1:]...)
			d.MarkNeedsSync()
			return
		}
	}
}

func (d *Document) Overlays() iter.Seq[dom.Element] {
	return func(yield func(dom.Element) bool) {
		for _, o := range d.overlays {
			if !yield(o.el) {
				return
			}
		}
	}
}

func (d *Document) Focus(el dom.Element) {
	if d.focusHandle != nil {
		d.focusHandle.Focus(el)
	}
}

func (d *Document) IsFocused(el dom.Element) bool {
	if d.focusHandle != nil {
		return d.focusHandle.IsFocused(el)
	}
	return false
}

func (d *Document) PushScope(scope *dom.FocusScope) {
	if d.focusHandle != nil {
		d.focusHandle.PushScope(scope)
	}
}

func (d *Document) PopScope() {
	if d.focusHandle != nil {
		d.focusHandle.PopScope()
	}
}

func (d *Document) ActiveScope() *dom.FocusScope {
	if d.focusHandle != nil {
		return d.focusHandle.ActiveScope()
	}
	return nil
}

func (d *Document) CurrentFocus() dom.Element {
	if d.focusHandle != nil {
		return d.focusHandle.Current()
	}
	return nil
}

func (d *Document) NextFocus() bool {
	if d.focusHandle != nil {
		return d.focusHandle.Next()
	}
	return false
}

func (d *Document) PreviousFocus() bool {
	if d.focusHandle != nil {
		return d.focusHandle.Previous()
	}
	return false
}

func (d *Document) SetFocusHandle(handle dom.FocusHandle) {
	d.focusHandle = handle
}

func (d *Document) Selection() dom.Selection {
	return d.selection
}

func (d *Document) Clipboard() terminal.Clipboard             { return d.clipboard }
func (d *Document) SetClipboardProvider(p terminal.Clipboard) { d.clipboard = p }

func (d *Document) Terminal() terminal.Terminal     { return d.terminal }
func (d *Document) SetTerminal(t terminal.Terminal) { d.terminal = t }

func (d *Document) DefaultView() dom.View     { return d.defaultView }
func (d *Document) SetDefaultView(v dom.View) { d.defaultView = v }

func (d *Document) CreateRange() dom.Range {
	return &Range{doc: d}
}

func (d *Document) sortOverlays() {
	slices.SortFunc(d.overlays, func(a, b OverlayRecord) int {
		if a.zIndex != b.zIndex {
			return a.zIndex - b.zIndex
		}
		return a.order - b.order
	})
}

// --- Mouse Handlers ---------------------------------------------------------

func (d *Document) handleMouseDown(ev event.Event) {
	mev, ok := ev.(*event.MouseEvent)
	if !ok || mev.Button != event.ButtonLeft {
		return
	}

	if d.defaultView == nil {
		return
	}

	node, runeOffset := d.defaultView.NodeAtPoint(mev.Screen.X, mev.Screen.Y)
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

func (d *Document) handleMouseMove(ev event.Event) {
	if !d.selectionDragging {
		return
	}

	mev, ok := ev.(*event.MouseEvent)
	if !ok {
		return
	}

	if d.defaultView == nil {
		return
	}

	currNode, currRuneOffset := d.defaultView.NodeAtPoint(mev.Screen.X, mev.Screen.Y)
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

func (d *Document) handleMouseUp(ev event.Event) {
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

func (d *Document) FindNodeAtByteOffset(root dom.Node, targetOffset int) (dom.Node, int) {
	currOffset := 0
	var walk func(dom.Node) (dom.Node, int, bool)
	walk = func(n dom.Node) (dom.Node, int, bool) {
		if t, ok := n.(dom.TextNode); ok {
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
	var lastText dom.TextNode
	var walkLast func(dom.Node)
	walkLast = func(n dom.Node) {
		if t, ok := n.(dom.TextNode); ok {
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

func (d *Document) comparePositions(nodeA dom.Node, offsetA int, nodeB dom.Node, offsetB int) int {
	if nodeA == nodeB {
		return offsetA - offsetB
	}

	var first dom.Node
	var walk func(dom.Node) bool
	walk = func(n dom.Node) bool {
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

func (d *Document) attachWalk(parent dom.Node, child dom.Node) {
	parentConnected := parent.IsConnected()
	d.walkAttach(child, parentConnected)
}

func (d *Document) walkAttach(n dom.Node, parentConnected bool) {
	b := asBase(n)
	if b == nil {
		return
	}

	// Steps 2 & 3: connection + registry + lifecycle.
	if parentConnected {
		b.connected = true
		d.registerNode(n)
		if lc, ok := n.(dom.Lifecycle); ok {
			lc.OnConnected()
		}
	}

	// Recurse into children (pre-order: parent handled before children).
	for child := range n.ChildNodes() {
		d.walkAttach(child, parentConnected)
	}
}

func (d *Document) detachWalk(parent dom.Node, child dom.Node) {
	parentConnected := parent.IsConnected()
	d.walkDetach(child, parentConnected)
}

func (d *Document) walkDetach(n dom.Node, parentConnected bool) {
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
		if lc, ok := n.(dom.Lifecycle); ok {
			lc.OnDisconnected()
		}
		// Step 2: unregister then flip connected flag.
		d.unregisterNode(n)
		b.connected = false
	}
}

// --- registry helpers -------------------------------------------------------

func (d *Document) registerNode(n dom.Node) {
	if el, ok := n.(dom.Element); ok {
		if id := el.ID(); id != "" {
			// Identity registry uses the current self pointer.
			d.byID[id] = el
		}
	}
}

func (d *Document) unregisterNode(n dom.Node) {
	if el, ok := n.(dom.Element); ok {
		if id := el.ID(); id != "" {
			delete(d.byID, id)
		}
	}
}

func (d *Document) registerID(id string, el dom.Element) {
	if id != "" {
		d.byID[id] = el
	}
}

func (d *Document) unregisterID(id string) {
	delete(d.byID, id)
}

func (d *Document) asBase() *BaseNode { return &d.BaseNode }

// --- Clipboard Handlers -----------------------------------------------------

func (d *Document) handleCopy(ev event.Event) {
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

	// Synchronize to the system clipboard if a provider is available.
	if d.clipboard != nil {
		if text := ce.Text(); text != "" {
			d.clipboard.WriteText(text)
		}
	}
}

func (d *Document) handlePaste(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	// If the items map is empty (e.g. a raw Ctrl+V that hasn't been handled
	// by a terminal extension), populate from system clipboard provider.
	if len(ce.Items) == 0 && d.clipboard != nil {
		d.clipboard.ReadText().Then(func(text string) {
			if text != "" {
				// Re-dispatch a new paste event with the retrieved text.
				// This ensures that listeners get the data they were expecting.
				newEv := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
				newEv.SetText(text)
				d.DispatchToTarget(newEv)
			}
		}, func(err error) {
			// Ignore errors for now.
		})
	}
}
