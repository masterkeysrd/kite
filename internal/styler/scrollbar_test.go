package styler_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

func TestStyle_ScrollbarCascade(t *testing.T) {
	base := style.Style{}.ScrollbarY(true).ScrollbarThumb('+', color.White)
	override := style.Style{}.ScrollbarX(true).ScrollbarTrack('.', color.Black)

	merged := base.Merge(override)

	sb := merged.Scrollbar.Value()
	if !sb.Y.UnwrapOr(false) {
		t.Error("expected Y to be true")
	}
	if !sb.X.UnwrapOr(false) {
		t.Error("expected X to be true")
	}
	if sb.ThumbGlyph.Value() != '+' {
		t.Errorf("expected ThumbGlyph to be '+', got %c", sb.ThumbGlyph.Value())
	}
	if sb.TrackGlyph.Value() != '.' {
		t.Errorf("expected TrackGlyph to be '.', got %c", sb.TrackGlyph.Value())
	}
}

func TestStyle_ResolverScrollbarDefaults(t *testing.T) {
	resolver := styler.NewResolver()
	node := &mockStyleNode{
		raw: style.Style{}.ScrollbarY(true),
	}

	computed := resolver.Resolve(node, nil)
	sb := computed.Scrollbar

	if !sb.Y.UnwrapOr(false) {
		t.Error("expected Y scrollbar")
	}
	if sb.TrackGlyph.Value() != style.DefaultScrollbarTrackVertical {
		t.Errorf("expected default vertical track glyph, got %c", sb.TrackGlyph.Value())
	}
	if sb.ThumbGlyph.Value() != style.DefaultScrollbarThumbVertical {
		t.Errorf("expected default vertical thumb glyph, got %c", sb.ThumbGlyph.Value())
	}
}

type mockStyleNode struct {
	raw      style.Style
	computed *style.Computed
}

func (m *mockStyleNode) RawStyle() style.Style              { return m.raw }
func (m *mockStyleNode) DefaultStyle() style.Style          { return style.Style{} }
func (m *mockStyleNode) IntrinsicStyle() style.Style        { return style.Style{} }
func (m *mockStyleNode) ComputedStyle() *style.Computed     { return m.computed }
func (m *mockStyleNode) SetComputedStyle(c *style.Computed) { m.computed = c }
func (m *mockStyleNode) IsDirtyStyle() bool                 { return true }
func (m *mockStyleNode) HasDirtyStyleChild() bool           { return false }
func (m *mockStyleNode) ClearDirtyStyle()                   {}
func (m *mockStyleNode) ClearChildNeedsStyle()              {}
func (m *mockStyleNode) StyleParent() style.StyleNode       { return nil }
func (m *mockStyleNode) StyleFirstChild() style.StyleNode   { return nil }
func (m *mockStyleNode) StyleNextSibling() style.StyleNode  { return nil }
