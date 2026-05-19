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

// BorderGlyphs defines the characters used to draw a border. The base fields
// (H, V, TL, TR, BL, BR) specify the default glyph set. The Override* fields
// are Optional; when set they replace the corresponding base field so callers
// can swap a single corner without redefining the entire glyph set.
//
// This supports the tee-junction pattern required by CardHeader / CardFooter
// OverrideBL / OverrideBR to "├" / "┤" while keeping
// the rest of the rounded glyph set intact.
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

// Border describes the per-side border properties of an element. Each side
// can have an independent width, style, and color. The Glyphs field is used
// when the style for a side is [BorderCustom] or when individual corner
// overrides are needed (e.g. tee-junction corners).
type Border struct {
	// Width holds the number of cells each side's border occupies.
	Width EdgeValues[int]
	// Style selects the glyph set for each side.
	Style EdgeValues[BorderStyle]
	// Color sets the foreground color for each side's border.
	Color EdgeValues[color.Color]
	// Glyphs provides per-corner override characters.
	Glyphs BorderGlyphs
}
