package paint

import (
	"github.com/masterkeysrd/kite/layout"
)

// PaintEngine handles the paint phase of the pipeline.
type PaintEngine struct{}

// NewPaintEngine creates a new PaintEngine.
func NewPaintEngine() *PaintEngine {
	return &PaintEngine{}
}

// Paint draws the immutable fragment tree onto the surface.
func (p *PaintEngine) Paint(frag *layout.Fragment, surface Surface) {
	// Implementation would go here.
}
