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
			Cursor: style.Some(style.Cursor{
				Shape: style.Some(style.CursorBar),
				Blink: style.Some(false),
				Color: style.Some[color.Color](color.RGBA{R: 255, G: 0, B: 0, A: 255}),
			}),
		},
	}
	parentRO := render.NewBox(parent, nil)
	parentRO.MarkDirty(render.DirtyStyle)

	child := &fakeNode{kind: 1}
	childRO := render.NewBox(child, nil)
	childRO.MarkDirty(render.DirtyStyle)

	parentComputed := resolver.Resolve(parentRO, nil)
	childComputed := resolver.Resolve(childRO, parentComputed)

	if childComputed.Cursor.Shape.UnwrapOr(style.CursorBlock) != style.CursorBar {
		t.Errorf("child: expected inherited Bar shape, got %v", childComputed.Cursor.Shape)
	}
	if childComputed.Cursor.Blink.UnwrapOr(true) != false {
		t.Errorf("child: expected inherited non-blinking, got %v", childComputed.Cursor.Blink)
	}
	if childComputed.Cursor.Color.UnwrapOr(nil) != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("child: expected inherited red color, got %v", childComputed.Cursor.Color)
	}
}

func TestCursorOverride(t *testing.T) {
	resolver := styler.NewResolver()

	parent := &fakeNode{
		kind: 1,
		rawStyle: style.Style{
			Cursor: style.Some(style.Cursor{
				Shape: style.Some(style.CursorBar),
			}),
		},
	}
	parentRO := render.NewBox(parent, nil)
	parentRO.MarkDirty(render.DirtyStyle)

	child := &fakeNode{
		kind: 1,
		rawStyle: style.Style{
			Cursor: style.Some(style.Cursor{
				Shape: style.Some(style.CursorBlock),
			}),
		},
	}
	childRO := render.NewBox(child, nil)
	childRO.MarkDirty(render.DirtyStyle)

	parentComputed := resolver.Resolve(parentRO, nil)
	childComputed := resolver.Resolve(childRO, parentComputed)

	if childComputed.Cursor.Shape.UnwrapOr(style.CursorBar) != style.CursorBlock {
		t.Errorf("child: expected overridden Block shape, got %v", childComputed.Cursor.Shape)
	}
}
