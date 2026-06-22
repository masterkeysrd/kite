package layout

import (
	"testing"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// TestTableFragmentBuilder_PercentWidthHints verifies that percentage width hints on
// TD cells are honoured by distributeTableWidth:
//   - Columns with a small hint (e.g. 1 %) shrink to their content max.
//   - The column with the large hint (e.g. 100 %) absorbs all remaining space.
func TestTableFragmentBuilder_PercentWidthHints(t *testing.T) {
	// Two meta columns (content max = 10 each) and one name column (content max = 20).
	// Table is 80 cells wide.
	// Expected: meta cols → 10 each; name col → 80 - 10 - 10 = 60.
	colMinMax := []MinMaxSizes{
		{Min: 5, Max: 10},
		{Min: 5, Max: 10},
		{Min: 10, Max: 20},
	}
	colPercent := []float32{1, 1, 100}

	node := &mockNode{style: &style.Computed{Display: style.DisplayTable}}
	space := NewConstraintSpaceBuilder(geometry.Size{80, 20}).ToConstraintSpace()
	builder := NewTableFragmentBuilder(node, space)
	builder.colMinMax = colMinMax
	builder.colPercent = colPercent

	widths := builder.distributeTableWidth(colMinMax, colPercent, 80)

	if widths[0] != 10 {
		t.Errorf("meta col 0: expected 10, got %d", widths[0])
	}
	if widths[1] != 10 {
		t.Errorf("meta col 1: expected 10, got %d", widths[1])
	}
	if widths[2] != 60 {
		t.Errorf("name col: expected 60, got %d", widths[2])
	}
}

func TestTableFragmentBuilder_DistributeSpan(t *testing.T) {
	node := &mockNode{
		style: &style.Computed{Display: style.DisplayTable},
	}
	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	builder := NewTableFragmentBuilder(node, space)

	// initialize grid with 3 columns, empty min/max
	builder.colMinMax = []MinMaxSizes{
		{Min: 10, Max: 20},
		{Min: 10, Max: 20},
		{Min: 10, Max: 20},
	}

	cell := &mockNode{
		cachedMinMax: MinMaxSizes{Min: 40, Max: 80},
		minMaxValid:  true,
		style:        &style.Computed{Display: style.DisplayTableCell},
	}

	builder.DistributeSpan(nil, cell, 0, 3)

	if builder.colMinMax[0].Min != 14 { // 40 - 30 = 10 extra. 10/3 = 3, rem 1. So 10+4 = 14, 10+3 = 13, 10+3 = 13
		t.Errorf("Expected col 0 min 14, got %d", builder.colMinMax[0].Min)
	}
	if builder.colMinMax[1].Min != 13 {
		t.Errorf("Expected col 1 min 13, got %d", builder.colMinMax[1].Min)
	}
	if builder.colMinMax[2].Min != 13 {
		t.Errorf("Expected col 2 min 13, got %d", builder.colMinMax[2].Min)
	}

	if builder.colMinMax[0].Max != 27 { // 80 - 60 = 20 extra. 20/3 = 6, rem 2. So 20+7 = 27, 20+7 = 27, 20+6 = 26
		t.Errorf("Expected col 0 max 27, got %d", builder.colMinMax[0].Max)
	}
	if builder.colMinMax[1].Max != 27 {
		t.Errorf("Expected col 1 max 27, got %d", builder.colMinMax[1].Max)
	}
	if builder.colMinMax[2].Max != 26 {
		t.Errorf("Expected col 2 max 26, got %d", builder.colMinMax[2].Max)
	}
}

// TestTableFragmentBuilder_GetCellShift verifies the horizontal border-collapse
// shift logic: when two adjacent cells both have a shared border, the second
// cell's X coordinate must be reduced by 1 so the border characters overlap.
// The first cell in any row never gets a shift because the row's own left border
// is handled by the caller as an initial X inset (matching block-layout insetX).
func TestTableFragmentBuilder_GetCellShift(t *testing.T) {
	tableNode := &mockNode{
		style: &style.Computed{Display: style.DisplayTable},
	}
	space := NewConstraintSpaceBuilder(geometry.Size{100, 100}).ToConstraintSpace()
	builder := NewTableFragmentBuilder(tableNode, space)

	t.Run("FirstCell_NoBorders_NoShift", func(t *testing.T) {
		builder.ResetRow()
		shift := builder.GetCellShift(0, 1, false, false)
		if shift != 0 {
			t.Errorf("expected shift 0 for first cell with no borders, got %d", shift)
		}
	})

	t.Run("FirstCell_WithBorders_NoShift", func(t *testing.T) {
		// The first cell's left border is handled by the row's initial X inset;
		// GetCellShift must not add any extra shift for column 0.
		builder.ResetRow()
		shift := builder.GetCellShift(0, 1, true, true)
		if shift != 0 {
			t.Errorf("expected shift 0 for first cell (row inset handles left border), got %d", shift)
		}
	})

	t.Run("SecondCell_BothHaveRightLeftBorder_ShiftOne", func(t *testing.T) {
		// Cell at col 0 has a right border; cell at col 1 has a left border.
		// Their shared edge should collapse: GetCellShift must return 1.
		builder.ResetRow()
		// Record that col 0 ended with a right border.
		_ = builder.GetCellShift(0, 1, false, true)
		// Now ask for the shift of the cell at col 1 with a left border.
		shift := builder.GetCellShift(1, 1, true, false)
		if shift != 1 {
			t.Errorf("expected shift 1 when adjacent cells share a border, got %d", shift)
		}
	})

	t.Run("SecondCell_NoRightBorderOnFirst_NoShift", func(t *testing.T) {
		// Cell at col 0 has NO right border: no collapse should occur.
		builder.ResetRow()
		_ = builder.GetCellShift(0, 1, false, false)
		shift := builder.GetCellShift(1, 1, true, false)
		if shift != 0 {
			t.Errorf("expected shift 0 when first cell has no right border, got %d", shift)
		}
	})

	t.Run("SecondCell_NoLeftBorderOnSecond_NoShift", func(t *testing.T) {
		// Cell at col 1 has NO left border: no collapse should occur.
		builder.ResetRow()
		_ = builder.GetCellShift(0, 1, false, true)
		shift := builder.GetCellShift(1, 1, false, false)
		if shift != 0 {
			t.Errorf("expected shift 0 when second cell has no left border, got %d", shift)
		}
	})

	t.Run("ResetRow_ClearsState", func(t *testing.T) {
		// After a ResetRow the right-border tracking is cleared.
		builder.ResetRow()
		_ = builder.GetCellShift(0, 1, false, true) // col 0 has right border
		builder.ResetRow()                          // reset clears tracking
		shift := builder.GetCellShift(1, 1, true, false)
		if shift != 0 {
			t.Errorf("expected shift 0 after ResetRow clears state, got %d", shift)
		}
	})
}
