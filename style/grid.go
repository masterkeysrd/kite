package style

// GridTrackSize is an alias to Dimension, allowing grid definitions
// to seamlessly use existing dimension types and helpers (e.g. Cells, Auto).
type GridTrackSize = Dimension

// Repeat generates a slice of GridTrackSize by repeating the provided sizes
// the specified number of times. It flattens the result into a single slice.
func Repeat(count int, sizes ...GridTrackSize) []GridTrackSize {
	if count <= 0 || len(sizes) == 0 {
		return nil
	}

	result := make([]GridTrackSize, 0, count*len(sizes))
	for i := 0; i < count; i++ {
		result = append(result, sizes...)
	}
	return result
}

// GridPlacement represents a column or row placement in a CSS Grid.
// It can specify a start line, an end line, or a span.
type GridPlacement struct {
	Span  int
	Start int
	End   int
}
