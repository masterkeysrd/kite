package element

// textControlBase is a generic base struct shared by InputElement and
// TextAreaElement. It centralises terminal coordinate math, scroll-cursor
// tracking, and default event handling so that fixes need only be applied
// once.
//
// Constraints:
//  1. textControlBase does not know about the UA shadow subtree structure;
//     it only knows about a single "uaDiv" element which holds the IFC line boxes.
//  2. It relies on the host element's Scroll/ScrollTo methods for actual
//     scroll state, as mandated by ADR-012. It must never call txa.Scroll/
//     ScrollTo DOM methods. textControlBase MUST NOT store scrollX/Y fields.
//  3. syncCallback MUST be called after any buffer mutation so that the
//     concrete element can rebuild its UA subtree and mark the render tree dirty.

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/editor"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/key"
)

type textControlBase[T Element] struct {
	host         T
	uaDiv        dom.Element
	buf          *editor.Buffer
	isMultiline  bool
	syncCallback func()

	// lastKnownCX/Y track the caret position in content-box coordinates at the
	// time of the last layout.
	lastKnownCX int
	lastKnownCY int

	// needsScrollIntoView is a flag set after a mutation or cursor move;
	// it triggers a ScrollCursorIntoView call during the commit/paint phase.
	needsScrollIntoView bool

	// lastSyncedOffset tracks the buffer offset at the time lastKnownCX/Y
	// were computed.
	lastSyncedOffset int

	// selectionStart and selectionEnd track the current local selection in
	// byte offsets. If selectionStart == selectionEnd, there is no selection.
	selectionStart int
	selectionEnd   int

	// isDragging is true during a mouse selection drag.
	isDragging bool

	// mouseMoveSub and mouseUpSub track the global document listeners during
	// a drag operation.
	mouseMoveSub event.Subscription
	mouseUpSub   event.Subscription

	// lastSyncedVersion tracks the buffer.Version() at the time of the last
	// full shadow-DOM rebuild.
	lastSyncedVersion int64

	// lastRenderedVersion tracks the buffer.Version() at the time of the last
	// paint phase.
	lastRenderedVersion int64
}

// initTextControlBase initialises the base with its dependencies.
// host is the outer dom.Element (the shadow host for the control).
// uaDiv is the inner block element whose IFC children form the text layout.
// buf is the editor buffer.
// isMultiline distinguishes textarea (true) from input (false).
// sync is the concrete element's sync callback.
func (b *textControlBase[T]) initTextControlBase(
	host T,
	uaDiv dom.Element,
	buf *editor.Buffer,
	isMultiline bool,
	sync func(),
) {
	b.host = host
	b.uaDiv = uaDiv
	b.buf = buf
	b.isMultiline = isMultiline
	b.syncCallback = sync
	b.lastSyncedVersion = -1   // force first sync
	b.lastRenderedVersion = -1 // force first paint
	b.lastSyncedOffset = -1    // force first cursor calc
	b.selectionStart = buf.ByteOffset()
	b.selectionEnd = buf.ByteOffset()
}

// wireTextControlEvents wires up default key bindings and mouse events.
func (b *textControlBase[T]) wireTextControlEvents() {
	b.host.AddEventListener(event.EventMouseDown, b.handleMouseDown)
	b.host.AddEventListener(event.EventKeyDown, b.handleKeyDown)
	b.host.AddEventListener(event.EventPaste, b.handlePaste)
	b.host.AddEventListener(event.EventCopy, b.handleCopy)
	b.host.AddEventListener(event.EventCut, b.handleCut)
}

// SetSelectionRange sets the selection to the given byte offsets.
// It moves the cursor to end and triggers a sync.
func (b *textControlBase[T]) SetSelectionRange(start, end int) {
	maxLen := len(b.buf.Value())
	if start < 0 {
		start = 0
	}
	if start > maxLen {
		start = maxLen
	}
	if end < 0 {
		end = 0
	}
	if end > maxLen {
		end = maxLen
	}

	b.selectionStart = start
	b.selectionEnd = end
	b.buf.SetOffset(end)
	b.syncCallback()
}

// SelectionRange returns the current selection start and end byte offsets.
func (b *textControlBase[T]) SelectionRange() (int, int) {
	return b.selectionStart, b.selectionEnd
}

// SelectedText returns the text currently covered by the local selection.
func (b *textControlBase[T]) SelectedText() string {
	if b.selectionStart == b.selectionEnd {
		return ""
	}
	start, end := b.selectionStart, b.selectionEnd
	if start > end {
		start, end = end, start
	}
	val := b.buf.Value()
	if start < 0 || end > len(val) {
		return ""
	}
	return val[start:end]
}

// --- cursor.Provider ---------------------------------------------------------

// CursorState implements cursor.Provider. It returns the terminal-cell
// coordinate of the caret within the host's content box, derived from the
// IFC fragment tree via cursor.FromTextFragment.
func (b *textControlBase[T]) CursorState() cursor.State {
	// If the buffer version has changed since the last time the cursor was
	// rendered, we must return true so that engine.updateHardwareCursor
	// triggers a repaint to recalculate the cursor position.
	if b.buf.Version() != b.lastRenderedVersion || b.buf.ByteOffset() != b.lastSyncedOffset {
		// Update cached coordinates from the live fragment tree.
		uaDivFrag := b.uaDivFragment()
		if uaDivFrag != nil {
			// Use the FromTextFragment helper to find cell coordinates for the
			// current buffer offset.
			cx, cy, ok := cursor.FromTextFragment(uaDivFrag, b.buf.ByteOffset())
			if ok {
				b.lastKnownCX = cx
				b.lastKnownCY = cy
				b.lastSyncedOffset = b.buf.ByteOffset()
				b.lastRenderedVersion = b.buf.Version()
			}
		}
	}

	// Hardware cursor is only visible if the host element is focused.
	focused := false
	if doc := b.host.OwnerDocument(); doc != nil {
		if fm, ok := doc.FocusManager().(interface{ IsFocused(dom.Node) bool }); ok {
			focused = fm.IsFocused(b.host)
		}
	}

	// Add the host's border and padding insets.
	ro := b.host.RenderObject()
	if ro == nil {
		return cursor.State{Visible: false}
	}
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	// Coordinates are relative to the host's content box origin.
	return cursor.State{
		Visible: focused,
		X:       insetLeft + b.lastKnownCX,
		Y:       insetTop + b.lastKnownCY,
		Shape:   cursor.ShapeBarBlink,
	}
}

// handleMouseDown handles a left-button mouse-down event, translating the
// screen-space click coordinates to a buffer byte offset while accounting for
// the host's border+padding inset and current scroll offset.
func (b *textControlBase[T]) handleMouseDown(ev event.Event) {
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return
	}
	if me.Button != event.ButtonLeft {
		return
	}

	// Stop propagation to prevent the Document from starting its own generic
	// selection drag.
	ev.StopPropagation()

	ro := b.host.RenderObject()
	if ro == nil || ro.Fragment() == nil {
		return
	}

	uaDivRO := b.uaDiv.RenderObject()
	if uaDivRO == nil || uaDivRO.Fragment() == nil {
		return
	}

	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	scrollX, scrollY := b.host.Scroll()

	targetX := me.Local.X - insetLeft + scrollX
	targetY := me.Local.Y - insetTop + scrollY

	offset := cursor.ByteOffsetAtPoint(uaDivRO.Fragment(), targetX, targetY)

	// Clamp to buffer length. Multiline controls have a trailing <br> in the
	// UA tree that can produce an off-by-one offset; single-line controls are
	// naturally bounded.
	if maxLen := len(b.buf.Value()); offset > maxLen {
		offset = maxLen
	}

	b.buf.SetOffset(offset)
	b.selectionStart = offset
	b.selectionEnd = offset
	b.isDragging = true

	// Register global document listeners for dragging.
	doc := b.host.OwnerDocument()
	b.clearDragSubscriptions()
	b.mouseMoveSub = doc.AddEventListener(event.EventMouseMove, b.handleMouseMove)
	b.mouseUpSub = doc.AddEventListener(event.EventMouseUp, b.handleMouseUp)

	b.syncCallback()
}

func (b *textControlBase[T]) handleMouseMove(ev event.Event) {
	if !b.isDragging {
		return
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok {
		return
	}

	ro := b.host.RenderObject()
	if ro == nil || ro.Fragment() == nil {
		return
	}

	uaDivRO := b.uaDiv.RenderObject()
	if uaDivRO == nil || uaDivRO.Fragment() == nil {
		return
	}

	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	scrollX, scrollY := b.host.Scroll()

	// Convert global mouse position to local coordinates.
	// We need to account for the host's absolute screen position.
	hostRect, ok := b.host.GetBoundingClientRect()
	if !ok {
		return
	}

	targetX := me.Screen.X - hostRect.Origin.X - insetLeft + scrollX
	targetY := me.Screen.Y - hostRect.Origin.Y - insetTop + scrollY

	offset := cursor.ByteOffsetAtPoint(uaDivRO.Fragment(), targetX, targetY)
	if maxLen := len(b.buf.Value()); offset > maxLen {
		offset = maxLen
	}

	b.selectionEnd = offset
	b.buf.SetOffset(offset)
	b.needsScrollIntoView = true
	b.syncCallback()
}

func (b *textControlBase[T]) handleMouseUp(_ event.Event) {
	if !b.isDragging {
		return
	}
	b.isDragging = false
	b.clearDragSubscriptions()
}

func (b *textControlBase[T]) clearDragSubscriptions() {
	if b.mouseMoveSub != nil {
		b.mouseMoveSub.Cancel()
		b.mouseMoveSub = nil
	}
	if b.mouseUpSub != nil {
		b.mouseUpSub.Cancel()
		b.mouseUpSub = nil
	}
}

// ProvidesCursor implements dom.Element.
func (b *textControlBase[T]) ProvidesCursor() bool {
	return true
}

// ScrollCursorIntoView implements dom.Element.
func (b *textControlBase[T]) ScrollCursorIntoView() {
	if !b.needsScrollIntoView {
		return
	}
	b.needsScrollIntoView = false

	// Ensure lastKnownCX/CY are up to date.
	b.CursorState()
	cx, cy := b.lastKnownCX, b.lastKnownCY

	// Host bounds.
	ro := b.host.RenderObject()
	if ro == nil {
		return
	}
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	width := ro.Fragment().Size.Width - bw.Left - bw.Right - cs.Padding.Left - cs.Padding.Right
	height := ro.Fragment().Size.Height - bw.Top - bw.Bottom - cs.Padding.Top - cs.Padding.Bottom

	// Current scroll.
	sx, sy := b.host.Scroll()

	// New scroll.
	nsx, nsy := sx, sy

	if cx < sx {
		nsx = cx
	} else if cx >= sx+width {
		nsx = cx - width + 1
	}

	if cy < sy {
		nsy = cy
	} else if cy >= sy+height {
		nsy = cy - height + 1
	}

	// Clamp to max possible scroll to handle shrinking content.
	maxSX, maxSY := layout.MaxScroll(ro.Fragment())
	nsx = max(0, min(nsx, maxSX))
	nsy = max(0, min(nsy, maxSY))

	if nsx != sx || nsy != sy {
		b.host.ScrollTo(nsx, nsy)
	}
}

// handleKeyDown processes a keydown event and routes it to the appropriate
// buffer operation.
func (b *textControlBase[T]) handleKeyDown(ev event.Event) {
	ke, ok := ev.(*event.KeyEvent)
	if !ok {
		return
	}

	shift := ke.Mod&key.ModShift != 0
	ctrl := ke.Mod&key.ModCtrl != 0

	switch {
	case ke.MatchString("backspace"):
		if !b.maybeDeleteSelection(ke) {
			b.buf.DeletePrevious()
		}
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("delete"):
		if !b.maybeDeleteSelection(ke) {
			b.buf.DeleteNext()
		}
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()
		b.syncCallback()
		ke.PreventDefault()
	case ke.Code == key.KeyLeft:
		b.buf.MoveLeft()
		if shift {
			b.selectionEnd = b.buf.ByteOffset()
		} else {
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
		}
		b.syncCallback()
		ke.PreventDefault()
	case ke.Code == key.KeyRight:
		b.buf.MoveRight()
		if shift {
			b.selectionEnd = b.buf.ByteOffset()
		} else {
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
		}
		b.syncCallback()
		ke.PreventDefault()
	case ke.Code == key.KeyUp:
		if b.isMultiline {
			b.moveUp()
			if shift {
				b.selectionEnd = b.buf.ByteOffset()
			} else {
				b.selectionStart = b.buf.ByteOffset()
				b.selectionEnd = b.buf.ByteOffset()
			}
			b.syncCallback()
		}
		ke.PreventDefault()
	case ke.Code == key.KeyDown:
		if b.isMultiline {
			b.moveDown()
			if shift {
				b.selectionEnd = b.buf.ByteOffset()
			} else {
				b.selectionStart = b.buf.ByteOffset()
				b.selectionEnd = b.buf.ByteOffset()
			}
			b.syncCallback()
		}
		ke.PreventDefault()
	case ke.Code == key.KeyEnter:
		if b.isMultiline {
			b.maybeDeleteSelection(ke)
			b.buf.Insert("\n")
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
			b.syncCallback()
			ke.PreventDefault()
		}
		// single-line: do not prevent so the engine can handle submit
	case ke.Code == key.KeyHome:
		b.buf.MoveToStart()
		if shift {
			b.selectionEnd = b.buf.ByteOffset()
		} else {
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
		}
		b.syncCallback()
		ke.PreventDefault()
	case ke.Code == key.KeyEnd:
		b.buf.MoveToEnd()
		if shift {
			b.selectionEnd = b.buf.ByteOffset()
		} else {
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
		}
		b.syncCallback()
		ke.PreventDefault()
	case (ke.Code == 'a' || ke.Code == 'A') && ctrl:
		b.selectionStart = 0
		b.selectionEnd = len(b.buf.Value())
		b.buf.SetOffset(b.selectionEnd)
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+w"), ke.MatchString("alt+backspace"):
		if !b.maybeDeleteSelection(ke) {
			b.buf.DeleteWordPrevious()
		}
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+k"):
		// Delete from cursor to end of buffer.
		if !b.maybeDeleteSelection(ke) {
			b.buf.DeleteWordNext()
		}
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+u"):
		// Delete from start of buffer to cursor.
		if !b.maybeDeleteSelection(ke) {
			b.buf.DeleteWordPrevious()
		}
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()
		b.syncCallback()
		ke.PreventDefault()
	default:
		// Printable character: insert if non-empty Text field and no ctrl/alt.
		if ke.Text != "" && !ctrl && ke.Mod&key.ModAlt == 0 && ke.Mod&key.ModMeta == 0 {
			b.maybeDeleteSelection(ke)
			b.buf.Insert(ke.Text)
			b.selectionStart = b.buf.ByteOffset()
			b.selectionEnd = b.buf.ByteOffset()
			b.syncCallback()
			ke.PreventDefault()
		}
	}
}

func (b *textControlBase[T]) handlePaste(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	text := ce.Text()
	if text == "" {
		return
	}

	b.maybeDeleteSelection(nil)
	b.buf.Insert(text)
	b.selectionStart = b.buf.ByteOffset()
	b.selectionEnd = b.buf.ByteOffset()
	b.syncCallback()

	// Dispatch TypeInput
	b.host.DispatchEvent(event.NewInput(b.buf.Value()))

	ev.PreventDefault()
	ev.StopPropagation()
}

func (b *textControlBase[T]) handleCopy(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	text := b.SelectedText()
	if text != "" {
		ce.SetText(text)
		ev.PreventDefault()
	}
}

func (b *textControlBase[T]) handleCut(ev event.Event) {
	ce, ok := ev.(*event.ClipboardEvent)
	if !ok {
		return
	}

	text := b.SelectedText()
	if text == "" {
		return
	}

	ce.SetText(text)
	b.buf.DeleteRange(b.selectionStart, b.selectionEnd)
	b.selectionStart = b.buf.ByteOffset()
	b.selectionEnd = b.buf.ByteOffset()
	b.syncCallback()

	// Dispatch TypeInput
	b.host.DispatchEvent(event.NewInput(b.buf.Value()))

	ev.PreventDefault()
}

func (b *textControlBase[T]) maybeDeleteSelection(_ *event.KeyEvent) bool {
	if b.selectionStart == b.selectionEnd {
		return false
	}
	b.buf.DeleteRange(b.selectionStart, b.selectionEnd)
	return true
}

// UpdateSelectionRange programmatically updates the document's selection
// based on the control's local selectionStart/selectionEnd offsets.
// It maps byte offsets to (Node, runeOffset) pairs within the UA subtree.
func (b *textControlBase[T]) UpdateSelectionRange() {
	doc := b.host.OwnerDocument()
	if doc == nil {
		return
	}

	sel := doc.Selection()
	if b.selectionStart == b.selectionEnd {
		// Ensure local selection state stays in sync with the buffer's cursor
		// position for programmatic moves (SetValue, manual Buffer().SetOffset).
		b.selectionStart = b.buf.ByteOffset()
		b.selectionEnd = b.buf.ByteOffset()

		// If we own the selection, clear it.
		// A simple way is to check if the first range starts in our uaDiv.
		if sel.RangeCount() > 0 {
			r := sel.GetRangeAt(0)
			if b.isNodeInUASubtree(r.StartContainer()) {
				sel.RemoveAllRanges()
			}
		}
		return
	}

	startOffset := b.selectionStart
	endOffset := b.selectionEnd
	if startOffset > endOffset {
		startOffset, endOffset = endOffset, startOffset
	}

	startNode, startRune := b.resolveOffset(startOffset)
	endNode, endRune := b.resolveOffset(endOffset)

	if startNode == nil || endNode == nil {
		return
	}

	// Optimization: Create the range and set its bounds BEFORE adding it to the
	// selection. This ensures we only trigger one 'selectionchange' event and
	// avoid redundant snapshot allocations in the dispatcher.
	r := doc.CreateRange()
	r.SetStart(startNode, startRune)
	r.SetEnd(endNode, endRune)

	sel.RemoveAllRanges()
	sel.AddRange(r)
}

func (b *textControlBase[T]) isNodeInUASubtree(n dom.Node) bool {
	return dom.IsUANode(n)
}

func (b *textControlBase[T]) resolveOffset(targetByteOffset int) (dom.Node, int) {
	currByte := 0
	childIdx := 0

	for child := b.uaDiv.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(dom.TextNode); ok {
			data := t.Data()
			byteLen := len(data)

			if targetByteOffset >= currByte && targetByteOffset < currByte+byteLen {
				// At start or strictly inside.
				rel := targetByteOffset - currByte
				runeOffset := 0
				for i := range data {
					if i >= rel {
						break
					}
					runeOffset++
				}
				return t, runeOffset
			}
			currByte += byteLen
		} else if el, ok := child.(dom.Element); ok && el.TagName() == "br" {
			// Content <br> represents exactly 1 byte (\n).
			// Placeholder <br> is 0 bytes.
			isPlaceholder := false
			if p, ok := el.(interface{ IsPlaceholderBr() bool }); ok {
				isPlaceholder = p.IsPlaceholderBr()
			}

			if !isPlaceholder {
				if currByte == targetByteOffset {
					// Position before this \n.
					return b.uaDiv, childIdx
				}
				currByte++ // \n
			} else {
				// Placeholder <br>.
				if currByte == targetByteOffset {
					return b.uaDiv, childIdx
				}
			}
		}
		childIdx++
	}

	// Fallback to end of uaDiv.
	return b.uaDiv, childIdx
}

// --- Vertical navigation helpers --------------------------------------------

// uaDivFragment returns the fragment for the inner ua-div, whose direct
// children are IFC line-boxes suitable for cursor.FromTextFragment.
func (b *textControlBase[T]) uaDivFragment() *layout.Fragment {
	ro := b.uaDiv.RenderObject()
	if ro == nil {
		return nil
	}
	return ro.Fragment()
}

func (b *textControlBase[T]) moveUp() {
	uaDivFrag := b.uaDivFragment()
	if uaDivFrag == nil {
		return
	}

	// We need a fresh cursor state calculation to ensure lastKnownCX/Y are up to date.
	b.CursorState()
	curX, curY := b.lastKnownCX, b.lastKnownCY

	// Standard IFC fragments: children are line-boxes.
	// Line 0 is at Y=0.
	if curY <= 0 {
		// Already on the first line.
		return
	}

	targetY := curY - 1
	offset := cursor.ByteOffsetAtPoint(uaDivFrag, curX, targetY)
	b.buf.SetOffset(offset)
}

func (b *textControlBase[T]) moveDown() {
	uaDivFrag := b.uaDivFragment()
	if uaDivFrag == nil {
		return
	}

	// We need a fresh cursor state calculation to ensure lastKnownCX/Y are up to date.
	b.CursorState()
	curX, curY := b.lastKnownCX, b.lastKnownCY

	// Check if target line exists.
	// IFC produces one fragment per line.
	lastLineY := 0
	for _, childLink := range uaDivFrag.Children {
		if childLink.Offset.Y > lastLineY {
			lastLineY = childLink.Offset.Y
		}
	}

	if curY >= lastLineY {
		// Already on the last line.
		return
	}

	targetY := curY + 1
	offset := cursor.ByteOffsetAtPoint(uaDivFrag, curX, targetY)
	if maxLen := len(b.buf.Value()); offset > maxLen {
		offset = maxLen
	}
	b.buf.SetOffset(offset)
}
