package render

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Inline represents an inline-level formatting object (like <span>).
type Inline struct {
	BaseRender

	logicalNode any
	eventTarget event.EventTarget

	// Styles
	computedStyle *style.Computed
	rawStyle      style.Style

	focusable bool
	disabled  bool
}

var _ Object = (*Inline)(nil)

// NewInline creates a new Inline render object.
func NewInline(logicalNode any, target event.EventTarget) *Inline {
	i := &Inline{
		logicalNode: logicalNode,
		eventTarget: target,
		// Default inline style
		computedStyle: &style.Computed{Display: style.DisplayInline},
	}
	i.Init(i)
	return i
}

func (i *Inline) EventTarget() event.EventTarget { return i.eventTarget }

func (i *Inline) Focusable() bool     { return i.focusable }
func (i *Inline) Disabled() bool      { return i.disabled }
func (i *Inline) SetFocusable(v bool) { i.focusable = v }
func (i *Inline) SetDisabled(v bool)  { i.disabled = v }

func (i *Inline) ComputedStyle() *style.Computed     { return i.computedStyle }
func (i *Inline) SetComputedStyle(c *style.Computed) { i.computedStyle = c }
func (i *Inline) Style() *style.Computed             { return i.computedStyle }

func (i *Inline) IsDetached() bool { return i.Parent() == nil }

func (i *Inline) RawStyle() style.Style { return i.rawStyle }
func (i *Inline) SetRawStyle(s style.Style) {
	i.rawStyle = s
	i.MarkDirty(DirtyStyle | DirtyLayout | DirtyPaint)
}
func (i *Inline) ElementDefaultStyle() style.Style { return style.Style{} }
func (i *Inline) IsDirtyStyle() bool               { return i.Flags()&DirtyStyle != 0 }
func (i *Inline) HasDirtyStyleChild() bool         { return i.Flags()&ChildNeedsStyle != 0 }
func (i *Inline) ClearDirtyStyle()                 { i.ClearDirty(DirtyStyle) }
func (i *Inline) ClearChildNeedsStyle()            { i.ClearDirty(ChildNeedsStyle) }
func (i *Inline) StyleParent() style.StyleNode {
	if i.Parent() != nil {
		return i.Parent()
	}
	return nil
}
func (i *Inline) StyleFirstChild() style.StyleNode {
	if i.FirstChild() != nil {
		return i.FirstChild()
	}
	return nil
}
func (i *Inline) StyleNextSibling() style.StyleNode {
	if i.NextSibling() != nil {
		return i.NextSibling()
	}
	return nil
}

func (i *Inline) LogicalNode() any { return i.logicalNode }
