package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

type mockOverlayNode struct {
	mockNode
	anchor    dom.Node
	placement geom.Placement
	flip      bool
}

func (m *mockOverlayNode) Anchor() dom.Node          { return m.anchor }
func (m *mockOverlayNode) Placement() geom.Placement { return m.placement }
func (m *mockOverlayNode) Flip() bool                { return m.flip }
func (m *mockOverlayNode) LogicalNode() dom.Node     { return m }

type mockAnchor struct {
	mockNode
	rect geom.Rect
}

func (a *mockAnchor) GetBoundingClientRect() (geom.Rect, bool) {
	return a.rect, true
}

func TestOverlayAlgorithm_BestFitChoosing(t *testing.T) {
	// Viewport 80x24.
	viewport := geom.Size{Width: 80, Height: 24}
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
	anchorV := &mockAnchor{rect: geom.Rect{Origin: geom.Point{X: 10, Y: 5}, Size: geom.Size{Width: 10, Height: 2}}}
	nodeV := &mockOverlayNode{
		mockNode: mockNode{
			style: &style.Computed{
				Display: style.DisplayInlineBlock,
			},
			firstChild: content,
		},
		anchor:    anchorV,
		placement: geom.PlacementTop,
		flip:      true,
	}

	algoV := GetAlgorithm(nodeV)
	algoV.Layout(nil, nodeV, space)

	// Expected Y: anchor.Y + anchor.Height = 5 + 2 = 7
	if nodeV.cachedFragment.Node != (Node)(nodeV) {
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
	offset geom.Point
}

func (m *capturedOffsetNode) SetOffset(p geom.Point) { m.offset = p }

func TestOverlayAlgorithm_BestFitLogic(t *testing.T) {
	viewport := geom.Size{Width: 80, Height: 24}
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
		anchor    geom.Rect
		placement geom.Placement
		expected  geom.Point
	}{
		{
			name:      "Vertical flip to Bottom (more space)",
			anchor:    geom.Rect{Origin: geom.Point{X: 10, Y: 5}, Size: geom.Size{Width: 10, Height: 2}},
			placement: geom.PlacementTop,
			expected:  geom.Point{X: 10, Y: 7}, // anchor.Y + anchor.H
		},
		{
			name:      "Vertical flip to Top (more space)",
			anchor:    geom.Rect{Origin: geom.Point{X: 10, Y: 15}, Size: geom.Size{Width: 10, Height: 2}},
			placement: geom.PlacementBottom,
			expected:  geom.Point{10, 5}, // anchor.Y - content.H
		},
		{
			name:      "Horizontal flip to Right (more space)",
			anchor:    geom.Rect{Origin: geom.Point{5, 10}, Size: geom.Size{2, 10}},
			placement: geom.PlacementLeft,
			expected:  geom.Point{7, 10}, // anchor.X + anchor.W
		},
		{
			name:      "Horizontal flip to Left (more space)",
			anchor:    geom.Rect{Origin: geom.Point{65, 10}, Size: geom.Size{2, 10}},
			placement: geom.PlacementRight,
			expected:  geom.Point{67, 10}, // Does not flip because it fits at Right.
		},
		{
			name:      "Horizontal flip to Right (more space) when at right edge",
			anchor:    geom.Rect{Origin: geom.Point{75, 10}, Size: geom.Size{2, 10}},
			placement: geom.PlacementRight,
			expected:  geom.Point{65, 10}, // Still flips to Left because Right overflows.
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

			algo := GetAlgorithm(node)

			// Manually set the cached fragment on the content node so OverlayAlgorithm
			// can measure it without running a full block layout.
			content.SetCachedLayout(ConstraintSpace{}, &Fragment{Size: geom.Size{10, 10}})
			// We must ALSO set it on the overlay node itself because OverlayAlgorithm
			// will try to use BlockAlgorithm on node.
			node.SetCachedLayout(space, &Fragment{Size: geom.Size{10, 10}})

			algo.Layout(nil, node, space)

			if node.offset != tt.expected {
				t.Errorf("expected offset %v, got %v", tt.expected, node.offset)
			}
		})
	}
}
