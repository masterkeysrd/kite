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

func TestTranslateProgramStatusSupportEvent(t *testing.T) {
	// Value event
	ev1 := uv.ProgramStatusSupportEvent{}
	raw1 := translateEvent(ev1)
	if _, ok := raw1.(*backend.RawProgramStatusSupportEvent); !ok {
		t.Fatalf("expected *backend.RawProgramStatusSupportEvent, got %T", raw1)
	}

	// Pointer event
	ev2 := &uv.ProgramStatusSupportEvent{}
	raw2 := translateEvent(ev2)
	if _, ok := raw2.(*backend.RawProgramStatusSupportEvent); !ok {
		t.Fatalf("expected *backend.RawProgramStatusSupportEvent, got %T", raw2)
	}
}

func TestBackend_SetProgramStatus(t *testing.T) {
	screen := uv.NewTerminalScreen(nil, nil)
	b := &Backend{
		screen: screen,
	}

	b.SetProgramStatus(&backend.ProgramStatus{
		State: backend.ProgramStateWorking,
		Title: "Task 1",
	})

	ps := screen.ProgramStatus()
	if ps == nil || ps.State != uv.ProgramStateWorking || ps.Title != "Task 1" {
		t.Fatalf("expected screen to have program status, got %#v", ps)
	}

	b.SetProgramStatus(nil)
	if screen.ProgramStatus() != nil {
		t.Fatalf("expected screen program status to be nil after clear, got %#v", screen.ProgramStatus())
	}
}
