package render

import (
	"unicode/utf8"

	"image/color"
	"reflect"

	"github.com/masterkeysrd/kite/dom"
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
	StartIndex() int
	EndIndex() int
}

// NodeOrder tracks the pre-order traversal boundaries of a node's subtree.
type NodeOrder struct {
	First int
	Last  int
}

type OverlaySelection struct {
	Element  dom.Element
	Fragment *layout.Fragment
	Offset   geom.Point
}

func isDescendant(parent dom.Node, child dom.Node) bool {
	if parent == nil || child == nil {
		return false
	}
	baseParent := getBase(parent)
	for curr := child; curr != nil; {
		if getBase(curr) == baseParent {
			return true
		}
		curr = curr.Parent()
	}
	return false
}

// ResolveSelection maps the active selection to physical screen rectangles.
func ResolveSelection(
	root *layout.Fragment,
	overlays []OverlaySelection,
	sel SelectionSource,
	nodeOrder map[any]NodeOrder,
) []SelectionRect {
	var rects []SelectionRect
	baseCache := make(map[any]any)
	for i := 0; i < sel.RangeCount(); i++ {
		rng := sel.GetRangeAt(i)
		if rng == nil || rng.IsCollapsed() {
			continue
		}
		nodeOffsets := make(map[any]int)

		startNode, _ := rng.StartContainerAny().(dom.Node)

		var activeOverlay *OverlaySelection
		for _, ov := range overlays {
			if isDescendant(ov.Element, startNode) {
				activeOverlay = &ov
				break
			}
		}

		if activeOverlay != nil {
			if activeOverlay.Fragment != nil {
				rects = append(rects, resolveRange(activeOverlay.Fragment, activeOverlay.Offset, rng, nodeOrder, nodeOffsets, baseCache)...)
			}
		} else {
			rects = append(rects, resolveRange(root, geom.Point{}, rng, nodeOrder, nodeOffsets, baseCache)...)
		}
	}
	return rects
}

func resolveRange(
	root *layout.Fragment,
	origin geom.Point,
	rng SelectionRange,
	nodeOrder map[any]NodeOrder,
	nodeOffsets map[any]int,
	baseCache map[any]any,
) []SelectionRect {
	var rects []SelectionRect

	startCont := rng.StartContainerAny()
	baseStart, ok := baseCache[startCont]
	if !ok {
		baseStart = getBase(startCont)
		baseCache[startCont] = baseStart
	}

	endCont := rng.EndContainerAny()
	baseEnd, ok := baseCache[endCont]
	if !ok {
		baseEnd = getBase(endCont)
		baseCache[endCont] = baseEnd
	}

	state := &walkState{
		rng:         rng,
		nodeOrder:   nodeOrder,
		nodeOffsets: nodeOffsets,
		baseCache:   baseCache,
		rects:       &rects,
		startIndex:  rng.StartIndex(),
		endIndex:    rng.EndIndex(),
		baseStart:   baseStart,
		baseEnd:     baseEnd,
	}

	walkFragments(root, origin, layout.InfiniteRect(), state)
	return rects
}

type walkState struct {
	rng         SelectionRange
	nodeOrder   map[any]NodeOrder
	nodeOffsets map[any]int
	baseCache   map[any]any
	rects       *[]SelectionRect

	startIndex int
	endIndex   int
	baseStart  any
	baseEnd    any

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
		base, ok := s.baseCache[ln]
		if !ok {
			base = getBase(ln)
			s.baseCache[ln] = base
		}

		order, hasOrder := s.nodeOrder[base]
		if hasOrder {
			// Check if we should stop: if this node starts at or after endIndex
			if order.First >= s.endIndex {
				s.stopped = true
				return
			}

			// Optimization: skip the entire subtree if it's strictly before the selection start.
			if order.Last < s.startIndex {
				return
			}
		}

		// B. Handle text fragments.
		if len(frag.Text) > 0 {
			sr := calculateTextSelectionRect(frag, origin, clip, ln, s)
			if sr != nil {
				*s.rects = append(*s.rects, *sr)
			}

			if frag.Node != nil && frag.Node.LogicalNode() != nil {
				// Only accumulate nodeOffsets if we actually need them for boundary resolution.
				if base == s.baseStart || base == s.baseEnd {
					count := 0
					for _, c := range frag.Text {
						count += utf8.RuneCount(c.Bytes)
					}
					s.nodeOffsets[base] += count
				}
			}
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

	for _, childLink := range frag.Children {
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
	if dn, ok := n.(dom.Node); ok {
		curr := dn
		for {
			if u := curr.Unwrap(); u != nil && u != curr {
				curr = u
			} else {
				break
			}
		}
		return curr
	}
	return getBaseReflect(n)
}

func getBaseReflect(n any) any {
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
	baseN, ok := s.baseCache[node]
	if !ok {
		baseN = getBase(node)
		s.baseCache[node] = baseN
	}

	startSel := -1
	if baseN == s.baseStart {
		startSel = s.rng.StartOffset()
	}

	endSel := 1000000000
	if baseN == s.baseEnd {
		endSel = s.rng.EndOffset()
	}

	firstSelectedX := -1
	lastSelectedX := -1

	// Fast path for fully selected nodes.
	if startSel == -1 && endSel == 1000000000 {
		firstSelectedX = 0
		lastSelectedX = 0
		for _, c := range frag.Text {
			lastSelectedX += c.CellWidth
		}
	} else {
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
			if baseN == s.baseEnd && clusterEnd >= endSel {
				s.stopped = true
			}

			x += cWidth
			runesSeen += cRunes
		}
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
