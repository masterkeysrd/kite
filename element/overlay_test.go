package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

func TestOverlay_Positioning(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Create an anchor at a fixed position
	anchor := element.Box().Style(style.S().
		Width(style.Cells(10)).
		Height(style.Cells(3)).
		Margin(5, 0, 0, 10))

	eng.Mount(anchor)

	// Create an overlay
	ovl := element.Overlay(
		element.Box().
			Style(style.S().
				Width(style.Cells(5)).
				Height(style.Cells(2)),
			),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: geom.PlacementBottom,
			ZIndex:    100,
		},
	)
	// Overlay attaches itself to the document in OnConnected
	eng.Document().AppendChild(ovl)

	// Run layout
	eng.Frame()

	// Verify anchor position
	anchorRect, _ := anchor.GetBoundingClientRect()
	expectedAnchorRect := geom.Rect{
		Origin: geom.Point{X: 10, Y: 5},
		Size:   geom.Size{Width: 10, Height: 3},
	}
	if anchorRect != expectedAnchorRect {
		t.Errorf("anchor rect mismatch: expected %v, got %v", expectedAnchorRect, anchorRect)
	}

	// Verify overlay position (Bottom placement)
	// Expected X: anchor.X = 10
	// Expected Y: anchor.Y + anchor.Height = 5 + 3 = 8
	ovlRO := eng.RenderObject(ovl)
	if ovlRO == nil {
		t.Fatal("overlay render object not created")
	}
	ovlOffset := ovlRO.Offset()
	expectedOvlOffset := geom.Point{X: 10, Y: 8}
	if ovlOffset != expectedOvlOffset {
		t.Errorf("overlay offset mismatch: expected %v, got %v", expectedOvlOffset, ovlOffset)
	}
}

func TestOverlay_Flipping(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Create an anchor near the bottom edge
	anchor := element.Box().Style(style.S().
		Width(style.Cells(10)).
		Height(style.Cells(3)).
		Margin(20, 0, 0, 10),
	)

	eng.Mount(anchor)

	// Create an overlay with Bottom placement and Flip enabled
	ovl := element.Overlay(
		element.Box().Style(style.S().
			Width(style.Cells(5)).
			Height(style.Cells(5)),
		),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: geom.PlacementBottom,
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

	ovlRO := eng.RenderObject(ovl)
	ovlOffset := ovlRO.Offset()
	expectedOvlOffset := geom.Point{X: 10, Y: 15}
	if ovlOffset != expectedOvlOffset {
		t.Errorf("overlay should have flipped to Top: expected %v, got %v", expectedOvlOffset, ovlOffset)
	}
}

func TestOverlay_BestFit(t *testing.T) {
	// Viewport 80x24.
	// Anchor at Y=10, Height=4. Top space: 10, Bottom space: 24 - 14 = 10.
	// If we have an overlay of height 15, it overflows both.
	// We want to test that it picks the side with more space if they are unequal.

	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	// Anchor at Y=5, Height=2. Top space: 5, Bottom space: 24 - 7 = 17.
	anchor := element.Box().Style(style.S().
		Width(style.Cells(10)).
		Height(style.Cells(2)).
		Margin(5, 0, 0, 10),
	)
	eng.Mount(anchor)

	// Overlay height 20.
	ovl := element.Overlay(
		element.Box().Style(style.S().
			Width(style.Cells(5)).
			Height(style.Cells(20)),
		),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: geom.PlacementTop, // Start at Top (space 5)
			Flip:      true,
		},
	)

	eng.Document().AppendChild(ovl)
	eng.Frame()

	// It should flip to Bottom because it has more space (17 vs 5).
	ovlRO := eng.RenderObject(ovl)
	ovlOffset := ovlRO.Offset()
	expectedY := 5 + 2 // anchor.Y + anchor.Height
	if ovlOffset.Y != expectedY {
		t.Errorf("overlay should have flipped to Bottom (more space): expected Y=%d, got %d", expectedY, ovlOffset.Y)
	}
}

func TestOverlay_HorizontalBestFit(t *testing.T) {
	// Viewport 80x24.
	// Anchor at X=5, Width=2. Left space: 5, Right space: 80 - 7 = 73.
	// Overlay width 50. Placement Left.
	// Should flip to Right because it has more space (73 vs 5).

	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	anchor := element.Box().Style(style.S().Width(style.Cells(2)).Height(style.Cells(3)).Margin(10, 0, 0, 5))
	eng.Mount(anchor)

	ovl := element.Overlay(
		element.Box().Style(style.S().Width(style.Cells(50)).Height(style.Cells(5))),
		element.OverlayConfig{
			Anchor:    anchor,
			Placement: geom.PlacementLeft,
			Flip:      true,
		},
	)
	eng.Document().AppendChild(ovl)
	eng.Frame()

	ovlRO := eng.RenderObject(ovl)
	ovlOffset := ovlRO.Offset()
	expectedX := 5 + 2 // anchor.X + anchor.Width
	if ovlOffset.X != expectedX {
		t.Errorf("overlay should have flipped to Right (more space): expected X=%d, got %d", expectedX, ovlOffset.X)
	}
}

func TestOverlay_SetConfig_UpdatesZIndex(t *testing.T) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	anchor := element.Box()
	eng.Mount(anchor)

	ovl1 := element.Overlay(
		element.Box(),
		element.OverlayConfig{
			Anchor: anchor,
			ZIndex: 100,
		},
	)
	ovl2 := element.Overlay(
		element.Box(),
		element.OverlayConfig{
			Anchor: anchor,
			ZIndex: 150,
		},
	)

	eng.Document().AppendChild(ovl1)
	eng.Document().AppendChild(ovl2)
	eng.Frame()

	// Initial order should be ovl1 (100) then ovl2 (150)
	var initial []dom.Element
	for el := range eng.Document().Overlays() {
		initial = append(initial, el)
	}
	if len(initial) != 2 || initial[0] != ovl1 || initial[1] != ovl2 {
		t.Fatalf("unexpected initial overlay order: %v", initial)
	}

	// Update configuration with a new zIndex for ovl1
	ovl1.SetConfig(element.OverlayConfig{
		Anchor: anchor,
		ZIndex: 200,
	})

	// Order should now be ovl2 (150) then ovl1 (200)
	var updated []dom.Element
	for el := range eng.Document().Overlays() {
		updated = append(updated, el)
	}
	if len(updated) != 2 || updated[0] != ovl2 || updated[1] != ovl1 {
		t.Errorf("unexpected updated overlay order: %v", updated)
	}
}

