package style

import "image/color"

// BorderStyle selects the pre-defined glyph set for a border.
type BorderStyle uint8

const (
	// BorderNone draws no border.
	BorderNone BorderStyle = iota
	// BorderSingle draws a single-line border (─ │ ┌ ┐ └ ┘).
	BorderSingle
	// BorderDouble draws a double-line border (═ ║ ╔ ╗ ╚ ╝).
	BorderDouble
	// BorderRounded draws a rounded border (─ │ ╭ ╮ ╰ ╯).
	BorderRounded
	// BorderThick draws a heavy-line border (━ ┃ ┏ ┓ ┗ ┛).
	BorderThick
	// BorderASCII draws an ASCII border (- | + + + +).
	BorderASCII
	// BorderCustom uses the glyphs supplied in [BorderGlyphs] verbatim.
	BorderCustom
)

// BorderGlyphsMap defines the standard box-drawing characters for each style.
var BorderGlyphsMap = map[BorderStyle]BorderGlyphs{
	BorderSingle:  {H: "─", V: "│", TL: "┌", TR: "┐", BL: "└", BR: "┘"},
	BorderDouble:  {H: "═", V: "║", TL: "╔", TR: "╗", BL: "╚", BR: "╝"},
	BorderRounded: {H: "─", V: "│", TL: "╭", TR: "╮", BL: "╰", BR: "╯"},
	BorderThick:   {H: "━", V: "┃", TL: "┏", TR: "┓", BL: "┗", BR: "┛"},
	BorderASCII:   {H: "-", V: "|", TL: "+", TR: "+", BL: "+", BR: "+"},
}

// BorderGlyphs defines the characters used to draw a border. The base fields
// (H, V, TL, TR, BL, BR) specify the default glyph set. The Override* fields
// are Optional; when set they replace the corresponding base field so callers
// can swap a single corner without redefining the entire glyph set.
type BorderGlyphs struct {
	// Base glyphs (active when the corresponding Override is unset).
	H, V   string // horizontal and vertical bar
	TL, TR string // top-left and top-right corners
	BL, BR string // bottom-left and bottom-right corners

	// Per-corner optional overrides. When set, these take precedence over
	// the matching base field.
	OverrideTL Optional[string]
	OverrideTR Optional[string]
	OverrideBL Optional[string]
	OverrideBR Optional[string]
}

// EffectiveTL returns the top-left corner glyph, applying any override.
func (g BorderGlyphs) EffectiveTL() string {
	if g.OverrideTL.IsSet() {
		return g.OverrideTL.Value()
	}
	return g.TL
}

// EffectiveTR returns the top-right corner glyph, applying any override.
func (g BorderGlyphs) EffectiveTR() string {
	if g.OverrideTR.IsSet() {
		return g.OverrideTR.Value()
	}
	return g.TR
}

// EffectiveBL returns the bottom-left corner glyph, applying any override.
func (g BorderGlyphs) EffectiveBL() string {
	if g.OverrideBL.IsSet() {
		return g.OverrideBL.Value()
	}
	return g.BL
}

// EffectiveBR returns the bottom-right corner glyph, applying any override.
func (g BorderGlyphs) EffectiveBR() string {
	if g.OverrideBR.IsSet() {
		return g.OverrideBR.Value()
	}
	return g.BR
}

// Border describes the per-side border properties of an element.
// In a terminal UI, borders always occupy exactly one cell if present.
type Border struct {
	// Edges defines which sides of the box have a border visible.
	Edges EdgeValues[bool]
	// Styles selects the glyph set for each side.
	Styles EdgeValues[BorderStyle]
	// Colors sets the foreground color for each side's border.
	Colors EdgeValues[color.Color]
	// Glyphs provides per-corner override characters.
	Glyphs BorderGlyphs
}

// SingleBorder returns a standard single-line border on all sides.
func SingleBorder() Border {
	return Border{
		Edges:  EdgeAll(true),
		Styles: EdgeAll(BorderSingle),
	}
}

// DoubleBorder returns a standard double-line border on all sides.
func DoubleBorder() Border {
	return Border{
		Edges:  EdgeAll(true),
		Styles: EdgeAll(BorderDouble),
	}
}

// RoundedBorder returns a single-line border with rounded corners.
func RoundedBorder() Border {
	return Border{
		Edges:  EdgeAll(true),
		Styles: EdgeAll(BorderRounded),
	}
}

// EmptyBorder returns a border with no edges visible.
func EmptyBorder() Border {
	return Border{}
}

// Some wraps the border in an [Optional].
func (b Border) Some() Optional[Border] {
	return Some(b)
}

// Color sets the color for all border edges.
func (b Border) Color(c color.Color) Border {
	b.Colors = EdgeAll(c)
	return b
}

// Style sets the border style for all edges.
func (b Border) Style(s BorderStyle) Border {
	b.Styles = EdgeAll(s)
	return b
}

// Top toggles the visibility of the top border edge.
func (b Border) Top(visible bool) Border {
	b.Edges.Top = visible
	return b
}

// Right toggles the visibility of the right border edge.
func (b Border) Right(visible bool) Border {
	b.Edges.Right = visible
	return b
}

// Bottom toggles the visibility of the bottom border edge.
func (b Border) Bottom(visible bool) Border {
	b.Edges.Bottom = visible
	return b
}

// Left toggles the visibility of the left border edge.
func (b Border) Left(visible bool) Border {
	b.Edges.Left = visible
	return b
}

// CornerOverride sets manual overrides for the four corners.
func (b Border) CornerOverride(tl, tr, bl, br string) Border {
	b.Glyphs.OverrideTL = Some(tl)
	b.Glyphs.OverrideTR = Some(tr)
	b.Glyphs.OverrideBL = Some(bl)
	b.Glyphs.OverrideBR = Some(br)
	return b
}

// Widths returns the integer width of each edge (1 or 0).
func (b Border) Widths() EdgeValues[int] {
	var w EdgeValues[int]
	if b.Edges.Top {
		w.Top = 1
	}
	if b.Edges.Right {
		w.Right = 1
	}
	if b.Edges.Bottom {
		w.Bottom = 1
	}
	if b.Edges.Left {
		w.Left = 1
	}
	return w
}
