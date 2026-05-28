package styler_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

type mockNode struct {
	raw      style.Style
	defaults style.Style
	computed *style.Computed
	parent   *mockNode
}

func (n *mockNode) RawStyle() style.Style       { return n.raw }
func (n *mockNode) DefaultStyle() style.Style   { return n.defaults }
func (n *mockNode) IntrinsicStyle() style.Style { return style.Style{} }
func (n *mockNode) IsDirtyStyle() bool          { return true }
func (n *mockNode) StyleParent() style.StyleNode {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *mockNode) StyleFirstChild() style.StyleNode  { return nil }
func (n *mockNode) StyleNextSibling() style.StyleNode { return nil }

func TestCursorInheritance(t *testing.T) {
	resolver := styler.NewResolver()

	parent := &mockNode{
		raw: style.Style{
			CursorShape: style.Some(style.CursorShapeBarSteady),
			CursorColor: style.Some[color.Color](color.RGBA{R: 255, G: 0, B: 0, A: 255}),
		},
	}
	child := &mockNode{parent: parent}

	parentComputed := resolver.Resolve(parent, nil)
	childComputed := resolver.Resolve(child, parentComputed)

	if childComputed.CursorShape != style.CursorShapeBarSteady {
		t.Errorf("child: expected inherited BarSteady shape, got %v", childComputed.CursorShape)
	}
	if childComputed.CursorColor != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("child: expected inherited red color, got %v", childComputed.CursorColor)
	}
}

func TestCursorOverride(t *testing.T) {
	resolver := styler.NewResolver()

	parent := &mockNode{
		raw: style.Style{
			CursorShape: style.Some(style.CursorShapeBarSteady),
		},
	}
	child := &mockNode{
		parent: parent,
		raw: style.Style{
			CursorShape: style.Some(style.CursorShapeBlockSteady),
		},
	}

	parentComputed := resolver.Resolve(parent, nil)
	childComputed := resolver.Resolve(child, parentComputed)

	if childComputed.CursorShape != style.CursorShapeBlockSteady {
		t.Errorf("child: expected overridden BlockSteady shape, got %v", childComputed.CursorShape)
	}
}
