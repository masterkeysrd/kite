package style

import (
	"image/color"

	"github.com/masterkeysrd/kite/geom"
)

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
	mediaRules    []MediaRule
	hasMediaRules bool
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

		mediaRules:    append(append([]MediaRule(nil), s.mediaRules...), override.mediaRules...),
		hasMediaRules: s.hasMediaRules || override.hasMediaRules,
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

// Gap sets the row and column gaps between flex/grid children.
// It expects 1 or 2 int values:
//
//   - Gap(n)    -> sets both row and column gap to n
//   - Gap(r, c) -> sets row gap to r and column gap to c
func (s Style) Gap(values ...int) Style {
	s.gap = Some(Gap(values...))
	return s
}

// FlexOpt returns the optional value of Flex.
func (s Style) FlexOpt() Optional[FlexItemValue] { return s.flex }

// Flex sets the grow, shrink, and basis for a flex item.
// It expects a mandatory grow (int), followed by optional shrink (int) and basis (Dimension):
//
//   - Flex(g)           -> grow=g, shrink=1, basis=Auto
//   - Flex(g, s)        -> grow=g, shrink=s, basis=Auto
//   - Flex(g, s, basis) -> grow=g, shrink=s, basis=basis
func (s Style) Flex(grow int, rest ...any) Style {
	s.flex = Some(Flex(grow, rest...))
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

// GridTemplateColumns sets the column track sizing functions.
// It expects variadic GridTrackSize values.
func (s Style) GridTemplateColumns(v ...GridTrackSize) Style {
	s.gridTemplateColumns = Some(v)
	return s
}

// GridTemplateRowsOpt returns the optional value of GridTemplateRows.
func (s Style) GridTemplateRowsOpt() Optional[[]GridTrackSize] { return s.gridTemplateRows }

// GridTemplateRows sets the row track sizing functions.
// It expects variadic GridTrackSize values.
func (s Style) GridTemplateRows(v ...GridTrackSize) Style {
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

// Padding sets the inner spacing between the element's content and its border.
// It expects 1, 2, or 4 int values (CSS shorthand):
//
//   - Padding(all)             -> all four sides equal
//   - Padding(vert, horiz)     -> top/bottom = vert, left/right = horiz
//   - Padding(top, right, bot, left)
func (s Style) Padding(values ...int) Style {
	s.padding = Some(Edges(values...))
	return s
}

// PaddingTop sets the top padding.
func (s Style) PaddingTop(v int) Style {
	p, _ := s.padding.Get()
	p.Top = v
	s.padding = Some(p)
	return s
}

// PaddingRight sets the right padding.
func (s Style) PaddingRight(v int) Style {
	p, _ := s.padding.Get()
	p.Right = v
	s.padding = Some(p)
	return s
}

// PaddingBottom sets the bottom padding.
func (s Style) PaddingBottom(v int) Style {
	p, _ := s.padding.Get()
	p.Bottom = v
	s.padding = Some(p)
	return s
}

// PaddingLeft sets the left padding.
func (s Style) PaddingLeft(v int) Style {
	p, _ := s.padding.Get()
	p.Left = v
	s.padding = Some(p)
	return s
}

// PaddingHorizontal sets the left and right paddings.
func (s Style) PaddingHorizontal(v int) Style {
	p, _ := s.padding.Get()
	p.Left = v
	p.Right = v
	s.padding = Some(p)
	return s
}

// PaddingVertical sets the top and bottom paddings.
func (s Style) PaddingVertical(v int) Style {
	p, _ := s.padding.Get()
	p.Top = v
	p.Bottom = v
	s.padding = Some(p)
	return s
}

// MarginOpt returns the optional value of Margin.
func (s Style) MarginOpt() Optional[EdgeValues[int]] { return s.margin }

// Margin sets the outer spacing between the element's border and its siblings.
// It expects 1, 2, or 4 int values (CSS shorthand):
//
//   - Margin(all)             -> all four sides equal
//   - Margin(vert, horiz)     -> top/bottom = vert, left/right = horiz
//   - Margin(top, right, bot, left)
func (s Style) Margin(values ...int) Style {
	s.margin = Some(Edges(values...))
	return s
}

// MarginTop sets the top margin.
func (s Style) MarginTop(v int) Style {
	m, _ := s.margin.Get()
	m.Top = v
	s.margin = Some(m)
	return s
}

// MarginRight sets the right margin.
func (s Style) MarginRight(v int) Style {
	m, _ := s.margin.Get()
	m.Right = v
	s.margin = Some(m)
	return s
}

// MarginBottom sets the bottom margin.
func (s Style) MarginBottom(v int) Style {
	m, _ := s.margin.Get()
	m.Bottom = v
	s.margin = Some(m)
	return s
}

// MarginLeft sets the left margin.
func (s Style) MarginLeft(v int) Style {
	m, _ := s.margin.Get()
	m.Left = v
	s.margin = Some(m)
	return s
}

// MarginHorizontal sets the left and right margins.
func (s Style) MarginHorizontal(v int) Style {
	m, _ := s.margin.Get()
	m.Left = v
	m.Right = v
	s.margin = Some(m)
	return s
}

// MarginVertical sets the top and bottom margins.
func (s Style) MarginVertical(v int) Style {
	m, _ := s.margin.Get()
	m.Top = v
	m.Bottom = v
	s.margin = Some(m)
	return s
}

// BorderOpt returns the optional value of Border.
func (s Style) BorderOpt() Optional[Border] { return s.border }

// Border sets the border properties. It accepts variadic arguments of types:
//   - bool: toggles all edges on/off
//   - BorderStyle: sets the style for all edges
//   - color.Color: sets the color for all edges
//   - BorderGlyphs: sets the glyphs for the entire border and switches to BorderCustom
//   - Border: replaces the entire border definition
func (s Style) Border(args ...any) Style {
	b, _ := s.border.Get()
	hasStyle := false
	for _, arg := range args {
		switch v := arg.(type) {
		case bool:
			b.Edges = EdgeAll(v)
		case BorderStyle:
			b.Styles = EdgeAll(v)
			hasStyle = true
		case BorderGlyphs:
			b.Glyphs = v
			b.Styles = EdgeAll(BorderCustom)
			hasStyle = true
		case color.Color:
			b.Colors = EdgeAll(v)
		case Border:
			b = v
			hasStyle = true
		}
	}
	if !hasStyle {
		applyDefaultStyleIfVisible(&b, "all")
	}
	s.border = Some(b)
	return s
}

// BorderTop toggles the visibility of the top border edge and optionally sets its style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color:
//
//   - BorderTop(true)                 -> enables top edge with BorderSingle
//   - BorderTop(true, BorderDouble)    -> enables top edge with BorderDouble
//   - BorderTop(true, BorderDouble, c) -> enables top edge with BorderDouble and color c
func (s Style) BorderTop(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Top = visible
	applyBorderSideArgs(&b, "top", args...)
	s.border = Some(b)
	return s
}

// BorderRight toggles the visibility of the right border edge and optionally sets its style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color.
func (s Style) BorderRight(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Right = visible
	applyBorderSideArgs(&b, "right", args...)
	s.border = Some(b)
	return s
}

// BorderBottom toggles the visibility of the bottom border edge and optionally sets its style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color.
func (s Style) BorderBottom(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Bottom = visible
	applyBorderSideArgs(&b, "bottom", args...)
	s.border = Some(b)
	return s
}

// BorderLeft toggles the visibility of the left border edge and optionally sets its style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color.
func (s Style) BorderLeft(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Left = visible
	applyBorderSideArgs(&b, "left", args...)
	s.border = Some(b)
	return s
}

// BorderHorizontal toggles the visibility of the left and right border edges and optionally sets their style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color.
func (s Style) BorderHorizontal(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Left = visible
	b.Edges.Right = visible
	applyBorderSideArgs(&b, "horizontal", args...)
	s.border = Some(b)
	return s
}

// BorderVertical toggles the visibility of the top and bottom border edges and optionally sets their style and color.
// It accepts optional arguments of types BorderStyle, BorderGlyphs, or color.Color.
func (s Style) BorderVertical(visible bool, args ...any) Style {
	b, _ := s.border.Get()
	b.Edges.Top = visible
	b.Edges.Bottom = visible
	applyBorderSideArgs(&b, "vertical", args...)
	s.border = Some(b)
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

// --- Border helpers ---------------------------------------------------------

// applyBorderSideArgs updates the border properties for a specific side or group of sides.
func applyBorderSideArgs(b *Border, side string, args ...any) {
	hasStyle := false
	for _, arg := range args {
		switch v := arg.(type) {
		case BorderStyle:
			hasStyle = true
			setSideStyle(b, side, v)
		case BorderGlyphs:
			hasStyle = true
			b.Glyphs = v
			setSideStyle(b, side, BorderCustom)
		case color.Color:
			setSideColor(b, side, v)
		}
	}

	// Default to BorderSingle if visible and no style was specified and current style is BorderNone
	if !hasStyle {
		applyDefaultStyleIfVisible(b, side)
	}
}

func setSideStyle(b *Border, side string, v BorderStyle) {
	switch side {
	case "top":
		b.Styles.Top = v
	case "right":
		b.Styles.Right = v
	case "bottom":
		b.Styles.Bottom = v
	case "left":
		b.Styles.Left = v
	case "horizontal":
		b.Styles.Left, b.Styles.Right = v, v
	case "vertical":
		b.Styles.Top, b.Styles.Bottom = v, v
	case "all":
		b.Styles = EdgeAll(v)
	}
}

func setSideColor(b *Border, side string, v color.Color) {
	switch side {
	case "top":
		b.Colors.Top = v
	case "right":
		b.Colors.Right = v
	case "bottom":
		b.Colors.Bottom = v
	case "left":
		b.Colors.Left = v
	case "horizontal":
		b.Colors.Left, b.Colors.Right = v, v
	case "vertical":
		b.Colors.Top, b.Colors.Bottom = v, v
	case "all":
		b.Colors = EdgeAll(v)
	}
}

func applyDefaultStyleIfVisible(b *Border, side string) {
	if side == "top" || side == "vertical" || side == "all" {
		if b.Edges.Top && b.Styles.Top == BorderNone {
			b.Styles.Top = BorderSingle
		}
	}
	if side == "right" || side == "horizontal" || side == "all" {
		if b.Edges.Right && b.Styles.Right == BorderNone {
			b.Styles.Right = BorderSingle
		}
	}
	if side == "bottom" || side == "vertical" || side == "all" {
		if b.Edges.Bottom && b.Styles.Bottom == BorderNone {
			b.Styles.Bottom = BorderSingle
		}
	}
	if side == "left" || side == "horizontal" || side == "all" {
		if b.Edges.Left && b.Styles.Left == BorderNone {
			b.Styles.Left = BorderSingle
		}
	}
}

// --- Media Queries -----------------------------------------------------------

// MediaQuery represents a viewport condition to dynamically match styles.
type MediaQuery struct {
	minWidth  int
	maxWidth  int
	minHeight int
	maxHeight int
}

// MediaRule associates a [MediaQuery] with an overriding [Style].
type MediaRule struct {
	Query MediaQuery
	Style Style
}

// Query returns an empty [MediaQuery] to start a query building chain.
func Query() MediaQuery {
	return MediaQuery{}
}

// MinWidth sets the minimum viewport width constraint.
func (q MediaQuery) MinWidth(w int) MediaQuery {
	q.minWidth = w
	return q
}

// MaxWidth sets the maximum viewport width constraint.
func (q MediaQuery) MaxWidth(w int) MediaQuery {
	q.maxWidth = w
	return q
}

// MinHeight sets the minimum viewport height constraint.
func (q MediaQuery) MinHeight(h int) MediaQuery {
	q.minHeight = h
	return q
}

// MaxHeight sets the maximum viewport height constraint.
func (q MediaQuery) MaxHeight(h int) MediaQuery {
	q.maxHeight = h
	return q
}

// Matches returns true if the given viewport size satisfies all constraints of the query.
func (q MediaQuery) Matches(viewport geom.Size) bool {
	if q.minWidth > 0 && viewport.Width < q.minWidth {
		return false
	}
	if q.maxWidth > 0 && viewport.Width > q.maxWidth {
		return false
	}
	if q.minHeight > 0 && viewport.Height < q.minHeight {
		return false
	}
	if q.maxHeight > 0 && viewport.Height > q.maxHeight {
		return false
	}
	return true
}

// Media registers a conditional media rule on the style. If the viewport matches
// the query, the ruleStyle will be merged on top of this style.
func (s Style) Media(q MediaQuery, ruleStyle Style) Style {
	s.mediaRules = append(s.mediaRules, MediaRule{Query: q, Style: ruleStyle})
	s.hasMediaRules = true
	return s
}

// HasMediaRules returns true if the style contains conditional media rules.
func (s Style) HasMediaRules() bool {
	return s.hasMediaRules
}

// EvaluateMedia matches all registered media rules against the given viewport size,
// merges the matching rule styles, and returns the resolved style.
func (s Style) EvaluateMedia(viewport geom.Size) Style {
	if !s.hasMediaRules {
		return s
	}
	res := s
	res.mediaRules = nil
	res.hasMediaRules = false

	for _, rule := range s.mediaRules {
		if rule.Query.Matches(viewport) {
			res = res.Merge(rule.Style)
		}
	}
	return res
}
