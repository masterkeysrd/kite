package render

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Overlay represents an anchored overlay render object.
type Overlay struct {
	Box
}

var _ Object = (*Overlay)(nil)

// NewOverlay creates a new Overlay render object.
func NewOverlay(logicalNode dom.Node, target event.EventTarget) *Overlay {
	o := &Overlay{}
	o.Init(o, logicalNode, target)
	// Overlays are usually block containers for their content
	o.SetComputedStyle(&style.Computed{Display: style.DisplayInlineBlock})
	return o
}
