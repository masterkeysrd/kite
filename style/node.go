// Package style defines the Style and Computed value types.
package style

import "image/color"

// DefaultStyle returns a copy of the baseline Computed for a kitex element:
// the value each property takes when neither the element nor any ancestor
// provides it. The returned value is a plain struct copy; callers may modify
// it freely.
func DefaultStyle() Computed {
	return Computed{
		// Display / flex -------------------------------------------------------
		Display:        DisplayBlock,     // block-level box by default
		ListStyleType:  ListStyleInherit, // inherit marker by default
		FlexDirection:  FlexRow,          // main axis is left → right
		FlexWrap:       FlexNoWrap,       // single-line by default
		JustifyContent: JustifyStart,     // pack toward main-start
		AlignItems:     AlignStretch,     // stretch items by default (CSS parity)
		AlignContent:   AlignStretch,     // stretch lines by default
		AlignSelf:      AlignStart,       // defers to parent AlignItems
		Gap:            GapValue{},       // no inter-child gap
		Flex:           FlexItemValue{Grow: 0, Shrink: 1, Basis: Auto},
		Order:          0, // default order

		GridTemplateColumns: nil,
		GridTemplateRows:    nil,
		GridColumnGap:       0,
		GridRowGap:          0,
		GridColumn:          GridPlacement{},
		GridRow:             GridPlacement{},

		// Box model ------------------------------------------------------------
		Width:     Auto,              // width determined by content / parent
		Height:    Auto,              // height determined by content / parent
		MinWidth:  Auto,              // CSS parity: default min-width is auto (min-content)
		MaxWidth:  Content,           // natural content width (no maximum)
		MinHeight: Auto,              // CSS parity: default min-height is auto (min-content)
		MaxHeight: Content,           // natural content height (no maximum)
		Padding:   EdgeValues[int]{}, // zero padding on all sides
		Margin:    EdgeValues[int]{}, // zero margin on all sides
		Border:    Border{},          // no border (BorderNone = 0)

		// Color / visual -------------------------------------------------------
		Foreground:    TerminalDefault,   // terminal's default foreground
		Background:    color.Transparent, // transparent; do not paint bg cell
		Bold:          false,
		Italic:        false,
		Underline:     false,
		Strikethrough: false,
		Reverse:       false,

		// Text -----------------------------------------------------------------
		TextAlign:    TextAlignLeft,      // left-align text
		TextWrap:     TextWrapWord,       // legacy wrap setting
		TextOverflow: TextOverflowClip,   // clip overflowing text
		WhiteSpace:   WhiteSpaceNormal,   // collapse whitespace, wrapping allowed
		WordBreak:    WordBreakNormal,    // break at the shaper's soft points
		OverflowWrap: OverflowWrapNormal, // no emergency intra-word breaking

		// Overflow / scroll ----------------------------------------------------
		OverflowX: OverflowVisible,
		OverflowY: OverflowVisible,
		Scrollbar: Scrollbar{},

		// Cursor ---------------------------------------------------------------
		CursorShape:         CursorShapeBlockBlink,
		CursorColor:         TerminalDefault,
		SelectionForeground: nil, // fallback to inversion
		SelectionBackground: nil, // fallback to inversion
	}
}
