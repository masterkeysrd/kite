package styler

import (
	"github.com/masterkeysrd/kite/style"
)

// CacheEntry holds the inputs and output of a single Resolve call for one
// node. Both fields are pointers; comparing them is O(1) and allocation-free.
type CacheEntry struct {
	Parent *style.Computed
	Result *style.Computed
}

// Resolver performs inheritance, default application, and computed-value
// resolution for a kitex style tree.
//
// Cache policy: Resolve returns the same *Computed pointer when the node does
// not have IsDirtyStyle set and the parent *Computed pointer is unchanged.
// The cache-hit path performs zero heap allocations.
//
// Single-threaded contract: Resolver is not safe for concurrent use.
type Resolver struct {
	defaults *style.Computed
	Cache    map[style.StyleNode]CacheEntry
}

// NewResolver creates a Resolver seeded with style.DefaultStyle as the baseline.
func NewResolver() *Resolver {
	d := style.DefaultStyle()
	return &Resolver{
		defaults: &d,
		Cache:    make(map[style.StyleNode]CacheEntry),
	}
}

// Resolve returns the Computed style for elem, inheriting from parent where
// appropriate and overlaying elem's own style on top.
//
// When the cache is warm (IsDirtyStyle is false and the parent pointer is
// unchanged), Resolve returns the cached pointer with zero allocations.
func (r *Resolver) Resolve(elem style.StyleNode, parent *style.Computed) *style.Computed {
	// Cache hit: no dirty flag and same parent pointer → return cached result.
	if entry, ok := r.Cache[elem]; ok {
		if !elem.IsDirtyStyle() && entry.Parent == parent {
			return entry.Result
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
		if c.ListStyleType == style.ListStyleInherit {
			c.ListStyleType = parent.ListStyleType
		}
		c.CursorShape = parent.CursorShape
		c.CursorColor = parent.CursorColor
		c.SelectionForeground = parent.SelectionForeground
		c.SelectionBackground = parent.SelectionBackground
	}

	// Layer 4: element's own author-set style — Optional fields only when IsSet.
	c = elem.RawStyle().Apply(c)

	// Layer 5: UA-mandated intrinsic style — highest precedence; authors cannot
	// override. Only applied when the element has intrinsic properties set
	// (sparse: most elements return an empty Style{} and Apply is a no-op).
	c = elem.IntrinsicStyle().Apply(c)

	// Apply sensible TUI defaults for glyphs if explicitly set to true but missing glyphs.
	if c.Scrollbar.X.UnwrapOr(false) || c.Scrollbar.Y.UnwrapOr(false) {
		if !c.Scrollbar.TrackGlyph.IsSet() {
			if c.Scrollbar.Y.UnwrapOr(false) {
				c.Scrollbar.TrackGlyph = style.Some(style.DefaultScrollbarTrackVertical)
			} else {
				c.Scrollbar.TrackGlyph = style.Some(style.DefaultScrollbarTrackHorizontal)
			}
		}
		if !c.Scrollbar.ThumbGlyph.IsSet() {
			if c.Scrollbar.Y.UnwrapOr(false) {
				c.Scrollbar.ThumbGlyph = style.Some(style.DefaultScrollbarThumbVertical)
			} else {
				c.Scrollbar.ThumbGlyph = style.Some(style.DefaultScrollbarThumbHorizontal)
			}
		}
	}

	result := new(style.Computed)
	*result = c
	r.Cache[elem] = CacheEntry{Parent: parent, Result: result}
	return result
}

// Invalidate removes the cached entry for elem, forcing a full re-resolve on
// the next Resolve call regardless of the DirtyStyle flag.
func (r *Resolver) Invalidate(elem style.StyleNode) { delete(r.Cache, elem) }
