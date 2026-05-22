package style

import "image/color"

// ---------------------------------------------------------------------------
// Property — enumeration of all style properties
// ---------------------------------------------------------------------------

// Property identifies a single style property. It is used in the Inheritable
// table and for cache invalidation tooling.
type Property uint8

const (
	PropDisplay        Property = iota
	PropFlexDirection           //nolint:revive
	PropFlexWrap                //nolint:revive
	PropJustifyContent          //nolint:revive
	PropAlignItems              //nolint:revive
	PropAlignContent            //nolint:revive
	PropAlignSelf               //nolint:revive
	PropGap                     //nolint:revive
	PropFlex                    //nolint:revive
	PropOrder                   //nolint:revive
	PropWidth                   //nolint:revive
	PropHeight                  //nolint:revive
	PropMinWidth                //nolint:revive
	PropMaxWidth                //nolint:revive
	PropMinHeight               //nolint:revive
	PropMaxHeight               //nolint:revive
	PropPadding                 //nolint:revive
	PropMargin                  //nolint:revive
	PropBorder                  //nolint:revive
	PropForeground              //nolint:revive
	PropBackground              //nolint:revive
	PropBold                    //nolint:revive
	PropItalic                  //nolint:revive
	PropUnderline               //nolint:revive
	PropStrikethrough           //nolint:revive
	PropReverse                 //nolint:revive
	PropTextAlign               //nolint:revive
	PropTextWrap                //nolint:revive
	PropTextOverflow            //nolint:revive
	PropWhiteSpace              //nolint:revive
	PropWordBreak               //nolint:revive
	PropOverflowWrap            //nolint:revive
	PropOverflowX               //nolint:revive
	PropOverflowY               //nolint:revive
	PropListStyleType           //nolint:revive
	PropCursorShape             //nolint:revive
	PropCursorColor             //nolint:revive
)

// Inheritable returns the set of properties that are inherited from a parent
// node's Computed style when the child does not explicitly set the property.
// This function is the single source of truth; the resolver's Resolve method
// implements the same set in the fast-path overlay loop.
//
// Inheritable properties:
// Foreground, Background, Bold, Italic, Underline, Strikethrough,
// TextWrap, TextOverflow, WhiteSpace, WordBreak, OverflowWrap,
// ListStyleType.
func Inheritable() map[Property]bool {
	return map[Property]bool{
		PropForeground:    true,
		PropBold:          true,
		PropItalic:        true,
		PropUnderline:     true,
		PropStrikethrough: true,
		PropTextWrap:      true,
		PropTextOverflow:  true,
		PropWhiteSpace:    true,
		PropWordBreak:     true,
		PropOverflowWrap:  true,
		PropListStyleType: true,
		PropCursorShape:   true,
		PropCursorColor:   true,
	}
}

// ---------------------------------------------------------------------------
// DefaultStyle
// ---------------------------------------------------------------------------

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

		// Cursor ---------------------------------------------------------------
		CursorShape: CursorShapeBlockBlink,
		CursorColor: TerminalDefault,
	}
}

// ---------------------------------------------------------------------------
// StyleNode — interface the resolver requires from any tree node
// ---------------------------------------------------------------------------

// StyleNode is the interface Resolver.Resolve and ResolveTree require from a
// render node. render.Object satisfies this interface when it embeds BaseRender.
//
// Navigation methods carry a "Style" prefix so they can coexist with
// render.Object's tree-navigation methods (which return render.Object).
type StyleNode interface {
	// RawStyle returns the author-set sparse style for this node. Called on
	// the full-resolve path only; cache hits bypass this.
	RawStyle() Style

	// DefaultStyle returns the element-type default style that the
	// resolver applies after the root DefaultStyle baseline and before parent
	// inheritance. Native elements return a sparse Style with only the
	// properties they override (e.g. Span returns Display:Inline; Box returns
	// an empty Style). A zero Style is valid and means "no element defaults".
	DefaultStyle() Style

	// IntrinsicStyle returns the UA-mandated sparse style for this node.
	// Properties set here have the highest cascade precedence and cannot be
	// overridden by the author's RawStyle. Most nodes return an empty Style{};
	// replaced and compound elements return a sparse Style with UA-forced
	// properties (e.g. Display:InlineBlock, OverflowX:Clip). See ADR-010.
	IntrinsicStyle() Style

	// ComputedStyle returns the previously resolved style, or nil if the
	// style phase has not yet visited this node.
	ComputedStyle() *Computed

	// SetComputedStyle stores the resolved style on this node.
	SetComputedStyle(*Computed)

	// IsDirtyStyle reports whether this node's style needs re-resolution.
	IsDirtyStyle() bool

	// HasDirtyStyleChild reports whether any descendant has DirtyStyle set
	// (i.e. the ChildNeedsStyle relay flag is set).
	HasDirtyStyleChild() bool

	// ClearDirtyStyle removes the DirtyStyle flag from this node.
	ClearDirtyStyle()

	// ClearChildNeedsStyle removes the ChildNeedsStyle relay flag from this
	// node after all dirty children in this subtree have been processed.
	ClearChildNeedsStyle()

	// StyleParent returns the parent node, or nil for the root.
	StyleParent() StyleNode

	// StyleFirstChild returns the first child, or nil.
	StyleFirstChild() StyleNode

	// StyleNextSibling returns the next sibling, or nil.
	StyleNextSibling() StyleNode
}

// ---------------------------------------------------------------------------
// Resolver
// ---------------------------------------------------------------------------

// cacheEntry holds the inputs and output of a single Resolve call for one
// node. Both fields are pointers; comparing them is O(1) and allocation-free.
type cacheEntry struct {
	parent *Computed
	result *Computed
}

// Resolver performs inheritance, default application, and computed-value
// resolution for a kitex style tree.
//
// Cache policy: Resolve returns the same *Computed pointer when the node does
// not have DirtyStyle set and the parent *Computed pointer is unchanged.
// The cache-hit path performs zero heap allocations.
//
// Single-threaded contract: Resolver is not safe for concurrent use.
type Resolver struct {
	defaults *Computed
	cache    map[StyleNode]cacheEntry
}

// NewResolver creates a Resolver seeded with DefaultStyle as the baseline.
func NewResolver() *Resolver {
	d := DefaultStyle()
	return &Resolver{
		defaults: &d,
		cache:    make(map[StyleNode]cacheEntry),
	}
}

// Resolve returns the Computed style for elem, inheriting from parent where
// appropriate and overlaying elem's own style on top.
//
// When the cache is warm (IsDirtyStyle is false and the parent pointer is
// unchanged), Resolve returns the cached pointer with zero allocations.
func (r *Resolver) Resolve(elem StyleNode, parent *Computed) *Computed {
	// Cache hit: no dirty flag and same parent pointer → return cached result.
	if entry, ok := r.cache[elem]; ok {
		if !elem.IsDirtyStyle() && entry.parent == parent {
			return entry.result
		}
	}

	// Full resolve — five-layer application:
	//   1. Root DefaultStyle baseline.
	//   2. Element-type defaults (e.g. Span → Display:Inline).     [OriginUADefault]
	//   3. Inheritable fields from parent Computed.                 [OriginInherited]
	//   4. Element's own author-set style.                         [OriginAuthor]
	//   5. UA-mandated intrinsic style (highest precedence).       [OriginUserAgent]

	// Layer 1: root baseline.
	c := *r.defaults

	// Layer 2: element-type defaults (applied before inheritance so that
	// parent colours can still override a Span's display default).
	c = elem.DefaultStyle().Apply(c)

	// Layer 3: overlay inheritable fields from parent (mirrors Inheritable()).
	if parent != nil {
		c.Foreground = parent.Foreground
		c.Bold = parent.Bold
		c.Italic = parent.Italic
		c.Underline = parent.Underline
		c.Strikethrough = parent.Strikethrough
		c.TextWrap = parent.TextWrap
		c.TextOverflow = parent.TextOverflow
		c.WhiteSpace = parent.WhiteSpace
		c.WordBreak = parent.WordBreak
		c.OverflowWrap = parent.OverflowWrap
		if c.ListStyleType == ListStyleInherit {
			c.ListStyleType = parent.ListStyleType
		}
		c.CursorShape = parent.CursorShape
		c.CursorColor = parent.CursorColor
	}

	// Layer 4: element's own author-set style — Optional fields only when IsSet.
	c = elem.RawStyle().Apply(c)

	// Layer 5: UA-mandated intrinsic style — highest precedence; authors cannot
	// override. Only applied when the element has intrinsic properties set
	// (sparse: most elements return an empty Style{} and Apply is a no-op).
	// Tagged as OriginUserAgent in the cascade-origin model (ADR-010).
	c = elem.IntrinsicStyle().Apply(c) // _ = OriginUserAgent

	// Computed-value resolution notes:
	//   TerminalDefault colours remain symbolic; the backend resolves them at
	//   paint time. Percent dimensions remain symbolic; layout resolves them
	//   against the parent's concrete size. Cells dimensions pass through as-is.

	result := new(Computed)
	*result = c
	r.cache[elem] = cacheEntry{parent: parent, result: result}
	return result
}

// Invalidate removes the cached entry for elem, forcing a full re-resolve on
// the next Resolve call regardless of the DirtyStyle flag.
func (r *Resolver) Invalidate(elem StyleNode) { delete(r.cache, elem) }

// ---------------------------------------------------------------------------
// ResolveTree — style phase walker
// ---------------------------------------------------------------------------

// ResolveTree runs the style phase on the subtree rooted at root. It walks
// top-down, gated on DirtyStyle | ChildNeedsStyle, resolves each dirty node,
// stores the result via SetComputedStyle, and clears dirty flags after visiting.
//
// ResolveTree is a no-op when root is nil or when neither DirtyStyle nor
// ChildNeedsStyle is set on root.
func ResolveTree(r *Resolver, root StyleNode) {
	if root == nil {
		return
	}
	if root.IsDirtyStyle() || root.HasDirtyStyleChild() {
		resolveSubtree(r, root, nil)
	}
}

// resolveSubtree resolves node (if dirty) then recursively visits children
// that carry style work. parent is the Computed of node's already-resolved
// parent; nil is passed for the root.
func resolveSubtree(r *Resolver, node StyleNode, parent *Computed) {
	oldComputed := node.ComputedStyle()
	computed := r.Resolve(node, parent)

	// Update node if style changed (either directly or via inheritance)
	if node.IsDirtyStyle() || computed != oldComputed {
		node.SetComputedStyle(computed)
		node.ClearDirtyStyle()
	}

	// Descend into children. If this node's computed style changed, we MUST
	// visit all children to propagate inheritance.
	force := (computed != oldComputed)

	for c := node.StyleFirstChild(); c != nil; c = c.StyleNextSibling() {
		if force || c.IsDirtyStyle() || c.HasDirtyStyleChild() {
			resolveSubtree(r, c, computed)
		}
	}

	// Clear the relay flag now that all dirty children have been processed.
	node.ClearChildNeedsStyle()
}
