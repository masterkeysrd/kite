package style

import "image/color"

// Style is a sparse set of visual and layout properties for a kitex element.
// Every field is [Optional] so that callers can compose styles without
// overwriting fields they did not intend to change.
//
// Construct via struct literals using [Some]:
//
//	s := Style{
//	    Display:       Some(DisplayFlex),
//	    FlexDirection: Some(FlexColumn),
//	    Gap:           Some(Gap(1)),
//	}
//
// Combine two styles with [Style.Merge].
type Style struct {
	// --- Flex / display -------------------------------------------------------

	// Display controls how the element participates in its parent's layout.
	Display Optional[Display]
	// ListStyleType defines the appearance of a list item marker.
	ListStyleType Optional[ListStyleType]
	// FlexDirection sets the main axis of a flex container.
	FlexDirection Optional[FlexDirection]
	// FlexWrap controls whether items wrap onto new lines.
	FlexWrap Optional[FlexWrap]
	// JustifyContent sets main-axis alignment inside a flex container.
	JustifyContent Optional[Justify]
	// AlignItems sets cross-axis alignment for all children.
	AlignItems Optional[Align]
	// AlignContent sets cross-axis alignment for multiple flex lines.
	AlignContent Optional[Align]
	// AlignSelf overrides the parent's AlignItems for this element.
	AlignSelf Optional[Align]
	// Gap sets the row and column gaps between flex children.
	Gap Optional[GapValue]
	// Flex sets the grow, shrink, and basis for a flex item.
	Flex Optional[FlexItemValue]
	// Order sets the order of a flex item.
	Order Optional[int]

	// --- Box model ------------------------------------------------------------

	// Width sets the preferred width of the element.
	Width Optional[Dimension]
	// Height sets the preferred height of the element.
	Height Optional[Dimension]
	// MinWidth sets the minimum width of the element.
	MinWidth Optional[Dimension]
	// MaxWidth sets the maximum width of the element.
	MaxWidth Optional[Dimension]
	// MinHeight sets the minimum height of the element.
	MinHeight Optional[Dimension]
	// MaxHeight sets the maximum height of the element.
	MaxHeight Optional[Dimension]
	// Padding is the inner spacing between the element's content and its border.
	Padding Optional[EdgeValues[int]]
	// Margin is the outer spacing between the element's border and its siblings.
	Margin Optional[EdgeValues[int]]
	// Border sets the element's border.
	Border Optional[Border]

	// --- Color / visual -------------------------------------------------------

	// Foreground sets the text (foreground) color.
	Foreground Optional[color.Color]
	// Background sets the cell background color.
	Background Optional[color.Color]
	// Bold enables bold text rendering.
	Bold Optional[bool]
	// Italic enables italic text rendering.
	Italic Optional[bool]
	// Underline enables underlined text.
	Underline Optional[bool]
	// Strikethrough enables strikethrough text.
	Strikethrough Optional[bool]
	// Reverse swaps foreground and background colors.
	Reverse Optional[bool]

	// --- Text -----------------------------------------------------------------

	// TextAlign controls horizontal text alignment within the element.
	TextAlign Optional[TextAlign]
	// TextWrap controls how text is wrapped at the element's width.
	TextWrap Optional[TextWrap]
	// TextOverflow controls how non-wrapping text is handled when it overflows.
	TextOverflow Optional[TextOverflow]
	// WhiteSpace controls whitespace preservation and whether wrapping is allowed.
	WhiteSpace Optional[WhiteSpace]
	// WordBreak controls whether breaks may occur between grapheme clusters.
	WordBreak Optional[WordBreak]
	// OverflowWrap controls emergency breaking inside unbreakable runs.
	OverflowWrap Optional[OverflowWrap]

	// --- Overflow / scroll ----------------------------------------------------

	// OverflowX controls how content that overflows on the horizontal axis is handled.
	OverflowX Optional[Overflow]
	// OverflowY controls how content that overflows on the vertical axis is handled.
	OverflowY Optional[Overflow]
}

// Apply returns a copy of base with every set field in s overlaid on top.
// It is the bridge between the sparse [Style] author model and the fully-resolved
// [Computed] type: the resolver calls it once per dirty node after applying
// inheritance, so all the per-property Optional checks live here rather than
// in the resolver.
func (s Style) Apply(base Computed) Computed {
	if s.Display.IsSet() {
		base.Display = s.Display.Value()
	}
	if s.ListStyleType.IsSet() {
		base.ListStyleType = s.ListStyleType.Value()
	}
	if s.FlexDirection.IsSet() {
		base.FlexDirection = s.FlexDirection.Value()
	}
	if s.FlexWrap.IsSet() {
		base.FlexWrap = s.FlexWrap.Value()
	}
	if s.JustifyContent.IsSet() {
		base.JustifyContent = s.JustifyContent.Value()
	}
	if s.AlignItems.IsSet() {
		base.AlignItems = s.AlignItems.Value()
	}
	if s.AlignContent.IsSet() {
		base.AlignContent = s.AlignContent.Value()
	}
	if s.AlignSelf.IsSet() {
		base.AlignSelf = s.AlignSelf.Value()
	}
	if s.Gap.IsSet() {
		base.Gap = s.Gap.Value()
	}
	if s.Flex.IsSet() {
		base.Flex = s.Flex.Value()
	}
	if s.Order.IsSet() {
		base.Order = s.Order.Value()
	}
	if s.Width.IsSet() {
		base.Width = s.Width.Value()
	}
	if s.Height.IsSet() {
		base.Height = s.Height.Value()
	}
	if s.MinWidth.IsSet() {
		base.MinWidth = s.MinWidth.Value()
	}
	if s.MaxWidth.IsSet() {
		base.MaxWidth = s.MaxWidth.Value()
	}
	if s.MinHeight.IsSet() {
		base.MinHeight = s.MinHeight.Value()
	}
	if s.MaxHeight.IsSet() {
		base.MaxHeight = s.MaxHeight.Value()
	}
	if s.Padding.IsSet() {
		base.Padding = s.Padding.Value()
	}
	if s.Margin.IsSet() {
		base.Margin = s.Margin.Value()
	}
	if s.Border.IsSet() {
		base.Border = s.Border.Value()
	}
	if s.Foreground.IsSet() {
		base.Foreground = s.Foreground.Value()
	}
	if s.Background.IsSet() {
		base.Background = s.Background.Value()
	}
	if s.Bold.IsSet() {
		base.Bold = s.Bold.Value()
	}
	if s.Italic.IsSet() {
		base.Italic = s.Italic.Value()
	}
	if s.Underline.IsSet() {
		base.Underline = s.Underline.Value()
	}
	if s.Strikethrough.IsSet() {
		base.Strikethrough = s.Strikethrough.Value()
	}
	if s.Reverse.IsSet() {
		base.Reverse = s.Reverse.Value()
	}
	if s.TextAlign.IsSet() {
		base.TextAlign = s.TextAlign.Value()
	}
	if s.TextWrap.IsSet() {
		base.TextWrap = s.TextWrap.Value()
	}
	if s.TextOverflow.IsSet() {
		base.TextOverflow = s.TextOverflow.Value()
	}
	if s.WhiteSpace.IsSet() {
		base.WhiteSpace = s.WhiteSpace.Value()
	}
	if s.WordBreak.IsSet() {
		base.WordBreak = s.WordBreak.Value()
	}
	if s.OverflowWrap.IsSet() {
		base.OverflowWrap = s.OverflowWrap.Value()
	}
	if s.OverflowX.IsSet() {
		base.OverflowX = s.OverflowX.Value()
	}
	if s.OverflowY.IsSet() {
		base.OverflowY = s.OverflowY.Value()
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
		Display:        s.Display.Merge(override.Display),
		FlexDirection:  s.FlexDirection.Merge(override.FlexDirection),
		FlexWrap:       s.FlexWrap.Merge(override.FlexWrap),
		JustifyContent: s.JustifyContent.Merge(override.JustifyContent),
		AlignItems:     s.AlignItems.Merge(override.AlignItems),
		AlignContent:   s.AlignContent.Merge(override.AlignContent),
		AlignSelf:      s.AlignSelf.Merge(override.AlignSelf),
		Gap:            s.Gap.Merge(override.Gap),
		Flex:           s.Flex.Merge(override.Flex),
		Order:          s.Order.Merge(override.Order),

		Width:     s.Width.Merge(override.Width),
		Height:    s.Height.Merge(override.Height),
		MinWidth:  s.MinWidth.Merge(override.MinWidth),
		MaxWidth:  s.MaxWidth.Merge(override.MaxWidth),
		MinHeight: s.MinHeight.Merge(override.MinHeight),
		MaxHeight: s.MaxHeight.Merge(override.MaxHeight),
		Padding:   s.Padding.Merge(override.Padding),
		Margin:    s.Margin.Merge(override.Margin),
		Border:    s.Border.Merge(override.Border),

		Foreground:    s.Foreground.Merge(override.Foreground),
		Background:    s.Background.Merge(override.Background),
		Bold:          s.Bold.Merge(override.Bold),
		Italic:        s.Italic.Merge(override.Italic),
		Underline:     s.Underline.Merge(override.Underline),
		Strikethrough: s.Strikethrough.Merge(override.Strikethrough),
		Reverse:       s.Reverse.Merge(override.Reverse),

		TextAlign:    s.TextAlign.Merge(override.TextAlign),
		TextWrap:     s.TextWrap.Merge(override.TextWrap),
		TextOverflow: s.TextOverflow.Merge(override.TextOverflow),
		WhiteSpace:   s.WhiteSpace.Merge(override.WhiteSpace),
		WordBreak:    s.WordBreak.Merge(override.WordBreak),
		OverflowWrap: s.OverflowWrap.Merge(override.OverflowWrap),

		OverflowX: s.OverflowX.Merge(override.OverflowX),
		OverflowY: s.OverflowY.Merge(override.OverflowY),
	}
}
