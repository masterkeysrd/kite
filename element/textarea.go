package element

// TextAreaElement implements a multi-line text-input widget as a UA shadow
// host (ADR-009).
//
// Architecture summary:
//   - The host gets a plain render.Box; the standard IFC handles shaping,
//     wrapping, and clipping.
//   - The UA shadow subtree follows the browser model:
//     ua-textarea-root → ua-div → [text, <br>, text, ..., <br placeholder>?]
//     Each '\n' in the buffer is a content BrElement; a placeholder BrElement
//     (zero bytes) is appended only when the value ends with '\n' or is empty,
//     so the cursor can sit on the trailing empty line.
//   - UA-mandated styles live in IntrinsicStyle(): display:inline-block,
//     overflow-y:scroll, overflow-wrap:break-word.
//   - Cursor positioning, scroll tracking, mouse hit-testing, and keyboard
//     routing are handled by the embedded textControlBase[TextAreaElement]
//     (ADR-013, TSK-029).
//
// See ADR-009, ADR-010, ADR-013, TSK-025, TSK-029.

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// TextAreaElement is a multi-line text-input widget implemented as a UA shadow
// host (ADR-009).
type TextAreaElement struct {
	elementBase[TextAreaElement]
	textControlBase[*TextAreaElement]

	// doc is the owning document, stored for creating new child nodes in
	// rebuildUASubtree() without needing to walk the DOM tree.
	doc dom.Document
}

// Compile-time interface assertions.
var (
	_ Element         = (*TextAreaElement)(nil)
	_ cursor.Provider = (*TextAreaElement)(nil)
)

// intrinsicTextAreaStyle is the UA-mandated style for TextAreaElement.
// Note: WhiteSpace is NOT pre-wrap here because the <br> model handles line
// breaks via BrElement rather than \n characters in text nodes.
var intrinsicTextAreaStyle = style.Style{
	Display:      style.Some(style.DisplayInlineBlock),
	OverflowY:    style.Some(style.OverflowScroll),
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

	txa := &TextAreaElement{
		doc: doc,
	}

	// Create the host DOM element.
	el := doc.CreateElement("textarea", txa)
	txa.initBase(el, txa, defaultTextAreaStyle, intrinsicTextAreaStyle)

	// Build the UA shadow subtree following the HTML model:
	//   ua-textarea-root → ua-div → [text1, <br>, text2, ..., <br>(trailing)]
	//
	// The ua-div is an unconstrained block that grows to fit content. The host
	// clips vertically (overflow-y: scroll) and the scroll offset pans content.
	uaDiv := doc.CreateElement("ua-textarea-div", nil)

	// Initialise the shared text-control base with the ua-div and buffer.
	txa.initTextControlBase(txa, uaDiv, buf, true /* multi-line */, txa.syncText)

	uaRoot := doc.CreateElement("ua-textarea-root", nil)
	uaRoot.AppendChild(uaDiv)
	el.AttachUARoot(uaRoot)

	// Populate the UA subtree with the initial value.
	txa.rebuildUASubtree()

	// Wire up default key bindings and mouse events (via textControlBase).
	txa.wireTextControlEvents()

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

// --- dom.Focusable -----------------------------------------------------------

func (txa *TextAreaElement) IsFocusable() bool { return true }
func (txa *TextAreaElement) Focus() {
	txa.needsScrollIntoView = true
}
func (txa *TextAreaElement) Blur() {}

// --- style.StyleNode overrides -----------------------------------------------

func (txa *TextAreaElement) IntrinsicStyle() style.Style {
	return intrinsicTextAreaStyle
}

// --- internal helpers --------------------------------------------------------

func (txa *TextAreaElement) syncText() {
	txa.needsScrollIntoView = true
	txa.rebuildUASubtree()
	if ro := txa.uaDiv.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
	}
	if ro := txa.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
		ro.MarkChildrenDirty()
	}
}

// rebuildUASubtree rebuilds the ua-div's children to match the current buffer
// value following the browser's textarea DOM model:
//
//	"abc"    →  text("abc")
//	"abc\n"  →  text("abc") <br> <br id=placeholder>
//	"abc\nX" →  text("abc") <br> text("X")
//	"\n"     →  <br> <br id=placeholder>
//	""       →  <br id=placeholder>
//
// Rules:
//   - Each '\n' in the buffer value produces one content <br>.
//   - A placeholder <br> (zero bytes, no cursor contribution) is appended
//     when and only when the value ends with '\n', OR the value is empty.
//     It ensures the textarea has height for the cursor on the empty last line.
//   - Lines that are empty strings produce no TextNode (only their <br>).
//
// All existing children of ua-div are removed before rebuilding.
// This is a "nuke and rebuild" strategy — correct but not optimal.
func (txa *TextAreaElement) rebuildUASubtree() {
	// Remove all existing children from the ua-div.
	for {
		child := txa.uaDiv.FirstChild()
		if child == nil {
			break
		}
		txa.uaDiv.RemoveChild(child)
	}

	value := txa.buf.Value()

	// Empty buffer: just a placeholder <br> so the textarea has height.
	if value == "" {
		txa.uaDiv.AppendChild(NewPlaceholderBr(txa.doc))
		return
	}

	// Split by '\n'. Each separator produces a content <br>.
	// A trailing '\n' produces an extra empty segment at the end.
	lines := splitLines(value)
	endsWithNewline := value[len(value)-1] == '\n'

	for i, line := range lines {
		// Add the text content of this line (skip if empty).
		if line != "" {
			txa.uaDiv.AppendChild(txa.doc.CreateTextNode(line, nil))
		}

		isLastSegment := i == len(lines)-1

		if !isLastSegment {
			// This segment is followed by a '\n' in the buffer → content <br>.
			txa.uaDiv.AppendChild(NewBr(txa.doc))
		} else if endsWithNewline {
			// The final segment is empty (value ends with '\n').
			// The content <br> for that '\n' was already appended on the
			// previous iteration. Now add the placeholder so the cursor
			// can sit on the empty last line.
			txa.uaDiv.AppendChild(NewPlaceholderBr(txa.doc))
		}
		// If isLastSegment && !endsWithNewline: the last line has content
		// and no trailing '\n' → no br needed at all.
	}
}

// splitLines splits s by \n and returns the segments. Unlike strings.Split,
// this function treats a trailing \n as producing one extra empty segment
// (matching the HTML textarea model where "a\n" has two lines: "a" and "").
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

// OnWheel implements event.Scrollable.
func (txa *TextAreaElement) OnWheel(e *event.WheelEvent) {
	txa.ScrollBy(e.DeltaX, e.DeltaY)
	e.StopPropagation()
}
