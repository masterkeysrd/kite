package style

import "image/color"

// Computed is the fully-resolved style produced by the style resolver (Task 06).
// Every field has a concrete value; there are no [Optional] wrappers. The
// render layer reads Computed exclusively; it never inspects [Style] directly.
type Computed struct {
	// --- Flex / display -------------------------------------------------------

	Display        Display
	ListStyleType  ListStyleType
	FlexDirection  FlexDirection
	FlexWrap       FlexWrap
	JustifyContent Justify
	AlignItems     Align
	AlignContent   Align
	AlignSelf      Align
	Gap            GapValue
	Flex           FlexItemValue
	Order          int

	// --- Box model ------------------------------------------------------------

	Width     Dimension
	Height    Dimension
	MinWidth  Dimension
	MaxWidth  Dimension
	MinHeight Dimension
	MaxHeight Dimension
	Padding   EdgeValues[int]
	Margin    EdgeValues[int]
	Border    Border

	// --- Color / visual -------------------------------------------------------

	// Foreground is never nil after resolution; it may be [TerminalDefault].
	Foreground color.Color
	// Background is never nil after resolution; it may be color.Transparent.
	Background    color.Color
	Bold          bool
	Italic        bool
	Underline     bool
	Strikethrough bool
	Reverse       bool

	// --- Text -----------------------------------------------------------------

	TextAlign    TextAlign
	TextWrap     TextWrap
	TextOverflow TextOverflow
	WhiteSpace   WhiteSpace
	WordBreak    WordBreak
	OverflowWrap OverflowWrap

	// --- Overflow / scroll ----------------------------------------------------

	OverflowX Overflow
	OverflowY Overflow
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
		c.OverflowY != other.OverflowY
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
		c.Reverse != other.Reverse
}
