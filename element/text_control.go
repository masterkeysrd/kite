package element

// textControlBase is a generic base struct shared by InputElement and
// TextAreaElement. It centralises terminal coordinate math, scroll-cursor
// tracking, and default event handling so that fixes need only be applied
// in one place.
//
// The type parameter T is the concrete element type (e.g. InputElement or
// TextAreaElement). This allows the base to hold a properly-typed host
// pointer for RenderObject / ComputedStyle / Scroll queries.
//
// Designed for TSK-029 / ADR-013.
//
// Rules:
//  1. textControlBase MUST NOT assume the structure of the text nodes inside
//     uaDiv. It purely uses uaDiv.RenderObject().Fragment() to ask the layout
//     engine for the resulting text-fragment geometry.
//  2. All scroll state is owned by the host element via the TSK-028 Scroll /
//     ScrollTo DOM methods. textControlBase MUST NOT store scrollX/Y fields.
//  3. syncCallback MUST be called after any buffer mutation so that the
//     concrete element can rebuild its UA subtree and mark the render tree dirty.

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
)

// textControlBase[T] holds the state shared by single-line and multi-line
// text controls. It must be embedded (not used standalone) by a concrete
// element that also embeds elementBase[T].
//
// Construction helpers: use initTextControlBase to populate the fields.
type textControlBase[T dom.Element] struct {
	// buf is the 1-D logical text model for this control.
	buf *editor.Buffer

	// uaDiv is the inner UA block element whose render object's fragment has
	// IFC line-boxes as direct children. cursor.FromTextFragment and
	// cursor.ByteOffsetAtPoint both require this level of the tree.
	uaDiv dom.Element

	// host is the outer dom.Element (the shadow host). We use it to query
	// RenderObject, ComputedStyle, and the Scroll/ScrollTo APIs.
	host T

	// needsScrollIntoView is true when the cursor has moved or text has
	// changed. ScrollCursorIntoView reads and clears this flag.
	needsScrollIntoView bool

	// isMultiline controls Up/Down/Enter handling. When false (Input), those
	// keys are either ignored or consumed so that the focus engine's default
	// spatial-navigation does not leave the field.
	isMultiline bool

	// syncCallback is called after every buffer mutation to let the concrete
	// element rebuild its UA subtree and mark the render tree dirty.
	syncCallback func()

	// lastKnownCX, lastKnownCY cache the most recent valid cursor position in
	// uaDiv-local coordinates. CursorState() updates these whenever
	// FromTextFragment returns ok=true, and falls back to them when the
	// fragment is stale (e.g. when called from a keydown handler before the
	// layout phase has run after a buffer mutation).
	lastKnownCX, lastKnownCY int

	// lastSyncedOffset tracks the buffer offset at the time lastKnownCX/Y
	// were computed.
	lastSyncedOffset int

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
}

// --- cursor.Provider ---------------------------------------------------------

// CursorState implements cursor.Provider. It returns the terminal-cell
// coordinate of the caret within the host's content box, derived from the
// IFC fragment tree via cursor.FromTextFragment.
//
// The coordinate is expressed relative to the host's border-box origin so
// that engine.updateHardwareCursor can subtract the host's absolute origin
// and the element's scroll offset to arrive at the physical screen position.
//
// If the host has not been laid out yet (no fragment), the cursor is placed
// at the top-left corner (0, 0) with Visible=true.
func (b *textControlBase[T]) CursorState() cursor.State {
	ro := b.host.RenderObject()
	if ro == nil || ro.Fragment() == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	uaDivRO := b.uaDiv.RenderObject()
	if uaDivRO == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	uaDivFrag := uaDivRO.Fragment()
	if uaDivFrag == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	offset := b.buf.ByteOffset()
	cx, cy := b.lastKnownCX, b.lastKnownCY

	// Recalculate only if offset changed or the fragment is new.
	if offset != b.lastSyncedOffset {
		if freshX, freshY, ok := cursor.FromTextFragment(uaDivFrag, offset); ok {
			cx, cy = freshX, freshY
			b.lastKnownCX, b.lastKnownCY = cx, cy
			b.lastSyncedOffset = offset
		}
	}

	// Add the host's inset (border + padding) so the returned state is
	// expressed relative to the host's border-box origin — matching the
	// convention expected by engine.updateHardwareCursor.
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	b.lastRenderedVersion = b.buf.Version()

	return cursor.State{
		Visible: true,
		X:       insetLeft + cx,
		Y:       insetTop + cy,
		Shape:   cursor.ShapeBarBlink,
	}
}

// --- Scroll engine -----------------------------------------------------------

// ScrollCursorIntoView ensures the caret is visible within the host's content
// viewport by adjusting the host's scroll offset. It is a no-op if
// needsScrollIntoView is false, or if the host has not been laid out yet.
//
// For single-line controls (isMultiline == false), contentH is effectively 1
// and Y-scrolling becomes a natural no-op without any special-case code.
func (b *textControlBase[T]) ScrollCursorIntoView() {
	if !b.needsScrollIntoView {
		return
	}
	b.needsScrollIntoView = false

	ro := b.host.RenderObject()
	if ro == nil || ro.Fragment() == nil {
		return
	}

	uaDivRO := b.uaDiv.RenderObject()
	if uaDivRO == nil {
		return
	}
	uaDivFrag := uaDivRO.Fragment()
	if uaDivFrag == nil {
		return
	}

	// Ensure we have current coordinates. CursorState() handles caching/recalc.
	state := b.CursorState()
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	// Translate state (host-local) back to uaDiv-local.
	cx := state.X - insetLeft
	cy := state.Y - insetTop

	contentW := max(0, ro.Fragment().Size.Width-bw.Left-bw.Right-cs.Padding.Left-cs.Padding.Right)
	contentH := max(0, ro.Fragment().Size.Height-bw.Top-bw.Bottom-cs.Padding.Top-cs.Padding.Bottom)

	scrollX, scrollY := b.host.Scroll()
	newX, newY := scrollX, scrollY

	// 1. Ensure cursor is visible in X.
	if cx < scrollX {
		newX = cx
	} else if cx >= scrollX+contentW {
		newX = cx - contentW + 1
	}

	// 2. Ensure cursor is visible in Y.
	if cy < scrollY {
		newY = cy
	} else if cy >= scrollY+contentH {
		newY = cy - contentH + 1
	}

	// 3. Clamp to allowed scroll range.
	maxScrollX, maxScrollY := layout.MaxScroll(ro.Fragment())

	if newX > maxScrollX {
		newX = maxScrollX
	}
	if newX < 0 {
		newX = 0
	}
	if newY > maxScrollY {
		newY = maxScrollY
	}
	if newY < 0 {
		newY = 0
	}

	if newX != scrollX || newY != scrollY {
		b.host.ScrollTo(newX, newY)
	}
}

// --- Event handlers ----------------------------------------------------------

// wireTextControlEvents registers the default keystroke and mouse handlers on
// the host element. This must be called once from the concrete element's
// constructor after initTextControlBase.
func (b *textControlBase[T]) wireTextControlEvents() {
	b.host.AddEventListener(event.EventKeyDown, b.handleKeyDown)
	b.host.AddEventListener(event.EventMouseDown, b.handleMouseDown)
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
	b.syncCallback()
}

// ProvidesCursor implements dom.Element.
func (b *textControlBase[T]) ProvidesCursor() bool {
	// If the buffer version has changed since the last time the cursor was
	// rendered, we must return true so that engine.updateHardwareCursor
	// triggers a repaint to recalculate the cursor position.
	if b.buf.Version() != b.lastRenderedVersion {
		if ro := b.host.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyPaint)
		}
	}
	return true
}

// handleKeyDown processes a keydown event and routes it to the appropriate
// buffer operation.
//
// Up, Down, and Enter are guarded by isMultiline:
//   - For single-line controls, Up and Down are consumed (PreventDefault) so
//     the engine's spatial-navigation does not shift focus away.
//   - Enter is silently ignored by single-line controls (not prevented, so the
//     engine may handle form submission in the future).
func (b *textControlBase[T]) handleKeyDown(ev event.Event) {
	ke, ok := ev.(*event.KeyEvent)
	if !ok {
		return
	}

	switch {
	case ke.MatchString("backspace"):
		b.buf.DeletePrevious()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("delete"):
		b.buf.DeleteNext()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("left"):
		b.buf.MoveLeft()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("right"):
		b.buf.MoveRight()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("up"):
		if b.isMultiline {
			b.moveUp()
			b.syncCallback()
		}
		ke.PreventDefault()
	case ke.MatchString("down"):
		if b.isMultiline {
			b.moveDown()
			b.syncCallback()
		}
		ke.PreventDefault()
	case ke.MatchString("enter"):
		if b.isMultiline {
			b.buf.Insert("\n")
			b.syncCallback()
			ke.PreventDefault()
		}
		// single-line: do not prevent so the engine can handle submit
	case ke.MatchString("home"), ke.MatchString("ctrl+a"):
		b.buf.MoveToStart()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("end"), ke.MatchString("ctrl+e"):
		b.buf.MoveToEnd()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+w"), ke.MatchString("alt+backspace"):
		b.buf.DeleteWordPrevious()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+k"):
		// Delete from cursor to end of buffer.
		b.buf.DeleteWordNext()
		b.syncCallback()
		ke.PreventDefault()
	case ke.MatchString("ctrl+u"):
		// Delete from start of buffer to cursor.
		b.buf.DeleteWordPrevious()
		b.syncCallback()
		ke.PreventDefault()
	default:
		// Printable character: insert if non-empty Text field and no ctrl/alt.
		if ke.Text != "" && (ke.Mod&key.ModCtrl == 0) && (ke.Mod&key.ModAlt == 0) {
			b.buf.Insert(ke.Text)
			b.syncCallback()
			ke.PreventDefault()
		}
	}
}

// --- Vertical navigation helpers --------------------------------------------

// uaDivFragment returns the fragment for the inner ua-div, whose direct
// children are IFC line-boxes suitable for cursor.FromTextFragment.
func (b *textControlBase[T]) uaDivFragment() *layout.Fragment {
	if b.uaDiv == nil {
		return nil
	}
	uaDivRO := b.uaDiv.RenderObject()
	if uaDivRO == nil {
		return nil
	}
	return uaDivRO.Fragment()
}

// moveUp moves the buffer cursor up one visual line in the IFC fragment tree.
func (b *textControlBase[T]) moveUp() {
	uaDivFrag := b.uaDivFragment()
	if uaDivFrag == nil || len(uaDivFrag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(uaDivFrag, b.buf.ByteOffset())
	if !ok {
		return
	}

	// Use the Y offset of the first line as the boundary for "Top of buffer".
	if curY <= uaDivFrag.Children[0].Offset.Y {
		b.buf.MoveToStart()
		return
	}

	targetY := curY - 1
	offset := cursor.ByteOffsetAtPoint(uaDivFrag, curX, targetY)
	if maxLen := len(b.buf.Value()); offset > maxLen {
		offset = maxLen
	}
	b.buf.SetOffset(offset)
}

// moveDown moves the buffer cursor down one visual line in the IFC fragment tree.
func (b *textControlBase[T]) moveDown() {
	uaDivFrag := b.uaDivFragment()
	if uaDivFrag == nil || len(uaDivFrag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(uaDivFrag, b.buf.ByteOffset())
	if !ok {
		return
	}

	// Guard: stop at the last content line. cursor.FromTextFragment at
	// len(buf.Value()) returns the Y of the last line that has actual content.
	// When curY is already there, pressing Down is a no-op.
	_, lastLineY, okLast := cursor.FromTextFragment(uaDivFrag, len(b.buf.Value()))
	if okLast && curY >= lastLineY {
		return
	}

	targetY := curY + 1
	offset := cursor.ByteOffsetAtPoint(uaDivFrag, curX, targetY)
	if maxLen := len(b.buf.Value()); offset > maxLen {
		offset = maxLen
	}
	b.buf.SetOffset(offset)
}
