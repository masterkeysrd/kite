package render

import (
	"unicode/utf8"

	"image/color"
	"reflect"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

// SelectionRect represents a physical rectangle of selected content.
type SelectionRect struct {
	Rect geom.Rect
	Fg   color.Color
	Bg   color.Color
}

// SelectionSource is an interface that provides access to selection ranges.
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

	state := &walkState{
		rng:         rng,
		nodeOrder:   nodeOrder,
		nodeOffsets: nodeOffsets,
		rects:       &rects,
	}

	walkFragments(root, geom.Point{}, layout.InfiniteRect(), state)
	return rects
}

type walkState struct {
	rng         SelectionRange
	nodeOrder   map[any]NodeOrder
	nodeOffsets map[any]int
	rects       *[]SelectionRect

	started bool
	stopped bool
}

func walkFragments(frag *layout.Fragment, origin geom.Point, clip geom.Rect, s *walkState) {
	if frag == nil || s.stopped {
		return
	}

	// 1. Determine logical node.
	var ln any
	if frag.Node != nil {
		ln = frag.Node.LogicalNode()
	} else if frag.ParentNode != nil {
		ln = frag.ParentNode.LogicalNode()
	}

	// 2. Handle boundary markers and text accumulation.
	if ln != nil {
		base := getBase(ln)
		order := s.nodeOrder[base]

		isStart := base == getBase(s.rng.StartContainerAny())

		// A. Check if we should start here.
		if !s.started && isStart {
			if _, ok := ln.(interface{ Data() string }); ok {
				// Text node start.
				s.started = true
			}
		}

		// B. Handle text fragments.
		if len(frag.Text) > 0 {
			if s.started {
				sr := calculateTextSelectionRect(frag, origin, clip, ln, s)
				if sr != nil {
					*s.rects = append(*s.rects, *sr)
				}
			}

			if frag.Node != nil && frag.Node.LogicalNode() != nil {
				count := 0
				for _, c := range frag.Text {
					count += utf8.RuneCount(c.Bytes)
				}
				s.nodeOffsets[base] += count
			}
		}

		// C. Check if we logically passed the end container.
		orderEnd := s.nodeOrder[getBase(s.rng.EndContainerAny())]
		if order.First > orderEnd.Last {
			s.stopped = true
			return
		}
	}

	// 3. Recurse children.
	scrollX, scrollY := 0, 0
	newClip := clip
	if frag.Node != nil && frag.Node.Style() != nil {
		cs := frag.Node.Style()
		if cs.OverflowX != style.OverflowVisible || cs.OverflowY != style.OverflowVisible {
			bw := cs.Border.Widths()
			pad := cs.Padding
			inset := geom.Rect{
				Origin: geom.Point{X: origin.X + bw.Left + pad.Left, Y: origin.Y + bw.Top + pad.Top},
				Size: geom.Size{
					Width:  max(0, frag.Size.Width-bw.Left-bw.Right-pad.Left-pad.Right),
					Height: max(0, frag.Size.Height-bw.Top-bw.Bottom-pad.Top-pad.Bottom),
				},
			}
			newClip = clip.Intersect(inset)
		}
		if el, ok := ln.(interface{ Scroll() (x, y int) }); ok {
			scrollX, scrollY = el.Scroll()
		}
	}

	for i, childLink := range frag.Children {
		if ln != nil {
			base := getBase(ln)
			if base == getBase(s.rng.StartContainerAny()) {
				if i >= s.rng.StartOffset() {
					s.started = true
				}
			}
			// Element end boundary check BEFORE walking the child.
			if base == getBase(s.rng.EndContainerAny()) {
				if i >= s.rng.EndOffset() {
					s.stopped = true
					return
				}
			}
		}

		childOrigin := geom.Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		walkFragments(childLink.Fragment, childOrigin, newClip, s)

		if s.stopped {
			return
		}
	}
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

func calculateTextSelectionRect(frag *layout.Fragment, origin geom.Point, clip geom.Rect, node any, s *walkState) *SelectionRect {
	baseN := getBase(node)
	baseStart := getBase(s.rng.StartContainerAny())
	baseEnd := getBase(s.rng.EndContainerAny())

	startSel := -1
	if baseN == baseStart {
		startSel = s.rng.StartOffset()
	}

	endSel := 1000000000
	if baseN == baseEnd {
		endSel = s.rng.EndOffset()
	}

	firstSelectedX := -1
	lastSelectedX := -1

	x := 0
	runesSeen := 0
	currentRuneOffset := s.nodeOffsets[baseN]

	for _, c := range frag.Text {
		cRunes := utf8.RuneCount(c.Bytes)
		cWidth := c.CellWidth

		clusterStart := currentRuneOffset + runesSeen
		clusterEnd := clusterStart + cRunes

		isSelected := true
		if startSel != -1 && clusterEnd <= startSel {
			isSelected = false
		}
		if endSel != 1000000000 && clusterStart >= endSel {
			isSelected = false
		}

		if isSelected {
			if firstSelectedX == -1 {
				firstSelectedX = x
			}
			lastSelectedX = x + cWidth
		}

		// If we reached the end boundary in this text node, mark walk as stopped.
		if baseN == baseEnd && clusterEnd >= endSel {
			s.stopped = true
		}

		x += cWidth
		runesSeen += cRunes
	}

	if firstSelectedX == -1 {
		return nil
	}

	resRect := geom.Rect{
		Origin: geom.Point{X: origin.X + firstSelectedX, Y: origin.Y},
		Size:   geom.Size{Width: lastSelectedX - firstSelectedX, Height: frag.Size.Height},
	}

	resRect = resRect.Intersect(clip)
	if resRect.Size.Width <= 0 || resRect.Size.Height <= 0 {
		return nil
	}

	var comp *style.Computed
	if frag.Node != nil {
		comp = frag.Node.Style()
	}
	if comp == nil {
		return nil
	}

	return &SelectionRect{
		Rect: resRect,
		Fg:   comp.SelectionForeground,
		Bg:   comp.SelectionBackground,
	}
}
