package style_test

import (
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestGridStyle_MergeAndResolve(t *testing.T) {
	s1 := style.Style{
		Display:             style.Some(style.DisplayGrid),
		GridTemplateColumns: style.Some(style.Repeat(2, style.Fr(1))),
		GridColumnGap:       style.Some(2),
	}
	s2 := style.Style{
		GridRowGap:          style.Some(1),
		GridTemplateColumns: style.Some([]style.GridTrackSize{style.Cells(10), style.Auto}),
	}

	merged := s1.Merge(s2)

	if merged.Display.Value() != style.DisplayGrid {
		t.Errorf("Expected DisplayGrid")
	}
	if merged.GridRowGap.Value() != 1 {
		t.Errorf("Expected GridRowGap 1")
	}
	if merged.GridColumnGap.Value() != 2 {
		t.Errorf("Expected GridColumnGap 2")
	}

	// Test resolver apply directly
	c := style.DefaultStyle()
	c = merged.Apply(c)

	if c.Display != style.DisplayGrid {
		t.Errorf("Resolver expected DisplayGrid, got %v", c.Display)
	}
	if c.GridColumnGap != 2 {
		t.Errorf("Resolver expected GridColumnGap 2, got %d", c.GridColumnGap)
	}
	if c.GridRowGap != 1 {
		t.Errorf("Resolver expected GridRowGap 1, got %d", c.GridRowGap)
	}

	expectedCols := []style.GridTrackSize{style.Cells(10), style.Auto}
	if !reflect.DeepEqual(c.GridTemplateColumns, expectedCols) {
		t.Errorf("Resolver expected GridTemplateColumns %v, got %v", expectedCols, c.GridTemplateColumns)
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name  string
		count int
		sizes []style.GridTrackSize
		want  []style.GridTrackSize
	}{
		{
			name:  "Repeat 3 Fr(1)",
			count: 3,
			sizes: []style.GridTrackSize{style.Fr(1)},
			want:  []style.GridTrackSize{style.Fr(1), style.Fr(1), style.Fr(1)},
		},
		{
			name:  "Repeat 2 Cells(10) Auto",
			count: 2,
			sizes: []style.GridTrackSize{style.Cells(10), style.Auto},
			want:  []style.GridTrackSize{style.Cells(10), style.Auto, style.Cells(10), style.Auto},
		},
		{
			name:  "Zero count",
			count: 0,
			sizes: []style.GridTrackSize{style.Fr(1)},
			want:  nil,
		},
		{
			name:  "Negative count",
			count: -1,
			sizes: []style.GridTrackSize{style.Fr(1)},
			want:  nil,
		},
		{
			name:  "Empty sizes",
			count: 3,
			sizes: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.Repeat(tt.count, tt.sizes...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Repeat(%d, %v) = %v, want %v", tt.count, tt.sizes, got, tt.want)
			}
		})
	}
}
