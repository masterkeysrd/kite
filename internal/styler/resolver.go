package styler

import (
	"github.com/masterkeysrd/kite/dom"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/render"
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
// not have DirtyStyle set and the parent *Computed pointer is unchanged.
// The cache-hit path performs zero heap allocations.
//
// Single-threaded contract: Resolver is not safe for concurrent use.
type Resolver struct {
	defaults *style.Computed
	cache    map[dom.Node]CacheEntry
}

// NewResolver creates a Resolver seeded with style.DefaultStyle as the baseline.
func NewResolver() *Resolver {
	d := style.DefaultStyle()
	return &Resolver{
		defaults: &d,
		cache:    make(map[dom.Node]CacheEntry),
	}
}

// ResolveTree recursively resolves styles for the given render subtree.
func (r *Resolver) ResolveTree(ro render.Object, parent *style.Computed, force bool) {
	flags := ro.Flags()
	if !force && flags&(render.DirtyStyle|render.ChildNeedsStyle) == 0 {
		return
	}

	computed := parent
	styleChanged := false
	n := ro.LogicalNode()

	if n == nil || n.Kind() == dom.KindDocument {
		// Document root or View uses DisplayBlock baseline.
		computed = &style.Computed{Display: style.DisplayBlock}
	} else if n.Kind() == dom.KindElement {
		if force || ro.Flags()&render.DirtyStyle != 0 {
			oldComputed := ro.ComputedStyle()

			computed = r.Resolve(ro, parent)
			if computed != oldComputed {
				ro.SetComputedStyle(computed)
				styleChanged = true
			}
			ro.ClearDirty(render.DirtyStyle)
		} else {
			computed = ro.ComputedStyle()
		}
	} else if n.Kind() == dom.KindText {
		// Text nodes just inherit parent style.
		if force || ro.ComputedStyle() != parent {
			ro.SetComputedStyle(parent)
			styleChanged = true
		}
	}

	// Recurse.
	if force || styleChanged || flags&render.ChildNeedsStyle != 0 {
		for child := ro.FirstChild(); child != nil; child = child.NextSibling() {
			r.ResolveTree(child, computed, force || styleChanged)
		}
	}
	ro.ClearDirty(render.ChildNeedsStyle)
}

// Resolve returns the Computed style for ro, inheriting from parent where
// appropriate and overlaying the logical node's own style on top.
//
// When the cache is warm (DirtyStyle is false and the parent pointer is
// unchanged), Resolve returns the cached pointer with zero allocations.
func (r *Resolver) Resolve(ro render.Object, parent *style.Computed) *style.Computed {
	n := ro.LogicalNode()
	if n == nil {
		return &style.Computed{Display: style.DisplayBlock}
	}

	de := internaldom.AsDirtyElement(n)

	var el *internaldom.Element
	var isEl bool

	type styleProvider interface {
		RawStyle() style.Style
		DefaultStyle() style.Style
		IntrinsicStyle() style.Style
	}

	var sp styleProvider

	if el, isEl = n.(*internaldom.Element); !isEl {
		if p, ok := n.(styleProvider); ok {
			sp = p
		} else if de != nil {
			sp = de
		} else if un := n.Unwrap(); un != nil {
			if p, ok := un.(styleProvider); ok {
				sp = p
			}
		}

		if sp == nil {
			// Fallback for nodes that don't provide styles (e.g. text nodes if
			// Resolve was called on them directly, though ResolveTree handles it).
			if parent != nil {
				return parent
			}
			return r.defaults
		}
	}

	// Cache hit: no dirty flag and same parent pointer → return cached result.
	if entry, ok := r.cache[n]; ok {
		// We check DirtyStyle from the render object as it's the primary driver now.
		if ro.Flags()&render.DirtyStyle == 0 && entry.Parent == parent {
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
	if isEl {
		c = el.DefaultStyle().Apply(c)
	} else {
		c = sp.DefaultStyle().Apply(c)
	}

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
	if isEl {
		c = el.RawStyle().Apply(c)
	} else {
		c = sp.RawStyle().Apply(c)
	}

	// Layer 5: UA-mandated intrinsic style — highest precedence; authors cannot
	// override. Only applied when the element has intrinsic properties set
	// (sparse: most elements return an empty Style{} and Apply is a no-op).
	var intrinsic style.Style
	if isEl {
		intrinsic = el.IntrinsicStyle()
	} else {
		intrinsic = sp.IntrinsicStyle()
	}
	c = intrinsic.Apply(c)

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
	r.cache[n] = CacheEntry{Parent: parent, Result: result}
	return result
}

// Invalidate removes the cached entry for node, forcing a full re-resolve on
// the next Resolve call regardless of the DirtyStyle flag.
func (r *Resolver) Invalidate(n dom.Node) { delete(r.cache, n) }
