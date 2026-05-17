package dom

// Outer returns the outer wrapper for el when one has been adopted; otherwise it
// returns el unchanged. This is useful when code needs the user-visible wrapper
// rather than the raw inner *element.
func Outer(n Node) Node {
	if n == nil {
		return nil
	}
	if a, ok := any(n).(adopter); ok {
		if outer := a.outerSelf(); outer != nil {
			return outer
		}
	}
	return n
}

// AdoptOuterTree runs an adoption walk over the subtree rooted at n.
// This is typically called by the engine during initial mount.
func AdoptOuterTree(n Node) {
	// Implementation would go here. For now, it's a stub.
}
