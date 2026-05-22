package paint

import (
	"image/color"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

// overflowClips reports whether the given Overflow value requires clipping
// descendant content. Visible does not clip; all other values do.
func overflowClips(o style.Overflow) bool {
	return o != style.OverflowVisible
}

// PaintEngine handles the paint phase of the pipeline.
type PaintEngine struct {
	DebugXRay bool
}

// NewPaintEngine creates a new PaintEngine.
func NewPaintEngine() *PaintEngine {
	return &PaintEngine{}
}

// Paint draws the immutable fragment tree onto the surface.
//
// resolveBorders is invoked exactly once on the root surface after the full
// fragment tree has been painted. It must never be called on a clipped
// sub-surface because the junction resolver must see the complete set of
// border cells across the entire viewport.
func (p *PaintEngine) Paint(frag *layout.Fragment, surface Surface) {
	if frag == nil {
		return
	}
	p.paintFragment(frag, layout.Point{}, surface)
	// resolveBorders runs once on the root surface. See invariant note above.
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

func isScrollContainer(o style.Overflow) bool {
	// overflow:clip creates a clipping boundary and supports programmatic
	// scroll offsets (via ScrollTo/ScrollBy) even though it hides the scrollbar.
	// This allows elements like <input> to pan their clipped content.
	return o == style.OverflowScroll || o == style.OverflowAuto || o == style.OverflowHidden || o == style.OverflowClip
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
			if cluster.BreakClass == text.BreakMandatory {
				continue
			}
			surface.Set(currentX, origin.Y, Cell{
				Content: string(cluster.Bytes),
				Width:   cluster.CellWidth,
				FG:      fg,
				BG:      bg,
			})
			currentX += cluster.CellWidth
		}
	}

	// 3. Compute child clip surface based on this fragment's overflow style.
	childSurface := p.computeChildSurface(frag, origin, surface)

	// 4. Handle Scroll translation (ADR-012).
	scrollX, scrollY := 0, 0
	if frag.Node != nil {
		if el, ok := frag.Node.LogicalNode().(dom.Element); ok {
			s := frag.Node.Style()
			if isScrollContainer(s.OverflowX) || isScrollContainer(s.OverflowY) {
				rawX, rawY := el.Scroll()

				// Clamping: stored scroll offset is raw author intent, we clamp to
				// actual content extent.
				bw := s.Border.Widths()
				contentW := max(0, frag.Size.Width-bw.Left-bw.Right-s.Padding.Left-s.Padding.Right)
				contentH := max(0, frag.Size.Height-bw.Top-bw.Bottom-s.Padding.Top-s.Padding.Bottom)

				extentW, extentH := 0, 0
				for _, childLink := range frag.Children {
					extentW = max(extentW, childLink.Offset.X+childLink.Fragment.Size.Width)
					extentH = max(extentH, childLink.Offset.Y+childLink.Fragment.Size.Height)
				}

				scrollX = max(0, min(rawX, extentW-contentW))
				scrollY = max(0, min(rawY, extentH-contentH))
			}
		}
	}

	// 5. X-Ray Mode (Task 33)
	if p.DebugXRay && frag.Node != nil {
		p.drawXRay(frag, origin, surface)
	}

	// 6. Recurse children (children are painted over parent).
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		p.paintFragment(childLink.Fragment, childOrigin, childSurface)
	}
}

func (p *PaintEngine) drawXRay(frag *layout.Fragment, origin layout.Point, surface Surface) {
	s := frag.Node.Style()
	if s == nil {
		return
	}

	bw := s.Border.Widths()
	pad := s.Padding
	mar := s.Margin

	// 1. Margin Box (Red border/tint)
	marginRect := layout.Rect{
		Origin: layout.Point{X: origin.X - mar.Left, Y: origin.Y - mar.Top},
		Size: layout.Size{
			Width:  frag.Size.Width + mar.Left + mar.Right,
			Height: frag.Size.Height + mar.Top + mar.Bottom,
		},
	}
	p.tintRect(marginRect, surface, color.RGBA{100, 0, 0, 255})

	// 2. Padding Box (Green border/tint)
	paddingRect := layout.Rect{
		Origin: layout.Point{X: origin.X + bw.Left, Y: origin.Y + bw.Top},
		Size: layout.Size{
			Width:  max(0, frag.Size.Width-bw.Left-bw.Right),
			Height: max(0, frag.Size.Height-bw.Top-bw.Bottom),
		},
	}
	p.tintRect(paddingRect, surface, color.RGBA{0, 100, 0, 255})

	// 3. Content Box (Blue border/tint)
	contentRect := layout.Rect{
		Origin: layout.Point{X: paddingRect.Origin.X + pad.Left, Y: paddingRect.Origin.Y + pad.Top},
		Size: layout.Size{
			Width:  max(0, paddingRect.Size.Width-pad.Left-pad.Right),
			Height: max(0, paddingRect.Size.Height-pad.Top-pad.Bottom),
		},
	}
	p.tintRect(contentRect, surface, color.RGBA{0, 0, 100, 255})
}

func (p *PaintEngine) tintRect(r layout.Rect, surface Surface, c color.Color) {
	for y := 0; y < r.Size.Height; y++ {
		for x := 0; x < r.Size.Width; x++ {
			absX, absY := r.Origin.X+x, r.Origin.Y+y
			cell := surface.CellAt(absX, absY)
			cell.BG = c
			surface.Set(absX, absY, cell)
		}
	}
}

// computeChildSurface returns the Surface that children of frag should paint
// onto. If neither overflow axis requires clipping, the parent surface is
// returned unchanged (zero-cost fast path). Otherwise a clipped sub-surface
// whose clip rect equals the fragment's content box (inset by border + padding)
// is returned; the axes whose overflow value is Visible remain unclipped by
// spanning the full border-box extent on that axis.
func (p *PaintEngine) computeChildSurface(frag *layout.Fragment, origin layout.Point, surface Surface) Surface {
	if frag == nil || frag.Node == nil || frag.Node.Style() == nil {
		return surface
	}

	s := frag.Node.Style()
	clipX := overflowClips(s.OverflowX)
	clipY := overflowClips(s.OverflowY)

	if !clipX && !clipY {
		// Fast path: nothing to clip.
		return surface
	}

	// Compute border widths (each edge is either 0 or 1).
	bw := s.Border.Widths()
	pad := s.Padding

	// Content-box insets from the fragment's border-box origin:
	//   inset = border + padding on each side.
	insetLeft := bw.Left + pad.Left
	insetTop := bw.Top + pad.Top
	insetRight := bw.Right + pad.Right
	insetBottom := bw.Bottom + pad.Bottom

	// Build the clip rect. For each axis that clips, use the content-box
	// inset (border + padding). For each axis that is Visible, extend to the
	// full surface bounds so that the Clip() call does not accidentally
	// constrain the unclipped axis.
	surfaceBounds := surface.Bounds()

	var clipRect layout.Rect

	if clipX {
		clipRect.Origin.X = origin.X + insetLeft
		clipRect.Size.Width = max(0, frag.Size.Width-insetLeft-insetRight)
	} else {
		clipRect.Origin.X = surfaceBounds.Origin.X
		clipRect.Size.Width = surfaceBounds.Size.Width
	}

	if clipY {
		clipRect.Origin.Y = origin.Y + insetTop
		clipRect.Size.Height = max(0, frag.Size.Height-insetTop-insetBottom)
	} else {
		clipRect.Origin.Y = surfaceBounds.Origin.Y
		clipRect.Size.Height = surfaceBounds.Size.Height
	}

	return surface.Clip(clipRect)
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
