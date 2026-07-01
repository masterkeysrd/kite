package collections

// DeleteAt removes the element at index i from the slice while maintaining order,
// and zeroing out the vacated slot at the end to prevent memory/reference leaks.
func DeleteAt[T any](slice []T, i int) []T {
	if i < 0 || i >= len(slice) {
		return slice
	}
	copy(slice[i:], slice[i+1:])
	var zero T
	slice[len(slice)-1] = zero
	return slice[:len(slice)-1]
}
