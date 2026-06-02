package style

import (
	"reflect"
	"testing"
)

func TestGridStyle_MergeAndResolve(t *testing.T) {
	s1 := Style{
		display:             Some(DisplayGrid),
		gridTemplateColumns: Some(Repeat(2, Fr(1))),
		gridColumnGap:       Some(2),
	}
	s2 := Style{
		gridRowGap:          Some(1),
		gridTemplateColumns: Some([]GridTrackSize{Cells(10), Auto}),
	}

	merged := s1.Merge(s2)

	if merged.display.Value() != DisplayGrid {
		t.Errorf("Expected DisplayGrid")
	}
	if merged.gridRowGap.Value() != 1 {
		t.Errorf("Expected GridRowGap 1")
	}
	if merged.gridColumnGap.Value() != 2 {
		t.Errorf("Expected GridColumnGap 2")
	}

	// Test resolver apply directly
	c := DefaultStyle()
	c = merged.Apply(c)

	if c.Display != DisplayGrid {
		t.Errorf("Resolver expected DisplayGrid, got %v", c.Display)
	}
	if c.GridColumnGap != 2 {
		t.Errorf("Resolver expected GridColumnGap 2, got %d", c.GridColumnGap)
	}
	if c.GridRowGap != 1 {
		t.Errorf("Resolver expected GridRowGap 1, got %d", c.GridRowGap)
	}

	expectedCols := []GridTrackSize{Cells(10), Auto}
	if !reflect.DeepEqual(c.GridTemplateColumns, expectedCols) {
		t.Errorf("Resolver expected GridTemplateColumns %v, got %v", expectedCols, c.GridTemplateColumns)
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name  string
		count int
		sizes []GridTrackSize
		want  []GridTrackSize
	}{
		{
			name:  "Repeat 3 Fr(1)",
			count: 3,
			sizes: []GridTrackSize{Fr(1)},
			want:  []GridTrackSize{Fr(1), Fr(1), Fr(1)},
		},
		{
			name:  "Repeat 2 Cells(10) Auto",
			count: 2,
			sizes: []GridTrackSize{Cells(10), Auto},
			want:  []GridTrackSize{Cells(10), Auto, Cells(10), Auto},
		},
		{
			name:  "Zero count",
			count: 0,
			sizes: []GridTrackSize{Fr(1)},
			want:  nil,
		},
		{
			name:  "Negative count",
			count: -1,
			sizes: []GridTrackSize{Fr(1)},
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
			got := Repeat(tt.count, tt.sizes...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Repeat(%d, %v) = %v, want %v", tt.count, tt.sizes, got, tt.want)
			}
		})
	}
}
