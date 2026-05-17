package style

import "image/color"

// Computed is the fully-resolved style produced by the style resolver (Task 06).
// Every field has a concrete value; there are no [Optional] wrappers. The
// render layer reads Computed exclusively; it never inspects [Style] directly.
type Computed struct {
	// --- Flex / display -------------------------------------------------------

	Display        Display
	FlexDirection  FlexDirection
	FlexWrap       FlexWrap
	JustifyContent Justify
	AlignItems     Align
	AlignContent   Align
	AlignSelf      Align
	Gap            GapValue
	Flex           FlexItemValue

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
