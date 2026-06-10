// Package style defines the Style and Computed value types used across the
// Kite engine. Style resolution is performed by the internal/styler package.
//
// # Style vs Computed
//
// Style fields are Optional[T] to distinguish unset from zero. Computed
// mirrors Style without Optional and is what the render layer reads.
// The package has no dependencies on other kitex packages.
//
// Style provides fluent shorthands for common property combinations.
//
// Methods like Gap, Padding, Margin, and Flex use variadic arguments to
// simplify common layouts.
//
//   - Gap(int...) accepts 1 or 2 integers.
//   - Padding(int...) and Margin(int...) accept 1, 2, or 4 integers (CSS shorthand).
//   - Flex(int, any...) accepts mandatory grow (int), optional shrink (int) and basis (Dimension).
//   - GridTemplateColumns(Dimension...) and GridTemplateRows(Dimension...) accept variadic track sizes.
//   - Border(any...) accepts bool, BorderStyle, color.Color, or BorderGlyphs.
//
// Example:
//
//	style.S().
//	    Padding(1, 2).   // 1 row vertical, 2 cols horizontal
//	    Gap(1).          // 1 cell between all flex children
//	    Flex(1).         // flex-grow: 1, shrink: 1, basis: auto
//	    GridTemplateColumns(style.Cells(10), style.Fr(1)).
//	    Border(true, style.BorderDouble) // All edges double border
package style
