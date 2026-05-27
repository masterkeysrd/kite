package layout

import (
	"github.com/masterkeysrd/kite/geom"
	geometry "github.com/masterkeysrd/kite/geom"
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
		GetBoundingClientRect() (geometry.Rect, bool)
	}); ok {
		anchorRect, found := anchorEl.GetBoundingClientRect()
		if found {
			x, y = a.calculatePosition(anchorRect, frag.Size, placement, flip, space.AvailableSize)
		}
	}

	node.SetOffset(geometry.Point{X: x, Y: y})

	return frag
}

func (a *OverlayAlgorithm) calculatePosition(anchor geometry.Rect, size geometry.Size, placement geom.Placement, flip bool, availableSize geometry.Size) (int, int) {
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
			case geom.PlacementTop, geom.PlacementBottom:
				if topSpace >= bottomSpace {
					return a.resolvePlacement(anchor, size, geom.PlacementTop)
				}
				return a.resolvePlacement(anchor, size, geom.PlacementBottom)
			case geom.PlacementLeft, geom.PlacementRight:
				if leftSpace >= rightSpace {
					return a.resolvePlacement(anchor, size, geom.PlacementLeft)
				}
				return a.resolvePlacement(anchor, size, geom.PlacementRight)
			}
		}
	}

	return x, y
}

func (a *OverlayAlgorithm) resolvePlacement(anchor geometry.Rect, size geometry.Size, placement geom.Placement) (int, int) {
	switch placement {
	case geom.PlacementTop:
		return anchor.Origin.X, anchor.Origin.Y - size.Height
	case geom.PlacementBottom:
		return anchor.Origin.X, anchor.Origin.Y + anchor.Size.Height
	case geom.PlacementLeft:
		return anchor.Origin.X - size.Width, anchor.Origin.Y
	case geom.PlacementRight:
		return anchor.Origin.X + anchor.Size.Width, anchor.Origin.Y
	default:
		return anchor.Origin.X, anchor.Origin.Y
	}
}

func (a *OverlayAlgorithm) oppositePlacement(p geom.Placement) geom.Placement {
	switch p {
	case geom.PlacementTop:
		return geom.PlacementBottom
	case geom.PlacementBottom:
		return geom.PlacementTop
	case geom.PlacementLeft:
		return geom.PlacementRight
	case geom.PlacementRight:
		return geom.PlacementLeft
	default:
		return p
	}
}

func (a *OverlayAlgorithm) ComputeMinMaxSizes(ctx *Context, node Node) MinMaxSizes {
	return blockAlgo.ComputeMinMaxSizes(ctx, node)
}
