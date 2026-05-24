package render

import (
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

// ResolveSelection maps the active selection to physical screen rectangles.
func ResolveSelection(root *layout.Fragment, sel SelectionSource, nodeOrder map[any]int) []SelectionRect {
	var rects []SelectionRect
	for i := 0; i < sel.RangeCount(); i++ {
		rng := sel.GetRangeAt(i)
		if rng == nil || rng.IsCollapsed() {
			continue
		}
		rects = append(rects, resolveRange(root, rng, nodeOrder)...)
	}
	return rects
}

func resolveRange(root *layout.Fragment, rng SelectionRange, nodeOrder map[any]int) []SelectionRect {
	var rects []SelectionRect
	walkFragments(root, layout.Point{}, layout.InfiniteRect(), rng, nodeOrder, &rects)
	return rects
}

func walkFragments(frag *layout.Fragment, origin layout.Point, clip layout.Rect, rng SelectionRange, nodeOrder map[any]int, rects *[]SelectionRect) {
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
	if frag.Node != nil {
		ln := frag.Node.LogicalNode()
		if ln != nil {
			// If it's a text fragment, we need to check if it's partially or fully selected.
			if len(frag.Text) > 0 {
				if isNodeInRange(ln, rng, nodeOrder) {
					// Calculate sub-rect for selected text.
					// For now, we return the full fragment rectangle if it's within the range.
					sr := calculateTextSelectionRect(frag, origin, newClip, ln, rng)
					if sr != nil {
						*rects = append(*rects, *sr)
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
		}
	}

	// 3. Recurse children.
	for _, childLink := range frag.Children {
		childOrigin := layout.Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		// If child fragment doesn't have a Node (like LineBox), use parent's Node
		// for selection check if needed, OR just recurse.
		// Usually text fragments ARE the ones with nodes.
		walkFragments(childLink.Fragment, childOrigin, newClip, rng, nodeOrder, rects)
	}
}

func isNodeInRange(n any, rng SelectionRange, nodeOrder map[any]int) bool {
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

	return orderStart <= orderN && orderN <= orderEnd
}

func isNodeFullySelected(n any, rng SelectionRange, nodeOrder map[any]int) bool {
	baseN := getBase(n)
	baseStart := getBase(rng.StartContainerAny())
	baseEnd := getBase(rng.EndContainerAny())

	if baseN == baseStart || baseN == baseEnd {
		return false
	}

	return isNodeInRange(n, rng, nodeOrder)
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
		next := results[0].Interface()
		if next == nil || next == v.Interface() {
			break
		}
		v = reflect.ValueOf(next)
	}
	return v.Interface()
}

func calculateTextSelectionRect(frag *layout.Fragment, origin layout.Point, clip layout.Rect, node any, rng SelectionRange) *SelectionRect {
	rect := layout.Rect{Origin: origin, Size: frag.Size}.Intersect(clip)
	if rect.Size.Width <= 0 || rect.Size.Height <= 0 {
		return nil
	}

	return &SelectionRect{
		Rect: rect,
		FG:   frag.Node.Style().SelectionForeground,
		BG:   frag.Node.Style().SelectionBackground,
	}
}
