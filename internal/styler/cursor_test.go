package styler_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

func TestCursorInheritance(t *testing.T) {
	resolver := styler.NewResolver()

	parent := &fakeNode{
		kind: 1, // Element
		rawStyle: style.Style{
			CursorShape: style.Some(style.CursorShapeBarSteady),
			CursorColor: style.Some[color.Color](color.RGBA{R: 255, G: 0, B: 0, A: 255}),
		},
	}
	parentRO := render.NewBox(parent, nil)
	parentRO.MarkDirty(render.DirtyStyle)

	child := &fakeNode{kind: 1}
	childRO := render.NewBox(child, nil)
	childRO.MarkDirty(render.DirtyStyle)

	parentComputed := resolver.Resolve(parentRO, nil)
	childComputed := resolver.Resolve(childRO, parentComputed)

	if childComputed.CursorShape != style.CursorShapeBarSteady {
		t.Errorf("child: expected inherited BarSteady shape, got %v", childComputed.CursorShape)
	}
	if childComputed.CursorColor != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("child: expected inherited red color, got %v", childComputed.CursorColor)
	}
}

func TestCursorOverride(t *testing.T) {
	resolver := styler.NewResolver()

	parent := &fakeNode{
		kind: 1,
		rawStyle: style.Style{
			CursorShape: style.Some(style.CursorShapeBarSteady),
		},
	}
	parentRO := render.NewBox(parent, nil)
	parentRO.MarkDirty(render.DirtyStyle)

	child := &fakeNode{
		kind: 1,
		rawStyle: style.Style{
			CursorShape: style.Some(style.CursorShapeBlockSteady),
		},
	}
	childRO := render.NewBox(child, nil)
	childRO.MarkDirty(render.DirtyStyle)

	parentComputed := resolver.Resolve(parentRO, nil)
	childComputed := resolver.Resolve(childRO, parentComputed)

	if childComputed.CursorShape != style.CursorShapeBlockSteady {
		t.Errorf("child: expected overridden BlockSteady shape, got %v", childComputed.CursorShape)
	}
}
