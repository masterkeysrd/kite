package render

// DirtyFlag is a bitmask that describes what work a render object (or its
// subtree) needs on the next frame.
//
// All mutations to flags must happen on the main goroutine; this package
// provides no atomic operations.
type DirtyFlag uint16

const (
	// Clean is the zero value; no work is pending.
	Clean DirtyFlag = 0

	// --- Self flags ----------------------------------------------------------

	// DirtyStyle means the computed style is stale and must be re-resolved
	// before layout.
	DirtyStyle DirtyFlag = 1 << 0

	// DirtyLayout means the box geometry is stale and must be re-measured.
	DirtyLayout DirtyFlag = 1 << 1

	// DirtyPaint means visible output is stale and the object must be redrawn.
	DirtyPaint DirtyFlag = 1 << 2

	// DirtyScroll means the scroll offset changed. Style and layout caches are
	// preserved; only the paint walk is needed. The paint phase clears
	// DirtyScroll together with DirtyPaint.
	DirtyScroll DirtyFlag = 1 << 3

	// --- Descendant relay flags ----------------------------------------------

	// ChildNeedsStyle means some descendant has DirtyStyle set.
	ChildNeedsStyle DirtyFlag = 1 << 5

	// ChildNeedsLayout means some descendant has DirtyLayout.
	ChildNeedsLayout DirtyFlag = 1 << 6

	// ChildNeedsPaint means some descendant has DirtyPaint or DirtyScroll.
	ChildNeedsPaint DirtyFlag = 1 << 7
)
