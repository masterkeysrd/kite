package paint

import (
	"image/color"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// FrameBuffer is the concrete Surface implementation used by the paint engine.
// It holds a 2-D grid of Cells indexed by (x, y), a monotonically-increasing
// frame version counter, and a dirty rectangle that grows as cells are written.
//
// FrameBuffer is not safe for concurrent use.
type FrameBuffer struct {
	cells      []Cell     // flat array: cells[y*width + x]
	bounds     geom.Rect  // absolute position and size of this buffer
	version    uint64     // incremented by each Flush / BeginFrame call
	dirty      *geom.Rect // nil when no cell has been written this frame
	blendCache map[blendKey]color.Color
}

// NewFrameBuffer creates a FrameBuffer positioned at origin (ox, oy) with the
// given width and height. All cells are initialised to zero values.
func NewFrameBuffer(ox, oy, width, height int) *FrameBuffer {
	return &FrameBuffer{
		cells: make([]Cell, width*height),
		bounds: geom.Rect{
			Origin: geom.Point{X: ox, Y: oy},
			Size:   geom.Size{Width: width, Height: height},
		},
		blendCache: make(map[blendKey]color.Color),
	}
}

// Reset clears the buffer content and resets the dirty rect.
func (fb *FrameBuffer) Reset() {
	for i := range fb.cells {
		fb.cells[i] = Cell{}
	}
	fb.dirty = nil
	if len(fb.blendCache) > 1024 {
		clear(fb.blendCache)
	}
}

// Set writes cell c into position (x, y). The coordinates are absolute
// (same space as Bounds). Calls outside the buffer's bounds are silently
// ignored.
//
// If c.BG is color.Transparent, the existing background color in the cell is
// preserved.
func (fb *FrameBuffer) Set(x, y int, c Cell) {
	if !fb.bounds.Contains(geom.Point{X: x, Y: y}) {
		return
	}
	lx := x - fb.bounds.Origin.X
	ly := y - fb.bounds.Origin.Y
	idx := ly*fb.bounds.Size.Width + lx

	if c.Bg == color.Transparent {
		c.Bg = fb.cells[idx].Bg
	} else if c.Bg != nil {
		c.Bg = fb.blendColors(fb.cells[idx].Bg, c.Bg)
	}

	fb.cells[idx] = c
	fb.growDirty(x, y)
}

// CellAt returns the cell at absolute position (x, y). If the position is out of
// bounds an empty Cell is returned.
func (fb *FrameBuffer) CellAt(x, y int) Cell {
	if !fb.bounds.Contains(geom.Point{X: x, Y: y}) {
		return Cell{}
	}
	lx := x - fb.bounds.Origin.X
	ly := y - fb.bounds.Origin.Y
	return fb.cells[ly*fb.bounds.Size.Width+lx]
}

// Bounds returns the full drawable area of the buffer.
func (fb *FrameBuffer) Bounds() geom.Rect { return fb.bounds }

// Clip returns a clippedSurface that restricts writes to the intersection of
// fb's bounds and r. The returned Surface still accepts absolute coordinates.
func (fb *FrameBuffer) Clip(r geom.Rect) Surface {
	clipped := fb.bounds.Intersect(r)
	return &clippedSurface{fb: fb, bounds: clipped}
}

// Version returns the current frame version counter.
func (fb *FrameBuffer) Version() uint64 { return fb.version }

// BumpVersion increments the frame version, signalling the start of a new
// frame. The dirty rect is also reset.
func (fb *FrameBuffer) BumpVersion() {
	fb.version++
	fb.dirty = nil
}

// DirtyRect returns the bounding rectangle of all cells written since the last
// BumpVersion call. If no cell has been written, ok is false.
func (fb *FrameBuffer) DirtyRect() (r geom.Rect, ok bool) {
	if fb.dirty == nil {
		return geom.Rect{}, false
	}
	return *fb.dirty, true
}

// growDirty expands the dirty rect to include the absolute position (x, y).
func (fb *FrameBuffer) growDirty(x, y int) {
	if fb.dirty == nil {
		fb.dirty = &geom.Rect{
			Origin: geom.Point{X: x, Y: y},
			Size:   geom.Size{Width: 1, Height: 1},
		}
		return
	}
	d := fb.dirty
	x2 := max(d.Origin.X+d.Size.Width, x+1)
	y2 := max(d.Origin.Y+d.Size.Height, y+1)
	d.Origin.X = min(d.Origin.X, x)
	d.Origin.Y = min(d.Origin.Y, y)
	d.Size.Width = x2 - d.Origin.X
	d.Size.Height = y2 - d.Origin.Y
}

// clippedSurface wraps a FrameBuffer and restricts Set calls to a sub-rect.
type clippedSurface struct {
	fb     *FrameBuffer
	bounds geom.Rect
}

// Set writes cell c at (x, y) only if the position lies within the clip rect.
func (cs *clippedSurface) Set(x, y int, c Cell) {
	if !cs.bounds.Contains(geom.Point{X: x, Y: y}) {
		return
	}
	cs.fb.Set(x, y, c)
}

// Bounds returns the clipped drawable area.
func (cs *clippedSurface) Bounds() geom.Rect { return cs.bounds }

// Clip returns a further-clipped surface whose bounds is the intersection of
// cs's bounds and r.
func (cs *clippedSurface) Clip(r geom.Rect) Surface {
	return &clippedSurface{fb: cs.fb, bounds: cs.bounds.Intersect(r)}
}

// CellAt returns the cell at absolute position (x, y) if it lies within the
// clip rect and the global framebuffer bounds.
func (cs *clippedSurface) CellAt(x, y int) Cell {
	if !cs.bounds.Contains(geom.Point{X: x, Y: y}) {
		return Cell{}
	}
	return cs.fb.CellAt(x, y)
}

type blendKey struct {
	parent color.RGBA
	child  color.RGBA
}

func toRGBA(c color.Color) color.RGBA {
	if c == nil || isTransparent(c) {
		return color.RGBA{R: 0, G: 0, B: 0, A: 0}
	}
	if style.IsTerminalDefault(c) {
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	if rgba, ok := c.(color.RGBA); ok {
		return rgba
	}
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

func (fb *FrameBuffer) blendColors(parent, child color.Color) color.Color {
	if child == nil || isTransparent(child) {
		return parent
	}
	if style.IsTerminalDefault(child) {
		return style.TerminalDefault
	}
	if parent == nil {
		parent = color.Transparent
	}

	pRGBA := toRGBA(parent)
	cRGBA := toRGBA(child)

	if pRGBA.A == 0 {
		pRGBA = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	if cRGBA.A == 255 {
		return child
	}

	if fb.blendCache == nil {
		fb.blendCache = make(map[blendKey]color.Color)
	}

	key := blendKey{
		parent: pRGBA,
		child:  cRGBA,
	}

	if cached, ok := fb.blendCache[key]; ok {
		return cached
	}

	a := uint16(cRGBA.A)
	invA := 255 - a

	r := uint8((uint16(cRGBA.R)*a + uint16(pRGBA.R)*invA) / 255)
	g := uint8((uint16(cRGBA.G)*a + uint16(pRGBA.G)*invA) / 255)
	b := uint8((uint16(cRGBA.B)*a + uint16(pRGBA.B)*invA) / 255)

	blended := color.RGBA{R: r, G: g, B: b, A: 255}
	fb.blendCache[key] = blended
	return blended
}
