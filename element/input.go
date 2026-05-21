package element

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
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
