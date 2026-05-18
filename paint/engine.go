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
	if frag == nil {
		return
	}

	// 1. Draw self (background and border).
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		if s.Background != nil && !isTransparent(s.Background) {
			p.fillRect(layout.Rect{
				Origin: origin,
				Size:   frag.Size,
			}, surface, " ", color.Transparent, s.Background)
		}

		// Render border (simplified ASCII border rendering).
		border := s.Border.Width
		if border.Top > 0 || border.Bottom > 0 || border.Left > 0 || border.Right > 0 {
			p.drawBorder(layout.Rect{
				Origin: origin,
				Size:   frag.Size,
			}, surface, style.Border{
				Width: border,
				Color: s.Border.Color,
			})
		}
	}

	// 2. Render text clusters for THIS fragment.
	if len(frag.Text) > 0 {
		fg, bg := p.getInheritedStyle(frag)

		currentX := origin.X
		for _, cluster := range frag.Text {
			surface.Set(currentX, origin.Y, Cell{
				Content: string(cluster.Bytes),
				Width:   cluster.CellWidth,
				FG:      fg,
				BG:      bg,
			})
			currentX += cluster.CellWidth
		}
	}

	// 3. Recurse children (children are painted over parent).
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X,
			Y: origin.Y + childLink.Offset.Y,
		}
		p.paintFragment(childLink.Fragment, childOrigin, surface)
	}
}

func (p *PaintEngine) getInheritedStyle(frag *layout.Fragment) (fg, bg color.Color) {
	// Default values
	fg = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	bg = color.Transparent

	// 1. Determine Foreground
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		if s.Foreground != nil && s.Foreground != style.TerminalDefault {
			fg = s.Foreground
		} else if frag.ParentNode != nil && frag.ParentNode.Style() != nil {
			ps := frag.ParentNode.Style()
			if ps.Foreground != nil && ps.Foreground != style.TerminalDefault {
				fg = ps.Foreground
			}
		}
	}

	// 2. Determine Background
	// Important: we only want to set a background if it's explicitly non-transparent.
	// Otherwise, we must return color.Transparent so surface.Set preserves the box background.
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		if s.Background != nil && !isTransparent(s.Background) {
			bg = s.Background
		}
	}

	if isTransparent(bg) && frag.ParentNode != nil && frag.ParentNode.Style() != nil {
		ps := frag.ParentNode.Style()
		if ps.Background != nil && !isTransparent(ps.Background) {
			bg = ps.Background
		}
	}

	return fg, bg
}

func isTransparent(c color.Color) bool {
	if c == nil || c == color.Transparent {
		return true
	}
	_, _, _, a := c.RGBA()
	return a == 0
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
	if borderCol == nil {
		borderCol = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
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
