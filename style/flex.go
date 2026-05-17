package style

// GapValue holds the inter-child gaps for both layout axes of a flex
// container.  Row is the block-axis gap (between lines); Column is the
// inline-axis gap (between items on the same line).
//
// Build via the [Gap] variadic constructor, not as a struct literal, so
// callers remain decoupled from the field layout.
type GapValue struct {
	Row    int
	Column int
}

// Gap returns a [GapValue].
//
//   - Gap(n)    → {Row: n, Column: n}
//   - Gap(r, c) → {Row: r, Column: c}
//
// Any other arity panics.
func Gap(values ...int) GapValue {
	switch len(values) {
	case 1:
		return GapValue{Row: values[0], Column: values[0]}
	case 2:
		return GapValue{Row: values[0], Column: values[1]}
	default:
		panic("style.Gap: expected 1 or 2 arguments")
	}
}

// FlexItemValue groups the grow/shrink/basis properties for a flex item.
//
// Build via the [Flex] variadic constructor, not as a struct literal.
type FlexItemValue struct {
	Grow   int
	Shrink int
	Basis  Dimension
}

// Flex returns a [FlexItemValue].
//
//   - Flex(g)           → {Grow: g, Shrink: 1, Basis: Auto}
//   - Flex(g, s)        → {Grow: g, Shrink: s, Basis: Auto}
//   - Flex(g, s, basis) → {Grow: g, Shrink: s, Basis: basis}
//
// grow and shrink are integers; basis is an optional [Dimension].
// Any other arity panics.
func Flex(grow int, rest ...any) FlexItemValue {
	item := FlexItemValue{Grow: grow, Shrink: 1, Basis: Auto}
	switch len(rest) {
	case 0:
		// grow only — defaults above apply
	case 1:
		item.Shrink = rest[0].(int)
	case 2:
		item.Shrink = rest[0].(int)
		item.Basis = rest[1].(Dimension)
	default:
		panic("style.Flex: expected 1, 2, or 3 arguments")
	}
	return item
}
