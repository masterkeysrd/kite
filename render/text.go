package render

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// Text represents a text-level render object.
type Text struct {
	BaseRender

	logicalNode any
	eventTarget event.EventTarget

	// Styles
	computedStyle *style.Computed
	rawStyle      style.Style
}

var _ Object = (*Text)(nil)

// NewText creates a new Text render object.
func NewText(logicalNode any, target event.EventTarget) *Text {
	t := &Text{
		logicalNode: logicalNode,
		eventTarget: target,
		// Text nodes inherit styles and typically have inline display
		computedStyle: &style.Computed{Display: style.DisplayInline},
	}
	t.Init(t)
	return t
}

func (t *Text) EventTarget() event.EventTarget { return t.eventTarget }

// Text nodes are not directly focusable or disabled in the same way as elements.
func (t *Text) Focusable() bool     { return false }
func (t *Text) Disabled() bool      { return false }
func (t *Text) SetFocusable(v bool) {}
func (t *Text) SetDisabled(v bool)  {}

func (t *Text) ComputedStyle() *style.Computed     { return t.computedStyle }
func (t *Text) SetComputedStyle(c *style.Computed) { t.computedStyle = c }
func (t *Text) Style() *style.Computed             { return t.computedStyle }

func (t *Text) IsDetached() bool { return t.Parent() == nil }

func (t *Text) RawStyle() style.Style { return t.rawStyle }
func (t *Text) SetRawStyle(s style.Style) {
	t.rawStyle = s
	t.MarkDirty(DirtyStyle | DirtyLayout | DirtyPaint)
}
func (t *Text) ElementDefaultStyle() style.Style { return style.Style{} }
func (t *Text) IsDirtyStyle() bool               { return t.Flags()&DirtyStyle != 0 }
func (t *Text) HasDirtyStyleChild() bool         { return t.Flags()&ChildNeedsStyle != 0 }
func (t *Text) ClearDirtyStyle()                 { t.ClearDirty(DirtyStyle) }
func (t *Text) ClearChildNeedsStyle()            { t.ClearDirty(ChildNeedsStyle) }

func (t *Text) StyleParent() style.StyleNode {
	if t.Parent() != nil {
		return t.Parent()
	}
	return nil
}
func (t *Text) StyleFirstChild() style.StyleNode { return nil }
func (t *Text) StyleNextSibling() style.StyleNode {
	if t.NextSibling() != nil {
		return t.NextSibling()
	}
	return nil
}

func (t *Text) LogicalNode() any { return t.logicalNode }

// Data returns the text content from the logical node.
func (t *Text) Data() string {
	if ts, ok := t.logicalNode.(interface{ Data() string }); ok {
		return ts.Data()
	}
	return ""
}

// Text nodes have no layout children.
func (t *Text) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		// No-op
	}
}
