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
//
// See ADR-009, ADR-010, TSK-024.
type InputElement struct {
	elementBase[InputElement]

	// buf is the 1-D logical text model.
	buf *editor.Buffer

	// uaText is the single UA-internal text node whose Data() mirrors buf.Value().
	// It is never nil after construction.
	uaText dom.TextNode

	// uaDiv is the inner UA block element (ua-input-div) that wraps uaText.
	// Its render object's fragment has the IFC line-boxes as direct children,
	// making it the correct root for cursor.FromTextFragment.
	uaDiv *uaInputDiv
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
	WhiteSpace: style.Some(style.WhiteSpacePre),
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
		buf:    buf,
		uaText: uaText,
		uaDiv:  uaInnerDiv,
	}

	// Create the host DOM element. The self-pointer is the InputElement so
	// that ADR-0036 identity resolution collapses to the outermost wrapper.
	el := doc.CreateElement("input", inp)

	inp.initBase(el, inp, defaultInputStyle, intrinsicInputStyle)

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
// The coordinate is expressed relative to the host's border-box origin so
// that engine.updateHardwareCursor can subtract the host's absolute origin
// and the element's scroll offset to arrive at the physical screen position.
//
// If the host has not been laid out yet (no fragment), the cursor is placed
// at the top-left corner.
func (inp *InputElement) CursorState() cursor.State {
	ro := inp.RenderObject()
	if ro == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	if ro.Fragment() == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	// Use the ua-div's render object so cursor.FromTextFragment receives a
	// fragment whose direct children are IFC line-boxes.
	uaDivRO := inp.uaDiv.RenderObject()
	if uaDivRO == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	uaDivFrag := uaDivRO.Fragment()
	if uaDivFrag == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	// cx, cy are relative to the ua-div's coordinate system (no border/padding
	// on the ua-div itself, so they equal the IFC-local column and row).
	cx, cy, _ := cursor.FromTextFragment(uaDivFrag, inp.buf.ByteOffset())

	// Add the host's inset (border + padding) so the returned state is
	// expressed relative to the host's border-box origin — matching the
	// convention expected by engine.updateHardwareCursor.
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	return cursor.State{
		Visible: true,
		X:       insetLeft + cx,
		Y:       insetTop + cy,
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
	if ro := inp.uaDiv.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
	}
	if ro := inp.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
		ro.MarkChildrenDirty()
	}
}

func (inp *InputElement) ScrollCursorIntoView() {
	ro := inp.RenderObject()
	if ro == nil {
		return
	}
	if ro.Fragment() == nil {
		return
	}

	// Use the ua-div's fragment: its direct children are IFC line-boxes, making
	// cursor.FromTextFragment return a position in the ua-div's coordinate
	// space (content-relative, no inset).
	uaDivRO := inp.uaDiv.RenderObject()
	if uaDivRO == nil {
		return
	}
	uaDivFrag := uaDivRO.Fragment()
	if uaDivFrag == nil {
		return
	}

	cx, _, ok := cursor.FromTextFragment(uaDivFrag, inp.buf.ByteOffset())
	if !ok {
		return
	}

	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	contentW := max(0, ro.Fragment().Size.Width-bw.Left-bw.Right-cs.Padding.Left-cs.Padding.Right)
	totalWidth := uaDivFrag.Size.Width

	scrollX, _ := inp.Scroll()

	// 1. Ensure cursor is visible.
	if cx < scrollX {
		scrollX = cx
	} else if cx >= scrollX+contentW {
		scrollX = cx - contentW + 1
	}

	// 2. Avoid empty space at the end if the content is wider than the viewport.
	// The cursor can be at totalWidth (one past the last character).
	if totalWidth >= contentW {
		if scrollX > totalWidth-contentW+1 {
			scrollX = totalWidth - contentW + 1
		}
	} else {
		// Content fits entirely, no scroll.
		scrollX = 0
	}

	inp.ScrollTo(scrollX, 0)
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
//     wrapping, and clipping.
//   - The UA shadow subtree follows the HTML model: ua-textarea-root →
//     ua-div → [text1, <br>, text2, ..., <br>(trailing)]. Each line of
//     text is a separate TextNode; <br> elements force mandatory line breaks.
//   - This removes the previous \n-in-text-node hack (white-space:pre-wrap).
//   - UA-mandated styles live in IntrinsicStyle(): display:inline-block,
//     overflow-y:scroll, overflow-wrap:break-word.
//   - Cursor positioning uses cursor.FromTextFragment on the inner div's
//     fragment (TSK-023). The virtual \n cluster emitted by InlineBr items
//     keeps buffer byte offsets consistent.
//   - Up/Down navigation is implemented by walking the fragment tree.
type TextAreaElement struct {
	elementBase[TextAreaElement]

	// buf is the 1-D logical text model.
	buf *editor.Buffer

	// uaDiv is the inner block element (ua-textarea-div) that holds the
	// text nodes and <br> elements. Its render object's fragment has the
	// IFC line-boxes as direct children.
	uaDiv dom.Element

	// doc is the owning document, stored for creating new child nodes in
	// syncText() without needing to walk the DOM tree.
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
		buf: buf,
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
	txa.uaDiv = uaDiv

	uaRoot := doc.CreateElement("ua-textarea-root", nil)
	uaRoot.AppendChild(uaDiv)
	el.AttachUARoot(uaRoot)

	// Populate the UA subtree with the initial value.
	txa.rebuildUASubtree()

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
	if ro.Fragment() == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	// Use the ua-div's render object fragment: its direct children are the
	// IFC line-boxes, which cursor.FromTextFragment requires.
	uaDivRO := txa.uaDiv.RenderObject()
	if uaDivRO == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}
	uaDivFrag := uaDivRO.Fragment()
	if uaDivFrag == nil {
		return cursor.State{Visible: true, X: 0, Y: 0, Shape: cursor.ShapeBarBlink}
	}

	cx, cy, _ := cursor.FromTextFragment(uaDivFrag, txa.buf.ByteOffset())

	// Add the host's inset (border + padding) to the cursor coordinates.
	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	insetLeft := bw.Left + cs.Padding.Left
	insetTop := bw.Top + cs.Padding.Top

	return cursor.State{
		Visible: true,
		X:       insetLeft + cx,
		Y:       insetTop + cy,
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
// "nuke and rebuild" strategy — it is correct but not optimal. A future
// optimisation could diff the existing subtree.
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
		ke.PreventDefault()
	case ke.MatchString("delete"):
		txa.buf.DeleteNext()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("left"):
		txa.buf.MoveLeft()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("right"):
		txa.buf.MoveRight()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("up"):
		txa.moveUp()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("down"):
		txa.moveDown()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("enter"):
		txa.buf.Insert("\n")
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("home"), ke.MatchString("ctrl+a"):
		txa.buf.MoveToStart()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("end"), ke.MatchString("ctrl+e"):
		txa.buf.MoveToEnd()
		txa.syncText()
		ke.PreventDefault()
	case ke.MatchString("ctrl+w"), ke.MatchString("alt+backspace"):
		txa.buf.DeleteWordPrevious()
		txa.syncText()
		ke.PreventDefault()
	default:
		if ke.Text != "" && (ke.Mod&key.ModCtrl == 0) && (ke.Mod&key.ModAlt == 0) {
			txa.buf.Insert(ke.Text)
			txa.syncText()
			ke.PreventDefault()
		}
	}
}

func (txa *TextAreaElement) moveUp() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	uaDivFrag := txa.uaDivFragment()
	if uaDivFrag == nil || len(uaDivFrag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(uaDivFrag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	// Use the Y offset of the first line as the boundary for "Top of buffer".
	if curY <= uaDivFrag.Children[0].Offset.Y {
		txa.buf.MoveToStart()
		return
	}

	targetY := curY - 1
	txa.buf.SetOffset(txa.offsetAtPoint(uaDivFrag, curX, targetY))
}

func (txa *TextAreaElement) moveDown() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	uaDivFrag := txa.uaDivFragment()
	if uaDivFrag == nil || len(uaDivFrag.Children) == 0 {
		return
	}

	curX, curY, ok := cursor.FromTextFragment(uaDivFrag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	targetY := curY + 1
	offset := txa.offsetAtPoint(uaDivFrag, curX, targetY)
	txa.buf.SetOffset(offset)
}

// uaDivFragment returns the fragment for the inner ua-div, whose direct
// children are IFC line-boxes suitable for cursor.FromTextFragment.
func (txa *TextAreaElement) uaDivFragment() *layout.Fragment {
	if txa.uaDiv == nil {
		return nil
	}
	uaDivRO := txa.uaDiv.RenderObject()
	if uaDivRO == nil {
		return nil
	}
	return uaDivRO.Fragment()
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

func (txa *TextAreaElement) ScrollCursorIntoView() {
	ro := txa.RenderObject()
	if ro == nil {
		return
	}
	if ro.Fragment() == nil {
		return
	}

	uaDivFrag := txa.uaDivFragment()
	if uaDivFrag == nil {
		return
	}

	cx, cy, ok := cursor.FromTextFragment(uaDivFrag, txa.buf.ByteOffset())
	if !ok {
		return
	}

	cs := ro.ComputedStyle()
	bw := cs.Border.Widths()
	contentW := max(0, ro.Fragment().Size.Width-bw.Left-bw.Right-cs.Padding.Left-cs.Padding.Right)
	contentH := max(0, ro.Fragment().Size.Height-bw.Top-bw.Bottom-cs.Padding.Top-cs.Padding.Bottom)

	scrollX, scrollY := txa.Scroll()

	newX, newY := scrollX, scrollY

	// 1. Ensure cursor is visible.
	if cx < scrollX {
		newX = cx
	} else if cx >= scrollX+contentW {
		newX = cx - contentW + 1
	}

	if cy < scrollY {
		newY = cy
	} else if cy >= scrollY+contentH {
		newY = cy - contentH + 1
	}

	// 2. Avoid empty space at the end if the content is wider/taller than the viewport.
	totalWidth := uaDivFrag.Size.Width
	totalHeight := uaDivFrag.Size.Height

	if totalWidth >= contentW {
		if newX > totalWidth-contentW+1 {
			newX = totalWidth - contentW + 1
		}
	} else {
		newX = 0
	}

	if totalHeight >= contentH {
		if newY > totalHeight-contentH+1 {
			newY = totalHeight - contentH + 1
		}
	} else {
		newY = 0
	}

	if newX != scrollX || newY != scrollY {
		txa.ScrollTo(newX, newY)
	}
}

// OnWheel implements event.Scrollable. Input disables wheel scrolling.
func (inp *InputElement) OnWheel(e *event.WheelEvent) {}

// OnWheel implements event.Scrollable.
func (txa *TextAreaElement) OnWheel(e *event.WheelEvent) {
	txa.ScrollBy(e.DeltaX, e.DeltaY)
	e.StopPropagation()
}
