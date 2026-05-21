package element

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

// InputElement is a single-line text-input widget implemented as a UA shadow
// host (ADR-009).
//
// Architecture summary:
//   - The host gets a plain render.Box; the standard IFC shapes the text.
//   - A single synthetic text node is attached via AttachUARoot() so the IFC
//     sees it during layout, but public traversal APIs (ChildNodes, Children,
//     GetElementByID) never expose it.
//   - UA-mandated styles live in IntrinsicStyle(): display:inline-block,
//     overflow-x:clip, overflow-y:clip, white-space:nowrap.
//   - Cursor positioning is delegated to cursor.FromTextFragment so no
//     bespoke cluster-walk lives on this type.
//
// See ADR-009, ADR-010, TSK-024.
type InputElement struct {
	elementBase[InputElement]

	// buf is the 1-D logical text model.
	buf *editor.Buffer

	// uaText is the single UA-internal text node whose Data() mirrors buf.Value().
	// It is never nil after construction.
	uaText dom.TextNode
}

// Compile-time interface assertions.
var (
	_ Element         = (*InputElement)(nil)
	_ cursor.Provider = (*InputElement)(nil)
)

// intrinsicInputStyle is the UA-mandated style shared by all InputElement
// instances. Set once at package init; never mutated.
var intrinsicInputStyle = style.Style{
	Display:    style.Some(style.DisplayInlineBlock),
	OverflowX:  style.Some(style.OverflowClip),
	OverflowY:  style.Some(style.OverflowClip),
	WhiteSpace: style.Some(style.WhiteSpaceNoWrap),
}

// defaultInputStyle holds the author-overridable defaults for an input.
// Height is intentionally absent: the block algorithm naturally produces
// border.Top + 1 content row + border.Bottom, which is the correct
// terminal height for a single-line bordered field. Hard-coding Height(1)
// would cap the outer box at 1 cell and crush the border rows.
var defaultInputStyle = style.Style{
	Width:   style.Some(style.Cells(20)),
	Padding: style.Some(style.EdgeValues[int]{}),
}

// NewInput creates a new InputElement owned by doc with an optional initial
// value.
func NewInput(doc dom.Document, initialValue string) *InputElement {
	buf := editor.NewBuffer(initialValue)

	// Create the UA text node with the initial buffer value.
	uaText := doc.CreateTextNode(buf.Value(), nil)

	inp := &InputElement{
		buf:    buf,
		uaText: uaText,
	}

	// Create the host DOM element. The self-pointer is the InputElement so
	// that ADR-0036 identity resolution collapses to the outermost wrapper.
	el := doc.CreateElement("input", inp)

	inp.initBase(el, inp, defaultInputStyle, intrinsicInputStyle)

	// Attach the UA shadow subtree. This stamps every node in the subtree with
	// the host's outer back-pointer and schedules a sync. The text node is the
	// direct child of the (implicit) UA root; we pass it directly since there
	// is no containing element needed for a single-node subtree.
	//
	// Per ADR-009 the UA root is the node passed to AttachUARoot. We want the
	// engine to walk the text node directly, so we create a minimal UA
	// container element and append the text node to it.
	uaRoot := doc.CreateElement("ua-input-root", nil)
	uaRoot.AppendChild(uaText)
	el.AttachUARoot(uaRoot)

	// Wire up default key bindings.
	inp.wireKeyListeners()

	return inp
}

// Input creates a new InputElement with the given initial value using the
// orphan document (for use in declarative tree construction).
func Input(initialValue string) *InputElement {
	return NewInput(orphanDocument, initialValue)
}

// Value returns the current text value of the input.
func (inp *InputElement) Value() string {
	return inp.buf.Value()
}

// SetValue replaces the buffer content with v, repositioning the cursor to
// the end of the new value.
func (inp *InputElement) SetValue(v string) *InputElement {
	inp.buf = editor.NewBuffer(v)
	inp.syncText()
	return inp
}

// Buffer returns the underlying editor.Buffer for direct manipulation.
// After any direct mutation, callers must call SyncBuffer() to propagate the
// change to the UA text node and the render tree.
func (inp *InputElement) Buffer() *editor.Buffer {
	return inp.buf
}

// SyncBuffer propagates the current buffer state to the UA text node and
// marks layout dirty. Call this after any direct mutation of Buffer().
func (inp *InputElement) SyncBuffer() {
	inp.syncText()
}

// --- cursor.Provider ---------------------------------------------------------

// CursorState implements cursor.Provider. It returns the terminal-cell
// coordinate of the caret within the host's content box, derived from the
// IFC fragment tree via cursor.FromTextFragment.
//
// If the host has not been laid out yet (no fragment), the cursor is placed
// at the top-left corner.
func (inp *InputElement) CursorState() cursor.State {
	ro := inp.RenderObject()
	if ro == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	frag := ro.Fragment()
	if frag == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	x, y, _ := cursor.FromTextFragment(frag, inp.buf.ByteOffset())
	return cursor.State{
		Visible: true,
		X:       x,
		Y:       y,
		Shape:   cursor.ShapeBarBlink,
	}
}

// --- dom.Focusable -----------------------------------------------------------

// IsFocusable always returns true for input elements.
func (inp *InputElement) IsFocusable() bool { return true }

// Focus is a no-op placeholder (focus management is handled by focus.Manager).
func (inp *InputElement) Focus() {}

// Blur is a no-op placeholder (focus management is handled by focus.Manager).
func (inp *InputElement) Blur() {}

// --- style.StyleNode overrides -----------------------------------------------

// IntrinsicStyle returns the UA-mandated forced styles for this element.
// These properties cannot be overridden by author styles (RawStyle). See ADR-010.
func (inp *InputElement) IntrinsicStyle() style.Style {
	return intrinsicInputStyle
}

// --- internal helpers --------------------------------------------------------

// syncText updates the UA text node to mirror the current buffer value and
// marks the render object dirty for layout and paint.
func (inp *InputElement) syncText() {
	inp.uaText.SetData(inp.buf.Value())
	if ro := inp.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
		ro.MarkChildrenDirty()
	}
}

// wireKeyListeners registers the default keystroke handlers on the host element.
// Each handler mutates the buffer and calls syncText.
func (inp *InputElement) wireKeyListeners() {
	inp.AddEventListener(event.EventKeyDown, inp.handleKeyDown)
}

// handleKeyDown processes a keydown event and routes it to the appropriate
// buffer operation.
func (inp *InputElement) handleKeyDown(ev event.Event) {
	ke, ok := ev.(*event.KeyEvent)
	if !ok {
		return
	}

	switch {
	case ke.MatchString("backspace"):
		inp.buf.DeletePrevious()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("delete"):
		inp.buf.DeleteNext()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("left"):
		inp.buf.MoveLeft()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("right"):
		inp.buf.MoveRight()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("up"), ke.MatchString("down"):
		// Single-line input: up/down have no meaning. Consume the event so
		// the engine's default spatial-navigation handler does not shift
		// focus away from this field.
		ke.PreventDefault()
	case ke.MatchString("home"), ke.MatchString("ctrl+a"):
		inp.buf.MoveToStart()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("end"), ke.MatchString("ctrl+e"):
		inp.buf.MoveToEnd()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("ctrl+w"), ke.MatchString("alt+backspace"):
		inp.buf.DeleteWordPrevious()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("ctrl+k"):
		// Delete from cursor to end of buffer.
		inp.buf.DeleteWordNext()
		inp.syncText()
		ke.PreventDefault()
	case ke.MatchString("ctrl+u"):
		// Delete from start of buffer to cursor.
		inp.buf.DeleteWordPrevious()
		inp.syncText()
		ke.PreventDefault()
	default:
		// Printable character: insert if non-empty Text field and no ctrl.
		if ke.Text != "" && (ke.Mod&key.ModCtrl == 0) && (ke.Mod&key.ModAlt == 0) {
			inp.buf.Insert(ke.Text)
			inp.syncText()
			ke.PreventDefault()
		}
	}
}

// --- TextAreaElement ---------------------------------------------------------

// TextAreaElement is a multi-line text-input widget implemented as a UA shadow
// host (ADR-009).
//
// Architecture summary:
//   - The host gets a plain render.Box; the standard IFC handles shaping,
//     wrapping (via white-space: pre-wrap), mandatory breaks, and clipping.
//   - A single synthetic text node is attached via AttachUARoot().
//   - UA-mandated styles live in IntrinsicStyle(): display:inline-block,
//     overflow-x:clip, overflow-y:scroll, white-space:pre-wrap,
//     overflow-wrap:break-word.
//   - Cursor positioning uses cursor.FromTextFragment (TSK-023).
//   - Up/Down navigation is implemented by walking the fragment tree.
type TextAreaElement struct {
	elementBase[TextAreaElement]

	// buf is the 1-D logical text model.
	buf *editor.Buffer

	// uaText is the single UA-internal text node whose Data() mirrors buf.Value().
	// It is never nil after construction.
	uaText dom.TextNode
}

// Compile-time interface assertions.
var (
	_ Element         = (*TextAreaElement)(nil)
	_ cursor.Provider = (*TextAreaElement)(nil)
)

// intrinsicTextAreaStyle is the UA-mandated style for TextAreaElement.
var intrinsicTextAreaStyle = style.Style{
	Display:      style.Some(style.DisplayInlineBlock),
	OverflowX:    style.Some(style.OverflowClip),
	OverflowY:    style.Some(style.OverflowScroll),
	WhiteSpace:   style.Some(style.WhiteSpacePreWrap),
	OverflowWrap: style.Some(style.OverflowWrapBreakWord),
}

// defaultTextAreaStyle holds the author-overridable defaults for a textarea.
var defaultTextAreaStyle = style.Style{
	Width:   style.Some(style.Cells(20)),
	Height:  style.Some(style.Cells(5)),
	Padding: style.Some(style.EdgeValues[int]{}),
}

// NewTextArea creates a new TextAreaElement owned by doc with an optional
// initial value.
func NewTextArea(doc dom.Document, initialValue string) *TextAreaElement {
	buf := editor.NewBuffer(initialValue)

	// Create the UA text node with the initial buffer value.
	uaText := doc.CreateTextNode(buf.Value(), nil)

	txa := &TextAreaElement{
		buf:    buf,
		uaText: uaText,
	}

	// Create the host DOM element.
	el := doc.CreateElement("textarea", txa)

	txa.initBase(el, txa, defaultTextAreaStyle, intrinsicTextAreaStyle)

	// Attach the UA shadow subtree.
	uaRoot := doc.CreateElement("ua-textarea-root", nil)
	uaRoot.AppendChild(uaText)
	el.AttachUARoot(uaRoot)

	// Wire up default key bindings.
	txa.wireKeyListeners()

	return txa
}

// TextArea creates a new TextAreaElement using the orphan document.
func TextArea(initialValue string) *TextAreaElement {
	return NewTextArea(orphanDocument, initialValue)
}

// Value returns the current text value.
func (txa *TextAreaElement) Value() string {
	return txa.buf.Value()
}

// SetValue replaces the buffer content.
func (txa *TextAreaElement) SetValue(v string) *TextAreaElement {
	txa.buf = editor.NewBuffer(v)
	txa.syncText()
	return txa
}

// Buffer returns the underlying editor.Buffer.
func (txa *TextAreaElement) Buffer() *editor.Buffer {
	return txa.buf
}

// SyncBuffer propagates the current buffer state to the UA text node.
func (txa *TextAreaElement) SyncBuffer() {
	txa.syncText()
}

// --- cursor.Provider ---------------------------------------------------------

// CursorState implements cursor.Provider.
func (txa *TextAreaElement) CursorState() cursor.State {
	ro := txa.RenderObject()
	if ro == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	frag := ro.Fragment()
	if frag == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	x, y, _ := cursor.FromTextFragment(frag, txa.buf.ByteOffset())
	return cursor.State{
		Visible: true,
		X:       x,
		Y:       y,
		Shape:   cursor.ShapeBarBlink,
	}
}

// --- dom.Focusable -----------------------------------------------------------

func (txa *TextAreaElement) IsFocusable() bool { return true }
func (txa *TextAreaElement) Focus()            {}
func (txa *TextAreaElement) Blur()             {}

// --- style.StyleNode overrides -----------------------------------------------

func (txa *TextAreaElement) IntrinsicStyle() style.Style {
	return intrinsicTextAreaStyle
}

// --- internal helpers --------------------------------------------------------

func (txa *TextAreaElement) syncText() {
	txa.uaText.SetData(txa.buf.Value())
	if ro := txa.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
		ro.MarkChildrenDirty()
	}
}

func (txa *TextAreaElement) wireKeyListeners() {
	txa.AddEventListener(event.EventKeyDown, txa.handleKeyDown)
}

func (txa *TextAreaElement) handleKeyDown(ev event.Event) {
	ke, ok := ev.(*event.KeyEvent)
	if !ok {
		return
	}

	switch {
	case ke.MatchString("backspace"):
		txa.buf.DeletePrevious()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("delete"):
		txa.buf.DeleteNext()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("left"):
		txa.buf.MoveLeft()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("right"):
		txa.buf.MoveRight()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("up"):
		txa.moveUp()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("down"):
		txa.moveDown()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("enter"):
		txa.buf.Insert("\n")
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("home"), ke.MatchString("ctrl+a"):
		txa.buf.MoveToStart()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("end"), ke.MatchString("ctrl+e"):
		txa.buf.MoveToEnd()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	case ke.MatchString("ctrl+w"), ke.MatchString("alt+backspace"):
		txa.buf.DeleteWordPrevious()
		txa.syncText()
		txa.scrollCursorIntoView()
		ke.PreventDefault()
	default:
		if ke.Text != "" && (ke.Mod&key.ModCtrl == 0) && (ke.Mod&key.ModAlt == 0) {
			txa.buf.Insert(ke.Text)
			txa.syncText()
			txa.scrollCursorIntoView()
			ke.PreventDefault()
		}
	}
}

func (txa *TextAreaElement) moveUp() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	frag := ro.Fragment()
	if frag == nil || len(frag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(frag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	// Use the Y offset of the first line as the boundary for "Top of buffer".
	if curY <= frag.Children[0].Offset.Y {
		txa.buf.MoveToStart()
		return
	}

	targetY := curY - 1
	txa.buf.SetOffset(txa.offsetAtPoint(frag, curX, targetY))
}

func (txa *TextAreaElement) moveDown() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	frag := ro.Fragment()
	if frag == nil || len(frag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(frag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	targetY := curY + 1
	offset := txa.offsetAtPoint(frag, curX, targetY)
	txa.buf.SetOffset(offset)
}

func (txa *TextAreaElement) offsetAtPoint(root *layout.Fragment, targetX, targetY int) int {
	runningBytes := 0
	for _, lineLink := range root.Children {
		lineBox := lineLink.Fragment
		lineBytes := txa.countLineBytes(lineBox)

		if lineLink.Offset.Y == targetY {
			return runningBytes + txa.resolveXOffset(lineBox, targetX-lineLink.Offset.X)
		}
		runningBytes += lineBytes
	}

	if targetY >= root.Size.Height {
		return len(txa.buf.Value())
	}
	if targetY < 0 {
		return 0
	}

	return txa.buf.ByteOffset()
}

func (txa *TextAreaElement) countLineBytes(lineBox *layout.Fragment) int {
	total := 0
	for _, childLink := range lineBox.Children {
		for _, c := range childLink.Fragment.Text {
			total += len(c.Bytes)
		}
	}
	return total
}

func (txa *TextAreaElement) resolveXOffset(lineBox *layout.Fragment, targetX int) int {
	bytesSeen := 0

	for _, childLink := range lineBox.Children {
		child := childLink.Fragment
		childX := childLink.Offset.X // relative to lineBox
		xInChild := 0

		if len(child.Text) > 0 {
			for _, c := range child.Text {
				// Stop before the mandatory break character (e.g. \n) because its
				// logical position is on this line, but its visual position
				// is effectively "after" the line. Offset-wise, the byte
				// after \n belongs to the next line.
				if c.BreakClass == text.BreakMandatory {
					return bytesSeen
				}

				// If we are already at or past the target visual column, return
				// the bytes accumulated so far.
				if childX+xInChild >= targetX {
					return bytesSeen
				}

				cw := clusterWidth(c)
				// If adding this cluster would take us past targetX, stop here.
				if cw > 0 && childX+xInChild+cw > targetX {
					return bytesSeen
				}
				bytesSeen += len(c.Bytes)
				xInChild += cw
			}
		} else {
			// Atomic inline: check if we should stop before or after it.
			if childX+child.Size.Width > targetX {
				return bytesSeen
			}
			if childX+child.Size.Width >= targetX {
				return bytesSeen
			}
		}
	}
	return bytesSeen
}

func clusterWidth(c text.Cluster) int {
	if c.CellWidth < 0 {
		return 0
	}
	return c.CellWidth
}

func (txa *TextAreaElement) scrollCursorIntoView() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	frag := ro.Fragment()
	if frag == nil {
		return
	}
	_, y, ok := cursor.FromTextFragment(frag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	txa.ScrollTo(0, y)
}
