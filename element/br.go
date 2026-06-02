package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// BrElement represents an HTML <br> element: a void inline element that
// forces a mandatory line break in the inline formatting context (IFC).
//
// Architecture summary:
//   - Display is "inline" (not a block container).
//   - The IFC builder detects BrElement via the brElement interface and emits
//     an InlineBr item, which the LineBreaker treats as a forced line end with
//     a '\n' byte cluster so that cursor.FromTextFragment counts the
//     corresponding '\n' byte from the buffer.
//   - A placeholder BrElement (placeholder == true) is detected via the
//     brPlaceholderElement interface and emits InlineBrPlaceholder instead.
//     The placeholder contributes ZERO bytes to the cursor offset model — it
//     only ensures the IFC produces an empty trailing line when the buffer
//     value ends with '\n', matching the browser's
//     <br id="placeholder"> convention.
//
// This is used by TextAreaElement's UA shadow subtree to represent newlines
// inserted via the Enter key, matching the HTML spec model.
type BrElement struct {
	elementBase[BrElement]
	placeholder bool
}

// Compile-time assertion.
var _ Element = (*BrElement)(nil)

// intrinsicBrStyle is the UA-mandated style for <br>: display inline so the
// IFC builder processes it inline rather than as a block child.
var intrinsicBrStyle = style.S().Display(style.DisplayInline)

// NewBr creates a new content BrElement owned by doc.
// It represents a '\n' character in the buffer.
func NewBr(doc dom.Document) *BrElement {
	br := &BrElement{}
	el := doc.CreateElement("br", br)
	br.initBase(el, br, style.S(), intrinsicBrStyle)
	return br
}

// NewPlaceholderBr creates a trailing placeholder BrElement owned by doc.
// It forces an empty line in the IFC but contributes ZERO bytes to the
// cursor byte-offset model — it is never backed by a '\n' in the buffer.
func NewPlaceholderBr(doc dom.Document) *BrElement {
	br := &BrElement{placeholder: true}
	el := doc.CreateElement("br", br)
	br.initBase(el, br, style.S(), intrinsicBrStyle)
	return br
}

// Br creates a new content BrElement using the orphan document.
func Br() *BrElement {
	return NewBr(orphanDocument)
}

// IsBr implements the layout.brElement interface. Returns true only for
// content brs (not placeholders), so the IFC emits a '\n' byte cluster.
func (b *BrElement) IsBr() bool { return !b.placeholder }

// IsPlaceholderBr implements the layout.brPlaceholderElement interface.
// Returns true only for placeholder brs, so the IFC sets hadForcedBreakAtEnd
// without emitting any byte cluster.
func (b *BrElement) IsPlaceholderBr() bool { return b.placeholder }

// IntrinsicStyle returns the UA-mandated style for <br>.
func (b *BrElement) IntrinsicStyle() style.Style { return intrinsicBrStyle }

// IsFocusable returns false: <br> is never a focus target.
func (b *BrElement) IsFocusable() bool { return false }
