package layout

import (
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestResolveTrackSizes(t *testing.T) {
	tests := []struct {
		name      string
		templates []style.GridTrackSize
		available int
		gap       int
		want      []int
	}{
		{
			name:      "Fixed sizes",
			templates: []style.GridTrackSize{style.Cells(10), style.Cells(20)},
			available: 100,
			gap:       0,
			want:      []int{10, 20},
		},
		{
			name:      "Percentage sizes",
			templates: []style.GridTrackSize{style.Percent(50), style.Percent(25)},
			available: 100,
			gap:       0,
			want:      []int{50, 25},
		},
		{
			name:      "Mixed with gap",
			templates: []style.GridTrackSize{style.Cells(10), style.Percent(50)},
			available: 100,
			gap:       5,
			// contentAvailable = 100 - 5 = 95
			// Percent(50) of 95 = 47
			want: []int{10, 47},
		},
		{
			name:      "Auto and Fr reserved",
			templates: []style.GridTrackSize{style.Auto, style.Fr(1)},
			available: 100,
			gap:       0,
			want:      []int{0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveTrackSizes(tt.templates, tt.available, tt.gap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTrackSizes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGridBuilder_PlaceItems(t *testing.T) {
	// Helper to create nodes with specific grid styles
	newNode := func(col, row style.GridPlacement) *mockNode {
		return &mockNode{
			style: &style.Computed{
				GridColumn: col,
				GridRow:    row,
			},
		}
	}

	tests := []struct {
		name     string
		tmplCols []style.GridTrackSize
		children []*mockNode
		want     []gridItem
	}{
		{
			name: "Fully explicit",
			children: []*mockNode{
				newNode(style.GridPlacement{Start: 1}, style.GridPlacement{Start: 1}),
				newNode(style.GridPlacement{Start: 2}, style.GridPlacement{Start: 2}),
			},
			want: []gridItem{
				{colStart: 0, colSpan: 1, rowStart: 0, rowSpan: 1},
				{colStart: 1, colSpan: 1, rowStart: 1, rowSpan: 1},
			},
		},
		{
			name: "Mixed explicit and implicit",
			children: []*mockNode{
				// Item 1: Explicitly at (1, 1)
				newNode(style.GridPlacement{Start: 1}, style.GridPlacement{Start: 1}),
				// Item 2: Auto
				newNode(style.GridPlacement{}, style.GridPlacement{}),
			},
			tmplCols: []style.GridTrackSize{style.Auto, style.Auto},
			want: []gridItem{
				{colStart: 0, colSpan: 1, rowStart: 0, rowSpan: 1},
				{colStart: 1, colSpan: 1, rowStart: 0, rowSpan: 1},
			},
		},
		{
			name: "Implicit with spans",
			children: []*mockNode{
				// Item 1: span 2 cols
				newNode(style.GridPlacement{Span: 2}, style.GridPlacement{}),
				// Item 2: auto
				newNode(style.GridPlacement{}, style.GridPlacement{}),
			},
			tmplCols: []style.GridTrackSize{style.Auto, style.Auto},
			want: []gridItem{
				{colStart: 0, colSpan: 2, rowStart: 0, rowSpan: 1},
				{colStart: 0, colSpan: 1, rowStart: 1, rowSpan: 1},
			},
		},
		{
			name: "Explicit overlaps / gaps",
			children: []*mockNode{
				// Item 1: (2, 1)
				newNode(style.GridPlacement{Start: 2}, style.GridPlacement{Start: 1}),
				// Item 2: Auto
				newNode(style.GridPlacement{}, style.GridPlacement{}),
			},
			want: []gridItem{
				{colStart: 1, colSpan: 1, rowStart: 0, rowSpan: 1},
				{colStart: 0, colSpan: 1, rowStart: 0, rowSpan: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Link children
			var first *mockNode
			if len(tt.children) > 0 {
				first = tt.children[0]
				for i := 0; i < len(tt.children)-1; i++ {
					tt.children[i].nextSibling = tt.children[i+1]
				}
			}

			parent := &mockNode{
				style: &style.Computed{
					GridTemplateColumns: tt.tmplCols,
				},
				firstChild: first,
			}

			builder := NewGridBuilder(parent, ConstraintSpace{})
			builder.PlaceItems()

			if len(builder.items) != len(tt.want) {
				t.Fatalf("expected %d items, got %d", len(tt.want), len(builder.items))
			}

			for i, item := range builder.items {
				want := tt.want[i]
				if item.colStart != want.colStart || item.colSpan != want.colSpan ||
					item.rowStart != want.rowStart || item.rowSpan != want.rowSpan {
					t.Errorf("item %d: got {col:%d span:%d, row:%d span:%d}, want {col:%d span:%d, row:%d span:%d}",
						i, item.colStart, item.colSpan, item.rowStart, item.rowSpan,
						want.colStart, want.colSpan, want.rowStart, want.rowSpan)
				}
			}
		})
	}
}
