package wind

// Optional wraps a value of type T to distinguish an explicitly set value
// (including the zero value) from an unset option.
type Optional[T any] struct {
	value T
	set   bool
}

// Some returns an Optional with v explicitly set.
func Some[T any](v T) Optional[T] {
	return Optional[T]{value: v, set: true}
}

// None returns an unset Optional.
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// IsSet reports whether a value has been explicitly set.
func (o Optional[T]) IsSet() bool {
	return o.set
}

// Value returns the stored value, or the zero value of T if unset.
func (o Optional[T]) Value() T {
	return o.value
}

// Get returns the stored value and a boolean reporting whether it was set.
func (o Optional[T]) Get() (T, bool) {
	return o.value, o.set
}

// UnwrapOr returns the stored value if set, otherwise fallback.
func (o Optional[T]) UnwrapOr(fallback T) T {
	if o.set {
		return o.value
	}
	return fallback
}
