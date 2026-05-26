package layout

import (
	"github.com/masterkeysrd/kite/style"
)

// OverlayAlgorithm implements anchored positioning and smart flipping for overlays.
type OverlayAlgorithm struct{}

func (a *OverlayAlgorithm) Layout(ctx *Context, node Node, space ConstraintSpace) *Fragment {
	defer ctx.Begin("Layout(Overlay)")()
	// 1. Measure content first (shrink-wrap)
	// Overlays typically shrink-wrap their content unless they have fixed sizes.
	contentSpace := ConstraintSpace{
		AvailableSize:   space.AvailableSize,
		ContainingSpace: space.ContainingSpace,
		ContainerSpace:  space.ContainerSpace,
	}
	if node.Style().Width.Kind() == style.KindAuto || node.Style().Width.Kind() == style.KindContent {
		contentSpace.IsFixedInlineSize = false
	} else {
		contentSpace.IsFixedInlineSize = space.IsFixedInlineSize
	}

	// We use BlockAlgorithm for the content of the overlay itself (it acts as a BFC).
	frag := blockAlgo.Layout(ctx, node, contentSpace)

	// 2. Determine physical position
	lever, ok := node.LogicalNode().(OverlayLever)
	if !ok {
		// If not an OverlayLever, just return the fragment at (0,0)
		return frag
	}

	anchor := lever.Anchor()
	placement := lever.Placement()
	flip := lever.Flip()

	// Default position
	x, y := 0, 0

	if anchorEl, ok := anchor.(interface {
		GetBoundingClientRect() (Rect, bool)
	}); ok {
		anchorRect, found := anchorEl.GetBoundingClientRect()
		if found {
			x, y = a.calculatePosition(anchorRect, frag.Size, placement, flip, space.AvailableSize)
		}
	}

	node.SetOffset(Point{X: x, Y: y})

	return frag
}

func (a *OverlayAlgorithm) calculatePosition(anchor Rect, size Size, placement OverlayPlacement, flip bool, availableSize Size) (int, int) {
	x, y := a.resolvePlacement(anchor, size, placement)

	if flip {
		// Check if it overflows viewport
		overflows := x < 0 || y < 0 || x+size.Width > availableSize.Width || y+size.Height > availableSize.Height

		if overflows {
			// Try opposite placement
			opposite := a.oppositePlacement(placement)
			nx, ny := a.resolvePlacement(anchor, size, opposite)

			// Check if opposite also overflows
			nOverflows := nx < 0 || ny < 0 || nx+size.Width > availableSize.Width || ny+size.Height > availableSize.Height

			if !nOverflows {
				return nx, ny
			}

			// If both overflow, we default to the side with the most available space.
			topSpace := anchor.Origin.Y
			bottomSpace := max(0, availableSize.Height-(anchor.Origin.Y+anchor.Size.Height))
			leftSpace := anchor.Origin.X
			rightSpace := max(0, availableSize.Width-(anchor.Origin.X+anchor.Size.Width))

			switch placement {
			case PlacementTop, PlacementBottom:
				if topSpace >= bottomSpace {
					return a.resolvePlacement(anchor, size, PlacementTop)
				}
				return a.resolvePlacement(anchor, size, PlacementBottom)
			case PlacementLeft, PlacementRight:
				if leftSpace >= rightSpace {
					return a.resolvePlacement(anchor, size, PlacementLeft)
				}
				return a.resolvePlacement(anchor, size, PlacementRight)
			}
		}
	}

	return x, y
}

func (a *OverlayAlgorithm) resolvePlacement(anchor Rect, size Size, placement OverlayPlacement) (int, int) {
	switch placement {
	case PlacementTop:
		return anchor.Origin.X, anchor.Origin.Y - size.Height
	case PlacementBottom:
		return anchor.Origin.X, anchor.Origin.Y + anchor.Size.Height
	case PlacementLeft:
		return anchor.Origin.X - size.Width, anchor.Origin.Y
	case PlacementRight:
		return anchor.Origin.X + anchor.Size.Width, anchor.Origin.Y
	default:
		return anchor.Origin.X, anchor.Origin.Y
	}
}

func (a *OverlayAlgorithm) oppositePlacement(p OverlayPlacement) OverlayPlacement {
	switch p {
	case PlacementTop:
		return PlacementBottom
	case PlacementBottom:
		return PlacementTop
	case PlacementLeft:
		return PlacementRight
	case PlacementRight:
		return PlacementLeft
	default:
		return p
	}
}

func (a *OverlayAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return blockAlgo.ComputeMinMaxSizes(ctx, node)
}
