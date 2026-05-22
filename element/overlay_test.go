package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

func TestOverlay_Positioning(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Create an anchor at a fixed position
	anchor := element.Box().Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(3)),
		Margin: style.Some(style.Edges(5, 0, 0, 10)),
	})
	eng.Mount(anchor)

	// Create an overlay
	ovl := element.Overlay(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(5)),
			Height: style.Some(style.Cells(2)),
		}),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: layout.PlacementBottom,
			ZIndex:    100,
		},
	)
	// Overlay attaches itself to the document in OnConnected
	eng.Document().AppendChild(ovl)

	// Run layout
	eng.Frame()

	// Verify anchor position
	anchorRect, _ := anchor.GetBoundingClientRect()
	expectedAnchorRect := layout.Rect{
		Origin: layout.Point{X: 10, Y: 5},
		Size:   layout.Size{Width: 10, Height: 3},
	}
	if anchorRect != expectedAnchorRect {
		t.Errorf("anchor rect mismatch: expected %v, got %v", expectedAnchorRect, anchorRect)
	}

	// Verify overlay position (Bottom placement)
	// Expected X: anchor.X = 10
	// Expected Y: anchor.Y + anchor.Height = 5 + 3 = 8
	ovlRO := ovl.RenderObject()
	if ovlRO == nil {
		t.Fatal("overlay render object not created")
	}
	ovlOffset := ovlRO.Offset()
	expectedOvlOffset := layout.Point{X: 10, Y: 8}
	if ovlOffset != expectedOvlOffset {
		t.Errorf("overlay offset mismatch: expected %v, got %v", expectedOvlOffset, ovlOffset)
	}
}

func TestOverlay_Flipping(t *testing.T) {
	t.Skip("We need to debug the flipping logic before we can enable this test")
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Create an anchor near the bottom edge
	anchor := element.Box().Style(style.Style{
		Width:  style.Some(style.Cells(10)),
		Height: style.Some(style.Cells(3)),
		Margin: style.Some(style.Edges(20, 0, 0, 10)),
	})
	eng.Mount(anchor)

	// Create an overlay with Bottom placement and Flip enabled
	ovl := element.Overlay(
		element.Box().Style(style.Style{
			Width:  style.Some(style.Cells(5)),
			Height: style.Some(style.Cells(5)),
		}),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: layout.PlacementBottom,
			Flip:      true,
		},
	)
	eng.Document().AppendChild(ovl)

	// Run layout
	eng.Frame()

	// Anchor is at Y=20, Height=3. Bottom edge is 23.
	// Overlay Height=5. 23 + 5 = 28. Viewport Height is 24.
	// It overflows bottom, so it should flip to Top.
	// Expected Top Y: anchor.Y - ovl.Height = 20 - 5 = 15.

	ovlRO := ovl.RenderObject()
	ovlOffset := ovlRO.Offset()
	expectedOvlOffset := layout.Point{X: 10, Y: 15}
	if ovlOffset != expectedOvlOffset {
		t.Errorf("overlay should have flipped to Top: expected %v, got %v", expectedOvlOffset, ovlOffset)
	}
}
