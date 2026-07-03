// Package geom defines the geometric value types used throughout the Kite
// layout engine. All coordinates and extents are expressed in terminal cells
// (integer grid positions).
//
// # Types
//
//   - Point: a position (X, Y) in terminal-cell coordinates.
//   - Size: a two-dimensional extent (width × height).
//   - Rect: an axis-aligned rectangle defined by an origin and a size.
//     Methods include Contains, Inset, Intersect, and Overlaps.
//
// # Edges
//
// Edges represents four-side inset values used by layout to compute padding,
// margin, and border boxes.
//
// # Placement
//
// Placement enumerates the four cardinal positions relative to an anchor,
// used by the overlay system to decide flip direction.
//
// Example:
//
//	rect := geom.Rect{
//	    Origin: geom.Point{X: 0, Y: 0},
//	    Size:   geom.Size{Width: 80, Height: 24},
//	}
//	content := rect.Inset(geom.Edges{Top: 1, Bottom: 1, Left: 2, Right: 2})
package geom
