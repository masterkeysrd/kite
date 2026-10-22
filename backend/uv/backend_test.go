package uv

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
	"testing"
	"time"

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

	// 2. Test InsertAbove(buf) queues when not started
	insertBuf := backend.NewBuffer(10, 2)
	insertBuf.Set(0, 0, backend.Cell{Content: "H", Width: 1})
	insertBuf.Set(1, 0, backend.Cell{Content: "i", Width: 1})
	insertBuf.Set(0, 1, backend.Cell{Content: "!", Width: 1})

	b.InsertAbove(insertBuf)

	if buf.Len() != 0 {
		t.Errorf("expected no immediate terminal write before start, got %q", buf.String())
	}
	if len(b.pendingInsertAbove) != 1 {
		t.Fatalf("expected 1 pending buffer, got %d", len(b.pendingInsertAbove))
	}

	// 3. Test processInsertAbove directly
	b.processInsertAbove(b.pendingInsertAbove[0])
	output := buf.String()
	if !strings.Contains(output, "Hi") || !strings.Contains(output, "!") {
		t.Errorf("expected inserted content to be written to screen, got %q", output)
	}
}

func TestBackend_PendingInsertAbove_PreStart(t *testing.T) {
	var buf bytes.Buffer
	screen := uv.NewTerminalScreen(&buf, nil)
	b := &Backend{
		screen:   screen,
		renderCh: make(chan renderRequest, 256),
		width:    80,
		height:   24,
	}

	buf1 := backend.NewBuffer(10, 1)
	buf1.Set(0, 0, backend.Cell{Content: "1", Width: 1})
	buf2 := backend.NewBuffer(10, 1)
	buf2.Set(0, 0, backend.Cell{Content: "2", Width: 1})

	// Before Start, InsertAbove must queue into pendingInsertAbove without writing to terminal
	b.InsertAbove(buf1)
	b.InsertAbove(buf2)

	if buf.Len() != 0 {
		t.Fatalf("expected nothing written to terminal before start, got %q", buf.String())
	}
	if len(b.pendingInsertAbove) != 2 {
		t.Fatalf("expected 2 pending buffers, got %d", len(b.pendingInsertAbove))
	}

	// Now start renderLoop and verify pendingInsertAbove is drained
	b.started.Store(true)
	b.renderWG.Add(1)
	go b.renderLoop()

	b.stopRenderLoop()

	if len(b.pendingInsertAbove) != 0 {
		t.Fatalf("expected pendingInsertAbove to be drained, got %d", len(b.pendingInsertAbove))
	}
	output := buf.String()
	if !strings.Contains(output, "1") || !strings.Contains(output, "2") {
		t.Errorf("expected pending buffers to be processed, got %q", output)
	}
}

func TestBackend_ProcessInsertAbove_TrimmingTrailingSpaces(t *testing.T) {
	var buf bytes.Buffer
	screen := uv.NewTerminalScreen(&buf, nil)
	b := &Backend{
		screen: screen,
		width:  40,
		height: 10,
	}

	insertBuf := backend.NewBuffer(40, 2)
	// Line 0: "Hello" followed by empty spaces
	for i, ch := range "Hello" {
		insertBuf.Set(i, 0, backend.Cell{Content: string(ch), Width: 1})
	}
	// Line 1: "World" followed by empty spaces
	for i, ch := range "World" {
		insertBuf.Set(i, 1, backend.Cell{Content: string(ch), Width: 1})
	}

	b.processInsertAbove(insertBuf)

	output := buf.String()
	if !strings.Contains(output, "Hello\x1b[K\r\n") {
		t.Errorf("expected trimmed line with Hello\\x1b[K\\r\\n, got %q", output)
	}
	if !strings.Contains(output, "World\x1b[K\r\n") {
		t.Errorf("expected trimmed line with World\\x1b[K\\r\\n, got %q", output)
	}
	if strings.Contains(output, "Hello      ") {
		t.Errorf("expected trailing spaces to be trimmed, but found trailing spaces in %q", output)
	}
}

func TestBackend_ProcessInsertAbove_PreservesStyledSpaces(t *testing.T) {
	var buf bytes.Buffer
	screen := uv.NewTerminalScreen(&buf, nil)
	b := &Backend{
		screen: screen,
		width:  10,
		height: 5,
	}

	insertBuf := backend.NewBuffer(10, 1)
	insertBuf.Set(0, 0, backend.Cell{Content: "A", Width: 1})
	// Trailing space with background styling
	insertBuf.Set(1, 0, backend.Cell{Content: " ", Width: 1, Bg: color.RGBA{R: 255, A: 255}})

	b.processInsertAbove(insertBuf)

	output := buf.String()
	if !strings.Contains(output, "A") {
		t.Errorf("expected output to contain 'A', got %q", output)
	}
	// Styled space ANSI escape sequence should be present
	if !strings.Contains(output, "48;2;255;0;0") && !strings.Contains(output, "48;5;") {
		t.Errorf("expected styled space ANSI escape sequence, got %q", output)
	}
}

func TestBackend_ProcessInsertAbove_LongBufferExceedingTerminalHeight(t *testing.T) {
	var buf bytes.Buffer
	screen := uv.NewTerminalScreen(&buf, nil)
	b := &Backend{
		screen: screen,
		width:  80,
		height: 10, // terminal height is small (10 lines)
	}

	totalLines := 60 // 60 lines exceeds terminal height 10
	insertBuf := backend.NewBuffer(80, totalLines)
	for y := 0; y < totalLines; y++ {
		lineText := fmt.Sprintf("Log line %d", y)
		for x, ch := range lineText {
			insertBuf.Set(x, y, backend.Cell{Content: string(ch), Width: 1})
		}
	}

	b.processInsertAbove(insertBuf)

	output := buf.String()
	for y := 0; y < totalLines; y++ {
		expected := fmt.Sprintf("Log line %d\x1b[K\r\n", y)
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but missing", expected)
		}
	}
}

func TestBackend_InsertAbove_RestoresActiveInlineFrame(t *testing.T) {
	var out bytes.Buffer
	screen := uv.NewTerminalScreen(&out, nil)
	b := &Backend{
		screen:   screen,
		renderCh: make(chan renderRequest, 256),
		width:    80,
		height:   24,
		inline:   true,
	}
	b.bufferPool.New = func() any {
		return backend.NewBuffer(80, 24)
	}
	b.started.Store(true)
	b.renderWG.Add(1)
	go b.renderLoop()

	// 1. Render an initial active inline frame (height 2)
	frameBuf := backend.NewBuffer(80, 2)
	frameBuf.Set(0, 0, backend.Cell{Content: "C", Width: 1}) // Composer
	frameBuf.Set(0, 1, backend.Cell{Content: "S", Width: 1}) // Status
	b.renderCh <- renderRequest{buffer: frameBuf}

	// Give renderLoop time to process the frame
	time.Sleep(15 * time.Millisecond)

	// 2. Call InsertAbove with a static item
	staticBuf := backend.NewBuffer(80, 1)
	staticBuf.Set(0, 0, backend.Cell{Content: "M", Width: 1}) // Message
	b.InsertAbove(staticBuf)

	b.stopRenderLoop()

	output := out.String()
	// Static message "M" must be in output
	if !strings.Contains(output, "M") {
		t.Errorf("expected static message 'M' in output, got %q", output)
	}
	// Active inline frame "C" and "S" must still be rendered after static message
	lastM := strings.LastIndex(output, "M")
	lastC := strings.LastIndex(output, "C")
	lastS := strings.LastIndex(output, "S")
	if lastC < lastM || lastS < lastM {
		t.Errorf("expected active frame ('C', 'S') to be re-rendered below 'M', indices M=%d, C=%d, S=%d", lastM, lastC, lastS)
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
