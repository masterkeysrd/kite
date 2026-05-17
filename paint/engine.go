package paint

import (
	"github.com/masterkeysrd/kite/render"
)

// PaintEngine handles the paint phase of the pipeline.
type PaintEngine struct{}

// NewPaintEngine creates a new PaintEngine.
func NewPaintEngine() *PaintEngine {
	return &PaintEngine{}
}

// Paint draws the render tree rooted at obj onto the surface.
func (p *PaintEngine) Paint(obj render.Object, surface Surface) {
	// Implementation would go here.
}
