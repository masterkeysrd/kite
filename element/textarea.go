package element

// TextAreaElement implements a multi-line text-input widget as a UA shadow
// host (ADR-009).
//
// Architecture summary:
//   - The host gets a plain render.Box; the standard IFC handles shaping,
//     wrapping, and clipping.
//   - The UA shadow subtree follows the HTML model:
//     ua-textarea-root → ua-div → [text1, <br>, text2, ..., <br>(trailing)].
//     Each line of text is a separate TextNode; <br> elements force mandatory
//     line breaks. A trailing <br> is always present so the textarea has
//     height even when the last line is empty.
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
// value. It follows the HTML textarea model:
//
//	ua-div → [text1, <br>, text2, ..., <br>(trailing)]
//
// Each line of text is a separate TextNode; <br> elements force mandatory
// line breaks in the IFC. A trailing <br> is always present so that the
// textarea has height even when the last line is empty.
//
// All existing children of ua-div are removed before rebuilding. This is a
// "nuke and rebuild" strategy — it is correct but not optimal.
func (txa *TextAreaElement) rebuildUASubtree() {
	// Remove all existing children from the ua-div.
	for {
		child := txa.uaDiv.FirstChild()
		if child == nil {
			break
		}
		txa.uaDiv.RemoveChild(child)
	}

	// Split the buffer content by \n to get individual lines.
	value := txa.buf.Value()
	lines := splitLines(value)

	// Add each line as a TextNode followed by a <br>.
	// The trailing <br> is always added (even after the last non-empty line),
	// matching the HTML spec's placeholder break.
	for _, line := range lines {
		if line != "" {
			textNode := txa.doc.CreateTextNode(line, nil)
			txa.uaDiv.AppendChild(textNode)
		}
		br := NewBr(txa.doc)
		txa.uaDiv.AppendChild(br)
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
