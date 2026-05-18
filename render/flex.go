package render

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Flex represents a flex container formatting object.
type Flex struct {
	BaseRender

	logicalNode any
	eventTarget event.EventTarget

	// Styles
	computedStyle *style.Computed
	rawStyle      style.Style

	focusable bool
	disabled  bool
}

var _ Object = (*Flex)(nil)

// NewFlex creates a new Flex render object.
func NewFlex(logicalNode any, target event.EventTarget) *Flex {
	f := &Flex{
		logicalNode: logicalNode,
		eventTarget: target,
		// Default flex style
		computedStyle: &style.Computed{Display: style.DisplayFlex},
	}
	f.Init(f)
	return f
}

func (f *Flex) EventTarget() event.EventTarget { return f.eventTarget }

func (f *Flex) Focusable() bool     { return f.focusable }
func (f *Flex) Disabled() bool      { return f.disabled }
func (f *Flex) SetFocusable(v bool) { f.focusable = v }
func (f *Flex) SetDisabled(v bool)  { f.disabled = v }

func (f *Flex) ComputedStyle() *style.Computed     { return f.computedStyle }
func (f *Flex) SetComputedStyle(c *style.Computed) { f.computedStyle = c }
func (f *Flex) Style() *style.Computed             { return f.computedStyle }

func (f *Flex) IsDetached() bool { return f.Parent() == nil }

func (f *Flex) RawStyle() style.Style { return f.rawStyle }
func (f *Flex) SetRawStyle(s style.Style) {
	f.rawStyle = s
	f.MarkDirty(DirtyStyle | DirtyLayout | DirtyPaint)
}
func (f *Flex) ElementDefaultStyle() style.Style { return style.Style{} }
func (f *Flex) IsDirtyStyle() bool               { return f.Flags()&DirtyStyle != 0 }
func (f *Flex) HasDirtyStyleChild() bool         { return f.Flags()&ChildNeedsStyle != 0 }
func (f *Flex) ClearDirtyStyle()                 { f.ClearDirty(DirtyStyle) }
func (f *Flex) ClearChildNeedsStyle()            { f.ClearDirty(ChildNeedsStyle) }
func (f *Flex) StyleParent() style.StyleNode {
	if f.Parent() != nil {
		return f.Parent()
	}
	return nil
}
func (f *Flex) StyleFirstChild() style.StyleNode {
	if f.FirstChild() != nil {
		return f.FirstChild()
	}
	return nil
}
func (f *Flex) StyleNextSibling() style.StyleNode {
	if f.NextSibling() != nil {
		return f.NextSibling()
	}
	return nil
}

func (f *Flex) LogicalNode() any { return f.logicalNode }
