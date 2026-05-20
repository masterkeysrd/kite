package style

import (
	"image/color"
	"testing"
)

type mockNode struct {
	raw      Style
	defaults Style
	computed *Computed
	parent   *mockNode
}

func (n *mockNode) RawStyle() Style              { return n.raw }
func (n *mockNode) DefaultStyle() Style          { return n.defaults }
func (n *mockNode) IntrinsicStyle() Style        { return Style{} }
func (n *mockNode) ComputedStyle() *Computed     { return n.computed }
func (n *mockNode) SetComputedStyle(c *Computed) { n.computed = c }
func (n *mockNode) IsDirtyStyle() bool           { return true }
func (n *mockNode) HasDirtyStyleChild() bool     { return false }
func (n *mockNode) ClearDirtyStyle()             {}
func (n *mockNode) ClearChildNeedsStyle()        {}
func (n *mockNode) StyleParent() StyleNode {
	if n.parent == nil {
		return nil
	}
	return n.parent
}
func (n *mockNode) StyleFirstChild() StyleNode  { return nil }
func (n *mockNode) StyleNextSibling() StyleNode { return nil }

func TestCursorInheritance(t *testing.T) {
	resolver := NewResolver()

	parent := &mockNode{
		raw: Style{
			CursorShape: Some(CursorShapeBarSteady),
			CursorColor: Some[color.Color](color.RGBA{R: 255, G: 0, B: 0, A: 255}),
		},
	}
	child := &mockNode{
		parent: parent,
	}

	parentComputed := resolver.Resolve(parent, nil)
	parent.SetComputedStyle(parentComputed)

	if parentComputed.CursorShape != CursorShapeBarSteady {
		t.Errorf("parent: expected CursorShapeBarSteady, got %v", parentComputed.CursorShape)
	}

	childComputed := resolver.Resolve(child, parentComputed)

	if childComputed.CursorShape != CursorShapeBarSteady {
		t.Errorf("child: expected inherited CursorShapeBarSteady, got %v", childComputed.CursorShape)
	}
	if childComputed.CursorColor != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("child: expected inherited red color, got %v", childComputed.CursorColor)
	}
}

func TestCursorOverride(t *testing.T) {
	resolver := NewResolver()

	parent := &mockNode{
		raw: Style{
			CursorShape: Some(CursorShapeBarSteady),
		},
	}
	child := &mockNode{
		parent: parent,
		raw: Style{
			CursorShape: Some(CursorShapeUnderlineBlink),
		},
	}

	parentComputed := resolver.Resolve(parent, nil)
	childComputed := resolver.Resolve(child, parentComputed)

	if childComputed.CursorShape != CursorShapeUnderlineBlink {
		t.Errorf("child: expected overridden CursorShapeUnderlineBlink, got %v", childComputed.CursorShape)
	}
}
