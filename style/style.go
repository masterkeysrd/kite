package style

import "image/color"

// Style is a sparse set of visual and layout properties for a kitex element.
// Every field is [Optional] so that callers can compose styles without
// overwriting fields they did not intend to change.
//
// Construct via struct literals using [Some]:
//
//	s := Style{
//	    display:       Some(DisplayFlex),
//	    flexDirection: Some(FlexColumn),
//	    gap:           Some(Gap(1)),
//	}
//
// Combine two styles with [Style.Merge].
type Style struct {
	// --- Flex / display -------------------------------------------------------

	// Display controls how the element participates in its parent's layout.
	display Optional[Display]
	// ListStyleType defines the appearance of a list item marker.
	listStyleType Optional[ListStyleType]
	// FlexDirection sets the main axis of a flex container.
	flexDirection Optional[FlexDirection]
	// FlexWrap controls whether items wrap onto new lines.
	flexWrap Optional[FlexWrap]
	// JustifyContent sets main-axis alignment inside a flex container.
	justifyContent Optional[Justify]
	// AlignItems sets cross-axis alignment for all children.
	alignItems Optional[Align]
	// AlignContent sets cross-axis alignment for multiple flex lines.
	alignContent Optional[Align]
	// AlignSelf overrides the parent's AlignItems for this element.
	alignSelf Optional[Align]
	// Gap sets the row and column gaps between flex children.
	gap Optional[GapValue]
	// Flex sets the grow, shrink, and basis for a flex item.
	flex Optional[FlexItemValue]
	// Order sets the order of a flex item.
	order Optional[int]

	// --- Grid -----------------------------------------------------------------

	// GridTemplateColumns defines the line names and track sizing functions of the grid columns.
	gridTemplateColumns Optional[[]GridTrackSize]
	// GridTemplateRows defines the line names and track sizing functions of the grid rows.
	gridTemplateRows Optional[[]GridTrackSize]
	// GridColumnGap specifies the size of the grid lines between columns.
	gridColumnGap Optional[int]
	// GridRowGap specifies the size of the grid lines between rows.
	gridRowGap Optional[int]
	// GridColumn specifies a grid item's size and location within the grid column.
	gridColumn Optional[GridPlacement]
	// GridRow specifies a grid item's size and location within the grid row.
	gridRow Optional[GridPlacement]

	// --- Box model ------------------------------------------------------------

	// Width sets the preferred width of the element.
	width Optional[Dimension]
	// Height sets the preferred height of the element.
	height Optional[Dimension]
	// MinWidth sets the minimum width of the element.
	minWidth Optional[Dimension]
	// MaxWidth sets the maximum width of the element.
	maxWidth Optional[Dimension]
	// MinHeight sets the minimum height of the element.
	minHeight Optional[Dimension]
	// MaxHeight sets the maximum height of the element.
	maxHeight Optional[Dimension]
	// Padding is the inner spacing between the element's content and its border.
	padding Optional[EdgeValues[int]]
	// Margin is the outer spacing between the element's border and its siblings.
	margin Optional[EdgeValues[int]]
	// Border sets the element's border.
	border Optional[Border]

	// --- Color / visual -------------------------------------------------------

	// Foreground sets the text (foreground) color.
	foreground Optional[color.Color]
	// Background sets the cell background color.
	background Optional[color.Color]
	// Bold enables bold text rendering.
	bold Optional[bool]
	// Italic enables italic text rendering.
	italic Optional[bool]
	// Underline enables underlined text.
	underline Optional[bool]
	// Strikethrough enables strikethrough text.
	strikethrough Optional[bool]
	// Reverse swaps foreground and background colors.
	reverse Optional[bool]

	// --- Selection ------------------------------------------------------------

	// SelectionForeground sets the text color of selected content.
	selectionForeground Optional[color.Color]
	// SelectionBackground sets the background color of selected content.
	selectionBackground Optional[color.Color]

	// --- Text -----------------------------------------------------------------

	// TextAlign controls horizontal text alignment within the element.
	textAlign Optional[TextAlign]
	// TextWrap controls how text is wrapped at the element's width.
	textWrap Optional[TextWrap]
	// TextOverflow controls how non-wrapping text is handled when it overflows.
	textOverflow Optional[TextOverflow]
	// WhiteSpace controls whitespace preservation and whether wrapping is allowed.
	whiteSpace Optional[WhiteSpace]
	// WordBreak controls whether breaks may occur between grapheme clusters.
	wordBreak Optional[WordBreak]
	// OverflowWrap controls emergency breaking inside unbreakable runs.
	overflowWrap Optional[OverflowWrap]

	// --- Overflow / scroll ----------------------------------------------------

	// OverflowX controls how content that overflows on the horizontal axis is handled.
	// Defaults to OverflowVisible.
	overflowX Optional[Overflow]
	// OverflowY controls how content that overflows on the vertical axis is handled.
	// Defaults to OverflowVisible.
	overflowY Optional[Overflow]
	// Scrollbar defines the visual appearance of scrollbars.
	scrollbar Optional[Scrollbar]

	// --- Cursor ---------------------------------------------------------------

	// Cursor defines the terminal hardware cursor properties when this element is focused.
	cursor Optional[Cursor]
}

// Apply returns a copy of base with every set field in s overlaid on top.
// It is the bridge between the sparse [Style] author model and the fully-resolved
// [Computed] type: the resolver calls it once per dirty node after applying
// inheritance, so all the per-property Optional checks live here rather than
// in the resolver.
func (s Style) Apply(base Computed) Computed {
	if s.display.IsSet() {
		base.Display = s.display.Value()
	}
	if s.listStyleType.IsSet() {
		base.ListStyleType = s.listStyleType.Value()
	}
	if s.flexDirection.IsSet() {
		base.FlexDirection = s.flexDirection.Value()
	}
	if s.flexWrap.IsSet() {
		base.FlexWrap = s.flexWrap.Value()
	}
	if s.justifyContent.IsSet() {
		base.JustifyContent = s.justifyContent.Value()
	}
	if s.alignItems.IsSet() {
		base.AlignItems = s.alignItems.Value()
	}
	if s.alignContent.IsSet() {
		base.AlignContent = s.alignContent.Value()
	}
	if s.alignSelf.IsSet() {
		base.AlignSelf = s.alignSelf.Value()
	}
	if s.gap.IsSet() {
		base.Gap = s.gap.Value()
	}
	if s.flex.IsSet() {
		base.Flex = s.flex.Value()
	}
	if s.order.IsSet() {
		base.Order = s.order.Value()
	}
	if s.gridTemplateColumns.IsSet() {
		base.GridTemplateColumns = s.gridTemplateColumns.Value()
	}
	if s.gridTemplateRows.IsSet() {
		base.GridTemplateRows = s.gridTemplateRows.Value()
	}
	if s.gridColumnGap.IsSet() {
		base.GridColumnGap = s.gridColumnGap.Value()
	}
	if s.gridRowGap.IsSet() {
		base.GridRowGap = s.gridRowGap.Value()
	}
	if s.gridColumn.IsSet() {
		base.GridColumn = s.gridColumn.Value()
	}
	if s.gridRow.IsSet() {
		base.GridRow = s.gridRow.Value()
	}
	if s.width.IsSet() {
		base.Width = s.width.Value()
	}
	if s.height.IsSet() {
		base.Height = s.height.Value()
	}
	if s.minWidth.IsSet() {
		base.MinWidth = s.minWidth.Value()
	}
	if s.maxWidth.IsSet() {
		base.MaxWidth = s.maxWidth.Value()
	}
	if s.minHeight.IsSet() {
		base.MinHeight = s.minHeight.Value()
	}
	if s.maxHeight.IsSet() {
		base.MaxHeight = s.maxHeight.Value()
	}
	if s.padding.IsSet() {
		base.Padding = s.padding.Value()
	}
	if s.margin.IsSet() {
		base.Margin = s.margin.Value()
	}
	if s.border.IsSet() {
		base.Border = s.border.Value()
	}
	if s.foreground.IsSet() {
		base.Foreground = s.foreground.Value()
	}
	if s.background.IsSet() {
		base.Background = s.background.Value()
	}
	if s.bold.IsSet() {
		base.Bold = s.bold.Value()
	}
	if s.italic.IsSet() {
		base.Italic = s.italic.Value()
	}
	if s.underline.IsSet() {
		base.Underline = s.underline.Value()
	}
	if s.strikethrough.IsSet() {
		base.Strikethrough = s.strikethrough.Value()
	}
	if s.reverse.IsSet() {
		base.Reverse = s.reverse.Value()
	}
	if s.selectionForeground.IsSet() {
		base.SelectionForeground = s.selectionForeground.Value()
	}
	if s.selectionBackground.IsSet() {
		base.SelectionBackground = s.selectionBackground.Value()
	}
	if s.cursor.IsSet() {
		base.Cursor = base.Cursor.Merge(s.cursor.Value())
	}
	if s.textAlign.IsSet() {
		base.TextAlign = s.textAlign.Value()
	}
	if s.textWrap.IsSet() {
		base.TextWrap = s.textWrap.Value()
	}
	if s.textOverflow.IsSet() {
		base.TextOverflow = s.textOverflow.Value()
	}
	if s.whiteSpace.IsSet() {
		base.WhiteSpace = s.whiteSpace.Value()
	}
	if s.wordBreak.IsSet() {
		base.WordBreak = s.wordBreak.Value()
	}
	if s.overflowWrap.IsSet() {
		base.OverflowWrap = s.overflowWrap.Value()
	}
	if s.overflowX.IsSet() {
		base.OverflowX = s.overflowX.Value()
	}
	if s.overflowY.IsSet() {
		base.OverflowY = s.overflowY.Value()
	}
	if s.scrollbar.IsSet() {
		base.Scrollbar = base.Scrollbar.Merge(s.scrollbar.Value())
	}
	return base
}

// Merge returns a new [Style] where each field is taken from override if it is
// set, otherwise from s. Neither s nor override is mutated.
//
// This implements field-level composition: callers can build up a final style
// from a base theme plus successive override layers:
//
//	final := theme.Merge(widgetDefault).Merge(userProvided)
func (s Style) Merge(override Style) Style {
	return Style{
		display:        s.display.Merge(override.display),
		flexDirection:  s.flexDirection.Merge(override.flexDirection),
		flexWrap:       s.flexWrap.Merge(override.flexWrap),
		justifyContent: s.justifyContent.Merge(override.justifyContent),
		alignItems:     s.alignItems.Merge(override.alignItems),
		alignContent:   s.alignContent.Merge(override.alignContent),
		alignSelf:      s.alignSelf.Merge(override.alignSelf),
		gap:            s.gap.Merge(override.gap),
		flex:           s.flex.Merge(override.flex),
		order:          s.order.Merge(override.order),

		gridTemplateColumns: s.gridTemplateColumns.Merge(override.gridTemplateColumns),
		gridTemplateRows:    s.gridTemplateRows.Merge(override.gridTemplateRows),
		gridColumnGap:       s.gridColumnGap.Merge(override.gridColumnGap),
		gridRowGap:          s.gridRowGap.Merge(override.gridRowGap),
		gridColumn:          s.gridColumn.Merge(override.gridColumn),
		gridRow:             s.gridRow.Merge(override.gridRow),

		width:     s.width.Merge(override.width),
		height:    s.height.Merge(override.height),
		minWidth:  s.minWidth.Merge(override.minWidth),
		maxWidth:  s.maxWidth.Merge(override.maxWidth),
		minHeight: s.minHeight.Merge(override.minHeight),
		maxHeight: s.maxHeight.Merge(override.maxHeight),
		padding:   s.padding.Merge(override.padding),
		margin:    s.margin.Merge(override.margin),
		border:    s.border.Merge(override.border),

		foreground:          s.foreground.Merge(override.foreground),
		background:          s.background.Merge(override.background),
		bold:                s.bold.Merge(override.bold),
		italic:              s.italic.Merge(override.italic),
		underline:           s.underline.Merge(override.underline),
		strikethrough:       s.strikethrough.Merge(override.strikethrough),
		reverse:             s.reverse.Merge(override.reverse),
		selectionForeground: s.selectionForeground.Merge(override.selectionForeground),
		selectionBackground: s.selectionBackground.Merge(override.selectionBackground),

		textAlign:    s.textAlign.Merge(override.textAlign),
		textWrap:     s.textWrap.Merge(override.textWrap),
		textOverflow: s.textOverflow.Merge(override.textOverflow),
		whiteSpace:   s.whiteSpace.Merge(override.whiteSpace),
		wordBreak:    s.wordBreak.Merge(override.wordBreak),
		overflowWrap: s.overflowWrap.Merge(override.overflowWrap),

		overflowX: s.overflowX.Merge(override.overflowX),
		overflowY: s.overflowY.Merge(override.overflowY),
		scrollbar: mergeOptionalScrollbar(s.scrollbar, override.scrollbar),
		cursor:    mergeOptionalCursor(s.cursor, override.cursor),
	}
}

func mergeOptionalCursor(base, override Optional[Cursor]) Optional[Cursor] {
	if !override.IsSet() {
		return base
	}
	if !base.IsSet() {
		return override
	}
	return Some(base.Value().Merge(override.Value()))
}

func mergeOptionalScrollbar(base, override Optional[Scrollbar]) Optional[Scrollbar] {
	if !override.IsSet() {
		return base
	}
	if !base.IsSet() {
		return override
	}
	return Some(base.Value().Merge(override.Value()))
}

// Overflow sets both OverflowX and OverflowY to v and returns the modified style.
func (s Style) Overflow(v Overflow) Style {
	s.overflowX = Some(v)
	s.overflowY = Some(v)
	return s
}

// ScrollbarX enables or disables the horizontal scrollbar.
func (s Style) ScrollbarX(v bool) Style {
	sb, _ := s.scrollbar.Get()
	sb.X = Some(v)
	s.scrollbar = Some(sb)
	return s
}

// ScrollbarY enables or disables the vertical scrollbar.
func (s Style) ScrollbarY(v bool) Style {
	sb, _ := s.scrollbar.Get()
	sb.Y = Some(v)
	s.scrollbar = Some(sb)
	return s
}

// ScrollbarThumb sets the glyph and color for the scrollbar thumb.
func (s Style) ScrollbarThumb(glyph rune, c color.Color) Style {
	sb, _ := s.scrollbar.Get()
	sb.ThumbGlyph = Some(glyph)
	sb.ThumbColor = Some(c)
	s.scrollbar = Some(sb)
	return s
}

// ScrollbarTrack sets the glyph and color for the scrollbar track.
func (s Style) ScrollbarTrack(glyph rune, c color.Color) Style {
	sb, _ := s.scrollbar.Get()
	sb.TrackGlyph = Some(glyph)
	sb.TrackColor = Some(c)
	s.scrollbar = Some(sb)
	return s
}

// --- Builder and Getter methods ---------------------------------------------

// S returns a new empty Style.
func S() Style { return Style{} }

// DisplayOpt returns the optional value of Display.
func (s Style) DisplayOpt() Optional[Display] { return s.display }

// Display sets the Display property.
func (s Style) Display(v Display) Style {
	s.display = Some(v)
	return s
}

// ListStyleTypeOpt returns the optional value of ListStyleType.
func (s Style) ListStyleTypeOpt() Optional[ListStyleType] { return s.listStyleType }

// ListStyleType sets the ListStyleType property.
func (s Style) ListStyleType(v ListStyleType) Style {
	s.listStyleType = Some(v)
	return s
}

// FlexDirectionOpt returns the optional value of FlexDirection.
func (s Style) FlexDirectionOpt() Optional[FlexDirection] { return s.flexDirection }

// FlexDirection sets the FlexDirection property.
func (s Style) FlexDirection(v FlexDirection) Style {
	s.flexDirection = Some(v)
	return s
}

// FlexWrapOpt returns the optional value of FlexWrap.
func (s Style) FlexWrapOpt() Optional[FlexWrap] { return s.flexWrap }

// FlexWrap sets the FlexWrap property.
func (s Style) FlexWrap(v FlexWrap) Style {
	s.flexWrap = Some(v)
	return s
}

// JustifyContentOpt returns the optional value of JustifyContent.
func (s Style) JustifyContentOpt() Optional[Justify] { return s.justifyContent }

// JustifyContent sets the JustifyContent property.
func (s Style) JustifyContent(v Justify) Style {
	s.justifyContent = Some(v)
	return s
}

// AlignItemsOpt returns the optional value of AlignItems.
func (s Style) AlignItemsOpt() Optional[Align] { return s.alignItems }

// AlignItems sets the AlignItems property.
func (s Style) AlignItems(v Align) Style {
	s.alignItems = Some(v)
	return s
}

// AlignContentOpt returns the optional value of AlignContent.
func (s Style) AlignContentOpt() Optional[Align] { return s.alignContent }

// AlignContent sets the AlignContent property.
func (s Style) AlignContent(v Align) Style {
	s.alignContent = Some(v)
	return s
}

// AlignSelfOpt returns the optional value of AlignSelf.
func (s Style) AlignSelfOpt() Optional[Align] { return s.alignSelf }

// AlignSelf sets the AlignSelf property.
func (s Style) AlignSelf(v Align) Style {
	s.alignSelf = Some(v)
	return s
}

// GapOpt returns the optional value of Gap.
func (s Style) GapOpt() Optional[GapValue] { return s.gap }

// Gap sets the Gap property.
func (s Style) Gap(v GapValue) Style {
	s.gap = Some(v)
	return s
}

// FlexOpt returns the optional value of Flex.
func (s Style) FlexOpt() Optional[FlexItemValue] { return s.flex }

// Flex sets the Flex property.
func (s Style) Flex(v FlexItemValue) Style {
	s.flex = Some(v)
	return s
}

// OrderOpt returns the optional value of Order.
func (s Style) OrderOpt() Optional[int] { return s.order }

// Order sets the Order property.
func (s Style) Order(v int) Style {
	s.order = Some(v)
	return s
}

// GridTemplateColumnsOpt returns the optional value of GridTemplateColumns.
func (s Style) GridTemplateColumnsOpt() Optional[[]GridTrackSize] { return s.gridTemplateColumns }

// GridTemplateColumns sets the GridTemplateColumns property.
func (s Style) GridTemplateColumns(v []GridTrackSize) Style {
	s.gridTemplateColumns = Some(v)
	return s
}

// GridTemplateRowsOpt returns the optional value of GridTemplateRows.
func (s Style) GridTemplateRowsOpt() Optional[[]GridTrackSize] { return s.gridTemplateRows }

// GridTemplateRows sets the GridTemplateRows property.
func (s Style) GridTemplateRows(v []GridTrackSize) Style {
	s.gridTemplateRows = Some(v)
	return s
}

// GridColumnGapOpt returns the optional value of GridColumnGap.
func (s Style) GridColumnGapOpt() Optional[int] { return s.gridColumnGap }

// GridColumnGap sets the GridColumnGap property.
func (s Style) GridColumnGap(v int) Style {
	s.gridColumnGap = Some(v)
	return s
}

// GridRowGapOpt returns the optional value of GridRowGap.
func (s Style) GridRowGapOpt() Optional[int] { return s.gridRowGap }

// GridRowGap sets the GridRowGap property.
func (s Style) GridRowGap(v int) Style {
	s.gridRowGap = Some(v)
	return s
}

// GridColumnOpt returns the optional value of GridColumn.
func (s Style) GridColumnOpt() Optional[GridPlacement] { return s.gridColumn }

// GridColumn sets the GridColumn property.
func (s Style) GridColumn(v GridPlacement) Style {
	s.gridColumn = Some(v)
	return s
}

// GridRowOpt returns the optional value of GridRow.
func (s Style) GridRowOpt() Optional[GridPlacement] { return s.gridRow }

// GridRow sets the GridRow property.
func (s Style) GridRow(v GridPlacement) Style {
	s.gridRow = Some(v)
	return s
}

// WidthOpt returns the optional value of Width.
func (s Style) WidthOpt() Optional[Dimension] { return s.width }

// Width sets the Width property.
func (s Style) Width(v Dimension) Style {
	s.width = Some(v)
	return s
}

// HeightOpt returns the optional value of Height.
func (s Style) HeightOpt() Optional[Dimension] { return s.height }

// Height sets the Height property.
func (s Style) Height(v Dimension) Style {
	s.height = Some(v)
	return s
}

// MinWidthOpt returns the optional value of MinWidth.
func (s Style) MinWidthOpt() Optional[Dimension] { return s.minWidth }

// MinWidth sets the MinWidth property.
func (s Style) MinWidth(v Dimension) Style {
	s.minWidth = Some(v)
	return s
}

// MaxWidthOpt returns the optional value of MaxWidth.
func (s Style) MaxWidthOpt() Optional[Dimension] { return s.maxWidth }

// MaxWidth sets the MaxWidth property.
func (s Style) MaxWidth(v Dimension) Style {
	s.maxWidth = Some(v)
	return s
}

// MinHeightOpt returns the optional value of MinHeight.
func (s Style) MinHeightOpt() Optional[Dimension] { return s.minHeight }

// MinHeight sets the MinHeight property.
func (s Style) MinHeight(v Dimension) Style {
	s.minHeight = Some(v)
	return s
}

// MaxHeightOpt returns the optional value of MaxHeight.
func (s Style) MaxHeightOpt() Optional[Dimension] { return s.maxHeight }

// MaxHeight sets the MaxHeight property.
func (s Style) MaxHeight(v Dimension) Style {
	s.maxHeight = Some(v)
	return s
}

// PaddingOpt returns the optional value of Padding.
func (s Style) PaddingOpt() Optional[EdgeValues[int]] { return s.padding }

// Padding sets the Padding property.
func (s Style) Padding(v EdgeValues[int]) Style {
	s.padding = Some(v)
	return s
}

// MarginOpt returns the optional value of Margin.
func (s Style) MarginOpt() Optional[EdgeValues[int]] { return s.margin }

// Margin sets the Margin property.
func (s Style) Margin(v EdgeValues[int]) Style {
	s.margin = Some(v)
	return s
}

// BorderOpt returns the optional value of Border.
func (s Style) BorderOpt() Optional[Border] { return s.border }

// Border sets the Border property.
func (s Style) Border(v Border) Style {
	s.border = Some(v)
	return s
}

// ForegroundOpt returns the optional value of Foreground.
func (s Style) ForegroundOpt() Optional[color.Color] { return s.foreground }

// Foreground sets the Foreground property.
func (s Style) Foreground(v color.Color) Style {
	s.foreground = Some(v)
	return s
}

// BackgroundOpt returns the optional value of Background.
func (s Style) BackgroundOpt() Optional[color.Color] { return s.background }

// Background sets the Background property.
func (s Style) Background(v color.Color) Style {
	s.background = Some(v)
	return s
}

// BoldOpt returns the optional value of Bold.
func (s Style) BoldOpt() Optional[bool] { return s.bold }

// Bold sets the Bold property.
func (s Style) Bold(v bool) Style {
	s.bold = Some(v)
	return s
}

// ItalicOpt returns the optional value of Italic.
func (s Style) ItalicOpt() Optional[bool] { return s.italic }

// Italic sets the Italic property.
func (s Style) Italic(v bool) Style {
	s.italic = Some(v)
	return s
}

// UnderlineOpt returns the optional value of Underline.
func (s Style) UnderlineOpt() Optional[bool] { return s.underline }

// Underline sets the Underline property.
func (s Style) Underline(v bool) Style {
	s.underline = Some(v)
	return s
}

// StrikethroughOpt returns the optional value of Strikethrough.
func (s Style) StrikethroughOpt() Optional[bool] { return s.strikethrough }

// Strikethrough sets the Strikethrough property.
func (s Style) Strikethrough(v bool) Style {
	s.strikethrough = Some(v)
	return s
}

// ReverseOpt returns the optional value of Reverse.
func (s Style) ReverseOpt() Optional[bool] { return s.reverse }

// Reverse sets the Reverse property.
func (s Style) Reverse(v bool) Style {
	s.reverse = Some(v)
	return s
}

// SelectionForegroundOpt returns the optional value of SelectionForeground.
func (s Style) SelectionForegroundOpt() Optional[color.Color] { return s.selectionForeground }

// SelectionForeground sets the SelectionForeground property.
func (s Style) SelectionForeground(v color.Color) Style {
	s.selectionForeground = Some(v)
	return s
}

// SelectionBackgroundOpt returns the optional value of SelectionBackground.
func (s Style) SelectionBackgroundOpt() Optional[color.Color] { return s.selectionBackground }

// SelectionBackground sets the SelectionBackground property.
func (s Style) SelectionBackground(v color.Color) Style {
	s.selectionBackground = Some(v)
	return s
}

// TextAlignOpt returns the optional value of TextAlign.
func (s Style) TextAlignOpt() Optional[TextAlign] { return s.textAlign }

// TextAlign sets the TextAlign property.
func (s Style) TextAlign(v TextAlign) Style {
	s.textAlign = Some(v)
	return s
}

// TextWrapOpt returns the optional value of TextWrap.
func (s Style) TextWrapOpt() Optional[TextWrap] { return s.textWrap }

// TextWrap sets the TextWrap property.
func (s Style) TextWrap(v TextWrap) Style {
	s.textWrap = Some(v)
	return s
}

// TextOverflowOpt returns the optional value of TextOverflow.
func (s Style) TextOverflowOpt() Optional[TextOverflow] { return s.textOverflow }

// TextOverflow sets the TextOverflow property.
func (s Style) TextOverflow(v TextOverflow) Style {
	s.textOverflow = Some(v)
	return s
}

// WhiteSpaceOpt returns the optional value of WhiteSpace.
func (s Style) WhiteSpaceOpt() Optional[WhiteSpace] { return s.whiteSpace }

// WhiteSpace sets the WhiteSpace property.
func (s Style) WhiteSpace(v WhiteSpace) Style {
	s.whiteSpace = Some(v)
	return s
}

// WordBreakOpt returns the optional value of WordBreak.
func (s Style) WordBreakOpt() Optional[WordBreak] { return s.wordBreak }

// WordBreak sets the WordBreak property.
func (s Style) WordBreak(v WordBreak) Style {
	s.wordBreak = Some(v)
	return s
}

// OverflowWrapOpt returns the optional value of OverflowWrap.
func (s Style) OverflowWrapOpt() Optional[OverflowWrap] { return s.overflowWrap }

// OverflowWrap sets the OverflowWrap property.
func (s Style) OverflowWrap(v OverflowWrap) Style {
	s.overflowWrap = Some(v)
	return s
}

// OverflowXOpt returns the optional value of OverflowX.
func (s Style) OverflowXOpt() Optional[Overflow] { return s.overflowX }

// OverflowX sets the OverflowX property.
func (s Style) OverflowX(v Overflow) Style {
	s.overflowX = Some(v)
	return s
}

// OverflowYOpt returns the optional value of OverflowY.
func (s Style) OverflowYOpt() Optional[Overflow] { return s.overflowY }

// OverflowY sets the OverflowY property.
func (s Style) OverflowY(v Overflow) Style {
	s.overflowY = Some(v)
	return s
}

// ScrollbarOpt returns the optional value of Scrollbar.
func (s Style) ScrollbarOpt() Optional[Scrollbar] { return s.scrollbar }

// Scrollbar sets the Scrollbar property.
func (s Style) Scrollbar(v Scrollbar) Style {
	s.scrollbar = Some(v)
	return s
}

// CursorOpt returns the optional value of Cursor.
func (s Style) CursorOpt() Optional[Cursor] { return s.cursor }

// Cursor sets the Cursor property.
func (s Style) Cursor(v Cursor) Style {
	s.cursor = Some(v)
	return s
}

// TopBorder toggles the visibility of the top border edge.
func (s Style) TopBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Top = visible
	s.border = Some(b)
	return s
}

// RightBorder toggles the visibility of the right border edge.
func (s Style) RightBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Right = visible
	s.border = Some(b)
	return s
}

// BottomBorder toggles the visibility of the bottom border edge.
func (s Style) BottomBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Bottom = visible
	s.border = Some(b)
	return s
}

// LeftBorder toggles the visibility of the left border edge.
func (s Style) LeftBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Left = visible
	s.border = Some(b)
	return s
}

// HorizontalBorder toggles the visibility of the left and right border edges.
func (s Style) HorizontalBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Left = visible
	b.Edges.Right = visible
	s.border = Some(b)
	return s
}

// VerticalBorder toggles the visibility of the top and bottom border edges.
func (s Style) VerticalBorder(visible bool) Style {
	b, _ := s.border.Get()
	b.Edges.Top = visible
	b.Edges.Bottom = visible
	s.border = Some(b)
	return s
}

// TopMargin sets the top margin.
func (s Style) TopMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Top = v
	s.margin = Some(m)
	return s
}

// RightMargin sets the right margin.
func (s Style) RightMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Right = v
	s.margin = Some(m)
	return s
}

// BottomMargin sets the bottom margin.
func (s Style) BottomMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Bottom = v
	s.margin = Some(m)
	return s
}

// LeftMargin sets the left margin.
func (s Style) LeftMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Left = v
	s.margin = Some(m)
	return s
}

// HorizontalMargin sets the left and right margins.
func (s Style) HorizontalMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Left = v
	m.Right = v
	s.margin = Some(m)
	return s
}

// VerticalMargin sets the top and bottom margins.
func (s Style) VerticalMargin(v int) Style {
	m, _ := s.margin.Get()
	m.Top = v
	m.Bottom = v
	s.margin = Some(m)
	return s
}

// TopPadding sets the top padding.
func (s Style) TopPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Top = v
	s.padding = Some(p)
	return s
}

// RightPadding sets the right padding.
func (s Style) RightPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Right = v
	s.padding = Some(p)
	return s
}

// BottomPadding sets the bottom padding.
func (s Style) BottomPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Bottom = v
	s.padding = Some(p)
	return s
}

// LeftPadding sets the left padding.
func (s Style) LeftPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Left = v
	s.padding = Some(p)
	return s
}

// HorizontalPadding sets the left and right paddings.
func (s Style) HorizontalPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Left = v
	p.Right = v
	s.padding = Some(p)
	return s
}

// VerticalPadding sets the top and bottom paddings.
func (s Style) VerticalPadding(v int) Style {
	p, _ := s.padding.Get()
	p.Top = v
	p.Bottom = v
	s.padding = Some(p)
	return s
}
