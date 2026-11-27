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
	for range count {
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

// Span creates a GridPlacement that spans n tracks.
func Span(n int) GridPlacement {
	return GridPlacement{Span: n}
}

// GridLine creates a GridPlacement anchored at the 1-based track line start.
func GridLine(start int) GridPlacement {
	return GridPlacement{Start: start}
}

// GridArea creates a GridPlacement spanning from 1-based line start to end.
func GridArea(start, end int) GridPlacement {
	return GridPlacement{Start: start, End: end}
}
