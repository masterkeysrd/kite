package uv

import (
	"bytes"
	"strings"
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

func TestBackend_InlineMode_And_InsertAbove(t *testing.T) {
	var buf bytes.Buffer
	screen := uv.NewTerminalScreen(&buf, nil)
	b := &Backend{
		screen: screen,
		width:  80,
		height: 24,
	}

	// 1. Test SetInline(true)
	b.SetInline(true)
	if !b.inline {
		t.Fatalf("expected b.inline to be true")
	}
	if screen.MouseMode() != uv.MouseModeNone {
		t.Errorf("expected MouseModeNone when inline, got %v", screen.MouseMode())
	}

	// 2. Test InsertAbove(buf) directly when not started
	insertBuf := backend.NewBuffer(10, 2)
	insertBuf.Set(0, 0, backend.Cell{Content: "H", Width: 1})
	insertBuf.Set(1, 0, backend.Cell{Content: "i", Width: 1})
	insertBuf.Set(0, 1, backend.Cell{Content: "!", Width: 1})

	b.InsertAbove(insertBuf)

	output := buf.String()
	if !strings.Contains(output, "Hi") || !strings.Contains(output, "!") {
		t.Errorf("expected inserted content to be written to screen, got %q", output)
	}
}

func TestBackend_InsertAbove_ThroughRenderCh(t *testing.T) {
	var out bytes.Buffer
	screen := uv.NewTerminalScreen(&out, nil)
	b := &Backend{
		screen:   screen,
		renderCh: make(chan renderRequest, 4),
		width:    80,
		height:   24,
	}
	b.started.Store(true)
	b.renderWG.Add(1)
	go b.renderLoop()

	insertBuf := backend.NewBuffer(5, 1)
	insertBuf.Set(0, 0, backend.Cell{Content: "A", Width: 1})
	b.InsertAbove(insertBuf)

	// Close render loop gracefully and wait for renderLoop to finish processing
	b.stopRenderLoop()

	if !strings.Contains(out.String(), "A") {
		t.Errorf("expected 'A' in output from render loop, got %q", out.String())
	}
}
