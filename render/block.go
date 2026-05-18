package render

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Block represents a standard block-level formatting object.
type Block struct {
	BaseRender

	logicalNode any
	eventTarget event.EventTarget

	// Styles
	computedStyle *style.Computed
	rawStyle      style.Style

	focusable bool
	disabled  bool
}

var _ Object = (*Block)(nil)

// NewBlock creates a new Block render object.
func NewBlock(logicalNode any, target event.EventTarget) *Block {
	b := &Block{
		logicalNode: logicalNode,
		eventTarget: target,
		// Default block style
		computedStyle: &style.Computed{Display: style.DisplayBlock},
	}
	b.Init(b)
	return b
}

func (b *Block) EventTarget() event.EventTarget { return b.eventTarget }

func (b *Block) Focusable() bool     { return b.focusable }
func (b *Block) Disabled() bool      { return b.disabled }
func (b *Block) SetFocusable(v bool) { b.focusable = v }
func (b *Block) SetDisabled(v bool)  { b.disabled = v }

func (b *Block) ComputedStyle() *style.Computed     { return b.computedStyle }
func (b *Block) SetComputedStyle(c *style.Computed) { b.computedStyle = c }
func (b *Block) Style() *style.Computed             { return b.computedStyle }

func (b *Block) IsDetached() bool { return b.Parent() == nil }

func (b *Block) RawStyle() style.Style { return b.rawStyle }
func (b *Block) SetRawStyle(s style.Style) {
	b.rawStyle = s
	b.MarkDirty(DirtyStyle | DirtyLayout | DirtyPaint)
}
func (b *Block) ElementDefaultStyle() style.Style { return style.Style{} }
func (b *Block) IsDirtyStyle() bool               { return b.Flags()&DirtyStyle != 0 }
func (b *Block) HasDirtyStyleChild() bool         { return b.Flags()&ChildNeedsStyle != 0 }
func (b *Block) ClearDirtyStyle()                 { b.ClearDirty(DirtyStyle) }
func (b *Block) ClearChildNeedsStyle()            { b.ClearDirty(ChildNeedsStyle) }
func (b *Block) StyleParent() style.StyleNode {
	if b.Parent() != nil {
		return b.Parent()
	}
	return nil
}
func (b *Block) StyleFirstChild() style.StyleNode {
	if b.FirstChild() != nil {
		return b.FirstChild()
	}
	return nil
}
func (b *Block) StyleNextSibling() style.StyleNode {
	if b.NextSibling() != nil {
		return b.NextSibling()
	}
	return nil
}

func (b *Block) LogicalNode() any { return b.logicalNode }
