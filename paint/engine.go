package paint

import (
	"image/color"
	"unsafe"

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

	borderPoints []layout.Point
	clipStack    []layout.Rect
	rootSurface  Surface
}

// NewPaintEngine creates a new PaintEngine.
func NewPaintEngine() *PaintEngine {
	return &PaintEngine{
		borderPoints: make([]layout.Point, 0, 1024),
		clipStack:    make([]layout.Rect, 0, 16),
	}
}

// PaintFragment draws the immutable fragment tree onto the surface at the given origin.
func (p *PaintEngine) PaintFragment(frag *layout.Fragment, origin layout.Point, surface Surface) {
	if frag == nil {
		return
	}
	p.rootSurface = surface
	p.clipStack = p.clipStack[:0]
	p.clipStack = append(p.clipStack, surface.Bounds())
	p.paintFragment(frag, origin)
}

// ResolveBorders resolves border junctions across the entire surface.
func (p *PaintEngine) ResolveBorders(surface Surface) {
	p.rootSurface = surface
	p.resolveBorders(surface)
}

// Paint draws the immutable fragment tree onto the surface and resolves borders.
func (p *PaintEngine) Paint(frag *layout.Fragment, surface Surface) {
	if frag == nil {
		return
	}
	p.borderPoints = p.borderPoints[:0]
	p.PaintFragment(frag, layout.Point{}, surface)
	p.ResolveBorders(surface)
}

func (p *PaintEngine) setCell(x, y int, c Cell) {
	// Check against the current clip stack top.
	clip := p.clipStack[len(p.clipStack)-1]
	if x < clip.Origin.X || y < clip.Origin.Y || x >= clip.Origin.X+clip.Size.Width || y >= clip.Origin.Y+clip.Size.Height {
		return
	}

	p.rootSurface.Set(x, y, c)
	if c.BorderStyle != BorderNone {
		p.borderPoints = append(p.borderPoints, layout.Point{X: x, Y: y})
	}
}

func (p *PaintEngine) paintFragment(frag *layout.Fragment, origin layout.Point) {
	if frag == nil {
		return
	}

	// 0. Frustum Culling: skip if fragment is entirely outside current clip bounds.
	clip := p.clipStack[len(p.clipStack)-1]
	fragRect := layout.Rect{Origin: origin, Size: frag.Size}
	if !clip.Overlaps(fragRect) {
		return
	}

	// 1. Draw self (background and border).
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()

		// Optimization: Skip rendering if no visual content and no clipping.
		hasVisuals := (s.Background != nil && !isTransparent(s.Background)) ||
			s.Border.Edges.Top || s.Border.Edges.Bottom || s.Border.Edges.Left || s.Border.Edges.Right ||
			len(frag.Text) > 0

		if hasVisuals {
			if s.Background != nil && !isTransparent(s.Background) {
				p.fillRect(layout.Rect{
					Origin: origin,
					Size:   frag.Size,
				}, " ", color.Transparent, s.Background)
			}

			// Render border.
			if s.Border.Edges.Top || s.Border.Edges.Bottom || s.Border.Edges.Left || s.Border.Edges.Right {
				_, bg := p.getInheritedStyle(frag)
				p.drawBorder(layout.Rect{
					Origin: origin,
					Size:   frag.Size,
				}, s.Border, bg)
			}
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
			// Use unsafe to avoid allocation for string conversion.
			content := unsafe.String(unsafe.SliceData(cluster.Bytes), len(cluster.Bytes))
			p.setCell(currentX, origin.Y, Cell{
				Content: content,
				Width:   cluster.CellWidth,
				FG:      fg,
				BG:      bg,
			})
			currentX += cluster.CellWidth
		}
	}

	// 3. Compute child clip stack based on this fragment's overflow style.
	p.pushChildClip(frag, origin)

	// 4. Handle Scroll translation (ADR-012).
	scrollX, scrollY := 0, 0
	if frag.Node != nil {
		if el, ok := frag.Node.LogicalNode().(dom.Element); ok {
			s := frag.Node.Style()
			if isScrollContainer(s.OverflowX) || isScrollContainer(s.OverflowY) {
				rawX, rawY := el.Scroll()
				maxSX, maxSY := layout.MaxScroll(frag)
				scrollX = max(0, min(rawX, maxSX))
				scrollY = max(0, min(rawY, maxSY))
			}
		}
	}

	// 5. X-Ray Mode (Task 33)
	if p.DebugXRay && frag.Node != nil {
		p.drawXRay(frag, origin)
	}

	// 6. Recurse children (children are painted over parent).
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		p.paintFragment(childLink.Fragment, childOrigin)
	}

	// Pop child clip.
	p.popClip()
}

func (p *PaintEngine) pushChildClip(frag *layout.Fragment, origin layout.Point) {
	parentClip := p.clipStack[len(p.clipStack)-1]

	if frag == nil || frag.Node == nil || frag.Node.Style() == nil {
		p.clipStack = append(p.clipStack, parentClip)
		return
	}

	s := frag.Node.Style()
	clipX := overflowClips(s.OverflowX)
	clipY := overflowClips(s.OverflowY)

	if !clipX && !clipY {
		p.clipStack = append(p.clipStack, parentClip)
		return
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

	var clipRect layout.Rect

	if clipX {
		clipRect.Origin.X = origin.X + insetLeft
		clipRect.Size.Width = max(0, frag.Size.Width-insetLeft-insetRight)
	} else {
		clipRect.Origin.X = parentClip.Origin.X
		clipRect.Size.Width = parentClip.Size.Width
	}

	if clipY {
		clipRect.Origin.Y = origin.Y + insetTop
		clipRect.Size.Height = max(0, frag.Size.Height-insetTop-insetBottom)
	} else {
		clipRect.Origin.Y = parentClip.Origin.Y
		clipRect.Size.Height = parentClip.Size.Height
	}

	p.clipStack = append(p.clipStack, parentClip.Intersect(clipRect))
}

func (p *PaintEngine) popClip() {
	if len(p.clipStack) > 0 {
		p.clipStack = p.clipStack[:len(p.clipStack)-1]
	}
}

func (p *PaintEngine) drawXRay(frag *layout.Fragment, origin layout.Point) {
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
	p.tintRect(marginRect, color.RGBA{100, 0, 0, 255})

	// 2. Padding Box (Green border/tint)
	paddingRect := layout.Rect{
		Origin: layout.Point{X: origin.X + bw.Left, Y: origin.Y + bw.Top},
		Size: layout.Size{
			Width:  max(0, frag.Size.Width-bw.Left-bw.Right),
			Height: max(0, frag.Size.Height-bw.Top-bw.Bottom),
		},
	}
	p.tintRect(paddingRect, color.RGBA{0, 100, 0, 255})

	// 3. Content Box (Blue border/tint)
	contentRect := layout.Rect{
		Origin: layout.Point{X: paddingRect.Origin.X + pad.Left, Y: paddingRect.Origin.Y + pad.Top},
		Size: layout.Size{
			Width:  max(0, paddingRect.Size.Width-pad.Left-pad.Right),
			Height: max(0, paddingRect.Size.Height-pad.Top-pad.Bottom),
		},
	}
	p.tintRect(contentRect, color.RGBA{0, 0, 100, 255})
}

func (p *PaintEngine) tintRect(r layout.Rect, c color.Color) {
	for y := 0; y < r.Size.Height; y++ {
		for x := 0; x < r.Size.Width; x++ {
			absX, absY := r.Origin.X+x, r.Origin.Y+y
			cell := p.rootSurface.CellAt(absX, absY)
			cell.BG = c
			p.setCell(absX, absY, cell)
		}
	}
}

func (p *PaintEngine) resolveBorders(surface Surface) {
	if len(p.borderPoints) == 0 {
		return
	}

	for _, pt := range p.borderPoints {
		x, y := pt.X, pt.Y
		c := surface.CellAt(x, y)
		if c.BorderStyle == BorderNone {
			continue
		}

		// Check cardinal neighbors
		up := surface.CellAt(x, y-1)
		down := surface.CellAt(x, y+1)
		left := surface.CellAt(x-1, y)
		right := surface.CellAt(x+1, y)

		mask := 0
		if up.BorderStyle == c.BorderStyle && colorsEqual(up.FG, c.FG) && colorsEqual(up.BG, c.BG) && p.connectsDown(up) {
			mask |= 8
		}
		if down.BorderStyle == c.BorderStyle && colorsEqual(down.FG, c.FG) && colorsEqual(down.BG, c.BG) && p.connectsUp(down) {
			mask |= 4
		}
		if left.BorderStyle == c.BorderStyle && colorsEqual(left.FG, c.FG) && colorsEqual(left.BG, c.BG) && p.connectsRight(left) {
			mask |= 2
		}
		if right.BorderStyle == c.BorderStyle && colorsEqual(right.FG, c.FG) && colorsEqual(right.BG, c.BG) && p.connectsLeft(right) {
			mask |= 1
		}

		newContent := p.getJunctionGlyph(c.BorderStyle, mask)
		if newContent != "" && newContent != c.Content {
			c.Content = newContent
			surface.Set(x, y, c)
		}
	}
}

func (p *PaintEngine) connectsUp(c Cell) bool {
	switch c.Content {
	case "│", "║", "┃", "|",
		"└", "╚", "┗", "╰",
		"┘", "╝", "┛", "╯",
		"┴", "╩", "┻",
		"├", "╠", "┣",
		"┤", "╣", "┫",
		"┼", "╬", "╋",
		"+":
		return true
	}
	return false
}

func (p *PaintEngine) connectsDown(c Cell) bool {
	switch c.Content {
	case "│", "║", "┃", "|",
		"┌", "╔", "┏", "╭",
		"┐", "╗", "┓", "╮",
		"┬", "╦", "┳",
		"├", "╠", "┣",
		"┤", "╣", "┫",
		"┼", "╬", "╋",
		"+":
		return true
	}
	return false
}

func (p *PaintEngine) connectsLeft(c Cell) bool {
	switch c.Content {
	case "─", "═", "━", "-",
		"┐", "╗", "┓", "╮",
		"┘", "╝", "┛", "╯",
		"┬", "╦", "┳",
		"┴", "╩", "┻",
		"┤", "╣", "┫",
		"┼", "╬", "╋",
		"+":
		return true
	}
	return false
}

func (p *PaintEngine) connectsRight(c Cell) bool {
	switch c.Content {
	case "─", "═", "━", "-",
		"┌", "╔", "┏", "╭",
		"└", "╚", "┗", "╰",
		"┬", "╦", "┳",
		"┴", "╩", "┻",
		"├", "╠", "┣",
		"┼", "╬", "╋",
		"+":
		return true
	}
	return false
}

func colorsEqual(c1, c2 color.Color) bool {
	if c1 == c2 {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}

	// Fast path for common case of RGBA colors.
	if rgba1, ok := c1.(color.RGBA); ok {
		if rgba2, ok := c2.(color.RGBA); ok {
			return rgba1 == rgba2
		}
	}

	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

var borderSingle = [16]string{
	0: "", 1: "─", 2: "─", 3: "─",
	4: "│", 5: "┌", 6: "┐", 7: "┬",
	8: "│", 9: "└", 10: "┘", 11: "┴",
	12: "│", 13: "├", 14: "┤", 15: "┼",
}

var borderDuble = [16]string{
	0: "", 1: "═", 2: "═", 3: "═",
	4: "║", 5: "╔", 6: "╗", 7: "╦",
	8: "║", 9: "╚", 10: "╝", 11: "╩",
	12: "║", 13: "╠", 14: "╣", 15: "╬",
}

var borderThick = [16]string{
	0: "", 1: "━", 2: "━", 3: "━",
	4: "┃", 5: "┏", 6: "┓", 7: "┳",
	8: "┃", 9: "┗", 10: "┛", 11: "┻",
	12: "┃", 13: "┣", 14: "┫", 15: "╋",
}

var borderASCII = [16]string{
	0: "", 1: "-", 2: "-", 3: "-",
	4: "|", 5: "+", 6: "+", 7: "+",
	8: "|", 9: "+", 10: "+", 11: "+",
	12: "|", 13: "+", 14: "+", 15: "+",
}

func (p *PaintEngine) getJunctionGlyph(style BorderStyle, mask int) string {
	switch style {
	case BorderDouble:
		return borderDuble[mask]
	case BorderThick:
		return borderThick[mask]
	case BorderAscii:
		return borderASCII[mask]
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
		return borderSingle[mask]
	}
}

func isScrollContainer(o style.Overflow) bool {
	// overflow:clip creates a clipping boundary and supports programmatic
	// scroll offsets (via ScrollTo/ScrollBy) even though it hides the scrollbar.
	// This allows elements like <input> to pan their clipped content.
	return o == style.OverflowScroll || o == style.OverflowAuto || o == style.OverflowHidden || o == style.OverflowClip
}

func (p *PaintEngine) fillRect(r layout.Rect, content string, fg, bg color.Color) {
	clip := p.clipStack[len(p.clipStack)-1]

	visibleRect := r.Intersect(clip)
	if visibleRect.Size.Width <= 0 || visibleRect.Size.Height <= 0 {
		return
	}

	for y := 0; y < visibleRect.Size.Height; y++ {
		for x := 0; x < visibleRect.Size.Width; x++ {
			p.setCell(visibleRect.Origin.X+x, visibleRect.Origin.Y+y, Cell{
				Content: content,
				Width:   1,
				FG:      fg,
				BG:      bg,
			})
		}
	}
}

func (p *PaintEngine) drawBorder(r layout.Rect, border style.Border, bg color.Color) {
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
			p.setCell(x+i, y, Cell{Content: glyphs.H, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
			p.setCell(x+i, y+height-1, Cell{Content: glyphs.H, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
			p.setCell(x, y+i, Cell{Content: glyphs.V, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
			p.setCell(x+width-1, y+i, Cell{Content: glyphs.V, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
		p.setCell(x, y, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
		p.setCell(x+width-1, y, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
		p.setCell(x, y+height-1, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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
		p.setCell(x+width-1, y+height-1, Cell{Content: glyph, Width: 1, FG: c, BG: bg, BorderStyle: bs})
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

	if rgba, ok := c.(color.RGBA); ok {
		return rgba.A == 0
	}

	_, _, _, a := c.RGBA()
	return a == 0
}
