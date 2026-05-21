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
//     an InlineBr item, which the LineBreaker treats as a forced line end.
//   - BrElement contributes zero bytes to the text buffer byte-offset model,
//     so cursor.FromTextFragment treats a <br> as transparent.
//
// This is used by TextAreaElement's UA shadow subtree to represent newlines
// inserted via the Enter key, matching the HTML spec model.
type BrElement struct {
	elementBase[BrElement]
}

// Compile-time assertion.
var _ Element = (*BrElement)(nil)

// intrinsicBrStyle is the UA-mandated style for <br>: display inline so the
// IFC builder processes it inline rather than as a block child.
var intrinsicBrStyle = style.Style{
	Display: style.Some(style.DisplayInline),
}

// NewBr creates a new BrElement owned by doc.
func NewBr(doc dom.Document) *BrElement {
	br := &BrElement{}
	el := doc.CreateElement("br", br)
	br.initBase(el, br, style.Style{}, intrinsicBrStyle)
	return br
}

// Br creates a new BrElement using the orphan document.
func Br() *BrElement {
	return NewBr(orphanDocument)
}

// IsBr implements the layout.brElement interface. The IFC builder uses this to
// emit an InlineBr item rather than treating <br> as a generic inline container.
func (b *BrElement) IsBr() bool { return true }

// IntrinsicStyle returns the UA-mandated style for <br>.
func (b *BrElement) IntrinsicStyle() style.Style { return intrinsicBrStyle }

// IsFocusable returns false: <br> is never a focus target.
func (b *BrElement) IsFocusable() bool { return false }
