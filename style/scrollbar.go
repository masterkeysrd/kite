package style

import (
	"image/color"
)

// Scrollbar defines the visual appearance and visibility of scrollbars.
// Screen real estate is precious in a TUI, so scrollbars are opt-in.
type Scrollbar struct {
	// X enables the horizontal scrollbar.
	X Optional[bool]
	// Y enables the vertical scrollbar.
	Y Optional[bool]
	// TrackGlyph is the rune used for the scrollbar track.
	TrackGlyph Optional[rune]
	// TrackColor is the color of the scrollbar track.
	TrackColor Optional[color.Color]
	// ThumbGlyph is the rune used for the scrollbar thumb.
	ThumbGlyph Optional[rune]
	// ThumbColor is the color of the scrollbar thumb.
	ThumbColor Optional[color.Color]
}

// Merge returns a new [Scrollbar] where each set field in override overlays
// the corresponding field in s.
func (s Scrollbar) Merge(override Scrollbar) Scrollbar {
	return Scrollbar{
		X:          s.X.Merge(override.X),
		Y:          s.Y.Merge(override.Y),
		TrackGlyph: s.TrackGlyph.Merge(override.TrackGlyph),
		TrackColor: s.TrackColor.Merge(override.TrackColor),
		ThumbGlyph: s.ThumbGlyph.Merge(override.ThumbGlyph),
		ThumbColor: s.ThumbColor.Merge(override.ThumbColor),
	}
}

// Default glyphs for scrollbars.
const (
	DefaultScrollbarTrackVertical   = '│'
	DefaultScrollbarThumbVertical   = '┃'
	DefaultScrollbarTrackHorizontal = '─'
	DefaultScrollbarThumbHorizontal = '━'
)
