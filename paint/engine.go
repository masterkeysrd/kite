package paint

import (
	"image/color"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// PaintEngine handles the paint phase of the pipeline.
type PaintEngine struct{}

// NewPaintEngine creates a new PaintEngine.
func NewPaintEngine() *PaintEngine {
	return &PaintEngine{}
}

// Paint draws the immutable fragment tree onto the surface.
func (p *PaintEngine) Paint(frag *layout.Fragment, surface Surface) {
	if frag == nil {
		return
	}
	p.paintFragment(frag, layout.Point{}, surface)
}

func (p *PaintEngine) paintFragment(frag *layout.Fragment, origin layout.Point, surface Surface) {
	// Draw background if there's a style.
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		if s.Background != nil {
			p.fillRect(layout.Rect{
				Origin: origin,
				Size:   frag.Size,
			}, surface, " ", color.Transparent, s.Background)
		}

		// Render border (simplified ASCII border rendering).
		border := frag.Node.Style().Border.Width
		if border.Top > 0 || border.Bottom > 0 || border.Left > 0 || border.Right > 0 {
			p.drawBorder(layout.Rect{
				Origin: origin,
				Size:   frag.Size,
			}, surface, style.Border{
				Width: border,
				Color: frag.Node.Style().Border.Color,
			})
		}
	}

	// Recurse children
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X,
			Y: origin.Y + childLink.Offset.Y,
		}
		p.paintFragment(childLink.Fragment, childOrigin, surface)
	}
}

func (p *PaintEngine) fillRect(r layout.Rect, surface Surface, content string, fg, bg color.Color) {
	for y := 0; y < r.Size.Height; y++ {
		for x := 0; x < r.Size.Width; x++ {
			surface.Set(r.Origin.X+x, r.Origin.Y+y, Cell{
				Content: content,
				Width:   1,
				FG:      fg,
				BG:      bg,
			})
		}
	}
}

func (p *PaintEngine) drawBorder(r layout.Rect, surface Surface, borderStyle style.Border) {
	// Simplified ASCII border rendering based on border width and color.
	borderCol := borderStyle.Color.Top // Simplify: use top color for all
	// Draw Top border
	for x := 0; x < r.Size.Width; x++ {
		surface.Set(r.Origin.X+x, r.Origin.Y, Cell{Content: "-", Width: 1, FG: borderCol})
	}
	// Draw Bottom border
	for x := 0; x < r.Size.Width; x++ {
		surface.Set(r.Origin.X+x, r.Origin.Y+r.Size.Height-1, Cell{Content: "-", Width: 1, FG: borderCol})
	}
	// Draw side borders
	for y := 0; y < r.Size.Height; y++ {
		surface.Set(r.Origin.X, r.Origin.Y+y, Cell{Content: "|", Width: 1, FG: borderCol})
		surface.Set(r.Origin.X+r.Size.Width-1, r.Origin.Y+y, Cell{Content: "|", Width: 1, FG: borderCol})
	}
	// Corners
	surface.Set(r.Origin.X, r.Origin.Y, Cell{Content: "+", Width: 1, FG: borderCol})
	surface.Set(r.Origin.X+r.Size.Width-1, r.Origin.Y, Cell{Content: "+", Width: 1, FG: borderCol})
	surface.Set(r.Origin.X, r.Origin.Y+r.Size.Height-1, Cell{Content: "+", Width: 1, FG: borderCol})
	surface.Set(r.Origin.X+r.Size.Width-1, r.Origin.Y+r.Size.Height-1, Cell{Content: "+", Width: 1, FG: borderCol})
}
