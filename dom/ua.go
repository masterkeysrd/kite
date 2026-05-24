package dom

import "iter"

// LayoutChildren returns an iterator that yields the engine-visible children of
// n. For nodes that hold a UA shadow subtree (via AttachUARoot), the iterator
// yields the public child list followed by the children of the UA root.
//
// LayoutChildren is the authoritative walker for the engine's Sync, Style,
// Layout, and Paint phases. It must never be exposed to author code; public
// author-facing traversal (ChildNodes, FirstChild, LastChild, Children) must
// remain UA-invisible.
//
// Zero-allocation fast path: when n has no UA subtree the iterator degrades to
// a plain ChildNodes() walk with no additional overhead beyond the public
// iterator.
func LayoutChildren(n Node) iter.Seq[Node] {
	// Check whether n hosts a UA subtree by resolving through wrappers.
	// We must unwrap to reach the concrete *element if n is a wrapper type.
	var uaRoot Node
	for cur := n; cur != nil; cur = cur.Unwrap() {
		if e, ok := cur.(*element); ok {
			uaRoot = e.uaRoot
			break
		}
	}

	if uaRoot == nil {
		// Fast path: no UA subtree — behave identically to ChildNodes().
		return n.ChildNodes()
	}

	// Slow path: union of public children and UA root's children.
	return func(yield func(Node) bool) {
		// Public children first.
		for child := range n.ChildNodes() {
			if !yield(child) {
				return
			}
		}
		// UA root's children second.
		for child := range uaRoot.ChildNodes() {
			if !yield(child) {
				return
			}
		}
	}
}

// IsUANode reports whether n is part of a UA shadow subtree. The check is O(1)
// because AttachUARoot stamps every node in the subtree with an inUASubtree
// flag at construction time.
func IsUANode(n Node) bool {
	if n == nil {
		return false
	}
	if b := asBase(n); b != nil {
		return b.inUASubtree
	}
	return false
}

// UARoot returns the UA shadow subtree root attached to el, or nil if el does
// not have one. This is an engine-internal accessor used by the Sync phase.
// It must not be called from author code.
func UARoot(el Element) Node {
	for cur := Node(el); cur != nil; cur = cur.Unwrap() {
		if e, ok := cur.(*element); ok {
			return e.uaRoot
		}
	}
	return nil
}
