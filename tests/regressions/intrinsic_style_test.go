// Package regressions – Intrinsic Style Layer regression tests (TSK-022).
//
// These tests verify that UA-mandated properties set via IntrinsicStyle()
// resist author overrides and are honoured end-to-end through the full engine
// pipeline (sync → style → layout → paint).
package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// intrinsicClipElement — a minimal replaced-element stub for regression tests
// ---------------------------------------------------------------------------

// intrinsicClipElement is a test element that UA-forces OverflowX:Clip via
// IntrinsicStyle(), modelling a replaced element like <input>. It also serves
// as a render.CustomObjectProvider to get its own render.Box.
type intrinsicClipElement struct {
	dom.Element
	rawStyle style.Style
}

func newIntrinsicClipElement(doc dom.Document, rawStyle style.Style) *intrinsicClipElement {
	e := &intrinsicClipElement{
		Element:  dom.NewElement(doc, "clip-input", nil),
		rawStyle: rawStyle,
	}
	return e
}

func (e *intrinsicClipElement) RawStyle() style.Style     { return e.rawStyle }
func (e *intrinsicClipElement) DefaultStyle() style.Style { return style.Style{} }
func (e *intrinsicClipElement) IntrinsicStyle() style.Style {
	return style.Style{
		OverflowX: style.Some(style.OverflowClip),
	}
}
func (e *intrinsicClipElement) Unwrap() dom.Node   { return e.Element }
func (e *intrinsicClipElement) IsDirtyStyle() bool { return false }

var _ render.CustomObjectProvider = (*intrinsicClipElement)(nil)

func (e *intrinsicClipElement) CreateRenderObject() render.Object {
	return render.NewBox(e, e)
}

// ---------------------------------------------------------------------------
// TestIntrinsicStyle_OverflowClipResistsAuthorOverride
// ---------------------------------------------------------------------------

// TestIntrinsicStyle_OverflowClipResistsAuthorOverride verifies that an
// element with intrinsic OverflowX:Clip resists an author attempt to set
// OverflowX:Visible. Content placed outside the element's content box must be
// clipped in the output framebuffer.
func TestIntrinsicStyle_OverflowClipResistsAuthorOverride(t *testing.T) {
	b := mock.New(40, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	// Author tries to set OverflowX:Visible — intrinsic layer must override.
	doc := dom.NewDocument()
	clipEl := newIntrinsicClipElement(doc, style.Style{
		Width:     style.Some(style.Cells(10)),
		Height:    style.Some(style.Cells(1)),
		OverflowX: style.Some(style.OverflowVisible), // author override — must lose
	})

	// Child: 30-cell wide red box — would spill without clipping.
	child := element.Box().Style(style.Style{
		Width:      style.Some(style.Cells(30)),
		Height:     style.Some(style.Cells(1)),
		Background: style.Some[color.Color](color.RGBA{255, 0, 0, 255}),
	})

	// Connect: clipEl → child using the DOM node directly.
	clipEl.AppendChild(child)

	// Wrap in a root box.
	root := dom.NewElement(doc, "root", nil)
	root.AppendChild(clipEl)
	eng.Mount(root)

	eng.Frame()
	fr := b.LastFrame()
	if fr.Surface == nil {
		t.Fatal("no frame surface produced")
	}

	// Verify the intrinsic clip won: cells 0..9 may have red, cells 10+ must not.
	for x := 10; x < 30; x++ {
		cell := fr.Surface.CellAt(x, 0)
		if cell.Bg == (color.RGBA{255, 0, 0, 255}) {
			t.Errorf("cell (%d,0): intrinsic OverflowX:Clip was bypassed — red content leaked outside the 10-cell boundary", x)
		}
	}
}
