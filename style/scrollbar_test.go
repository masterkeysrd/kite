package style

import (
	"image/color"
	"testing"
)

func TestStyle_ScrollbarCascade(t *testing.T) {
	base := Style{}.ScrollbarY(true).ScrollbarThumb('+', color.White)
	override := Style{}.ScrollbarX(true).ScrollbarTrack('.', color.Black)

	merged := base.Merge(override)

	sb := merged.Scrollbar.Value()
	if !sb.Y.UnwrapOr(false) {
		t.Errorf("expected ScrollbarY to be true")
	}
	if !sb.X.UnwrapOr(false) {
		t.Errorf("expected ScrollbarX to be true")
	}
	if sb.ThumbGlyph.Value() != '+' {
		t.Errorf("expected ThumbGlyph to be '+', got %c", sb.ThumbGlyph.Value())
	}
	if sb.TrackGlyph.Value() != '.' {
		t.Errorf("expected TrackGlyph to be '.', got %c", sb.TrackGlyph.Value())
	}
}

func TestStyle_ResolverScrollbarDefaults(t *testing.T) {
	resolver := NewResolver()
	node := &mockStyleNode{
		raw: Style{}.ScrollbarY(true),
	}

	computed := resolver.Resolve(node, nil)
	sb := computed.Scrollbar

	if sb.TrackGlyph.Value() != DefaultScrollbarTrackVertical {
		t.Errorf("expected default vertical track glyph, got %c", sb.TrackGlyph.Value())
	}
	if sb.ThumbGlyph.Value() != DefaultScrollbarThumbVertical {
		t.Errorf("expected default vertical thumb glyph, got %c", sb.ThumbGlyph.Value())
	}
}

type mockStyleNode struct {
	raw      Style
	computed *Computed
}

func (m *mockStyleNode) RawStyle() Style              { return m.raw }
func (m *mockStyleNode) DefaultStyle() Style          { return Style{} }
func (m *mockStyleNode) IntrinsicStyle() Style        { return Style{} }
func (m *mockStyleNode) ComputedStyle() *Computed     { return m.computed }
func (m *mockStyleNode) SetComputedStyle(c *Computed) { m.computed = c }
func (m *mockStyleNode) IsDirtyStyle() bool           { return true }
func (m *mockStyleNode) HasDirtyStyleChild() bool     { return false }
func (m *mockStyleNode) ClearDirtyStyle()             {}
func (m *mockStyleNode) ClearChildNeedsStyle()        {}
func (m *mockStyleNode) StyleParent() StyleNode       { return nil }
func (m *mockStyleNode) StyleFirstChild() StyleNode   { return nil }
func (m *mockStyleNode) StyleNextSibling() StyleNode  { return nil }
