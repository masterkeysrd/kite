package element_test

// Integration test for TSK-022: Intrinsic Style Layer (ADR-010).
//
// An input-like element forces Display:InlineBlock and OverflowX:Clip via
// IntrinsicStyle(). Even when the author sets Display:Block via RawStyle(),
// the resolved Computed.Display must remain InlineBlock.

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// intrinsicElement — a test element that forces UA-mandated properties
// ---------------------------------------------------------------------------

// intrinsicElement mimics a replaced element (e.g. <input>) that UA-mandates
// Display:InlineBlock and OverflowX:Clip regardless of what the author sets.
type intrinsicElement struct {
	dom.Element
	rawStyle       style.Style
	intrinsicStyle style.Style
}

func newIntrinsicElement(doc dom.Document) *intrinsicElement {
	e := &intrinsicElement{
		Element: dom.NewElement(doc, "test-input", nil),
		intrinsicStyle: style.Style{
			Display:   style.Some(style.DisplayInlineBlock),
			OverflowX: style.Some(style.OverflowClip),
		},
	}
	// Register self as outer so the engine recognises it.
	return e
}

func (e *intrinsicElement) RawStyle() style.Style       { return e.rawStyle }
func (e *intrinsicElement) DefaultStyle() style.Style   { return style.Style{} }
func (e *intrinsicElement) IntrinsicStyle() style.Style { return e.intrinsicStyle }
func (e *intrinsicElement) Unwrap() dom.Node            { return e.Element }
func (e *intrinsicElement) IsDirtyStyle() bool          { return false }

var _ render.CustomObjectProvider = (*intrinsicElement)(nil)

// CreateRenderObject provides a render.Box that the engine will use for this node.
func (e *intrinsicElement) CreateRenderObject() render.Object {
	return render.NewBox(e, e)
}

// ---------------------------------------------------------------------------
// TestIntrinsicStyle_Integration_InlineBlockForced
// ---------------------------------------------------------------------------

// TestIntrinsicStyle_Integration_InlineBlockForced verifies that after the
// full engine style-resolution pass (Sync → Style → Layout), a node whose
// IntrinsicStyle() forces Display:InlineBlock ends up with
// Computed.Display == DisplayInlineBlock even when the author set
// RawStyle Display:Block.
func TestIntrinsicStyle_Integration_InlineBlockForced(t *testing.T) {
	b := mock.New(80, 5)
	eng := engine.New(b, engine.Options{})
	defer eng.Stop()

	doc := dom.NewDocument()
	el := newIntrinsicElement(doc)

	// Author tries to override Display to Block.
	el.rawStyle = style.Style{
		Display: style.Some(style.DisplayBlock),
	}

	// Mount via a wrapper box so the engine sees it.
	box := dom.NewElement(doc, "box", nil)
	box.AppendChild(el)
	eng.Mount(box)
	eng.Frame()

	ro := eng.RenderObject(el)
	if ro == nil {
		t.Fatal("render object not created for intrinsicElement")
	}

	cs := ro.ComputedStyle()
	if cs == nil {
		t.Fatal("computed style is nil after engine frame")
	}

	if cs.Display != style.DisplayInlineBlock {
		t.Errorf("Display = %v, want DisplayInlineBlock (intrinsic must win over author Block)", cs.Display)
	}
	if cs.OverflowX != style.OverflowClip {
		t.Errorf("OverflowX = %v, want OverflowClip (intrinsic must win)", cs.OverflowX)
	}
}
