package uv

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/masterkeysrd/kite/backend"
)

func TestPopulateUVCell(t *testing.T) {
	var uvCell uv.Cell

	// 1. Populate a cell with Reverse attribute
	c1 := backend.Cell{
		Content: "A",
		Style:   backend.CellReverse,
	}
	populateUVCell(&uvCell, c1)

	if uvCell.Style.Attrs&uv.AttrReverse == 0 {
		t.Errorf("Expected uv.AttrReverse to be set, but got Attrs=%d", uvCell.Style.Attrs)
	}

	// 2. Populate another cell without Reverse attribute.
	// Since uvCell is reused, if populateUVCell doesn't clear Attrs, AttrReverse will leak!
	c2 := backend.Cell{
		Content: "B",
		Style:   0,
	}
	populateUVCell(&uvCell, c2)

	if uvCell.Style.Attrs&uv.AttrReverse != 0 {
		t.Errorf("Style leaked: expected AttrReverse to be cleared, but got Attrs=%d", uvCell.Style.Attrs)
	}
}
