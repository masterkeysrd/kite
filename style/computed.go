package style

import "image/color"

// Computed is the fully-resolved style produced by the style resolver (Task 06).
// Every field has a concrete value; there are no [Optional] wrappers. The
// render layer reads Computed exclusively; it never inspects [Style] directly.
type Computed struct {
	// --- Flex / display -------------------------------------------------------

	Display        Display       `json:"display"`
	ListStyleType  ListStyleType `json:"listStyleType"`
	FlexDirection  FlexDirection `json:"flexDirection"`
	FlexWrap       FlexWrap      `json:"flexWrap"`
	JustifyContent Justify       `json:"justifyContent"`
	AlignItems     Align         `json:"alignItems"`
	AlignContent   Align         `json:"alignContent"`
	AlignSelf      Align         `json:"alignSelf"`
	Gap            GapValue      `json:"gap"`
	Flex           FlexItemValue `json:"flex"`
	Order          int           `json:"order"`

	// --- Box model ------------------------------------------------------------

	Width     Dimension       `json:"width"`
	Height    Dimension       `json:"height"`
	MinWidth  Dimension       `json:"minWidth"`
	MaxWidth  Dimension       `json:"maxWidth"`
	MinHeight Dimension       `json:"minHeight"`
	MaxHeight Dimension       `json:"maxHeight"`
	Padding   EdgeValues[int] `json:"padding"`
	Margin    EdgeValues[int] `json:"margin"`
	Border    Border          `json:"border"`

	// --- Color / visual -------------------------------------------------------

	// Foreground is never nil after resolution; it may be [TerminalDefault].
	Foreground color.Color `json:"foreground"`
	// Background is never nil after resolution; it may be color.Transparent.
	Background    color.Color `json:"background"`
	Bold          bool        `json:"bold"`
	Italic        bool        `json:"italic"`
	Underline     bool        `json:"underline"`
	Strikethrough bool        `json:"strikethrough"`
	Reverse       bool        `json:"reverse"`

	// --- Selection ------------------------------------------------------------

	SelectionForeground color.Color `json:"selectionForeground"`
	SelectionBackground color.Color `json:"selectionBackground"`

	// --- Text -----------------------------------------------------------------

	TextAlign    TextAlign    `json:"textAlign"`
	TextWrap     TextWrap     `json:"textWrap"`
	TextOverflow TextOverflow `json:"textOverflow"`
	WhiteSpace   WhiteSpace   `json:"whiteSpace"`
	WordBreak    WordBreak    `json:"wordBreak"`
	OverflowWrap OverflowWrap `json:"overflowWrap"`

	// --- Overflow / scroll ----------------------------------------------------

	OverflowX Overflow  `json:"overflowX"`
	OverflowY Overflow  `json:"overflowY"`
	Scrollbar Scrollbar `json:"scrollbar"`

	// --- Cursor ---------------------------------------------------------------

	CursorShape CursorShape `json:"cursorShape"`
	CursorColor color.Color `json:"cursorColor"`
}

// AffectsLayout returns true if the change between c and other requires a re-layout.
func (c *Computed) AffectsLayout(other *Computed) bool {
	if c == other {
		return false
	}
	if other == nil {
		return true
	}
	return c.Display != other.Display ||
		c.ListStyleType != other.ListStyleType ||
		c.FlexDirection != other.FlexDirection ||
		c.FlexWrap != other.FlexWrap ||
		c.JustifyContent != other.JustifyContent ||
		c.AlignItems != other.AlignItems ||
		c.AlignContent != other.AlignContent ||
		c.AlignSelf != other.AlignSelf ||
		c.Gap != other.Gap ||
		c.Flex != other.Flex ||
		c.Order != other.Order ||
		c.Width != other.Width ||
		c.Height != other.Height ||
		c.MinWidth != other.MinWidth ||
		c.MaxWidth != other.MaxWidth ||
		c.MinHeight != other.MinHeight ||
		c.MaxHeight != other.MaxHeight ||
		c.Padding != other.Padding ||
		c.Margin != other.Margin ||
		c.Border != other.Border ||
		c.TextAlign != other.TextAlign ||
		c.TextWrap != other.TextWrap ||
		c.TextOverflow != other.TextOverflow ||
		c.WhiteSpace != other.WhiteSpace ||
		c.WordBreak != other.WordBreak ||
		c.OverflowWrap != other.OverflowWrap ||
		c.OverflowX != other.OverflowX ||
		c.OverflowY != other.OverflowY ||
		c.Scrollbar.X != other.Scrollbar.X ||
		c.Scrollbar.Y != other.Scrollbar.Y ||
		c.CursorShape != other.CursorShape ||
		c.CursorColor != other.CursorColor
}

// AffectsPaint returns true if the change between c and other requires a repaint
// (assuming AffectsLayout returned false).
func (c *Computed) AffectsPaint(other *Computed) bool {
	if c == other {
		return false
	}
	if other == nil {
		return true
	}
	return c.Foreground != other.Foreground ||
		c.Background != other.Background ||
		c.Bold != other.Bold ||
		c.Italic != other.Italic ||
		c.Underline != other.Underline ||
		c.Strikethrough != other.Strikethrough ||
		c.Reverse != other.Reverse ||
		c.SelectionForeground != other.SelectionForeground ||
		c.SelectionBackground != other.SelectionBackground ||
		c.Scrollbar.TrackGlyph != other.Scrollbar.TrackGlyph ||
		c.Scrollbar.TrackColor != other.Scrollbar.TrackColor ||
		c.Scrollbar.ThumbGlyph != other.Scrollbar.ThumbGlyph ||
		c.Scrollbar.ThumbColor != other.Scrollbar.ThumbColor
}
