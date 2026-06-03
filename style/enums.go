package style

// Display controls how an element participates in its parent's layout.
type Display uint8

const (
	// DisplayBlock renders the element as a block-level box.
	DisplayBlock Display = iota
	// DisplayFlex renders the element as a flex container.
	DisplayFlex
	// DisplayInline renders the element as an inline-level box that
	// participates in the surrounding inline-formatting context.
	DisplayInline
	// DisplayInlineBlock renders the element as an inline-level box that
	// establishes its own block-formatting context internally.
	DisplayInlineBlock
	// DisplayInlineFlex renders the element as an inline-level box that
	// establishes its own flex-formatting context internally.
	DisplayInlineFlex
	// DisplayNone removes the element from the layout entirely.
	DisplayNone
	// DisplayListItem renders the element as a list item with a marker.
	DisplayListItem
	// DisplayTable renders the element as a block-level table.
	DisplayTable
	// DisplayGrid renders the element as a block-level grid container.
	DisplayGrid
	// DisplayTableHeaderGroup renders the element as a table header group.
	DisplayTableHeaderGroup
	// DisplayTableRowGroup renders the element as a table row group.
	DisplayTableRowGroup
	// DisplayTableFooterGroup renders the element as a table footer group.
	DisplayTableFooterGroup
	// DisplayTableRow renders the element as a table row.
	DisplayTableRow
	// DisplayTableCell renders the element as a table cell.
	DisplayTableCell
)

// FlexDirection defines the main axis of a flex container.
type FlexDirection uint8

const (
	// FlexRow lays children out left-to-right (default).
	FlexRow FlexDirection = iota
	// FlexColumn lays children out top-to-bottom.
	FlexColumn
	// FlexRowReverse lays children out right-to-left.
	FlexRowReverse
	// FlexColumnReverse lays children out bottom-to-top.
	FlexColumnReverse
)

// FlexWrap controls whether flex items wrap onto multiple lines.
type FlexWrap uint8

const (
	// FlexNoWrap keeps all items on one line (default).
	FlexNoWrap FlexWrap = iota
	// FlexWrapOn wraps items onto new lines when the line overflows.
	FlexWrapOn
)

// Justify controls main-axis alignment inside a flex container.
type Justify uint8

const (
	// JustifyStart packs items toward the main-start edge.
	JustifyStart Justify = iota
	// JustifyEnd packs items toward the main-end edge.
	JustifyEnd
	// JustifyCenter centers items along the main axis.
	JustifyCenter
	// JustifyBetween distributes items with space between them.
	JustifyBetween
	// JustifyAround distributes items with space around them.
	JustifyAround
	// JustifyEvenly distributes items with equal space between and around them.
	JustifyEvenly
)

// Align controls cross-axis alignment.
type Align uint8

const (
	// AlignAuto defers to the parent's AlignItems.
	AlignAuto Align = iota
	// AlignStart aligns items at the cross-start edge.
	AlignStart
	// AlignEnd aligns items at the cross-end edge.
	AlignEnd
	// AlignCenter centers items along the cross axis.
	AlignCenter
	// AlignStretch stretches items to fill the cross axis.
	AlignStretch
	// AlignBaseline aligns items along their text baselines.
	AlignBaseline
)

// TextAlign controls horizontal text alignment within a block.
type TextAlign uint8

const (
	// TextAlignLeft aligns text to the left edge.
	TextAlignLeft TextAlign = iota
	// TextAlignCenter centers text horizontally.
	TextAlignCenter
	// TextAlignRight aligns text to the right edge.
	TextAlignRight
)

// TextWrap controls how text is wrapped when it exceeds the element's width.
type TextWrap uint8

const (
	// TextWrapWord wraps at word boundaries.
	TextWrapWord TextWrap = iota
	// TextWrapChar wraps at any character boundary.
	TextWrapChar
	// TextWrapNone disables wrapping; text may overflow.
	TextWrapNone
)

// TextOverflow controls how non-wrapping text is rendered when it overflows.
type TextOverflow uint8

const (
	// TextOverflowClip clips overflowing text at the element boundary.
	TextOverflowClip TextOverflow = iota
	// TextOverflowEllipsis replaces overflowing text with "…".
	TextOverflowEllipsis
)

// WhiteSpace controls whether whitespace is collapsed or preserved and whether
// soft wrapping is allowed.
type WhiteSpace uint8

const (
	// WhiteSpaceNormal collapses whitespace and allows wrapping.
	WhiteSpaceNormal WhiteSpace = iota
	// WhiteSpacePre preserves whitespace and only breaks at mandatory breaks.
	WhiteSpacePre
	// WhiteSpacePreWrap preserves whitespace and allows wrapping.
	WhiteSpacePreWrap
	// WhiteSpaceNoWrap collapses whitespace and disables soft wrapping.
	WhiteSpaceNoWrap
)

// WordBreak controls whether grapheme-cluster boundaries may be used as soft
// break opportunities.
type WordBreak uint8

const (
	// WordBreakNormal breaks at the shaper's soft / mandatory opportunities.
	WordBreakNormal WordBreak = iota
	// WordBreakBreakAll allows a break before every grapheme cluster.
	WordBreakBreakAll
	// WordBreakKeepAll suppresses BreakAnywhere opportunities.
	WordBreakKeepAll
)

// OverflowWrap controls emergency breaking inside an otherwise-unbreakable run.
type OverflowWrap uint8

const (
	// OverflowWrapNormal never breaks inside an unbreakable run.
	OverflowWrapNormal OverflowWrap = iota
	// OverflowWrapAnywhere allows emergency breaking at any cluster boundary.
	OverflowWrapAnywhere
	// OverflowWrapBreakWord allows emergency breaking without lowering
	// min-content width.
	OverflowWrapBreakWord
)

// Overflow controls how overflowing content is handled on a single axis.
type Overflow uint8

const (
	// OverflowVisible allows content to overflow visibly.
	OverflowVisible Overflow = iota
	// OverflowHidden hides overflowing content.
	OverflowHidden
	// OverflowScroll enables scrolling for overflowing content.
	OverflowScroll
	// OverflowClip clips overflowing content without scrolling.
	OverflowClip
	// OverflowAuto enables scrolling when content overflows.
	OverflowAuto
)

// ListStyleType defines the appearance of a list item marker.
type ListStyleType uint8

const (
	// ListStyleInherit inherits the marker from the parent.
	ListStyleInherit ListStyleType = iota
	// ListStyleNone renders no marker.
	ListStyleNone
	// ListStyleDisc renders a solid bullet (•).
	ListStyleDisc
	// ListStyleCircle renders a hollow circle (○).
	ListStyleCircle
	// ListStyleSquare renders a solid square (■).
	ListStyleSquare
	// ListStyleDecimal renders a decimal number (1., 2., 3.).
	ListStyleDecimal
)
