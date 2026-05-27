package geom

import "github.com/masterkeysrd/kite/style"

// Point is a position in terminal-cell coordinates.
type Point struct {
	X, Y int
}

// Size represents a two-dimensional extent (width × height) in terminal cells.
type Size struct {
	Width  int
	Height int
}

// Rect is an axis-aligned rectangle defined by an origin and a size, measured
// in terminal cells. All layout arithmetic uses integer cells (border-box).
type Rect struct {
	Origin Point
	Size   Size
}

// Contains reports whether p lies inside r.
// The left/top edges are inclusive; the right/bottom edges are exclusive.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.Origin.X && p.X < r.Origin.X+r.Size.Width &&
		p.Y >= r.Origin.Y && p.Y < r.Origin.Y+r.Size.Height
}

// Inset returns a new Rect shrunk on all four sides by e. Width and Height
// are clamped to zero if the insets exceed the available space.
func (r Rect) Inset(e style.EdgeValues[int]) Rect {
	return Rect{
		Origin: Point{
			X: r.Origin.X + e.Left,
			Y: r.Origin.Y + e.Top,
		},
		Size: Size{
			Width:  max(0, r.Size.Width-e.Left-e.Right),
			Height: max(0, r.Size.Height-e.Top-e.Bottom),
		},
	}
}

// Intersect returns the largest Rect contained in both r and other.
// If the rectangles do not overlap, a zero Rect is returned.
func (r Rect) Intersect(other Rect) Rect {
	x1 := max(r.Origin.X, other.Origin.X)
	y1 := max(r.Origin.Y, other.Origin.Y)
	x2 := min(r.Origin.X+r.Size.Width, other.Origin.X+other.Size.Width)
	y2 := min(r.Origin.Y+r.Size.Height, other.Origin.Y+other.Size.Height)
	if x2 <= x1 || y2 <= y1 {
		return Rect{}
	}
	return Rect{
		Origin: Point{X: x1, Y: y1},
		Size:   Size{Width: x2 - x1, Height: y2 - y1},
	}
}

// Overlaps reports whether r and other have a non-empty intersection.
func (r Rect) Overlaps(other Rect) bool {
	x1 := max(r.Origin.X, other.Origin.X)
	y1 := max(r.Origin.Y, other.Origin.Y)
	x2 := min(r.Origin.X+r.Size.Width, other.Origin.X+other.Size.Width)
	y2 := min(r.Origin.Y+r.Size.Height, other.Origin.Y+other.Size.Height)
	return x2 > x1 && y2 > y1
}

type Placement int

const (
	PlacementTop Placement = iota
	PlacementBottom
	PlacementLeft
	PlacementRight
)
