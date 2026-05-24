package render

import (
	"unicode/utf8"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
	"image/color"
	"reflect"
)

// SelectionRect represents a physical rectangle of selected content.
// This is a mirror of paint.SelectionRect to avoid circular dependencies.
type SelectionRect struct {
	Rect layout.Rect
	FG   color.Color
	BG   color.Color
}

// SelectionSource is an interface that provides access to selection ranges.
// This avoids a circular dependency on the dom package.
type SelectionSource interface {
	RangeCount() int
	GetRangeAt(index int) SelectionRange
}

// SelectionRange is an interface for a selection range.
type SelectionRange interface {
	StartContainerAny() any
	EndContainerAny() any
	StartOffset() int
	EndOffset() int
	IsCollapsed() bool
}

// NodeOrder tracks the pre-order traversal boundaries of a node's subtree.
type NodeOrder struct {
	First int
	Last  int
}

// ResolveSelection maps the active selection to physical screen rectangles.
func ResolveSelection(root *layout.Fragment, sel SelectionSource, nodeOrder map[any]NodeOrder) []SelectionRect {
	var rects []SelectionRect
	for i := 0; i < sel.RangeCount(); i++ {
		rng := sel.GetRangeAt(i)
		if rng == nil || rng.IsCollapsed() {
			continue
		}
		nodeOffsets := make(map[any]int)
		rects = append(rects, resolveRange(root, rng, nodeOrder, nodeOffsets)...)
	}
	return rects
}

func resolveRange(root *layout.Fragment, rng SelectionRange, nodeOrder map[any]NodeOrder, nodeOffsets map[any]int) []SelectionRect {
	var rects []SelectionRect
	walkFragments(root, layout.Point{}, layout.InfiniteRect(), rng, nodeOrder, nodeOffsets, &rects)
	return rects
}

func walkFragments(frag *layout.Fragment, origin layout.Point, clip layout.Rect, rng SelectionRange, nodeOrder map[any]NodeOrder, nodeOffsets map[any]int, rects *[]SelectionRect) {
	if frag == nil {
		return
	}

	// 1. Update clip if this fragment clips.
	newClip := clip
	scrollX, scrollY := 0, 0
	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		if s.OverflowX != style.OverflowVisible || s.OverflowY != style.OverflowVisible {
			bw := s.Border.Widths()
			pad := s.Padding
			inset := layout.Rect{
				Origin: layout.Point{X: origin.X + bw.Left + pad.Left, Y: origin.Y + bw.Top + pad.Top},
				Size: layout.Size{
					Width:  max(0, frag.Size.Width-bw.Left-bw.Right-pad.Left-pad.Right),
					Height: max(0, frag.Size.Height-bw.Top-bw.Bottom-pad.Top-pad.Bottom),
				},
			}
			newClip = clip.Intersect(inset)
		}

		// Scroll offset
		if ln := frag.Node.LogicalNode(); ln != nil {
			if el, ok := ln.(interface{ Scroll() (x, y int) }); ok {
				rawX, rawY := el.Scroll()
				maxSX, maxSY := layout.MaxScroll(frag)
				scrollX = max(0, min(rawX, maxSX))
				scrollY = max(0, min(rawY, maxSY))
			}
		}
	}

	// 2. Check if this fragment's node is within the range.
	// If the fragment doesn't have a node, it might be a synthesized fragment
	// (like a LineBox or a list marker) that should inherit the parent's
	// offset state if it contains text.
	var ln any
	if frag.Node != nil {
		ln = frag.Node.LogicalNode()
	}

	// For list markers and synthesized text that might be attached to an inline,
	// we check ParentNode.
	if ln == nil && frag.ParentNode != nil {
		ln = frag.ParentNode.LogicalNode()
	}

	if ln != nil {
		base := getBase(ln)
		// If it's a text fragment, we need to check if it's partially or fully selected.
		if len(frag.Text) > 0 {
			if isNodeInRange(ln, rng, nodeOrder) {
				// Calculate sub-rect for selected text.
				sr := calculateTextSelectionRect(frag, origin, newClip, ln, rng, nodeOffsets[base], nodeOrder)
				if sr != nil {
					*rects = append(*rects, *sr)
				}
			}
			// Always update offset to stay in sync with the node's logical text.
			// List markers (which carry ParentNode) should NOT advance the logical node's offset,
			// because they are synthesized content "outside" the logical buffer.
			if frag.Node != nil && frag.Node.LogicalNode() != nil {
				for _, c := range frag.Text {
					nodeOffsets[base] += utf8.RuneCount(c.Bytes)
				}
			}
		} else {
			// Atomic inlines or other elements.
			// If fully selected, add the whole border box (clipped).
			if isNodeFullySelected(ln, rng, nodeOrder) {
				rect := layout.Rect{Origin: origin, Size: frag.Size}.Intersect(newClip)
				if rect.Size.Width > 0 && rect.Size.Height > 0 {
					*rects = append(*rects, SelectionRect{
						Rect: rect,
						FG:   frag.Node.Style().SelectionForeground,
						BG:   frag.Node.Style().SelectionBackground,
					})
				}
			}
		}
	} else if len(frag.Text) > 0 {
		// Fragment with text but no logical node (e.g., list marker or synthesized content).
		// These are currently not selectable since they don't map to the DOM.
	}

	// 3. Recurse children.
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		walkFragments(childLink.Fragment, childOrigin, newClip, rng, nodeOrder, nodeOffsets, rects)
	}
}

func isNodeInRange(n any, rng SelectionRange, nodeOrder map[any]NodeOrder) bool {
	if nodeOrder == nil {
		return false
	}

	baseN := getBase(n)
	baseStart := getBase(rng.StartContainerAny())
	baseEnd := getBase(rng.EndContainerAny())

	orderN, okN := nodeOrder[baseN]
	orderStart, okStart := nodeOrder[baseStart]
	orderEnd, okEnd := nodeOrder[baseEnd]

	if !okN || !okStart || !okEnd {
		return false
	}

	// In range means either:
	// 1. N is the start or end container.
	// 2. N is between start and end in pre-order walk.
	// 3. N contains the start or end container.
	return orderStart.First <= orderN.Last && orderN.First <= orderEnd.Last
}

func isNodeFullySelected(n any, rng SelectionRange, nodeOrder map[any]NodeOrder) bool {
	if nodeOrder == nil {
		return false
	}

	baseN := getBase(n)
	baseStart := getBase(rng.StartContainerAny())
	baseEnd := getBase(rng.EndContainerAny())

	orderN, okN := nodeOrder[baseN]
	orderStart, okStart := nodeOrder[baseStart]
	orderEnd, okEnd := nodeOrder[baseEnd]

	if !okN || !okStart || !okEnd {
		return false
	}

	// Fully selected means the entire subtree of N is within the range [start, end].
	// Ancestor check using walk indices:
	// N is ancestor of X if orderN.First <= orderX.First && orderX.First <= orderN.Last.
	isAncestorOfStart := orderN.First <= orderStart.First && orderStart.First <= orderN.Last
	isAncestorOfEnd := orderN.First <= orderEnd.First && orderEnd.First <= orderN.Last

	if isAncestorOfStart || isAncestorOfEnd {
		return false
	}

	// If not an ancestor of either, it's fully selected if it's "between" them.
	return orderStart.First <= orderN.First && orderN.Last <= orderEnd.Last
}

func getBase(n any) any {
	if n == nil {
		return nil
	}
	v := reflect.ValueOf(n)
	for {
		method := v.MethodByName("Unwrap")
		if !method.IsValid() {
			break
		}
		// Ensure it's a method with no arguments and at least one return value.
		if method.Type().NumIn() != 0 || method.Type().NumOut() == 0 {
			break
		}
		results := method.Call(nil)
		if len(results) == 0 {
			break
		}
		next := results[0].Interface()
		if next == nil || next == v.Interface() {
			break
		}
		v = reflect.ValueOf(next)
	}
	return v.Interface()
}

func calculateTextSelectionRect(frag *layout.Fragment, origin layout.Point, clip layout.Rect, node any, rng SelectionRange, currentRuneOffset int, nodeOrder map[any]NodeOrder) *SelectionRect {
	baseN := getBase(node)
	baseStart := getBase(rng.StartContainerAny())
	baseEnd := getBase(rng.EndContainerAny())

	startSel := 0
	if baseN == baseStart {
		startSel = rng.StartOffset()
	}

	endSel := 1000000000
	if baseN == baseEnd {
		endSel = rng.EndOffset()
	}

	// If this is a synthesized fragment (associated with an element but not carrying its own text in the buffer),
	// and the element is strictly between start and end, we select the whole thing.
	isSynthesized := frag.Node == nil && frag.ParentNode != nil
	if isSynthesized {
		orderN, okN := nodeOrder[baseN]
		orderStart, okStart := nodeOrder[baseStart]
		orderEnd, okEnd := nodeOrder[baseEnd]

		if okN && okStart && okEnd {
			isAncestorOfStart := orderN.First <= orderStart.First && orderStart.First <= orderN.Last
			isAncestorOfEnd := orderN.First <= orderEnd.First && orderEnd.First <= orderN.Last

			if !isAncestorOfStart && !isAncestorOfEnd {
				// Node is between start and end.
				if orderStart.First <= orderN.First && orderN.Last <= orderEnd.Last {
					startSel = -1
					endSel = 1000000000
				}
			} else if isAncestorOfStart && !isAncestorOfEnd {
				// Node contains start but not end.
				// For synthesized marker/content, we select it if it's "after" the start offset.
				// Markers are usually index 0.
				if rng.StartOffset() == 0 {
					startSel = -1
				} else {
					return nil
				}
			} else if !isAncestorOfStart && isAncestorOfEnd {
				// Node contains end but not start.
				// Select if "before" the end offset.
				if rng.EndOffset() > 0 {
					endSel = 1000000000
				} else {
					return nil
				}
			} else {
				// Node contains both start and end.
				// This shouldn't happen for a marker unless it's very large,
				// but check offsets.
				if rng.StartOffset() == 0 && rng.EndOffset() > 0 {
					startSel = -1
					endSel = 1000000000
				} else {
					return nil
				}
			}
		}
	}

	firstSelectedX := -1
	lastSelectedX := -1

	x := 0
	runesSeen := 0
	for _, c := range frag.Text {
		cRunes := utf8.RuneCount(c.Bytes)
		cWidth := c.CellWidth

		clusterStart := currentRuneOffset + runesSeen
		clusterEnd := clusterStart + cRunes

		isSelected := true
		if clusterEnd <= startSel {
			isSelected = false
		}
		if clusterStart >= endSel {
			isSelected = false
		}

		if isSelected {
			if firstSelectedX == -1 {
				firstSelectedX = x
			}
			lastSelectedX = x + cWidth
		}

		x += cWidth
		runesSeen += cRunes
	}

	if firstSelectedX == -1 {
		return nil
	}

	resRect := layout.Rect{
		Origin: layout.Point{X: origin.X + firstSelectedX, Y: origin.Y},
		Size:   layout.Size{Width: lastSelectedX - firstSelectedX, Height: frag.Size.Height},
	}

	resRect = resRect.Intersect(clip)
	if resRect.Size.Width <= 0 || resRect.Size.Height <= 0 {
		return nil
	}

	// Use node's style if available, otherwise fallback to fragment's node style.
	var s *style.Computed
	if n, ok := node.(interface{ Style() *style.Computed }); ok {
		s = n.Style()
	} else if frag.Node != nil {
		s = frag.Node.Style()
	}

	if s == nil {
		return nil
	}

	return &SelectionRect{
		Rect: resRect,
		FG:   s.SelectionForeground,
		BG:   s.SelectionBackground,
	}
}
