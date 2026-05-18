package render

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// BaseRender provides a default implementation for many render.Object methods.
type BaseRender struct {
	self         Object
	parent       Object
	firstChild   Object
	lastChild    Object
	next         Object
	prev         Object
	flags        DirtyFlag
	cachedSpace  layout.ConstraintSpace
	cachedFrag   *layout.Fragment
	cachedMinMax layout.MinMaxSizes
	minMaxValid  bool
}

// Init sets the self-pointer for the BaseRender so it can pass the correct interface
// when linking children.
func (b *BaseRender) Init(self Object) {
	b.self = self
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
		if f&(DirtyLayout|DirtyStructure|ChildNeedsLayout) != 0 {
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

	for child := range b.Children() {
		child.ClearDirtyRecursive(f)
	}

	// After clearing children, we can clear our own relay flags if no children need them anymore.
	// In a real implementation, we might check if ANY child still has dirty flags.
	// For now, let's just clear them since this is a recursive clear of everything.
	relay := Clean
	if f&DirtyStyle != 0 {
		relay |= ChildNeedsStyle
	}
	if f&(DirtyLayout|DirtyStructure) != 0 {
		relay |= ChildNeedsLayout
	}
	if f&DirtyPaint != 0 {
		relay |= ChildNeedsPaint
	}
	if relay != Clean {
		b.ClearDirty(relay)
	}
}

func (b *BaseRender) MarkChildrenDirty() {
	b.MarkDirty(DirtyStructure | DirtyLayout | DirtyStyle)
}

func (b *BaseRender) Parent() Object          { return b.parent }
func (b *BaseRender) FirstChild() Object      { return b.firstChild }
func (b *BaseRender) LastChild() Object       { return b.lastChild }
func (b *BaseRender) NextSibling() Object     { return b.next }
func (b *BaseRender) PreviousSibling() Object { return b.prev }

func (b *BaseRender) Children() iter.Seq[Object] {
	return func(yield func(Object) bool) {
		for c := b.firstChild; c != nil; c = c.NextSibling() {
			if !yield(c) {
				return
			}
		}
	}
}

func (b *BaseRender) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {
		for c := b.firstChild; c != nil; c = c.NextSibling() {
			if !yield(c.(layout.Node)) {
				return
			}
		}
	}
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
	b.MarkDirty(DirtyStructure | DirtyLayout | DirtyPaint | DirtyStyle | ChildNeedsStyle)
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
	b.MarkDirty(DirtyStructure | DirtyLayout | DirtyPaint | DirtyStyle | ChildNeedsStyle)
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
	b.cachedSpace = space
	b.cachedFrag = frag
	// Successfully measured; clean the dirty flag.
	b.ClearDirty(DirtyLayout)
}

func (b *BaseRender) CachedMinMaxSizes() (layout.MinMaxSizes, bool) {
	// If the layout tree is dirty, the intrinsic sizes may be invalid.
	if b.flags&DirtyLayout != 0 {
		return layout.MinMaxSizes{}, false
	}
	return b.cachedMinMax, b.minMaxValid
}

func (b *BaseRender) SetCachedMinMaxSizes(sizes layout.MinMaxSizes) {
	b.cachedMinMax = sizes
	b.minMaxValid = true
}

// RenderView is the root of a render tree. It represents the viewport.
type RenderView struct {
	BaseRender
	logicalNode  any
	viewportSize layout.Size
	overlays     []Object
}

// NewRenderView creates a new RenderView.
func NewRenderView() *RenderView {
	v := &RenderView{}
	v.Init(v)
	return v
}

// ViewportSize returns the current viewport dimensions.
func (v *RenderView) ViewportSize() layout.Size {
	return v.viewportSize
}

// SetViewportSize updates the viewport dimensions.
func (v *RenderView) SetViewportSize(sz layout.Size) {
	v.viewportSize = sz
	v.MarkDirty(DirtyLayout | DirtyPaint)
}

// Overlays returns the list of overlay render trees.
func (v *RenderView) Overlays() []Object {
	return v.overlays
}

// EventTarget implementation
func (v *RenderView) EventTarget() event.EventTarget { return nil }

func (v *RenderView) Focusable() bool     { return false }
func (v *RenderView) Disabled() bool      { return false }
func (v *RenderView) SetFocusable(b bool) {}
func (v *RenderView) SetDisabled(b bool)  {}

func (v *RenderView) Style() *style.Computed {
	return &style.Computed{Display: style.DisplayBlock}
}
func (v *RenderView) ComputedStyle() *style.Computed {
	return v.Style()
}
func (v *RenderView) SetComputedStyle(*style.Computed) {}

func (v *RenderView) IsDetached() bool { return false }

// StyleNode implementation
func (v *RenderView) RawStyle() style.Style             { return style.Style{} }
func (v *RenderView) SetRawStyle(s style.Style)         {}
func (v *RenderView) ElementDefaultStyle() style.Style  { return style.Style{} }
func (v *RenderView) IsDirtyStyle() bool                { return v.flags&DirtyStyle != 0 }
func (v *RenderView) HasDirtyStyleChild() bool          { return v.flags&ChildNeedsStyle != 0 }
func (v *RenderView) ClearDirtyStyle()                  { v.ClearDirty(DirtyStyle) }
func (v *RenderView) ClearChildNeedsStyle()             { v.ClearDirty(ChildNeedsStyle) }
func (v *RenderView) StyleParent() style.StyleNode      { return nil }
func (v *RenderView) StyleFirstChild() style.StyleNode  { return v.FirstChild() }
func (v *RenderView) StyleNextSibling() style.StyleNode { return v.NextSibling() }

// layout.Node implementation
func (v *RenderView) LogicalNode() any     { return v.logicalNode }
func (v *RenderView) SetLogicalNode(n any) { v.logicalNode = n }

// LayoutPhase runs the layout process for the given subtree using the LayoutNG-inspired architecture.
func LayoutPhase(root Object, available layout.Size) {
	// 1. Build the constraint space for the viewport.
	// The viewport forces a fixed size.
	space := layout.NewConstraintSpaceBuilder(available).
		SetIsFixedInlineSize(true).
		SetIsFixedBlockSize(true).
		ToConstraintSpace()

	// 2. Wrap the root in the formatting context algorithm.
	algo := layout.NewAlgorithm(root, space)

	// 3. Execute the layout pass.
	// This will recursively visit children and cache fragments internally.
	algo.Layout()
}

// Unlink removes obj from its parent.
func Unlink(obj Object) {
	// Implementation would go here.
}

// Attach sets the back-pointer from a logical element to its render object.
func Attach(logical any, ro Object) {
	// Implementation would go here.
}
