package engine

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

type cursorProvidingRenderObject struct {
	render.BaseRender
	state cursor.State
}

func (c *cursorProvidingRenderObject) CursorState() cursor.State {
	return c.state
}

// Ensure it implements cursor.Provider
var _ cursor.Provider = (*cursorProvidingRenderObject)(nil)

type focusableElement struct {
	dom.Element
}

func (f *focusableElement) IsFocusable() bool { return true }
func (f *focusableElement) Focus()            {}
func (f *focusableElement) Blur()             {}
func (f *focusableElement) TabIndex() int     { return 0 }
func (f *focusableElement) Unwrap() dom.Node  { return f.Element }

func TestEngineCursorIntegration(t *testing.T) {
	b := mock.New(80, 24)
	e := New(b, Options{})

	// Create a document and a node.
	doc := e.Document()
	fe := &focusableElement{}
	el := doc.CreateElement("input", fe)
	fe.Element = el
	doc.AppendChild(fe)

	myRO := &cursorProvidingRenderObject{
		state: cursor.State{
			Visible: true,
			X:       2,
			Y:       1,
			Style: style.Cursor{
				Shape: style.Some(style.CursorBar),
				Blink: style.Some(true),
			},
		},
	}
	// Initialize BaseRender
	myRO.Init(myRO, fe, fe)
	myRO.SetComputedStyle(&style.Computed{})

	// Manually attach it to the element
	e.setRenderObject(fe, myRO)

	// Focus the element
	if ok := e.focusManager.SetFocus(fe, focus.ReasonProgrammatic); !ok {
		t.Fatalf("failed to focus element")
	}

	// STABILIZE: Run EnsureFreshLayout now to clear all dirty flags from setup.
	// This will create real (but empty/zero) fragments.
	e.EnsureFreshLayout()

	// MOCK: Now overwrite with our manual fragments. Since the tree is now clean,
	// subsequent EnsureFreshLayout calls (e.g. in updateHardwareCursor) will
	// see it's clean and return early, preserving our mocks.
	frag := &layout.Fragment{
		Node: myRO,
		Size: geom.Size{Width: 10, Height: 1},
	}
	myRO.SetCachedLayout(layout.ConstraintSpace{}, frag)

	viewFrag := &layout.Fragment{
		Node: e.renderView,
		Size: geom.Size{Width: 80, Height: 24},
		Children: []layout.FragmentLink{
			{
				Fragment: frag,
				Offset:   geom.Point{X: 5, Y: 10},
			},
		},
	}
	e.renderView.SetCachedLayout(layout.ConstraintSpace{}, viewFrag)

	// Run updateHardwareCursor. It will call EnsureFreshLayout, which should
	// be a no-op because we just stabilized.
	e.updateHardwareCursor(true)

	// Verify mock backend received the cursor state.
	if !b.Cursor.Visible {
		t.Errorf("expected cursor to be visible")
	}
	// Absolute position should be ViewOffset + LocalOffset = (5+2, 10+1) = (7, 11)
	if b.Cursor.X != 7 || b.Cursor.Y != 11 {
		t.Errorf("expected cursor pos (7, 11), got (%d, %d)", b.Cursor.X, b.Cursor.Y)
	}
	if b.Cursor.Shape != backend.CursorBar {
		t.Errorf("expected CursorBar, got %v", b.Cursor.Shape)
	}
}

func TestEngineCursorOffScreenHiding(t *testing.T) {
	b := mock.New(80, 24)
	e := New(b, Options{})

	// Create a document and a node.
	doc := e.Document()
	fe := &focusableElement{}
	el := doc.CreateElement("input", fe)
	fe.Element = el
	doc.AppendChild(fe)

	myRO := &cursorProvidingRenderObject{
		state: cursor.State{
			Visible: true,
			X:       6, // 75 + 6 = 81 (off-screen)
			Y:       0,
			Style: style.Cursor{
				Shape: style.Some(style.CursorBar),
				Blink: style.Some(true),
			},
		},
	}
	myRO.Init(myRO, fe, fe)
	myRO.SetComputedStyle(&style.Computed{})

	e.setRenderObject(fe, myRO)

	if ok := e.focusManager.SetFocus(fe, focus.ReasonProgrammatic); !ok {
		t.Fatalf("failed to focus element")
	}

	e.EnsureFreshLayout()

	frag := &layout.Fragment{
		Node: myRO,
		Size: geom.Size{Width: 10, Height: 1},
	}
	myRO.SetCachedLayout(layout.ConstraintSpace{}, frag)

	viewFrag := &layout.Fragment{
		Node: e.renderView,
		Size: geom.Size{Width: 80, Height: 24},
		Children: []layout.FragmentLink{
			{
				Fragment: frag,
				Offset:   geom.Point{X: 75, Y: 10},
			},
		},
	}
	e.renderView.SetCachedLayout(layout.ConstraintSpace{}, viewFrag)

	// Run updateHardwareCursor. It should hide the cursor because it is off-screen.
	e.updateHardwareCursor(true)

	if b.Cursor.Visible {
		t.Errorf("expected cursor to be hidden when X=81 is off-screen")
	}

	// Move cursor to X=2 (75 + 2 = 77, on-screen).
	myRO.state.X = 2
	e.updateHardwareCursor(true)

	if !b.Cursor.Visible {
		t.Errorf("expected cursor to be visible when X=77 is on-screen")
	}
	if b.Cursor.X != 77 || b.Cursor.Y != 10 {
		t.Errorf("expected cursor pos (77, 10), got (%d, %d)", b.Cursor.X, b.Cursor.Y)
	}
}
