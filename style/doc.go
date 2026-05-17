// Package style defines the Style and Computed value types and the Resolver
// that performs inheritance, default application, and computed-value
// resolution for kitex (kite v2).
//
// Style fields are Optional[T] to distinguish unset from zero. Computed
// mirrors Style without Optional and is what the render layer reads.
// The package has no dependencies on other kitex packages.
package style
