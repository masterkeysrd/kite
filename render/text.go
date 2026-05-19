package render

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

var _ layout.InlineLever = (*Text)(nil)

// Text represents a text-level render object.
type Text struct {
	BaseRender
}

var _ Object = (*Text)(nil)

// NewText creates a new Text render object.
func NewText(logicalNode any, target event.EventTarget) *Text {
	t := &Text{}
	t.Init(t, logicalNode, target)
	// Text nodes inherit styles and typically have inline display
	t.SetComputedStyle(&style.Computed{Display: style.DisplayInline})
	return t
}

func (t *Text) StyleFirstChild() style.StyleNode { return nil }

// Data returns the text content from the logical node.
func (t *Text) Data() string {
	if ts, ok := t.logicalNode.(interface{ Data() string }); ok {
		return ts.Data()
	}
	return ""
}

func (t *Text) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		// No-op
	}
}

func (t *Text) IsInlineLevel() bool {
	return true
}
