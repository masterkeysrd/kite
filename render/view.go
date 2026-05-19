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

	logicalNode   any
	eventTarget   event.EventTarget
	computedStyle *style.Computed
	rawStyle      style.Style
	elementStyle  style.Style
}

// Init sets the self-pointer and logical identity for the BaseRender.
func (b *BaseRender) Init(self Object, logicalNode any, target event.EventTarget) {
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

	for child := range b.Children() {
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
func (b *BaseRender) LogicalNode() any               { return b.logicalNode }
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

func (b *BaseRender) RawStyle() style.Style { return b.rawStyle }
func (b *BaseRender) SetRawStyle(s style.Style) {
	b.rawStyle = s
	b.MarkDirty(DirtyStyle)
}

func (b *BaseRender) ElementDefaultStyle() style.Style { return b.elementStyle }

func (b *BaseRender) SetElementDefaultStyle(s style.Style) {
	b.elementStyle = s
	b.MarkDirty(DirtyStyle)
}

func (b *BaseRender) IsDirtyStyle() bool                { return b.Flags()&DirtyStyle != 0 }
func (b *BaseRender) HasDirtyStyleChild() bool          { return b.Flags()&ChildNeedsStyle != 0 }
func (b *BaseRender) ClearDirtyStyle()                  { b.ClearDirty(DirtyStyle) }
func (b *BaseRender) ClearChildNeedsStyle()             { b.ClearDirty(ChildNeedsStyle) }
func (b *BaseRender) StyleParent() style.StyleNode      { return b.parent }
func (b *BaseRender) StyleFirstChild() style.StyleNode  { return b.firstChild }
func (b *BaseRender) StyleNextSibling() style.StyleNode { return b.next }

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
	viewportSize layout.Size
	overlays     []Object
}

// NewRenderView creates a new RenderView.
func NewRenderView() *RenderView {
	v := &RenderView{}
	v.Init(v, nil, nil)
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

func (v *RenderView) Style() *style.Computed {
	return &style.Computed{Display: style.DisplayBlock}
}

func (v *RenderView) ComputedStyle() *style.Computed {
	return v.Style()
}

func (v *RenderView) SetComputedStyle(*style.Computed) {}

func (v *RenderView) IsDetached() bool { return false }

func (v *RenderView) StyleParent() style.StyleNode { return nil }

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
	if obj == nil || obj.Parent() == nil {
		return
	}
	obj.Parent().RemoveChild(obj)
}

// Attach sets the back-pointer from a logical element to its render object.
func Attach(logical any, ro Object) {
	if host, ok := logical.(HostNode); ok {
		host.SetRenderObject(ro)
	}
}
