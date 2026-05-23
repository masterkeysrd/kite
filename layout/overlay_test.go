package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

type mockOverlayNode struct {
	mockNode
	anchor    any
	placement OverlayPlacement
	flip      bool
}

func (m *mockOverlayNode) Anchor() any                 { return m.anchor }
func (m *mockOverlayNode) Placement() OverlayPlacement { return m.placement }
func (m *mockOverlayNode) Flip() bool                  { return m.flip }
func (m *mockOverlayNode) LogicalNode() any            { return m }

type mockAnchor struct {
	rect Rect
}

func (a *mockAnchor) GetBoundingClientRect() (Rect, bool) {
	return a.rect, true
}

func TestOverlayAlgorithm_BestFitChoosing(t *testing.T) {
	// Viewport 80x24.
	viewport := Size{80, 24}
	space := NewConstraintSpaceBuilder(viewport).
		SetContainingSpace(viewport).
		SetContainerSpace(viewport).
		ToConstraintSpace()

	// Content 10x10.
	content := &mockNode{
		style: &style.Computed{
			Width:  style.Cells(10),
			Height: style.Cells(10),
		},
	}

	// 1. Vertical Best Fit: Anchor at Y=5, Height=2.
	// Top space: 5, Bottom space: 24 - 7 = 17.
	// Primary placement: Top.
	// Should flip to Bottom.
	anchorV := &mockAnchor{rect: Rect{Origin: Point{10, 5}, Size: Size{10, 2}}}
	nodeV := &mockOverlayNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInlineBlock,
			},
			firstChild: content,
		},
		anchor:    anchorV,
		placement: PlacementTop,
		flip:      true,
	}

	algoV := &OverlayAlgorithm{Node: nodeV, Space: space}
	algoV.Layout()

	// Expected Y: anchor.Y + anchor.Height = 5 + 2 = 7
	if nodeV.cachedFragment.Node != nodeV {
		t.Fatal("Layout did not store fragment on node")
	}
	if nodeV.cachedSpace != space {
		t.Fatal("Layout did not store space on node")
	}

	// Wait, OverlayAlgorithm sets the offset on the Node directly via SetOffset.
	// We need to capture that in mockNode.
}

type capturedOffsetNode struct {
	mockOverlayNode
	offset Point
}

func (m *capturedOffsetNode) SetOffset(p Point) { m.offset = p }

func TestOverlayAlgorithm_BestFitLogic(t *testing.T) {
	viewport := Size{80, 24}
	space := NewConstraintSpaceBuilder(viewport).
		SetContainingSpace(viewport).
		SetContainerSpace(viewport).
		ToConstraintSpace()

	contentStyle := &style.Computed{
		Width:  style.Cells(10),
		Height: style.Cells(10),
	}

	tests := []struct {
		name      string
		anchor    Rect
		placement OverlayPlacement
		expected  Point
	}{
		{
			name:      "Vertical flip to Bottom (more space)",
			anchor:    Rect{Origin: Point{10, 5}, Size: Size{10, 2}},
			placement: PlacementTop,
			expected:  Point{10, 7}, // anchor.Y + anchor.H
		},
		{
			name:      "Vertical flip to Top (more space)",
			anchor:    Rect{Origin: Point{10, 15}, Size: Size{10, 2}},
			placement: PlacementBottom,
			expected:  Point{10, 5}, // anchor.Y - content.H
		},
		{
			name:      "Horizontal flip to Right (more space)",
			anchor:    Rect{Origin: Point{5, 10}, Size: Size{2, 10}},
			placement: PlacementLeft,
			expected:  Point{7, 10}, // anchor.X + anchor.W
		},
		{
			name:      "Horizontal flip to Left (more space)",
			anchor:    Rect{Origin: Point{65, 10}, Size: Size{2, 10}},
			placement: PlacementRight,
			expected:  Point{67, 10}, // Does not flip because it fits at Right.
		},
		{
			name:      "Horizontal flip to Right (more space) when at right edge",
			anchor:    Rect{Origin: Point{75, 10}, Size: Size{2, 10}},
			placement: PlacementRight,
			expected:  Point{65, 10}, // Still flips to Left because Right overflows.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &mockNode{style: contentStyle}
			node := &capturedOffsetNode{
				mockOverlayNode: mockOverlayNode{
					mockNode: mockNode{
						style: &style.Computed{
							Display: style.DisplayInlineBlock,
						},
						firstChild: content,
					},
					anchor:    &mockAnchor{rect: tt.anchor},
					placement: tt.placement,
					flip:      true,
				},
			}

			algo := &OverlayAlgorithm{Node: node, Space: space}

			// Manually set the cached fragment on the content node so OverlayAlgorithm
			// can measure it without running a full block layout.
			content.SetCachedLayout(ConstraintSpace{}, &Fragment{Size: Size{10, 10}})
			// We must ALSO set it on the overlay node itself because OverlayAlgorithm
			// will try to use BlockAlgorithm on node.
			node.SetCachedLayout(space, &Fragment{Size: Size{10, 10}})

			algo.Layout()

			if node.offset != tt.expected {
				t.Errorf("expected offset %v, got %v", tt.expected, node.offset)
			}
		})
	}
}
