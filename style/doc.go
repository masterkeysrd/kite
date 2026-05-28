// Package style defines the Style and Computed value types used across the
// Kite engine. Style resolution is performed by the internal/styler package.
//
// # Style vs Computed
//
// Style fields are Optional[T] to distinguish unset from zero. Computed
// mirrors Style without Optional and is what the render layer reads.
// The package has no dependencies on other kitex packages.
//
// Style provides fluent shorthands for common property combinations. For
// example, Style.Overflow(v) sets both OverflowX and OverflowY to the same
// value.
package style
