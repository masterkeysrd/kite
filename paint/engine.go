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
	p.resolveBorders(surface)
}

func (p *PaintEngine) resolveBorders(surface Surface) {
	bounds := surface.Bounds()
	for y := bounds.Origin.Y; y < bounds.Origin.Y+bounds.Size.Height; y++ {
		for x := bounds.Origin.X; x < bounds.Origin.X+bounds.Size.Width; x++ {
			c := surface.CellAt(x, y)
			if c.BorderStyle == BorderNone {
				continue
			}

			// Check cardinal neighbors
			up := surface.CellAt(x, y-1).BorderStyle
			down := surface.CellAt(x, y+1).BorderStyle
			left := surface.CellAt(x-1, y).BorderStyle
			right := surface.CellAt(x+1, y).BorderStyle

			mask := 0
			if up != BorderNone {
				mask |= 8
			}
			if down != BorderNone {
				mask |= 4
			}
			if left != BorderNone {
				mask |= 2
			}
			if right != BorderNone {
				mask |= 1
			}

			// Find dominant style (Heaviest Style Wins)
			dominantStyle := max(right, max(left, max(down, max(up, c.BorderStyle))))
			newContent := p.getJunctionGlyph(dominantStyle, mask)
			if newContent != "" && newContent != c.Content {
				c.Content = newContent
				surface.Set(x, y, c)
			}
		}
	}
}

func (p *PaintEngine) getJunctionGlyph(style BorderStyle, mask int) string {
	switch style {
	case BorderDouble:
		glyphs := [16]string{
			0: "", 1: "═", 2: "═", 3: "═",
			4: "║", 5: "╔", 6: "╗", 7: "╦",
			8: "║", 9: "╚", 10: "╝", 11: "╩",
			12: "║", 13: "╠", 14: "╣", 15: "╬",
		}
		return glyphs[mask]
	case BorderThick:
		glyphs := [16]string{
			0: "", 1: "━", 2: "━", 3: "━",
			4: "┃", 5: "┏", 6: "┓", 7: "┳",
			8: "┃", 9: "┗", 10: "┛", 11: "┻",
			12: "┃", 13: "┣", 14: "┫", 15: "╋",
		}
		return glyphs[mask]
	case BorderAscii:
		glyphs := [16]string{
			0: "", 1: "-", 2: "-", 3: "-",
			4: "|", 5: "+", 6: "+", 7: "+",
			8: "|", 9: "+", 10: "+", 11: "+",
			12: "|", 13: "+", 14: "+", 15: "+",
		}
		return glyphs[mask]
	case BorderRounded:
		// Rounded uses single-line junctions for non-corners
		if mask == 5 {
			return "╭"
		}
		if mask == 6 {
			return "╮"
		}
		if mask == 9 {
			return "╰"
		}
		if mask == 10 {
			return "╯"
		}
		fallthrough
	default: // BorderSingle
		glyphs := [16]string{
			0: "", 1: "─", 2: "─", 3: "─",
			4: "│", 5: "┌", 6: "┐", 7: "┬",
			8: "│", 9: "└", 10: "┘", 11: "┴",
			12: "│", 13: "├", 14: "┤", 15: "┼",
		}
		return glyphs[mask]
	}
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

		// Render border.
		if s.Border.Edges.Top || s.Border.Edges.Bottom || s.Border.Edges.Left || s.Border.Edges.Right {
			_, bg := p.getInheritedStyle(frag)
			p.drawBorder(layout.Rect{
				Origin: origin,
				Size:   frag.Size,
			}, surface, s.Border, bg)
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

func (p *PaintEngine) drawBorder(r layout.Rect, surface Surface, border style.Border, bg color.Color) {
	width := r.Size.Width
	height := r.Size.Height
	x := r.Origin.X
	y := r.Origin.Y

	// Helper to get glyphs for a side
	getGlyphs := func(s style.BorderStyle) style.BorderGlyphs {
		if s == style.BorderCustom {
			return border.Glyphs
		}
		return style.BorderGlyphsMap[s]
	}

	mapStyle := func(s style.BorderStyle) BorderStyle {
		switch s {
		case style.BorderNone:
			return BorderNone
		case style.BorderSingle:
			return BorderSingle
		case style.BorderDouble:
			return BorderDouble
		case style.BorderRounded:
			return BorderRounded
		case style.BorderThick:
			return BorderThick
		case style.BorderASCII:
			return BorderAscii
		default:
			return BorderSingle
		}
	}

	// Draw Edges
	if border.Edges.Top {
		glyphs := getGlyphs(border.Styles.Top)
		bs := mapStyle(border.Styles.Top)
		c := border.Colors.Top
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		for i := range width {
			surface.Set(x+i, y, Cell{Content: glyphs.H, Width: 1, FG: c, BG: bg, BorderStyle: bs})
		}
	}
	if border.Edges.Bottom {
		glyphs := getGlyphs(border.Styles.Bottom)
		bs := mapStyle(border.Styles.Bottom)
		c := border.Colors.Bottom
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		for i := range width {
			surface.Set(x+i, y+height-1, Cell{Content: glyphs.H, Width: 1, FG: c, BG: bg, BorderStyle: bs})
		}
	}
	if border.Edges.Left {
		glyphs := getGlyphs(border.Styles.Left)
		bs := mapStyle(border.Styles.Left)
		c := border.Colors.Left
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		for i := range height {
			surface.Set(x, y+i, Cell{Content: glyphs.V, Width: 1, FG: c, BG: bg, BorderStyle: bs})
		}
	}
	if border.Edges.Right {
		glyphs := getGlyphs(border.Styles.Right)
		bs := mapStyle(border.Styles.Right)
		c := border.Colors.Right
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		for i := range height {
			surface.Set(x+width-1, y+i, Cell{Content: glyphs.V, Width: 1, FG: c, BG: bg, BorderStyle: bs})
		}
	}

	// Draw Corners
	// Top-Left
	if border.Edges.Top && border.Edges.Left {
		glyph := border.Glyphs.EffectiveTL()
		if glyph == "" {
			glyph = getGlyphs(border.Styles.Top).TL
		}
		bs := mapStyle(border.Styles.Top)
		c := border.Colors.Top
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		surface.Set(x, y, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
	}
	// Top-Right
	if border.Edges.Top && border.Edges.Right {
		glyph := border.Glyphs.EffectiveTR()
		if glyph == "" {
			glyph = getGlyphs(border.Styles.Top).TR
		}
		bs := mapStyle(border.Styles.Top)
		c := border.Colors.Top
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		surface.Set(x+width-1, y, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
	}
	// Bottom-Left
	if border.Edges.Bottom && border.Edges.Left {
		glyph := border.Glyphs.EffectiveBL()
		if glyph == "" {
			glyph = getGlyphs(border.Styles.Bottom).BL
		}
		bs := mapStyle(border.Styles.Bottom)
		c := border.Colors.Bottom
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		surface.Set(x, y+height-1, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
	}
	// Bottom-Right
	if border.Edges.Bottom && border.Edges.Right {
		glyph := border.Glyphs.EffectiveBR()
		if glyph == "" {
			glyph = getGlyphs(border.Styles.Bottom).BR
		}
		bs := mapStyle(border.Styles.Bottom)
		c := border.Colors.Bottom
		if c == nil {
			c = color.RGBA{255, 255, 255, 255}
		}
		surface.Set(x+width-1, y+height-1, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
	}
}
