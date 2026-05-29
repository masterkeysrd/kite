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
	"github.com/masterkeysrd/kite/event"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/text"
	"github.com/masterkeysrd/kite/style"
)

// TextAreaElement is a multi-line text-input widget implemented as a UA shadow
// host (ADR-009).
type TextAreaElement struct {
	elementBase[TextAreaElement]
	textControlBase[*TextAreaElement]

	name     string
	disabled bool
}

// uaTextAreaDiv is the inner UA block element that wraps the text content.
// It has Width:Content so it doesn't stretch to the host's width.
type uaTextAreaDiv struct {
	dom.Element
}

func (d *uaTextAreaDiv) Unwrap() dom.Node { return d.Element }
func (d *uaTextAreaDiv) DefaultStyle() style.Style {
	return style.Style{Width: style.Some(style.Auto)}
}
func (d *uaTextAreaDiv) RawStyle() style.Style       { return style.Style{} }
func (d *uaTextAreaDiv) IntrinsicStyle() style.Style { return style.Style{} }
func (d *uaTextAreaDiv) IsDirtyStyle() bool {
	if de := internaldom.AsDirtyElement(d.Element); de != nil {
		return de.IsDirtyStyle()
	}
	return false
}

// Compile-time interface assertions.
var (
	_ Element         = (*TextAreaElement)(nil)
	_ cursor.Provider = (*TextAreaElement)(nil)
	_ dom.FormControl = (*TextAreaElement)(nil)
)

// intrinsicTextAreaStyle is the UA-mandated style for TextAreaElement.
var intrinsicTextAreaStyle = style.Style{
	OverflowX:    style.Some(style.OverflowClip),
	OverflowY:    style.Some(style.OverflowAuto),
	OverflowWrap: style.Some(style.OverflowWrapBreakWord),
	WhiteSpace:   style.Some(style.WhiteSpacePreWrap),
}

// defaultTextAreaStyle holds the author-overridable defaults for a textarea.
var defaultTextAreaStyle = style.Style{
	Display:   style.Some(style.DisplayInlineBlock),
	Width:     style.Some(style.Cells(20)),
	Height:    style.Some(style.Cells(5)),
	Padding:   style.Some(style.EdgeValues[int]{}),
	Scrollbar: style.Some(style.Scrollbar{Y: style.Some(true)}),
}

// NewTextArea creates a new TextAreaElement owned by doc with an optional
// initial value.
func NewTextArea(doc dom.Document, initialValue string) *TextAreaElement {
	buf := text.NewBuffer(initialValue)

	txa := &TextAreaElement{}

	// Create the host DOM element.
	el := doc.CreateElement("textarea", txa)
	txa.initBase(el, txa, defaultTextAreaStyle, intrinsicTextAreaStyle)

	// Build the UA shadow subtree following the HTML model:
	//   ua-textarea-root → ua-div → [text1, <br>, text2, ..., <br>(trailing)]
	//
	// The ua-div is an unconstrained block that grows to fit content. The host
	// clips vertically (overflow-y: scroll) and the scroll offset pans content.
	uaDiv := &uaTextAreaDiv{doc.CreateElement("ua-textarea-div", nil)}

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
func (txa *TextAreaElement) Value() any {
	return txa.buf.Value()
}

// WithName sets the form control name and returns the TextAreaElement.
func (txa *TextAreaElement) WithName(name string) *TextAreaElement {
	txa.name = name
	return txa
}

// Name returns the form control name.
func (txa *TextAreaElement) Name() string {
	return txa.name
}

// TextContent returns the current text value of the textarea, satisfying the dom.Node interface.
func (txa *TextAreaElement) TextContent() string {
	return txa.buf.Value()
}

// SetValue replaces the buffer content.
func (txa *TextAreaElement) SetValue(v string) *TextAreaElement {
	txa.buf = text.NewBuffer(v)
	txa.syncText()
	return txa
}

// Buffer returns the underlying text.Buffer.
func (txa *TextAreaElement) Buffer() *text.Buffer {
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

// --- Style resolution overrides ----------------------------------------------

func (txa *TextAreaElement) IntrinsicStyle() style.Style {
	return intrinsicTextAreaStyle
}

// --- internal helpers --------------------------------------------------------

func (txa *TextAreaElement) syncText() {
	txa.needsScrollIntoView = true

	// Only rebuild shadow DOM if the content actually changed.
	v := txa.buf.Version()
	if v != txa.lastSyncedVersion {
		txa.rebuildUASubtree()
		txa.lastSyncedVersion = v

		if d := internaldom.AsDirty(txa.uaDiv); d != nil {
			d.MarkNeedsSync()
		}
		if d := internaldom.AsDirty(txa); d != nil {
			d.MarkNeedsSync()
		}
	} else {
		// Just cursor move: only need to repaint to update hardware cursor.
		if d := internaldom.AsDirty(txa); d != nil {
			d.MarkNeedsSync()
		}
	}
	txa.UpdateSelectionRange()
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
// rebuildUASubtree rebuilds the ua-div's children to match the current buffer
// value following the browser's textarea DOM model. It uses an incremental
// strategy, reusing existing children where possible to avoid massive
// destruction and re-allocation on every content change.
func (txa *TextAreaElement) rebuildUASubtree() {
	value := txa.buf.Value()

	doc := txa.OwnerDocument()
	if doc == nil {
		doc = orphanDocument
	}

	// Empty buffer: just a placeholder <br> so the textarea has height.
	if value == "" {
		txa.syncChildren([]dom.Node{NewPlaceholderBr(doc)})
		return
	}

	// Split by '\n'. Each separator produces a content <br>.
	lines := splitLines(value)
	endsWithNewline := value[len(value)-1] == '\n'

	var newChildren []dom.Node
	for i, line := range lines {
		if line != "" {
			newChildren = append(newChildren, doc.CreateTextNode(line, nil))
		}

		isLastSegment := i == len(lines)-1
		if !isLastSegment {
			newChildren = append(newChildren, NewBr(doc))
		} else if endsWithNewline {
			newChildren = append(newChildren, NewPlaceholderBr(doc))
		}
	}

	txa.syncChildren(newChildren)
}

func (txa *TextAreaElement) syncChildren(newChildren []dom.Node) {
	uaDiv := txa.uaDiv

	// Strategy:
	// 1. Iterate over newChildren.
	// 2. If an existing child at index i exists:
	//    - If it's the same kind, update it (for TextNode).
	//    - If different kind, replace it.
	// 3. If no existing child, append.
	// 4. Remove any remaining trailing existing children.

	currentChild := uaDiv.FirstChild()
	for _, newNode := range newChildren {
		if currentChild != nil {
			nextChild := currentChild.NextSibling()
			if !txa.tryUpdateNode(currentChild, newNode) {
				uaDiv.ReplaceChild(newNode, currentChild)
			}
			currentChild = nextChild
		} else {
			uaDiv.AppendChild(newNode)
		}
	}

	// Remove trailing.
	for currentChild != nil {
		nextChild := currentChild.NextSibling()
		uaDiv.RemoveChild(currentChild)
		currentChild = nextChild
	}
}

func (txa *TextAreaElement) tryUpdateNode(existing, newNode dom.Node) bool {
	if existing.Kind() != newNode.Kind() {
		return false
	}
	// Also check if BrElement is placeholder vs content.
	if existing.NodeName() != newNode.NodeName() {
		return false
	}

	switch e := existing.(type) {
	case dom.TextNode:
		n := newNode.(dom.TextNode)
		if e.Data() != n.Data() {
			e.SetData(n.Data())
		}
		return true
	case dom.Element:
		// If it's a <br>, we must ensure the placeholder status matches.
		if e.TagName() == "br" {
			eb, ok := e.(interface{ IsPlaceholderBr() bool })
			nb, ok2 := newNode.(interface{ IsPlaceholderBr() bool })
			if ok && ok2 {
				return eb.IsPlaceholderBr() == nb.IsPlaceholderBr()
			}
		}
		return true
	}
	return false
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
	oldX, oldY := txa.Scroll()
	txa.ScrollBy(e.DeltaX, e.DeltaY)
	newX, newY := txa.Scroll()

	if newX != oldX || newY != oldY {
		e.StopPropagation()
	}
}

func (txa *TextAreaElement) IsDisabled() bool   { return txa.disabled }
func (txa *TextAreaElement) SetDisabled(v bool) { txa.disabled = v }
func (txa *TextAreaElement) Disabled(v bool) *TextAreaElement {
	txa.disabled = v
	return txa
}
