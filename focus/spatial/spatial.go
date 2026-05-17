package spatial

import (
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
)

// Direction represents the four cardinal directions for spatial navigation.
type Direction uint8

const (
	// DirectionUp navigates toward the top of the screen.
	DirectionUp Direction = iota
	// DirectionDown navigates toward the bottom of the screen.
	DirectionDown
	// DirectionLeft navigates toward the left side of the screen.
	DirectionLeft
	// DirectionRight navigates toward the right side of the screen.
	DirectionRight
)

// offAxisPenalty is the multiplier applied to the off-axis distance when
// scoring candidates. A higher value penalizes diagonal candidates more
// aggressively. See ADR-0011.
const offAxisPenalty = 2.0

// Navigate moves focus in dir within the active scope of m.
// Returns true if focus moved to a new element, false if no suitable
// candidate was found (focus is unchanged).
//
// On success, focus is set with focus.ReasonKeyboard.
func Navigate(m *focus.Manager, dir Direction) bool {
	scope := m.ActiveScope()
	if scope == nil {
		return false
	}

	// Determine the anchor: current focus, or fall back to autofocus /
	// first focusable in DOM order.
	current := m.Current()
	if current == nil {
		if scope.Autofocus != nil && focus.IsFocusable(scope.Autofocus, scope) {
			current = scope.Autofocus
		} else {
			current = firstFocusable(scope)
		}
	}
	if current == nil {
		return false
	}

	best := bestCandidate(scope, current, dir)
	if best == nil {
		return false
	}
	return m.Focus(best, focus.ReasonKeyboard)
}

// Candidates returns the focusable nodes in dir from the current focus,
// ranked by suitability (lowest score first). It is exposed for advanced use
// such as showing a navigation preview. Most callers want Navigate.
func Candidates(m *focus.Manager, dir Direction) []render.Object {
	scope := m.ActiveScope()
	if scope == nil {
		return nil
	}

	current := m.Current()
	if current == nil {
		if scope.Autofocus != nil && focus.IsFocusable(scope.Autofocus, scope) {
			current = scope.Autofocus
		} else {
			current = firstFocusable(scope)
		}
	}
	if current == nil {
		return nil
	}

	return rankedCandidates(scope, current, dir)
}

// --- helpers -----------------------------------------------------------------

// firstFocusable returns the first focusable object in DOM order within scope,
// or nil if there are none.
func firstFocusable(scope *focus.Scope) render.Object {
	if scope == nil || scope.Root == nil {
		return nil
	}
	return firstFocusableInSubtree(scope.Root, scope)
}

// firstFocusableInSubtree walks the subtree rooted at root and returns the
// first focusable node in DOM pre-order, or nil.
func firstFocusableInSubtree(root render.Object, scope *focus.Scope) render.Object {
	for n := root; n != nil; n = nextPreOrder(n, root) {
		if focus.IsFocusable(n, scope) {
			return n
		}
	}
	return nil
}

// bestCandidate returns the best (lowest-score) focusable candidate in dir
// from current within scope, or nil if none qualify.
//
// This function is on the hot path of Navigate and must not allocate.
func bestCandidate(scope *focus.Scope, current render.Object, dir Direction) render.Object {
	if scope == nil || scope.Root == nil {
		return nil
	}

	curBounds, ok := layout.AbsoluteBounds(scope.Root.Fragment(), current)
	if !ok {
		return nil
	}

	var bestNode render.Object
	const maxScore = 1<<53 - 1 // large sentinel; avoids math.MaxFloat64 import
	bestScore := float64(maxScore)

	for n := scope.Root; n != nil; n = nextPreOrder(n, scope.Root) {
		if n == current {
			continue
		}
		if !focus.IsFocusable(n, scope) {
			continue
		}
		nb, found := layout.AbsoluteBounds(scope.Root.Fragment(), n)
		if !found {
			continue
		}
		if !inHalfPlane(curBounds, nb, dir) {
			continue
		}
		s := score(curBounds, nb, dir)
		if s < bestScore {
			bestScore = s
			bestNode = n
		}
	}

	return bestNode
}

// rankedCandidates returns all focusable candidates in dir from current,
// sorted ascending by score. Ties preserve DOM order.
func rankedCandidates(scope *focus.Scope, current render.Object, dir Direction) []render.Object {
	if scope == nil || scope.Root == nil {
		return nil
	}

	curBounds, ok := layout.AbsoluteBounds(scope.Root.Fragment(), current)
	if !ok {
		return nil
	}

	type entry struct {
		node  render.Object
		score float64
	}
	var entries []entry

	for n := scope.Root; n != nil; n = nextPreOrder(n, scope.Root) {
		if n == current {
			continue
		}
		if !focus.IsFocusable(n, scope) {
			continue
		}
		nb, found := layout.AbsoluteBounds(scope.Root.Fragment(), n)
		if !found {
			continue
		}
		if !inHalfPlane(curBounds, nb, dir) {
			continue
		}
		entries = append(entries, entry{node: n, score: score(curBounds, nb, dir)})
	}

	// Stable insertion sort — the input is already in DOM order, so equal
	// scores preserve DOM order without allocating a sort.Interface.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].score < entries[j-1].score; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}

	out := make([]render.Object, len(entries))
	for i, e := range entries {
		out[i] = e.node
	}
	return out
}

// nextPreOrder returns the next node in pre-order DFS traversal within the
// subtree rooted at root. Returns nil when the traversal is complete.
//
// This iterative approach avoids the allocation overhead of recursive closures.
func nextPreOrder(n, root render.Object) render.Object {
	// Descend into first child if present.
	if c := n.FirstChild(); c != nil {
		return c
	}
	// Otherwise move to next sibling, backtracking up the tree.
	for n != root {
		if s := n.NextSibling(); s != nil {
			return s
		}
		n = n.Parent()
		if n == nil {
			break
		}
	}
	return nil
}

// inHalfPlane reports whether candidate cb lies in the directional half-plane
// from the current node's bounding box curB.
//
// The half-plane test uses the near edges:
//   - Up:    candidate's bottom edge (cb.MaxY) < curB's top edge (curB.MinY)
//   - Down:  candidate's top edge (cb.MinY) > curB's bottom edge (curB.MaxY)
//   - Left:  candidate's right edge (cb.MaxX) < curB's left edge (curB.MinX)
//   - Right: candidate's left edge (cb.MinX) > curB's right edge (curB.MaxX)
func inHalfPlane(curB, cb layout.Rect, dir Direction) bool {
	switch dir {
	case DirectionUp:
		return rectMaxY(cb) <= rectMinY(curB)
	case DirectionDown:
		return rectMinY(cb) >= rectMaxY(curB)
	case DirectionLeft:
		return rectMaxX(cb) <= rectMinX(curB)
	case DirectionRight:
		return rectMinX(cb) >= rectMaxX(curB)
	default:
		return false
	}
}

// score computes the navigation score for candidate cb relative to current
// curB in dir. Lower scores win.
//
//	score = primaryDistance + offAxisPenalty * offAxisDistance
//
// Both distances are measured between the nearest edges of the bounding boxes.
func score(curB, cb layout.Rect, dir Direction) float64 {
	var primary, offAxis float64
	switch dir {
	case DirectionUp:
		// Primary: vertical distance (curB top → cb bottom)
		primary = float64(rectMinY(curB) - rectMaxY(cb))
		// Off-axis: horizontal overlap gap, or 0 if they overlap.
		offAxis = float64(horizontalGap(curB, cb))
	case DirectionDown:
		// Primary: vertical distance (cb top → curB bottom)
		primary = float64(rectMinY(cb) - rectMaxY(curB))
		offAxis = float64(horizontalGap(curB, cb))
	case DirectionLeft:
		// Primary: horizontal distance (curB left → cb right)
		primary = float64(rectMinX(curB) - rectMaxX(cb))
		offAxis = float64(verticalGap(curB, cb))
	case DirectionRight:
		// Primary: horizontal distance (cb left → curB right)
		primary = float64(rectMinX(cb) - rectMaxX(curB))
		offAxis = float64(verticalGap(curB, cb))
	}
	if primary < 0 {
		primary = 0
	}
	if offAxis < 0 {
		offAxis = 0
	}
	return primary + offAxisPenalty*offAxis
}

// horizontalGap returns the off-axis (horizontal) gap between two rects.
// Returns 0 if they overlap horizontally.
func horizontalGap(a, b layout.Rect) int {
	// Overlap region on X axis: max(minX) to min(maxX)
	overlapLeft := max(rectMinX(a), rectMinX(b))
	overlapRight := min(rectMaxX(a), rectMaxX(b))
	if overlapRight > overlapLeft {
		return 0
	}
	// No overlap: gap is the distance between the nearest horizontal edges.
	return overlapLeft - overlapRight
}

// verticalGap returns the off-axis (vertical) gap between two rects.
// Returns 0 if they overlap vertically.
func verticalGap(a, b layout.Rect) int {
	overlapTop := max(rectMinY(a), rectMinY(b))
	overlapBot := min(rectMaxY(a), rectMaxY(b))
	if overlapBot > overlapTop {
		return 0
	}
	return overlapTop - overlapBot
}

// --- Rect edge helpers -------------------------------------------------------

func rectMinX(r layout.Rect) int { return r.Origin.X }
func rectMaxX(r layout.Rect) int { return r.Origin.X + r.Size.Width }
func rectMinY(r layout.Rect) int { return r.Origin.Y }
func rectMaxY(r layout.Rect) int { return r.Origin.Y + r.Size.Height }
