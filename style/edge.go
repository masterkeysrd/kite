package style

// EdgeValues holds independent values for the four sides of a box: top,
// right, bottom, and left. It is used for padding, margin, and per-side border
// properties.
type EdgeValues[T any] struct {
	Top, Right, Bottom, Left T
}

// EdgeAll returns an [EdgeValues] where all four sides are set to v.
func EdgeAll[T any](v T) EdgeValues[T] {
	return EdgeValues[T]{Top: v, Right: v, Bottom: v, Left: v}
}

// Edges is a variadic constructor for [EdgeValues[T]] following CSS shorthand
// conventions:
//
//   - 1 arg  – Edges(all)             → all four sides equal
//   - 2 args – Edges(vertical, horizontal) → top/bottom = v, left/right = h
//   - 4 args – Edges(top, right, bottom, left)
//
// Any other number of arguments returns the zero [EdgeValues].
func Edges[T any](vals ...T) EdgeValues[T] {
	switch len(vals) {
	case 1:
		v := vals[0]
		return EdgeValues[T]{Top: v, Right: v, Bottom: v, Left: v}
	case 2:
		v, h := vals[0], vals[1]
		return EdgeValues[T]{Top: v, Right: h, Bottom: v, Left: h}
	case 4:
		return EdgeValues[T]{Top: vals[0], Right: vals[1], Bottom: vals[2], Left: vals[3]}
	default:
		return EdgeValues[T]{}
	}
}
