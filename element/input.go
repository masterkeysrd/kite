package element

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// uaInputDiv is the inner UA block element inside the Input host's shadow
// subtree. It has Width:MaxContent so the block algorithm does not constrain
// it to the available width; the host's overflow:clip boundary and scroll
// offset control what is visible.
type uaInputDiv struct {
	dom.Element
}

// Unwrap returns the underlying DOM element so that the DOM tree internals
// (asBase, InsertBefore, etc.) can navigate to the concrete baseNode.
func (d *uaInputDiv) Unwrap() dom.Node { return d.Element }

// DefaultStyle returns Width:MaxContent so the div grows to accommodate its
// text content rather than stretching to the host's available width.
func (d *uaInputDiv) DefaultStyle() style.Style {
	return style.Style{Width: style.Some(style.MaxContent)}
}

// InputElement is a single-line text-input widget implemented as a UA shadow
// host (ADR-009).
//
// Architecture summary:
//   - The host gets a plain render.Box; the standard IFC shapes the text.
//   - A UA shadow subtree is attached via AttachUARoot(): the structure is
//     ua-input-root → ua-input-div → TextNode. The div is an unconstrained
//     block that grows as the user types; the host clips it via overflow:clip
//     and translates it via ScrollTo/ScrollBy.
//   - UA-mandated styles live in IntrinsicStyle(): display:inline-block,
//     overflow-x:clip, overflow-y:clip, white-space:nowrap.
//   - Cursor positioning uses cursor.FromTextFragment on the inner div's
//     fragment, whose direct children are the IFC line-boxes.
//   - Common text-editing mechanics (CursorState, ScrollCursorIntoView,
//     handleMouseDown, handleKeyDown) are implemented by the embedded
//     textControlBase[InputElement]. See ADR-013, TSK-029.
//
// See ADR-009, ADR-010, ADR-013, TSK-024, TSK-029.
type InputElement struct {
	elementBase[InputElement]
	textControlBase[*InputElement]

	// uaText is the single UA-internal text node whose Data() mirrors buf.Value().
	// It is never nil after construction.
	uaText dom.TextNode

	// uaDiv is the inner UA block element (ua-input-div) that wraps uaText.
	// Its render object's fragment has the IFC line-boxes as direct children,
	// making it the correct root for cursor.FromTextFragment and
	// scroll-offset calculations.
	//
	// This field is also stored in textControlBase.uaDiv (as dom.Element) for
	// the shared geometry math; we keep a typed copy here for syncText.
	uaInputDivEl *uaInputDiv
}

// Compile-time interface assertions.
var (
	_ Element         = (*InputElement)(nil)
	_ cursor.Provider = (*InputElement)(nil)
)

// intrinsicInputStyle is the UA-mandated style shared by all InputElement
// instances. Set once at package init; never mutated.
var intrinsicInputStyle = style.Style{
	OverflowX:  style.Some(style.OverflowClip),
	OverflowY:  style.Some(style.OverflowClip),
	WhiteSpace: style.Some(style.WhiteSpacePre),
}

// defaultInputStyle holds the author-overridable defaults for an input.
// Height is intentionally absent: the block algorithm naturally produces
// border.Top + 1 content row + border.Bottom, which is the correct
// terminal height for a single-line bordered field. Hard-coding Height(1)
// would cap the outer box at 1 cell and crush the border rows.
var defaultInputStyle = style.Style{
	Display: style.Some(style.DisplayInlineBlock),
	Width:   style.Some(style.Cells(20)),
	Padding: style.Some(style.EdgeValues[int]{}),
}

// NewInput creates a new InputElement owned by doc with an optional initial
// value.
func NewInput(doc dom.Document, initialValue string) *InputElement {
	buf := editor.NewBuffer(initialValue)

	// Create the UA text node with the initial buffer value.
	uaText := doc.CreateTextNode(buf.Value(), nil)

	// Create the inner UA div that wraps the text node. This div is an
	// unconstrained block container: it grows as the user types while the
	// host clips it via overflow:clip. Its render object fragment has the
	// IFC line-boxes as direct children, making it the correct root for
	// cursor.FromTextFragment and scroll-offset calculations.
	//
	// The *uaInputDiv wrapper is stored directly as the DOM child (not the
	// underlying raw element) so that the engine's createRenderObject sets
	// logicalNode = *uaInputDiv, enabling BaseRender.DefaultStyle() to pick
	// up the Width:MaxContent override via duck-typing.
	uaInnerDiv := &uaInputDiv{}
	uaInnerDivEl := doc.CreateElement("ua-input-div", uaInnerDiv)
	uaInnerDiv.Element = uaInnerDivEl
	uaInnerDiv.AppendChild(uaText) // appends to uaInnerDivEl via embedding

	inp := &InputElement{
		uaText:       uaText,
		uaInputDivEl: uaInnerDiv,
	}

	// Create the host DOM element. The self-pointer is the InputElement so
	// that ADR-0036 identity resolution collapses to the outermost wrapper.
	el := doc.CreateElement("input", inp)

	inp.initBase(el, inp, defaultInputStyle, intrinsicInputStyle)

	// Initialise the shared text-control base.
	inp.initTextControlBase(inp, uaInnerDiv, buf, false /* single-line */, inp.syncText)

	// Attach the UA shadow subtree.
	//
	// Structure: ua-input-root → ua-input-div → TextNode.
	//
	// LayoutChildren(host) yields ua-input-root.ChildNodes() = [ua-input-div],
	// so the host's block algorithm sees ua-input-div as its sole block child.
	// The ua-input-div then contains the TextNode, which the IFC shapes into
	// line-boxes. The host clips the ua-input-div at its border+padding boundary
	// and uses its scroll offset to pan the content.
	uaRoot := doc.CreateElement("ua-input-root", nil)
	uaRoot.AppendChild(uaInnerDiv) // the *uaInputDiv is the DOM tree node
	el.AttachUARoot(uaRoot)

	// Wire up default key bindings and mouse events (via textControlBase).
	inp.wireTextControlEvents()

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

// TextContent returns the current text value of the input, satisfying the dom.Node interface.
func (inp *InputElement) TextContent() string {
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

// --- dom.Focusable -----------------------------------------------------------

// IsFocusable always returns true for input elements.
func (inp *InputElement) IsFocusable() bool { return true }

// Focus is a no-op placeholder (focus management is handled by focus.Manager).
func (inp *InputElement) Focus() {
	inp.needsScrollIntoView = true
}

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
	inp.needsScrollIntoView = true

	// Only rebuild shadow DOM if the content actually changed.
	v := inp.buf.Version()
	if v != inp.lastSyncedVersion {
		inp.uaText.SetData(inp.buf.Value())
		inp.lastSyncedVersion = v

		if ro := inp.uaInputDivEl.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
		}
		if ro := inp.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
			ro.MarkChildrenDirty()
		}
	} else {
		// Just cursor move: only need to repaint to update hardware cursor.
		if ro := inp.RenderObject(); ro != nil {
			ro.MarkDirty(render.DirtyPaint)
		}
	}
}

// OnWheel implements event.Scrollable. Input disables wheel scrolling.
func (inp *InputElement) OnWheel(e *event.WheelEvent) {}
