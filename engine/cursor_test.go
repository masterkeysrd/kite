package engine

import (
	"testing"

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

func TestEngineCursorIntegration(t *testing.T) {
	b := mock.New(80, 24)
	e := New(b, Options{})

	// Create a document and a node.
	doc := e.Document()
	fe := &focusableElement{}
	el := doc.CreateElement("input", fe)
	fe.Element = el
	doc.AppendChild(el)

	if el.Parent() != doc {
		t.Fatalf("el parent is not doc: %v", el.Parent())
	}

	myRO := &cursorProvidingRenderObject{
		state: cursor.State{
			Visible: true,
			X:       2,
			Y:       1,
			Shape:   cursor.ShapeBarBlink,
		},
	}
	// Initialize BaseRender
	myRO.Init(myRO, el, el)
	myRO.SetComputedStyle(&style.Computed{})

	// Manually attach it to the element
	el.SetRenderObject(myRO)

	// In kite, AbsoluteBounds traverses the fragment tree.
	// We need myRO to have a fragment and be in the tree.

	// We need a fragment for AbsoluteBounds to work
	frag := &layout.Fragment{
		Node: myRO,
		Size: geom.Size{Width: 10, Height: 1},
	}
	myRO.SetCachedLayout(layout.ConstraintSpace{}, frag)

	// Attach to renderView
	e.renderView.InsertChild(myRO, nil)

	// We also need to manually construct the fragment tree for the renderView
	// because AbsoluteBounds walks from root.Fragment().
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

	// Focus the element
	if ok := e.focusManager.SetFocus(fe, focus.ReasonProgrammatic); !ok {
		t.Fatalf("failed to focus element")
	}

	// Run updateHardwareCursor
	e.updateHardwareCursor(true)

	// Verify mock backend received the cursor state.
	if !b.Cursor.Visible {
		t.Errorf("expected cursor to be visible")
	}
	// Absolute position should be ViewOffset + LocalOffset = (5+2, 10+1) = (7, 11)
	if b.Cursor.X != 7 || b.Cursor.Y != 11 {
		t.Errorf("expected cursor pos (7, 11), got (%d, %d)", b.Cursor.X, b.Cursor.Y)
	}
	if b.Cursor.Shape != cursor.ShapeBarBlink {
		t.Errorf("expected ShapeBarBlink, got %v", b.Cursor.Shape)
	}
}
