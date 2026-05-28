package paint

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/layout/text"
	"github.com/masterkeysrd/kite/style"
)

type mockNode struct {
	layout.Node
	s *style.Computed
}

func (m *mockNode) Style() *style.Computed { return m.s }
func (m *mockNode) LogicalNode() dom.Node  { return nil }

func TestPaint_InheritedStyle(t *testing.T) {
	pe := &PaintEngine{}

	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}

	tests := []struct {
		name        string
		nodeStyle   *style.Computed
		parentStyle *style.Computed
		wantFG      color.Color
		wantBG      color.Color
	}{
		{
			name: "Default style",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			wantFG: color.RGBA{255, 255, 255, 255}, // Default fallback
			wantBG: color.Transparent,
		},
		{
			name: "Explicit foreground on node",
			nodeStyle: &style.Computed{
				Foreground: red,
				Background: color.Transparent,
			},
			wantFG: red,
			wantBG: color.Transparent,
		},
		{
			name: "Inherit foreground from parent",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			parentStyle: &style.Computed{
				Foreground: blue,
			},
			wantFG: blue,
			wantBG: color.Transparent,
		},
		{
			name: "Explicit background on parent",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			parentStyle: &style.Computed{
				Background: blue,
			},
			wantFG: color.RGBA{255, 255, 255, 255},
			wantBG: blue,
		},
		{
			name: "Node background overrides parent",
			nodeStyle: &style.Computed{
				Background: red,
			},
			parentStyle: &style.Computed{
				Background: blue,
			},
			wantFG: color.RGBA{255, 255, 255, 255},
			wantBG: red,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frag := &layout.Fragment{
				Node: &mockNode{s: tt.nodeStyle},
			}
			if tt.parentStyle != nil {
				frag.ParentNode = &mockNode{s: tt.parentStyle}
			}

			gotFG, gotBG := pe.getInheritedStyle(frag)
			if gotFG != tt.wantFG {
				t.Errorf("gotFG = %v, want %v", gotFG, tt.wantFG)
			}
			if gotBG != tt.wantBG {
				t.Errorf("gotBG = %v, want %v", gotBG, tt.wantBG)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Overflow clipping tests
// -----------------------------------------------------------------------------

// makeTextFrag creates a simple fragment with a text cluster at the given
// size/offset. The text cluster has CellWidth == len(content) to keep things
// simple for testing. The node carries the given computed style.
func makeBoxFrag(size geom.Size, s *style.Computed, children ...layout.FragmentLink) *layout.Fragment {
	return &layout.Fragment{
		Size:     size,
		Node:     &mockNode{s: s},
		Children: children,
	}
}

func makeTextFrag(content string, s *style.Computed) *layout.Fragment {
	var clusters []text.Cluster
	for i, r := range content {
		_ = i
		clusters = append(clusters, text.Cluster{Bytes: []byte(string(r)), CellWidth: 1})
	}
	return &layout.Fragment{
		Size: geom.Size{Width: len(content), Height: 1},
		Node: &mockNode{s: s},
		Text: clusters,
	}
}

// TestOverflowClips_OverflowClips tests the helper function directly.
func TestOverflowClips(t *testing.T) {
	if overflowClips(style.OverflowVisible) {
		t.Error("OverflowVisible should NOT clip")
	}
	for _, o := range []style.Overflow{style.OverflowHidden, style.OverflowClip, style.OverflowScroll} {
		if !overflowClips(o) {
			t.Errorf("overflow %v should clip", o)
		}
	}
}

// TestPaint_OverflowVisible_NoClip verifies that OverflowVisible does not clip
// children that exceed the parent's bounds (regression guard).
func TestPaint_OverflowVisible_NoClip(t *testing.T) {
	// Parent: 5×1, no overflow → child text of 10 chars should fully paint.
	parentStyle := &style.Computed{OverflowX: style.OverflowVisible, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("0123456789", childStyle) // 10 cells wide
	parentFrag := makeBoxFrag(
		geom.Size{Width: 5, Height: 1},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 0, Y: 0}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 1)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// All 10 cells should be painted.
	for x := 0; x < 10; x++ {
		c := fb.CellAt(x, 0)
		if c.Content == "" {
			t.Errorf("OverflowVisible: cell (%d,0) should be painted, got empty", x)
		}
	}
}

// TestPaint_OverflowHidden_ClipsToContentBox verifies that OverflowHidden clips
// children to the content box of the parent.
func TestPaint_OverflowHidden_ClipsToContentBox(t *testing.T) {
	// Parent: 5×1, overflow-x: hidden → child text of 10 chars should only
	// paint 5 cells.
	parentStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("0123456789", childStyle) // 10 cells wide
	parentFrag := makeBoxFrag(
		geom.Size{Width: 5, Height: 1},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 0, Y: 0}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 1)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// Only cells 0..4 should be painted; 5..9 must remain empty.
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("OverflowHidden: cell (%d,0) should be painted", x)
		}
	}
	for x := 5; x < 10; x++ {
		if fb.CellAt(x, 0).Content != "" {
			t.Errorf("OverflowHidden: cell (%d,0) should be clipped but got %q", x, fb.CellAt(x, 0).Content)
		}
	}
}

// TestPaint_OverflowClip_BehavesLikeHidden verifies that OverflowClip produces
// the same visual result as OverflowHidden for the paint phase.
func TestPaint_OverflowClip_BehavesLikeHidden(t *testing.T) {
	parentStyle := &style.Computed{OverflowX: style.OverflowClip, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("ABCDEFGHIJ", childStyle) // 10 cells wide
	parentFrag := makeBoxFrag(
		geom.Size{Width: 5, Height: 1},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 0, Y: 0}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 1)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("OverflowClip: cell (%d,0) should be painted", x)
		}
	}
	for x := 5; x < 10; x++ {
		if fb.CellAt(x, 0).Content != "" {
			t.Errorf("OverflowClip: cell (%d,0) should be clipped but got %q", x, fb.CellAt(x, 0).Content)
		}
	}
}

// TestPaint_AsymmetricOverflow_HiddenXVisibleY clips only horizontal overflow.
func TestPaint_AsymmetricOverflow_HiddenXVisibleY(t *testing.T) {
	// Parent: 5×3, overflow-x: hidden, overflow-y: visible.
	// Child text is 10 wide × 5 tall (overflows both axes).
	// Only horizontal spill is clipped.
	parentStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	// Build a 5-row child that is 10 cells wide.
	var childChildren []layout.FragmentLink
	for row := 0; row < 5; row++ {
		childChildren = append(childChildren, layout.FragmentLink{
			Offset:   geom.Point{X: 0, Y: row},
			Fragment: makeTextFrag("0123456789", childStyle),
		})
	}
	childFrag := makeBoxFrag(geom.Size{Width: 10, Height: 5}, childStyle, childChildren...)
	parentFrag := makeBoxFrag(
		geom.Size{Width: 5, Height: 3},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 6)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// X >= 5 must be empty (horizontal clip).
	for row := 0; row < 5; row++ {
		for x := 5; x < 10; x++ {
			if fb.CellAt(x, row).Content != "" {
				t.Errorf("AsymmetricX: cell (%d,%d) must be clipped, got %q", x, row, fb.CellAt(x, row).Content)
			}
		}
	}
	// Y >= 3 may still be painted (vertical visible).
	paintedBelow := false
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 4).Content != "" {
			paintedBelow = true
			break
		}
	}
	if !paintedBelow {
		t.Error("AsymmetricX: vertical overflow should NOT be clipped (OverflowY: Visible)")
	}
}

// TestPaint_BorderIntegrity_OverflowHidden verifies that a fragment's own border
// is not clipped by its own overflow property.
func TestPaint_BorderIntegrity_OverflowHidden(t *testing.T) {
	// Parent: 5×3, single border, overflow: hidden.
	// The border occupies the outer perimeter. An oversized child should be
	// clipped, but the parent's own border should remain intact.
	parentStyle := &style.Computed{
		OverflowX: style.OverflowHidden,
		OverflowY: style.OverflowHidden,
		Border:    style.SingleBorder(),
	}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("0123456789", childStyle) // 10 cells wide
	parentFrag := makeBoxFrag(
		geom.Size{Width: 5, Height: 3},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 1, Y: 1}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 5)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// Border top-left corner must be present.
	if fb.CellAt(0, 0).BorderStyle == BorderNone {
		t.Error("BorderIntegrity: top-left border cell should be set")
	}
	// Border top-right corner must be present.
	if fb.CellAt(4, 0).BorderStyle == BorderNone {
		t.Error("BorderIntegrity: top-right border cell should be set")
	}
	// Border bottom-left corner must be present.
	if fb.CellAt(0, 2).BorderStyle == BorderNone {
		t.Error("BorderIntegrity: bottom-left border cell should be set")
	}
	// Border bottom-right corner must be present.
	if fb.CellAt(4, 2).BorderStyle == BorderNone {
		t.Error("BorderIntegrity: bottom-right border cell should be set")
	}
	// Content area (inside border) — child text starts at x=1 (border width),
	// so clip region is x=[1,3]. Cells 4..9 (after the border-right) must be empty.
	// The content-box is x=[1..3] (border=1 each side → width=3).
	for x := 4; x < 10; x++ {
		if fb.CellAt(x, 1).Content != "" && fb.CellAt(x, 1).BorderStyle == BorderNone {
			t.Errorf("BorderIntegrity: cell (%d,1) should be clipped, got %q", x, fb.CellAt(x, 1).Content)
		}
	}
}

// TestPaint_PaddingContributesToClipRect verifies that padding is accounted for
// in the content-box clip rect computation.
func TestPaint_PaddingContributesToClipRect(t *testing.T) {
	// Parent: 8 wide × 1 tall, padding-left = 2, overflow-x: hidden.
	// Content box starts at x=2, width = 8-2 = 6.
	// A child offset to (2,0) with 10 chars wide should clip at x=8.
	parentStyle := &style.Computed{
		OverflowX: style.OverflowHidden,
		OverflowY: style.OverflowVisible,
		Padding:   style.EdgeValues[int]{Left: 2},
	}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("0123456789", childStyle) // 10 cells wide
	parentFrag := makeBoxFrag(
		geom.Size{Width: 8, Height: 1},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 2, Y: 0}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 15, 1)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// Cells 2..7 (content box) must be painted.
	for x := 2; x < 8; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("Padding: cell (%d,0) should be painted", x)
		}
	}
	// Cells 8..11 must be clipped.
	for x := 8; x < 12; x++ {
		if fb.CellAt(x, 0).Content != "" {
			t.Errorf("Padding: cell (%d,0) should be clipped, got %q", x, fb.CellAt(x, 0).Content)
		}
	}
}

// TestPaint_NestedOverflow_IntersectsClipRects verifies that nested overflow
// boxes compose clip rects via intersection (grandchild is clipped by both).
func TestPaint_NestedOverflow_IntersectsClipRects(t *testing.T) {
	// Layout:
	//   outer (10 wide, overflow-x: hidden) → clips to x=[0,9]
	//   inner (6 wide at offset 2, overflow-x: hidden) → clips to x=[2,7]
	//   grandchild text (20 chars at offset 2 inside inner = abs x=4)
	// Grandchild can only paint x=[4,7] (intersection).
	outerStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowVisible}
	innerStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowVisible}
	grandStyle := &style.Computed{}

	grandFrag := makeTextFrag("01234567890123456789", grandStyle) // 20 cells
	innerFrag := makeBoxFrag(
		geom.Size{Width: 6, Height: 1},
		innerStyle,
		layout.FragmentLink{Offset: geom.Point{X: 2, Y: 0}, Fragment: grandFrag},
	)
	outerFrag := makeBoxFrag(
		geom.Size{Width: 10, Height: 1},
		outerStyle,
		layout.FragmentLink{Offset: geom.Point{X: 2, Y: 0}, Fragment: innerFrag},
	)

	fb := NewFrameBuffer(0, 0, 25, 1)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, outerFrag, geom.Point{}, fb)

	// inner clip rect: x=[2,7] (outer origin 0 + inner offset 2; width 6 → 2+6=8, but outer
	// clip starts at 0 and ends at 10, so inner clips to [2,8)).
	// grandchild text starts at abs x = 0+2(outer→inner) + 2(inner→grand) = 4.
	// inner content-box: x=[2,8) but outer clip intersects → [2,10) → still [2,8).
	// grandchild paints 20 chars from x=4 → clipped at x=8.
	for x := 4; x < 8; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("NestedClip: cell (%d,0) should be painted", x)
		}
	}
	for x := 8; x < 24; x++ {
		if fb.CellAt(x, 0).Content != "" {
			t.Errorf("NestedClip: cell (%d,0) should be clipped, got %q", x, fb.CellAt(x, 0).Content)
		}
	}
}

// TestPaint_ZeroSizedContentBox drops all descendant paint when border+padding
// consume the entire fragment size.
func TestPaint_ZeroSizedContentBox(t *testing.T) {
	// Parent: 2×1, border on both sides (1+1=2 ≥ width), overflow-x: hidden.
	// Content-box width = max(0, 2-1-1) = 0 → clip surface is empty.
	parentStyle := &style.Computed{
		OverflowX: style.OverflowHidden,
		OverflowY: style.OverflowHidden,
		Border:    style.SingleBorder(),
	}
	childStyle := &style.Computed{}

	childFrag := makeTextFrag("Hello", childStyle)
	parentFrag := makeBoxFrag(
		geom.Size{Width: 2, Height: 1},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 1, Y: 0}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 10, 2)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// No text content from the child should appear (zero-width content box).
	for x := 0; x < 10; x++ {
		c := fb.CellAt(x, 0)
		if c.Content != "" && c.BorderStyle == BorderNone {
			t.Errorf("ZeroSizedContentBox: non-border cell (%d,0) should be empty, got %q", x, c.Content)
		}
	}
}

// TestPaint_Integration_HiddenOverflow10Wide tests that a 10-wide box with
// overflow-x: hidden containing a 30-char text line paints only 10 cells.
func TestPaint_Integration_HiddenOverflow10Wide(t *testing.T) {
	parentStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	text30 := "012345678901234567890123456789" // 30 chars
	childFrag := makeTextFrag(text30, childStyle)
	parentFrag := makeBoxFrag(
		geom.Size{Width: 10, Height: 3},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 40, 3)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// Cells 0..9 should be painted.
	for x := 0; x < 10; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("Integration: cell (%d,0) should be painted", x)
		}
	}
	// Cells 10..29 must remain empty.
	for x := 10; x < 30; x++ {
		if fb.CellAt(x, 0).Content != "" {
			t.Errorf("Integration: cell (%d,0) must be clipped, got %q", x, fb.CellAt(x, 0).Content)
		}
	}
}

// TestPaint_Integration_VisibleOverflow10Wide verifies that OverflowVisible
// allows a 30-char text to spill past a 10-wide container.
func TestPaint_Integration_VisibleOverflow10Wide(t *testing.T) {
	parentStyle := &style.Computed{OverflowX: style.OverflowVisible, OverflowY: style.OverflowVisible}
	childStyle := &style.Computed{}

	text30 := "012345678901234567890123456789" // 30 chars
	childFrag := makeTextFrag(text30, childStyle)
	parentFrag := makeBoxFrag(
		geom.Size{Width: 10, Height: 3},
		parentStyle,
		layout.FragmentLink{Offset: geom.Point{}, Fragment: childFrag},
	)

	fb := NewFrameBuffer(0, 0, 40, 3)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, parentFrag, geom.Point{}, fb)

	// All 30 cells should be painted (spill allowed).
	for x := 0; x < 30; x++ {
		if fb.CellAt(x, 0).Content == "" {
			t.Errorf("VisibleOverflow: cell (%d,0) should be painted", x)
		}
	}
}

func TestPaint_IsTransparent(t *testing.T) {
	tests := []struct {
		name string
		c    color.Color
		want bool
	}{
		{"Nil", nil, true},
		{"Transparent", color.Transparent, true},
		{"RGBA 0", color.RGBA{0, 0, 0, 0}, true},
		{"RGBA 1", color.RGBA{0, 0, 0, 1}, false},
		{"Opaque Red", color.RGBA{255, 0, 0, 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTransparent(tt.c); got != tt.want {
				t.Errorf("isTransparent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaint_DebugXRay(t *testing.T) {
	// Content: 1x1
	// Padding: 1 -> Padding Box: 3x3
	// Border: 1 -> Border Box: 5x5
	// Margin: 1 -> Margin Box: 7x7
	// Origin (border-box): (1,1)

	s := &style.Computed{
		Margin:  style.EdgeAll(1),
		Padding: style.EdgeAll(1),
		Border:  style.SingleBorder(),
	}

	frag := makeBoxFrag(geom.Size{Width: 5, Height: 5}, s)

	fb := NewFrameBuffer(0, 0, 7, 7)
	pe := NewPaintEngine()
	pe.DebugXRay = true

	pe.PaintFragment(nil, frag, geom.Point{X: 1, Y: 1}, fb)

	// Check colors
	// Margin area & Border area: Red (100, 0, 0)
	// Padding area: Green (0, 100, 0)
	// Content area: Blue (0, 0, 100)

	marginColor := color.RGBA{100, 0, 0, 255}
	paddingColor := color.RGBA{0, 100, 0, 255}
	contentColor := color.RGBA{0, 0, 100, 255}

	// Cell (0,0) should be margin color
	if c := fb.CellAt(0, 0).BG; c != marginColor {
		t.Errorf("expected margin color at (0,0), got %v", c)
	}

	// Cell (1,1) is border, should be margin color (since it's outside padding box)
	if c := fb.CellAt(1, 1).BG; c != marginColor {
		t.Errorf("expected margin color at (1,1) [border], got %v", c)
	}

	// Cell (2,2) should be padding color
	if c := fb.CellAt(2, 2).BG; c != paddingColor {
		t.Errorf("expected padding color at (2,2), got %v", c)
	}

	// Cell (3,3) should be content color
	if c := fb.CellAt(3, 3).BG; c != contentColor {
		t.Errorf("expected content color at (3,3), got %v", c)
	}
}

func TestPaint_DebugXRay_Clipping(t *testing.T) {
	// Parent: 5x5, overflow: hidden
	// Child: at (4,4) with margin 2.
	// Child's Margin Box starts at (2,2) relative to parent origin.
	// Parent size 5x5.
	// Child margin box should be clipped by parent content box at x=5, y=5.

	parentStyle := &style.Computed{OverflowX: style.OverflowHidden, OverflowY: style.OverflowHidden}
	childStyle := &style.Computed{Margin: style.EdgeAll(2)}

	childFrag := makeBoxFrag(geom.Size{Width: 1, Height: 1}, childStyle)
	parentFrag := makeBoxFrag(geom.Size{Width: 5, Height: 5}, parentStyle,
		layout.FragmentLink{Offset: geom.Point{X: 4, Y: 4}, Fragment: childFrag})

	fb := NewFrameBuffer(0, 0, 10, 10)
	pe := NewPaintEngine()
	pe.DebugXRay = true

	pe.PaintFragment(nil, parentFrag, geom.Point{X: 0, Y: 0}, fb)

	marginColor := color.RGBA{100, 0, 0, 255}

	// (2,2) should be margin color (it's inside parent 5x5)
	if c := fb.CellAt(2, 2).BG; c != marginColor {
		t.Errorf("expected margin color at (2,2), got %v", c)
	}

	// (5,2) should be nil (clipped by parent)
	if c := fb.CellAt(5, 2).BG; c != nil {
		t.Errorf("expected clipped at (5,2), got %v", c)
	}
}

// =============================================================================
// drawBorder clip-stack tests
//
// These tests exercise the fix applied in paint/engine.go that clips the four
// border-drawing loops to the current clipStack rectangle.  Before the fix,
// every loop iterated over the full fragment width/height regardless of the
// visible clip region.  When a fragment had a pathologically large height
// (e.g. Percent(100) resolving to ≈ InfiniteBlockSize ≈ 1 073 741 823) the
// loops would iterate billions of times, hanging the paint engine.
//
// Each test drives PaintFragment so the clipStack is initialised exactly as
// the engine does in production (initialised to the framebuffer bounds).
// =============================================================================

// makeBorderedFrag builds a fragment with a single-line border and the given
// logical size.
func makeBorderedFrag(size geom.Size) *layout.Fragment {
	s := &style.Computed{
		Border: style.SingleBorder(),
	}
	return &layout.Fragment{
		Size: size,
		Node: &mockNode{s: s},
	}
}

// TestDrawBorder_FullyVisible verifies that when a bordered fragment fits
// entirely within the clip rect all four edges are drawn.
func TestDrawBorder_FullyVisible(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 5, Height: 3})
	fb := NewFrameBuffer(0, 0, 10, 5)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{}, fb)

	// Top edge: y=0, x in [0,4]
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 0).BorderStyle == BorderNone {
			t.Errorf("FullyVisible: expected border at (%d,0)", x)
		}
	}
	// Bottom edge: y=2
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 2).BorderStyle == BorderNone {
			t.Errorf("FullyVisible: expected border at (%d,2)", x)
		}
	}
	// Left edge: x=0
	for y := 0; y < 3; y++ {
		if fb.CellAt(0, y).BorderStyle == BorderNone {
			t.Errorf("FullyVisible: expected border at (0,%d)", y)
		}
	}
	// Right edge: x=4
	for y := 0; y < 3; y++ {
		if fb.CellAt(4, y).BorderStyle == BorderNone {
			t.Errorf("FullyVisible: expected border at (4,%d)", y)
		}
	}
}

// TestDrawBorder_TopEdgeClipped verifies that the top border row is not drawn
// when it falls above the clip rect (framebuffer starts at y=1).
func TestDrawBorder_TopEdgeClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 5, Height: 3})
	// Framebuffer starts at absolute y=1; the fragment is painted at y=0.
	fb := NewFrameBuffer(0, 1, 5, 3)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{X: 0, Y: 0}, fb)

	// y=0 is outside the framebuffer → CellAt returns an empty Cell.
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 0).BorderStyle != BorderNone {
			t.Errorf("TopEdgeClipped: unexpected border at (%d,0)", x)
		}
	}
	// Bottom edge (y=2) is inside the framebuffer.
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 2).BorderStyle == BorderNone {
			t.Errorf("TopEdgeClipped: expected bottom border at (%d,2)", x)
		}
	}
}

// TestDrawBorder_BottomEdgeClipped verifies that the bottom border row is not
// drawn when it falls below the clip rect (framebuffer only 3 rows, fragment 4 tall).
func TestDrawBorder_BottomEdgeClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 5, Height: 4})
	fb := NewFrameBuffer(0, 0, 5, 3) // rows 0-2 visible; row 3 clipped
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{}, fb)

	// Bottom edge (y=3) must not appear.
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 3).BorderStyle != BorderNone {
			t.Errorf("BottomEdgeClipped: unexpected border at (%d,3)", x)
		}
	}
	// Top edge (y=0) must still be present.
	for x := 0; x < 5; x++ {
		if fb.CellAt(x, 0).BorderStyle == BorderNone {
			t.Errorf("BottomEdgeClipped: expected top border at (%d,0)", x)
		}
	}
}

// TestDrawBorder_LeftEdgeClipped verifies that the left border column is not
// drawn when the fragment is painted at x=-1 (left column outside framebuffer).
func TestDrawBorder_LeftEdgeClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 5, Height: 3})
	fb := NewFrameBuffer(0, 0, 10, 5)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{X: -1, Y: 0}, fb)

	// x=-1 is outside framebuffer → empty.
	for y := 0; y < 3; y++ {
		if fb.CellAt(-1, y).BorderStyle != BorderNone {
			t.Errorf("LeftEdgeClipped: unexpected border at (-1,%d)", y)
		}
	}
	// Right edge is at abs x = -1+5-1 = 3, which is inside the framebuffer.
	for y := 0; y < 3; y++ {
		if fb.CellAt(3, y).BorderStyle == BorderNone {
			t.Errorf("LeftEdgeClipped: expected right border at (3,%d)", y)
		}
	}
}

// TestDrawBorder_RightEdgeClipped verifies that the right border column is not
// drawn when it falls beyond the framebuffer (fragment 8 wide, framebuffer 5 wide).
func TestDrawBorder_RightEdgeClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 8, Height: 3})
	fb := NewFrameBuffer(0, 0, 5, 3) // columns 5-7 clipped
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{}, fb)

	// Right edge at x=7 must not appear.
	for y := 0; y < 3; y++ {
		if fb.CellAt(7, y).BorderStyle != BorderNone {
			t.Errorf("RightEdgeClipped: unexpected border at (7,%d)", y)
		}
	}
	// Left edge (x=0) must still be present.
	for y := 0; y < 3; y++ {
		if fb.CellAt(0, y).BorderStyle == BorderNone {
			t.Errorf("RightEdgeClipped: expected left border at (0,%d)", y)
		}
	}
}

// TestDrawBorder_HorizontalEdgesPartiallyClipped verifies that horizontal
// edges (top/bottom) are drawn only for the x-range that overlaps the clip.
//
// Fragment: 10 wide × 3 tall, painted at x=0.
// Framebuffer: 6 wide, starting at absolute x=2 → clip x=[2,7].
// Expected: top/bottom border only at x=2..7.
func TestDrawBorder_HorizontalEdgesPartiallyClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 10, Height: 3})
	fb := NewFrameBuffer(2, 0, 6, 3) // absolute x: 2..7
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{X: 0, Y: 0}, fb)

	// x=2..7 (inside framebuffer) on top row should have borders.
	for x := 2; x < 8; x++ {
		if fb.CellAt(x, 0).BorderStyle == BorderNone {
			t.Errorf("HorizPartialClip: expected top border at (%d,0)", x)
		}
	}
	// x=0..1 are outside framebuffer → empty.
	for x := 0; x < 2; x++ {
		if fb.CellAt(x, 0).BorderStyle != BorderNone {
			t.Errorf("HorizPartialClip: unexpected border at (%d,0)", x)
		}
	}
	// x=8..9 are outside framebuffer → empty.
	for x := 8; x < 10; x++ {
		if fb.CellAt(x, 0).BorderStyle != BorderNone {
			t.Errorf("HorizPartialClip: unexpected border at (%d,0)", x)
		}
	}
}

// TestDrawBorder_VerticalEdgesPartiallyClipped verifies that vertical edges
// (left/right) are drawn only for the y-range that overlaps the clip.
//
// Fragment: 3 wide × 10 tall, painted at y=0.
// Framebuffer: 3 wide × 6 tall, starting at absolute y=2 → clip y=[2,7].
// Expected: left/right border only at y=2..7.
func TestDrawBorder_VerticalEdgesPartiallyClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 3, Height: 10})
	fb := NewFrameBuffer(0, 2, 3, 6) // absolute y: 2..7
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{X: 0, Y: 0}, fb)

	// Rows 2..7 should have left border.
	for y := 2; y < 8; y++ {
		if fb.CellAt(0, y).BorderStyle == BorderNone {
			t.Errorf("VertPartialClip: expected left border at (0,%d)", y)
		}
	}
	// Rows 0..1 outside framebuffer → empty.
	for y := 0; y < 2; y++ {
		if fb.CellAt(0, y).BorderStyle != BorderNone {
			t.Errorf("VertPartialClip: unexpected border at (0,%d)", y)
		}
	}
	// Rows 8..9 outside framebuffer → empty.
	for y := 8; y < 10; y++ {
		if fb.CellAt(0, y).BorderStyle != BorderNone {
			t.Errorf("VertPartialClip: unexpected border at (0,%d)", y)
		}
	}
}

// TestDrawBorder_HugeHeightClippedToViewport is the direct regression test for
// the infinite-loop hang: before the fix, a fragment with height ≈ 1 billion
// caused the vertical border loops to iterate billions of times.
//
// This test uses height=1_000_000 inside a 5-row viewport. It MUST complete in
// milliseconds; a re-introduced regression would cause a multi-second hang that
// the Go test timeout would catch.
func TestDrawBorder_HugeHeightClippedToViewport(t *testing.T) {
	const hugeHeight = 1_000_000
	const viewportH = 5
	const viewportW = 20

	frag := makeBorderedFrag(geom.Size{Width: viewportW, Height: hugeHeight})
	fb := NewFrameBuffer(0, 0, viewportW, viewportH)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{}, fb)

	// Top border row (y=0) is within the viewport.
	for x := 0; x < viewportW; x++ {
		if fb.CellAt(x, 0).BorderStyle == BorderNone {
			t.Errorf("HugeHeight: expected top border at (%d,0)", x)
		}
	}
	// Rows 1..viewportH-2 are interior: only left (x=0) and right (x=w-1) borders.
	for y := 1; y < viewportH-1; y++ {
		if fb.CellAt(0, y).BorderStyle == BorderNone {
			t.Errorf("HugeHeight: expected left border at (0,%d)", y)
		}
		if fb.CellAt(viewportW-1, y).BorderStyle == BorderNone {
			t.Errorf("HugeHeight: expected right border at (%d,%d)", viewportW-1, y)
		}
		for x := 1; x < viewportW-1; x++ {
			if fb.CellAt(x, y).BorderStyle != BorderNone {
				t.Errorf("HugeHeight: unexpected interior border at (%d,%d)", x, y)
			}
		}
	}
	// Bottom border (y=hugeHeight-1) is far outside the viewport → not written.
	for x := 0; x < viewportW; x++ {
		if fb.CellAt(x, hugeHeight-1).BorderStyle != BorderNone {
			t.Errorf("HugeHeight: unexpected bottom border at (%d,%d)", x, hugeHeight-1)
		}
	}
}

// TestDrawBorder_HugeWidthClippedToViewport mirrors the height regression test
// for the horizontal axis.
func TestDrawBorder_HugeWidthClippedToViewport(t *testing.T) {
	const hugeWidth = 1_000_000
	const viewportW = 5
	const viewportH = 20

	frag := makeBorderedFrag(geom.Size{Width: hugeWidth, Height: viewportH})
	fb := NewFrameBuffer(0, 0, viewportW, viewportH)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{}, fb)

	// Left edge (x=0) is in viewport for all rows.
	for y := 0; y < viewportH; y++ {
		if fb.CellAt(0, y).BorderStyle == BorderNone {
			t.Errorf("HugeWidth: expected left border at (0,%d)", y)
		}
	}
	// Top edge: columns 1..viewportW-1 in row 0 should be border.
	for x := 1; x < viewportW; x++ {
		if fb.CellAt(x, 0).BorderStyle == BorderNone {
			t.Errorf("HugeWidth: expected top border at (%d,0)", x)
		}
	}
	// Right border (x=hugeWidth-1) is far outside → not written.
	for y := 0; y < viewportH; y++ {
		if fb.CellAt(hugeWidth-1, y).BorderStyle != BorderNone {
			t.Errorf("HugeWidth: unexpected right border at (%d,%d)", hugeWidth-1, y)
		}
	}
}

// TestDrawBorder_AllEdgesClipped verifies that no cells are written when the
// entire fragment is painted off-screen (origin far below the framebuffer).
func TestDrawBorder_AllEdgesClipped(t *testing.T) {
	frag := makeBorderedFrag(geom.Size{Width: 5, Height: 3})
	fb := NewFrameBuffer(0, 0, 10, 5)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, frag, geom.Point{X: 0, Y: 10}, fb) // entirely below viewport

	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			if fb.CellAt(x, y).BorderStyle != BorderNone {
				t.Errorf("AllEdgesClipped: unexpected border at (%d,%d)", x, y)
			}
		}
	}
}

// TestDrawBorder_NestedClipRestrictsBorder verifies that a child border is
// clipped by the ancestor's overflow:hidden content-box, not just the global
// framebuffer bounds.
func TestDrawBorder_NestedClipRestrictsBorder(t *testing.T) {
	childStyle := &style.Computed{
		Border: style.SingleBorder(),
	}
	childFrag := &layout.Fragment{
		Size: geom.Size{Width: 8, Height: 4},
		Node: &mockNode{s: childStyle},
	}

	outerStyle := &style.Computed{
		Border:    style.SingleBorder(),
		OverflowX: style.OverflowHidden,
		OverflowY: style.OverflowHidden,
	}
	outerFrag := &layout.Fragment{
		Size: geom.Size{Width: 10, Height: 5},
		Node: &mockNode{s: outerStyle},
		Children: []layout.FragmentLink{
			{Offset: geom.Point{X: 1, Y: 1}, Fragment: childFrag},
		},
	}

	fb := NewFrameBuffer(0, 0, 15, 8)
	pe := NewPaintEngine()
	pe.PaintFragment(nil, outerFrag, geom.Point{}, fb)

	// Child's top border is at abs y=1 (outer origin 0 + child offset 1).
	// Outer content-box clip starts at y=1 (1-cell border), so y=1 is visible.
	for x := 1; x < 9; x++ {
		if fb.CellAt(x, 1).BorderStyle == BorderNone {
			t.Errorf("NestedClip: expected child top border at (%d,1)", x)
		}
	}
	// Nothing should be painted beyond the outer fragment's bounds (y >= 5).
	for x := 0; x < 15; x++ {
		if fb.CellAt(x, 5).BorderStyle != BorderNone {
			t.Errorf("NestedClip: unexpected border outside outer at (%d,5)", x)
		}
	}
}
