package render

import (
	"iter"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

// BaseRender provides a default implementation for many render.Object methods.
type BaseRender struct {
	self              Object
	parent            Object
	firstChild        Object
	lastChild         Object
	next              Object
	prev              Object
	flags             DirtyFlag
	cachedSpace       layout.ConstraintSpace
	cachedFrag        *layout.Fragment
	cachedMinMax      layout.MinMaxSizes
	minMaxValid       bool
	cachedBlockWidth  int
	cachedBlockHeight int
	blockSizeValid    bool

	logicalNode   dom.Node
	eventTarget   event.EventTarget
	computedStyle *style.Computed
	offset        geom.Point
}

// Init sets the self-pointer and logical identity for the BaseRender.
func (b *BaseRender) Init(self Object, logicalNode dom.Node, target event.EventTarget) {
	b.self = self
	b.logicalNode = logicalNode
	b.eventTarget = target
	b.flags = DirtyStyle | DirtyLayout | DirtyPaint
}

func (b *BaseRender) selfObject() Object {
	if b.self != nil {
		return b.self
	}
	// Fallback that shouldn't happen if Init is called.
	return nil
}

func (b *BaseRender) Flags() DirtyFlag { return b.flags }

func (b *BaseRender) MarkDirty(f DirtyFlag) {
	b.flags |= f
	if b.parent != nil {
		relay := Clean
		if f&(DirtyStyle|ChildNeedsStyle) != 0 {
			relay |= ChildNeedsStyle
		}
		if f&(DirtyLayout|ChildNeedsLayout) != 0 {
			relay |= ChildNeedsLayout
		}
		if f&(DirtyPaint|DirtyScroll|ChildNeedsPaint) != 0 {
			relay |= ChildNeedsPaint
		}
		if relay != Clean {
			b.parent.MarkDirty(relay)
		}
	}
}

func (b *BaseRender) ClearDirty(f DirtyFlag) {
	b.flags &^= f
}

func (b *BaseRender) ClearDirtyRecursive(f DirtyFlag) {
	b.ClearDirty(f)

	for child := b.FirstChild(); child != nil; child = child.NextSibling() {
		child.ClearDirtyRecursive(f)
	}

	// After clearing children, we can clear our own relay flags if no children need them anymore.
	relay := Clean
	if f&DirtyStyle != 0 {
		relay |= ChildNeedsStyle
	}
	if f&DirtyLayout != 0 {
		relay |= ChildNeedsLayout
	}
	if f&(DirtyPaint|DirtyScroll) != 0 {
		relay |= ChildNeedsPaint
	}
	if relay != Clean {
		b.ClearDirty(relay)
	}
}

func (b *BaseRender) MarkChildrenDirty() {
	b.MarkDirty(DirtyLayout | DirtyStyle)
}

func (b *BaseRender) Parent() Object      { return b.parent }
func (b *BaseRender) FirstChild() Object  { return b.firstChild }
func (b *BaseRender) LastChild() Object   { return b.lastChild }
func (b *BaseRender) NextSibling() Object { return b.next }
func (b *BaseRender) PreviousSibling() Object {
	return b.prev
}

func (b *BaseRender) EventTarget() event.EventTarget { return b.eventTarget }

func (b *BaseRender) IsDetached() bool               { return b.parent == nil }
func (b *BaseRender) LogicalNode() dom.Node          { return b.logicalNode }
func (b *BaseRender) ComputedStyle() *style.Computed { return b.computedStyle }
func (b *BaseRender) Style() *style.Computed         { return b.computedStyle }

func (b *BaseRender) SetComputedStyle(c *style.Computed) {
	if b.computedStyle != nil {
		if b.computedStyle.AffectsLayout(c) {
			b.MarkDirty(DirtyLayout | DirtyPaint)
		} else if b.computedStyle.AffectsPaint(c) {
			b.MarkDirty(DirtyPaint)
		}
	} else {
		// First time initialization
		b.MarkDirty(DirtyLayout | DirtyPaint)
	}
	b.computedStyle = c
}

func (b *BaseRender) IsDirtyPaint() bool       { return b.Flags()&(DirtyPaint|DirtyScroll) != 0 }
func (b *BaseRender) HasChildNeedsPaint() bool { return b.Flags()&ChildNeedsPaint != 0 }

func (b *BaseRender) Offset() geom.Point {
	offset := b.offset
	if b.parent != nil {
		pOffset := b.parent.Offset()
		offset.X += pOffset.X
		offset.Y += pOffset.Y
	}
	return offset
}

func (b *BaseRender) SetOffset(p geom.Point) {
	if b.offset != p {
		b.MarkDirty(DirtyPaint)
		b.offset = p
	}
}

func (b *BaseRender) IsAnonymous() bool {
	return false
}

func (b *BaseRender) MaxScroll() (x, y int) {
	if b.cachedFrag == nil {
		return 0, 0
	}
	return layout.MaxScroll(b.cachedFrag)
}

func (b *BaseRender) Children() iter.Seq[Object] {
	return func(yield func(Object) bool) {
		for c := b.firstChild; c != nil; c = c.NextSibling() {
			if !yield(c) {
				return
			}
		}
	}
}

func (b *BaseRender) FirstLayoutChild() layout.Node {
	if b.firstChild == nil {
		return nil
	}
	return b.firstChild.(layout.Node)
}

func (b *BaseRender) NextLayoutSibling(child layout.Node) layout.Node {
	next := child.(Object).NextSibling()
	if next == nil {
		return nil
	}
	return next.(layout.Node)
}

type linker interface {
	setParent(Object)
	setNext(Object)
	setPrev(Object)
}

func (b *BaseRender) setParent(p Object) { b.parent = p }
func (b *BaseRender) setNext(n Object)   { b.next = n }
func (b *BaseRender) setPrev(p Object)   { b.prev = p }

func (b *BaseRender) InsertChild(child, before Object) {
	c, ok := child.(linker)
	if !ok {
		return
	}
	c.setParent(b.selfObject())
	if before == nil {
		c.setPrev(b.lastChild)
		c.setNext(nil)
		if b.lastChild != nil {
			b.lastChild.(linker).setNext(child)
		} else {
			b.firstChild = child
		}
		b.lastChild = child
	} else {
		prev := before.PreviousSibling()
		c.setPrev(prev)
		c.setNext(before)
		before.(linker).setPrev(child)
		if prev != nil {
			prev.(linker).setNext(child)
		} else {
			b.firstChild = child
		}
	}

	// Propagate child's existing flags up the new parent chain.
	if childFlags := child.Flags(); childFlags != Clean {
		b.MarkDirty(childFlags)
	}

	b.MarkDirty(DirtyLayout | DirtyPaint | DirtyStyle | ChildNeedsStyle)
}

func (b *BaseRender) RemoveChild(child Object) {
	c, ok := child.(linker)
	if !ok || child.Parent() != b.selfObject() {
		return
	}
	prev := child.PreviousSibling()
	next := child.NextSibling()
	if prev != nil {
		prev.(linker).setNext(next)
	} else {
		b.firstChild = next
	}
	if next != nil {
		next.(linker).setPrev(prev)
	} else {
		b.lastChild = prev
	}
	c.setParent(nil)
	c.setNext(nil)
	c.setPrev(nil)
	b.MarkDirty(DirtyLayout | DirtyPaint | DirtyStyle | ChildNeedsStyle)
}

// selfObject returns the concrete render object wrapping this BaseRender.
// Since BaseRender doesn't know its wrapper, it must be provided by the wrapper.
// We will modify BaseRender to store a self Object pointer, initialized on creation.

func (b *BaseRender) IsDirtyLayout() bool { return b.flags&DirtyLayout != 0 }
func (b *BaseRender) ClearDirtyLayout()   { b.ClearDirty(DirtyLayout) }

func (b *BaseRender) Fragment() *layout.Fragment { return b.cachedFrag }

func (b *BaseRender) CachedLayout(space layout.ConstraintSpace) *layout.Fragment {
	// If the layout tree is dirty, the cache is invalid.
	if b.flags&(DirtyLayout|ChildNeedsLayout) != 0 {
		return nil
	}
	// Return fragment only if the incoming constraints match exactly.
	if b.cachedSpace == space {
		return b.cachedFrag
	}
	return nil
}

func (b *BaseRender) SetCachedLayout(space layout.ConstraintSpace, frag *layout.Fragment) {
	if b.cachedFrag != frag {
		b.MarkDirty(DirtyPaint)
	}
	b.cachedSpace = space
	b.cachedFrag = frag
	// Successfully measured; clean the dirty flag and ensure parents know we are clean.
	b.ClearDirty(DirtyLayout)
}

func (b *BaseRender) CachedMinMaxSizes() (layout.MinMaxSizes, bool) {
	// If the layout tree is dirty, the intrinsic sizes may be invalid.
	if b.flags&(DirtyLayout|ChildNeedsLayout) != 0 {
		return layout.MinMaxSizes{}, false
	}
	return b.cachedMinMax, b.minMaxValid
}

func (b *BaseRender) SetCachedMinMaxSizes(sizes layout.MinMaxSizes) {
	b.cachedMinMax = sizes
	b.minMaxValid = true
}

func (b *BaseRender) CachedBlockSize(width int) (int, bool) {
	if b.flags&(DirtyLayout|ChildNeedsLayout) != 0 {
		return 0, false
	}
	if b.blockSizeValid && b.cachedBlockWidth == width {
		return b.cachedBlockHeight, true
	}
	return 0, false
}

func (b *BaseRender) SetCachedBlockSize(width, height int) {
	b.cachedBlockWidth = width
	b.cachedBlockHeight = height
	b.blockSizeValid = true
}

// RenderView is the root of a render tree. It represents the viewport.
type RenderView struct {
	BaseRender
	viewportSize geom.Size
	overlays     []Object
}

// NewRenderView creates a new RenderView.
func NewRenderView() *RenderView {
	v := &RenderView{}
	v.Init(v, nil, nil)
	return v
}

// ViewportSize returns the current viewport dimensions.
func (v *RenderView) ViewportSize() geom.Size {
	return v.viewportSize
}

// SetViewportSize updates the viewport dimensions.
func (v *RenderView) SetViewportSize(sz geom.Size) {
	if v.viewportSize == sz {
		return
	}
	v.viewportSize = sz
	v.MarkDirty(DirtyLayout | DirtyPaint)

	// Invalidate styles that contain media rules.
	invalidateMediaStyles(v)
	for _, overlay := range v.overlays {
		invalidateMediaStyles(overlay)
	}
}

// Overlays returns the list of overlay render trees.
func (v *RenderView) Overlays() []Object {
	return v.overlays
}

// SetOverlays replaces the list of overlay render trees.
func (v *RenderView) SetOverlays(overlays []Object) {
	v.overlays = overlays
	v.MarkDirty(DirtyLayout | DirtyPaint)
}

func (v *RenderView) Style() *style.Computed {
	return &style.Computed{Display: style.DisplayBlock}
}

func (v *RenderView) ComputedStyle() *style.Computed {
	return v.Style()
}

func (v *RenderView) SetComputedStyle(*style.Computed) {}

func (v *RenderView) IsDetached() bool { return false }

func (v *RenderView) SetLogicalNode(n dom.Node) { v.logicalNode = n }

// LayoutPhase runs the layout process for the given subtree using the LayoutNG-inspired architecture.
func LayoutPhase(ctx *layout.Context, root Object, available geom.Size) {
	// 1. Build the constraint space for the viewport.
	// The viewport forces a fixed size.
	// The viewport has no border/padding, so ContainingSpace and ContainerSpace are equal.
	space := layout.NewConstraintSpaceBuilder(available).
		SetContainingSpace(available).
		SetContainerSpace(available).
		SetIsFixedInlineSize(true).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	// 2. Wrap the root in the formatting context algorithm.
	algo := layout.GetAlgorithm(root)

	// 3. Execute the layout pass.
	// This will recursively visit children and cache fragments internally.
	frag := algo.Layout(ctx, root, space)
	propagateOffsets(frag)

	// 4. Layout overlays.
	if rv, ok := root.(*RenderView); ok {
		for _, overlay := range rv.Overlays() {
			comp := overlay.ComputedStyle()
			avail := available
			if comp != nil {
				avail.Width = max(0, avail.Width-comp.Margin.Left-comp.Margin.Right)
				avail.Height = max(0, avail.Height-comp.Margin.Top-comp.Margin.Bottom)
			}

			osb := layout.NewConstraintSpaceBuilder(avail).
				SetContainingSpace(available).
				SetContainerSpace(available)

			if comp != nil {
				if comp.Width.Kind() == style.KindPercent && comp.Width.PercentValue() == 100 {
					osb.SetIsFixedInlineSize(true)
				}
				if comp.Height.Kind() == style.KindPercent && comp.Height.PercentValue() == 100 {
					osb.SetIsFixedBlockSize(true)
				}
			}
			overlaySpace := osb.ToConstraintSpace()

			overlayAlgo := layout.GetAlgorithm(overlay)
			frag := overlayAlgo.Layout(ctx, overlay, overlaySpace)
			overlay.SetCachedLayout(overlaySpace, frag)
			propagateOffsets(frag)

			// If the overlay doesn't have a custom positioner (like OverlayAlgorithm),
			// we fallback to margin-based positioning relative to the viewport.
			if _, ok := overlay.LogicalNode().(layout.OverlayLever); !ok {
				if cs := overlay.ComputedStyle(); cs != nil {
					// Use margins for absolute positioning relative to viewport.
					x, y := cs.Margin.Left, cs.Margin.Top
					overlay.SetOffset(geom.Point{X: x, Y: y})
				}
			}
		}
	}
}

func propagateOffsets(frag *layout.Fragment) {
	propagateOffsetsAccum(frag, geom.Point{})
}

func propagateOffsetsAccum(frag *layout.Fragment, accum geom.Point) {
	if frag == nil {
		return
	}
	for _, link := range frag.Children {
		offset := geom.Point{X: accum.X + link.Offset.X, Y: accum.Y + link.Offset.Y}
		if ro, ok := link.Fragment.Node.(Object); ok {
			ro.SetOffset(offset)
			if ro.Flags()&(DirtyLayout|ChildNeedsLayout) != 0 {
				propagateOffsetsAccum(link.Fragment, geom.Point{})
			}
		} else {
			propagateOffsetsAccum(link.Fragment, offset)
		}
	}
}

// Unlink removes obj from its parent.
func Unlink(obj Object) {
	if obj == nil || obj.Parent() == nil {
		return
	}
	obj.Parent().RemoveChild(obj)
}

func invalidateMediaStyles(obj Object) {
	if obj == nil {
		return
	}
	n := obj.LogicalNode()
	if n != nil {
		type styleProvider interface {
			RawStyle() style.Style
		}
		var sp styleProvider
		if p, ok := n.(styleProvider); ok {
			sp = p
		} else if un := n.Unwrap(); un != nil {
			if p, ok := un.(styleProvider); ok {
				sp = p
			}
		}
		if sp != nil && sp.RawStyle().HasMediaRules() {
			obj.MarkDirty(DirtyStyle)
		}
	}
	for child := obj.FirstChild(); child != nil; child = child.NextSibling() {
		invalidateMediaStyles(child)
	}
}
