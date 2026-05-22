package testenv

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/paint"
)

var update = flag.Bool("update", false, "update golden files")

// Environment provides ergonomic tools for testing Kite components headless.
type Environment struct {
	Engine  *engine.Engine
	Backend *mock.Backend
}

// DispatchKey sends a key down event to the target EventTarget by building
// the ancestor path (root -> target) and dispatching through the event
// system. Useful for tests to simulate keyboard input against nodes.
func (e *Environment) DispatchKey(target event.EventTarget, k key.Key) {
	ev := event.NewKeyEvent(event.EventKeyDown, k)

	var path []event.EventTarget
	curr := target
	for curr != nil {
		path = append(path, curr)
		if n, ok := curr.(dom.Node); ok {
			p := n.Parent()
			if p == nil {
				break
			}
			curr = p
		} else {
			break
		}
	}

	// Reverse to root -> target
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	d := event.NewDispatcher()
	d.Dispatch(ev, path)
}

// New creates a new test environment wrapping the given engine.
func New(eng *engine.Engine) *Environment {
	return &Environment{
		Engine: eng,
	}
}

// Default creates a new test environment with a mock backend of the given
// dimensions and a default engine.
func Default(width, height int) *Environment {
	b := mock.New(width, height)
	eng := engine.New(b, engine.Options{})
	return &Environment{
		Engine:  eng,
		Backend: b,
	}
}

// Teardown gracefully stops the engine.
func (e *Environment) Teardown() {
	e.Engine.Stop()
}

// Close is an alias for Teardown.
func (e *Environment) Close() {
	e.Teardown()
}

// Document returns the logical document root.
func (e *Environment) Document() dom.Document {
	return e.Engine.Document()
}

// Mount appends n as the body of the document.
func (e *Environment) Mount(n dom.Node) {
	if el, ok := n.(dom.Element); ok {
		e.Engine.Mount(el)
	}
}

// Flush blocks until the engine completes a frame, allowing assertions on the newly painted state.
func (e *Environment) Flush() {
	e.Engine.Frame()
}

// RenderFrame is an alias for Flush.
func (e *Environment) RenderFrame() {
	e.Flush()
}

// GetNodeByID returns the element with the given ID from the logical DOM.
func (e *Environment) GetNodeByID(id string) dom.Element {
	el := e.Engine.Document().GetElementByID(id)
	if el != nil {
		return el
	}
	return e.QuerySelector("#" + id)
}

// QuerySelector returns the first element matching the selector.
// Supports simple tag name ("div"), ID ("#id"), and class (".class") matching.
func (e *Environment) QuerySelector(selector string) dom.Element {
	return e.Engine.Document().QuerySelector(selector)
}

// SendKey simulates a key event.
func (e *Environment) SendKey(k key.Key) {
	e.Engine.ProcessRawEvent(&event.RawKeyEvent{
		Key: k,
	})
}

// HasFocus reports whether the given logical node currently has focus.
func (e *Environment) HasFocus(n dom.Node) bool {
	return e.Engine.FocusManager().Current() == n
}

// Type simulates typing the given text.
func (e *Environment) Type(text string) {
	for _, r := range text {
		e.SendKey(key.Key{
			Code: r,
			Text: string(r),
		})
	}
}

// Click simulates a mouse click at (x, y).
func (e *Environment) Click(x, y int) {
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		Button: event.ButtonLeft,
	})
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		Button: event.ButtonLeft,
		Up:     true,
	})
}

// Wheel simulates a mouse wheel event at (x, y).
func (e *Environment) Wheel(x, y, dx, dy int) {
	e.Engine.ProcessRawEvent(&event.RawMouseEvent{
		X:      x,
		Y:      y,
		DeltaX: dx,
		DeltaY: dy,
	})
}

// ScrollTo sets the scroll offset of an element.
func (e *Environment) ScrollTo(el dom.Element, x, y int) {
	el.ScrollTo(x, y)
}

// ShowOverlay adds el to the top layer at the specified z-index.
func (e *Environment) ShowOverlay(el dom.Element, zIndex int) {
	e.Engine.Document().ShowOverlay(el, zIndex)
}

// HideOverlay removes el from the top layer.
func (e *Environment) HideOverlay(el dom.Element) {
	e.Engine.Document().HideOverlay(el)
}

// Overlays returns an iterator over all active overlays.
func (e *Environment) Overlays() iter.Seq[dom.Element] {
	return e.Engine.Document().Overlays()
}

// QueryOverlay returns the first element matching the selector in any active overlay.
func (e *Environment) QueryOverlay(selector string) dom.Element {
	for overlay := range e.Engine.Document().Overlays() {
		if found := overlay.QuerySelector(selector); found != nil {
			return found
		}
	}
	return nil
}

// MatchGolden compares the current framebuffer against a stored snapshot.
func (e *Environment) MatchGolden(t *testing.T, filename string) {
	t.Helper()

	actual, expected, goldenPath, actualPath, err := e.matchGolden(filename)
	if err != nil {
		t.Fatalf("MatchGolden failed: %v", err)
	}

	if expected == nil {
		// New golden file created or updated
		t.Logf("golden file %s created/updated", goldenPath)
		return
	}

	if string(actual) != string(expected) {
		t.Errorf("framebuffer does not match golden file %s", goldenPath)
		t.Errorf("actual output written to %s", actualPath)
	}
}

func (e *Environment) matchGolden(filename string) (actual, expected []byte, goldenPath, actualPath string, err error) {
	frame := e.Backend.LastFrame()
	if frame.Surface == nil {
		return nil, nil, "", "", fmt.Errorf("no frame has been painted")
	}

	fb := frame.Surface
	bounds := fb.Bounds()
	width := bounds.Size.Width
	height := bounds.Size.Height

	type goldenCell struct {
		Content string `json:"c"`
		FG      string `json:"fg,omitempty"`
		BG      string `json:"bg,omitempty"`
		Attrs   uint8  `json:"a,omitempty"`
	}

	type goldenFrame struct {
		Width  int            `json:"width"`
		Height int            `json:"height"`
		Cells  [][]goldenCell `json:"cells"`
	}

	gf := goldenFrame{
		Width:  width,
		Height: height,
		Cells:  make([][]goldenCell, height),
	}

	for y := range height {
		gf.Cells[y] = make([]goldenCell, width)
		for x := range width {
			cell := fb.CellAt(x+bounds.Origin.X, y+bounds.Origin.Y)
			gf.Cells[y][x] = goldenCell{
				Content: cell.Content,
				FG:      colorToHex(cell.FG),
				BG:      colorToHex(cell.BG),
				Attrs:   uint8(cell.Attrs),
			}
		}
	}

	actual, err = json.MarshalIndent(gf, "", "  ")
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to marshal actual frame: %w", err)
	}

	goldenPath = filepath.Join("testdata", filename+".golden")
	actualPath = filepath.Join("testdata", filename+".actual")

	_, statErr := os.Stat(goldenPath)
	if *update || os.IsNotExist(statErr) {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to create testdata dir: %w", err)
		}
		err = os.WriteFile(goldenPath, actual, 0644)
		if err != nil {
			return nil, nil, "", "", fmt.Errorf("failed to write golden file: %w", err)
		}
		return actual, nil, goldenPath, actualPath, nil
	}

	expected, err = os.ReadFile(goldenPath)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to read golden file: %w", err)
	}

	if string(actual) != string(expected) {
		_ = os.WriteFile(actualPath, actual, 0644)
	}

	return actual, expected, goldenPath, actualPath, nil
}

func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// DumpANSI translates the current FrameBuffer into a raw string of ANSI escape codes.
func (e *Environment) DumpANSI() string {
	frame := e.Backend.LastFrame()
	if frame.Surface == nil {
		return ""
	}

	fb := frame.Surface
	bounds := fb.Bounds()
	var sb strings.Builder

	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; x++ {
			cell := fb.CellAt(x+bounds.Origin.X, y+bounds.Origin.Y)
			// ANSI Escape codes
			if cell.FG != nil {
				r, g, b, _ := cell.FG.RGBA()
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm", uint8(r>>8), uint8(g>>8), uint8(b>>8))
			}
			if cell.BG != nil {
				r, g, b, _ := cell.BG.RGBA()
				fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm", uint8(r>>8), uint8(g>>8), uint8(b>>8))
			}
			if cell.Attrs&paint.AttrBold != 0 {
				sb.WriteString("\x1b[1m")
			}
			if cell.Attrs&paint.AttrItalic != 0 {
				sb.WriteString("\x1b[3m")
			}
			if cell.Attrs&paint.AttrUnderline != 0 {
				sb.WriteString("\x1b[4m")
			}
			if cell.Attrs&paint.AttrInverse != 0 {
				sb.WriteString("\x1b[7m")
			}

			content := cell.Content
			if content == "" {
				content = " "
			}
			sb.WriteString(content)
			sb.WriteString("\x1b[0m")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// DumpHTML translates the current FrameBuffer into a standalone HTML file.
func (e *Environment) DumpHTML() string {
	frame := e.Backend.LastFrame()
	if frame.Surface == nil {
		return ""
	}

	fb := frame.Surface
	bounds := fb.Bounds()
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<style>\n")
	sb.WriteString("body { background: #000; color: #fff; font-family: monospace; white-space: pre; line-height: 1; }\n")
	sb.WriteString(".cell { display: inline-block; width: 1ch; }\n")
	sb.WriteString("</style>\n</head>\n<body>\n")

	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; x++ {
			cell := fb.CellAt(x+bounds.Origin.X, y+bounds.Origin.Y)
			style := ""
			if cell.FG != nil {
				style += fmt.Sprintf("color: %s; ", colorToHex(cell.FG))
			}
			if cell.BG != nil {
				style += fmt.Sprintf("background-color: %s; ", colorToHex(cell.BG))
			}
			if cell.Attrs&paint.AttrBold != 0 {
				style += "font-weight: bold; "
			}
			if cell.Attrs&paint.AttrItalic != 0 {
				style += "font-style: italic; "
			}
			if cell.Attrs&paint.AttrUnderline != 0 {
				style += "text-decoration: underline; "
			}
			// Inverse is hard in HTML without knowing original colors, but we can swap them if both are set
			if cell.Attrs&paint.AttrInverse != 0 {
				// Simplified inverse
				style += "filter: invert(100%); "
			}

			content := cell.Content
			if content == "" {
				content = " "
			}
			// Escape HTML entities
			content = strings.ReplaceAll(content, "&", "&amp;")
			content = strings.ReplaceAll(content, "<", "&lt;")
			content = strings.ReplaceAll(content, ">", "&gt;")

			if style != "" {
				fmt.Fprintf(&sb, "<span style=\"%s\">%s</span>", strings.TrimSpace(style), content)
			} else {
				sb.WriteString(content)
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("</body>\n</html>")
	return sb.String()
}

// DumpText translates the current FrameBuffer into a plain text representation.
func (e *Environment) DumpText() string {
	frame := e.Backend.LastFrame()
	if frame.Surface == nil {
		return ""
	}

	fb := frame.Surface
	bounds := fb.Bounds()
	var sb strings.Builder

	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; x++ {
			cell := fb.CellAt(x+bounds.Origin.X, y+bounds.Origin.Y)
			content := cell.Content
			if content == "" {
				content = " "
			}
			sb.WriteString(content)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
