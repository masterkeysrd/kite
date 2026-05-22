package layout

import (
	"math"

	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

// InfiniteBlockSize is the height used when probing a node's intrinsic block
// size. It represents an unconstrained vertical axis: the node should size to
// its content, not to the available height. math.MaxInt32/2 avoids overflow
// when arithmetic is performed on top of it.
const InfiniteBlockSize = math.MaxInt32 / 2

// Fragment represents the immutable output of a layout algorithm.
// Once created, a Fragment's fields must never be modified. This immutability
// allows fragments to be cached and reused across layout passes.
type Fragment struct {
	// Size is the computed physical dimensions of this fragment in terminal cells.
	Size Size

	// Node is the layout node that generated this fragment.
	Node Node

	// Children contains the positioned child fragments relative to this fragment.
	Children []FragmentLink

	// Text contains the shaped clusters if this fragment represents a text run.
	Text []text.Cluster

	// ParentNode is the containing inline element for text fragments (for style inheritance).
	ParentNode Node

	// BreakToken is the token to resume layout in the next fragmentainer.
	BreakToken *BreakToken
}

// BreakToken represents the state needed to resume layout of a node in a new
// fragmentainer (e.g., the next page or column).
type BreakToken struct {
	Node        Node
	ChildIndex  int
	InlineToken any // For resuming text layout
}

// FragmentLink connects a child Fragment to its parent at a specific physical offset.
// Positioning information is stored here rather than inside the Fragment itself,
// allowing the exact same Fragment to be reused in different positions.
type FragmentLink struct {
	// Offset is the physical position of the child relative to the parent fragment's origin.
	Offset Point

	// Fragment is the immutable child fragment.
	Fragment *Fragment
}

// ConstraintSpace defines the inputs for a layout operation. It encapsulates the
// physical size constraints alongside any additional context required during the
// layout walk (e.g., parent reference sizes, break tokens).
//
// The three size fields serve distinct roles (ADR-018):
//   - AvailableSize: per-child space after subtracting margins (or an explicit size).
//   - ContainerSpace: parent's content-box (ContainingSpace − border − padding).
//     KindPercent dimensions resolve against this field, and it is the base for
//     computing per-child AvailableSize. In CSS, percentage widths/heights always
//     resolve against the containing block's content area.
//   - ContainingSpace: parent's resolved border-box. Carries the parent's total
//     outer size for algorithms that require it (e.g. intrinsic sizing, positioning).
type ConstraintSpace struct {
	// AvailableSize is the ideal size the node should consume, provided by the parent.
	AvailableSize Size

	// ContainingSpace is the parent's resolved border-box dimensions.
	ContainingSpace Size

	// ContainerSpace is the parent's content-box dimensions
	// (ContainingSpace minus border minus padding).
	// KindPercent resolution and per-child AvailableSize derive from this field.
	ContainerSpace Size

	// IsFixedInlineSize indicates the inline size (width) is pre-determined.
	IsFixedInlineSize bool

	// IsFixedBlockSize indicates the block size (height) is pre-determined.
	IsFixedBlockSize bool

	// BreakToken is the token to resume layout from.
	BreakToken *BreakToken
}

// MinMaxSizes represents the intrinsic minimum and maximum widths of a node.
type MinMaxSizes struct {
	Min, Max int
}

// Encompass expands the min/max bounds to fit another MinMaxSizes.
func (m *MinMaxSizes) Encompass(other MinMaxSizes) {
	m.Min = max(m.Min, other.Min)
	m.Max = max(m.Max, other.Max)
}

// EncompassSize expands the min/max bounds to fit an explicit value.
func (m *MinMaxSizes) EncompassSize(value int) {
	m.Min = max(m.Min, value)
	m.Max = max(m.Max, value)
}

// Constrain caps the boundaries (min/max) to a specific value.
func (m MinMaxSizes) Constrain(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: min(m.Min, value),
		Max: min(m.Max, value),
	}
}

// Add shifts both min and max sizes simultaneously.
func (m MinMaxSizes) Add(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: m.Min + value,
		Max: m.Max + value,
	}
}

// Subtract shifts both min and max sizes simultaneously.
func (m MinMaxSizes) Subtract(value int) MinMaxSizes {
	return MinMaxSizes{
		Min: max(0, m.Min-value),
		Max: max(0, m.Max-value),
	}
}

// Algorithm is the interface that all LayoutNG-inspired layout formatters must implement.
type Algorithm interface {
	// Layout computes and returns an immutable Fragment based on the underlying node and constraints.
	Layout() *Fragment

	// ComputeMinMaxSizes calculates the intrinsic minimum and maximum sizes of the node.
	ComputeMinMaxSizes() MinMaxSizes
}

// NewAlgorithm returns the appropriate layout algorithm for the given node and constraints.
func NewAlgorithm(node Node, space ConstraintSpace) Algorithm {
	if _, ok := node.LogicalNode().(OverlayLever); ok {
		return &OverlayAlgorithm{Node: node, Space: space}
	}
	switch node.Style().Display {
	case style.DisplayFlex, style.DisplayInlineFlex:
		return &FlexAlgorithm{Node: node, Space: space}
	case style.DisplayTable:
		return &TableAlgorithm{Node: node, Space: space}
	case style.DisplayTableHeaderGroup, style.DisplayTableRowGroup, style.DisplayTableFooterGroup:
		return &TableSectionAlgorithm{Node: node, Space: space}
	case style.DisplayTableRow:
		return &TableRowAlgorithm{Node: node, Space: space}
	case style.DisplayTableCell:
		// Cells just act as BFCs with rigid constraints passed by the row.
		return &BlockAlgorithm{Node: node, Space: space}
	case style.DisplayListItem:
		return &ListAlgorithm{Node: node, Space: space}
	default:
		return &BlockAlgorithm{Node: node, Space: space}
	}
}

// IntrinsicMinMaxSizes computes the intrinsic min/max widths for a node by selecting
// the correct algorithm based on its display style.
func IntrinsicMinMaxSizes(node Node) MinMaxSizes {
	if sizes, ok := node.CachedMinMaxSizes(); ok {
		return sizes
	}
	// Note: We pass an empty ConstraintSpace as intrinsic sizes should not
	// depend on parent constraints.
	algo := NewAlgorithm(node, ConstraintSpace{})
	return algo.ComputeMinMaxSizes()
}

// IntrinsicBlockSize returns the intrinsic block size (height) of a node given an
// available inline size (width).
func IntrinsicBlockSize(node Node, availableWidth int) int {
	// For now, we just run a probe layout. In the future, this should be cached.
	// ContainerSpace and ContainingSpace must be set to the probe width so that
	// children with KindPercent widths resolve correctly inside the probe.
	// Without this, a child with width:100% would resolve to 0 (ContainerSpace.Width=0),
	// causing the IFC to place each character on its own line and return a wildly
	// inflated block height.
	probeSize := Size{Width: availableWidth, Height: InfiniteBlockSize}
	space := NewConstraintSpaceBuilder(probeSize).
		SetContainerSpace(probeSize).
		SetContainingSpace(probeSize).
		ToConstraintSpace()
	algo := NewAlgorithm(node, space)
	return algo.Layout().Size.Height
}

// AbsoluteBounds traverses the fragment tree starting at root and computes the absolute
// bounding rectangle of the target node. Returns the rect and true if found, or a zero
// rect and false if the node is not present in the tree.
func AbsoluteBounds(root *Fragment, target Node) (Rect, bool) {
	if root == nil {
		return Rect{}, false
	}
	if root.Node == target {
		return Rect{Origin: Point{0, 0}, Size: root.Size}, true
	}
	for _, childLink := range root.Children {
		if rect, found := AbsoluteBounds(childLink.Fragment, target); found {
			// Add this link's offset to the child's absolute origin.
			rect.Origin.X += childLink.Offset.X
			rect.Origin.Y += childLink.Offset.Y
			return rect, true
		}
	}
	return Rect{}, false
}

// ScrolledAbsoluteBounds returns the absolute bounding box of target, shifted
// by all ancestor scroll offsets and clipped by all ancestor overflow regions.
//
// It returns:
//   - rect: the absolute border-box of target (scrolled).
//   - clip: the absolute content-box clip rectangle of the nearest clipping
//     ancestor (intersected with all further clipping ancestors).
//   - found: true if target was found in the subtree.
func ScrolledAbsoluteBounds(root *Fragment, target Node) (rect Rect, clip Rect, found bool) {
	return scrolledAbsoluteBounds(root, target, Point{0, 0}, InfiniteRect())
}

type scrollableElement interface {
	Scroll() (x, y int)
}

func scrolledAbsoluteBounds(frag *Fragment, target Node, origin Point, currentClip Rect) (Rect, Rect, bool) {
	if frag == nil {
		return Rect{}, Rect{}, false
	}

	// 1. If this is the target, we found it.
	if frag.Node == target {
		return Rect{Origin: origin, Size: frag.Size}, currentClip, true
	}

	// 2. Compute the new clip rect if this fragment clips.
	newClip := currentClip
	scrollX, scrollY := 0, 0

	if frag.Node != nil && frag.Node.Style() != nil {
		s := frag.Node.Style()
		clipX := s.OverflowX != style.OverflowVisible
		clipY := s.OverflowY != style.OverflowVisible

		if clipX || clipY {
			bw := s.Border.Widths()
			pad := s.Padding
			insetLeft := bw.Left + pad.Left
			insetTop := bw.Top + pad.Top
			insetRight := bw.Right + pad.Right
			insetBottom := bw.Bottom + pad.Bottom

			var fragClip Rect
			if clipX {
				fragClip.Origin.X = origin.X + insetLeft
				fragClip.Size.Width = max(0, frag.Size.Width-insetLeft-insetRight)
			} else {
				fragClip.Origin.X = currentClip.Origin.X
				fragClip.Size.Width = currentClip.Size.Width
			}

			if clipY {
				fragClip.Origin.Y = origin.Y + insetTop
				fragClip.Size.Height = max(0, frag.Size.Height-insetTop-insetBottom)
			} else {
				fragClip.Origin.Y = currentClip.Origin.Y
				fragClip.Size.Height = currentClip.Size.Height
			}
			newClip = currentClip.Intersect(fragClip)
		}

		// 3. Compute scroll translation if this is a scroll container.
		// overflow:clip is included: it creates a clip boundary and supports
		// programmatic scroll offsets even without scrollbars.
		isScrollX := s.OverflowX == style.OverflowScroll || s.OverflowX == style.OverflowAuto || s.OverflowX == style.OverflowHidden || s.OverflowX == style.OverflowClip
		isScrollY := s.OverflowY == style.OverflowScroll || s.OverflowY == style.OverflowAuto || s.OverflowY == style.OverflowHidden || s.OverflowY == style.OverflowClip

		if (isScrollX || isScrollY) && frag.Node.LogicalNode() != nil {
			if el, ok := frag.Node.LogicalNode().(scrollableElement); ok {
				rawX, rawY := el.Scroll()
				maxSX, maxSY := MaxScroll(frag)
				scrollX = max(0, min(rawX, maxSX))
				scrollY = max(0, min(rawY, maxSY))
			}
		}
	}

	// 4. Recurse.
	for _, childLink := range frag.Children {
		childOrigin := Point{
			X: origin.X + childLink.Offset.X - scrollX,
			Y: origin.Y + childLink.Offset.Y - scrollY,
		}
		if r, c, found := scrolledAbsoluteBounds(childLink.Fragment, target, childOrigin, newClip); found {
			return r, c, true
		}
	}

	return Rect{}, Rect{}, false
}

// MaxScroll calculates the maximum horizontal and vertical scroll offsets
// for a fragment, clamped to its content extent.
func MaxScroll(frag *Fragment) (x, y int) {
	if frag == nil || frag.Node == nil || frag.Node.Style() == nil {
		return 0, 0
	}

	s := frag.Node.Style()
	bw := s.Border.Widths()
	pad := s.Padding

	// Content-box insets from the fragment's border-box origin.
	insetLeft := bw.Left + pad.Left
	insetTop := bw.Top + pad.Top

	// Content box size.
	contentW := max(0, frag.Size.Width-bw.Left-bw.Right-pad.Left-pad.Right)
	contentH := max(0, frag.Size.Height-bw.Top-bw.Bottom-pad.Top-pad.Bottom)

	// Content extent (union of child fragments).
	extentW, extentH := 0, 0
	for _, childLink := range frag.Children {
		extentW = max(extentW, childLink.Offset.X+childLink.Fragment.Size.Width-insetLeft)
		extentH = max(extentH, childLink.Offset.Y+childLink.Fragment.Size.Height-insetTop)
	}

	maxSX := max(0, extentW-contentW)
	maxSY := max(0, extentH-contentH)

	// Inputs and TextAreas (elements providing a cursor) need 1 extra cell of
	// horizontal scroll so the caret can sit after the last character.
	isCursorProvider := false
	if ln := frag.Node.LogicalNode(); ln != nil {
		if el, ok := ln.(interface{ ProvidesCursor() bool }); ok {
			isCursorProvider = el.ProvidesCursor()
		}
	}
	if isCursorProvider {
		maxSX++
	}

	return maxSX, maxSY
}
