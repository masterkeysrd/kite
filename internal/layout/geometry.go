package layout

import geometry "github.com/masterkeysrd/kite/geom"

// Constraints defines the minimum and maximum dimensions allowed for a box.
type Constraints struct {
	Min, Max geometry.Size
}

// MeasureResult is the output of a node's Measure method.
type MeasureResult struct {
	Size geometry.Size
}

// InfiniteRect returns a rectangle that covers the entire signed integer coordinate space.
func InfiniteRect() geometry.Rect {
	const inf = 1e9 // Large enough for terminal grids
	return geometry.Rect{
		Origin: geometry.Point{X: -inf, Y: -inf},
		Size:   geometry.Size{Width: 2 * inf, Height: 2 * inf},
	}
}
