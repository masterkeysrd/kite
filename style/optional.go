package style

// Optional wraps a value of type T to distinguish "unset" from the zero
// value. Every field of [Style] is Optional so that callers can compose styles
// without clobbering fields they did not intend to set.
type Optional[T any] struct {
	value T
	set   bool
}

// Some returns an Optional with v set.
func Some[T any](v T) Optional[T] { return Optional[T]{value: v, set: true} }

// None returns an unset Optional.
func None[T any]() Optional[T] { return Optional[T]{} }

// Set assigns v and marks the Optional as set.
func (o *Optional[T]) Set(v T) {
	o.value = v
	o.set = true
}

// Unset clears the value and marks the Optional as unset.
func (o *Optional[T]) Unset() {
	var zero T
	o.value = zero
	o.set = false
}

// Value returns the stored value. If the Optional is unset the zero value of T
// is returned.
func (o Optional[T]) Value() T { return o.value }

// UnwrapOr returns the stored value if it is set, otherwise it returns fallback.
func (o Optional[T]) UnwrapOr(fallback T) T {
	if o.set {
		return o.value
	}
	return fallback
}

// IsSet reports whether a value has been explicitly set.
func (o Optional[T]) IsSet() bool { return o.set }

// Merge returns other if it is set, otherwise it returns o. This is the
// building block used by [Style.Merge] to combine two styles field-by-field.
func (o Optional[T]) Merge(other Optional[T]) Optional[T] {
	if other.set {
		return other
	}
	return o
}
