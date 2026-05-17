package render

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// BaseRender provides a default implementation for many render.Object methods.
type BaseRender struct {
	parent      Object
	flags       DirtyFlag
	cachedSpace layout.ConstraintSpace
	cachedFrag  *layout.Fragment
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

func (b *BaseRender) MarkChildrenDirty() {
	b.MarkDirty(DirtyStructure | DirtyLayout)
}

func (b *BaseRender) Parent() Object { return b.parent }

func (b *BaseRender) Fragment() *layout.Fragment { return b.cachedFrag }

func (b *BaseRender) CachedLayout(space layout.ConstraintSpace) *layout.Fragment {
	// If the layout tree is dirty, the cache is invalid.
	if b.flags&DirtyLayout != 0 {
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

// RenderView is the root of a render tree. It represents the viewport.
type RenderView struct {
	BaseRender
	viewportSize layout.Size
	overlays     []Object
}

// NewRenderView creates a new RenderView.
func NewRenderView() *RenderView {
	return &RenderView{}
}

// ViewportSize returns the current viewport dimensions.
func (v *RenderView) ViewportSize() layout.Size {
	return v.viewportSize
}

// SetViewportSize updates the viewport dimensions.
func (v *RenderView) SetViewportSize(sz layout.Size) {
	v.viewportSize = sz
	v.MarkDirty(DirtyLayout)
}

// Overlays returns the list of overlay render trees.
func (v *RenderView) Overlays() []Object {
	return v.overlays
}

// EventTarget implementation
func (v *RenderView) EventTarget() event.EventTarget { return nil }

// Tree navigation
func (v *RenderView) FirstChild() Object      { return nil }
func (v *RenderView) LastChild() Object       { return nil }
func (v *RenderView) NextSibling() Object     { return nil }
func (v *RenderView) PreviousSibling() Object { return nil }
func (v *RenderView) Children() iter.Seq[Object] {
	return func(yield func(Object) bool) {}
}

func (v *RenderView) Focusable() bool { return false }
func (v *RenderView) Disabled() bool  { return false }

func (v *RenderView) Style() *style.Computed {
	return &style.Computed{Display: style.DisplayBlock}
}
func (v *RenderView) ComputedStyle() *style.Computed {
	return v.Style()
}
func (v *RenderView) SetComputedStyle(*style.Computed) {}

func (v *RenderView) IsDetached() bool { return false }

// StyleNode implementation
func (v *RenderView) RawStyle() style.Style            { return style.Style{} }
func (v *RenderView) ElementDefaultStyle() style.Style { return style.Style{} }
func (v *RenderView) IsDirtyStyle() bool               { return v.flags&DirtyStyle != 0 }
func (v *RenderView) HasDirtyStyleChild() bool         { return v.flags&ChildNeedsStyle != 0 }
func (v *RenderView) ClearDirtyStyle()                 { v.ClearDirty(DirtyStyle) }
func (v *RenderView) ClearChildNeedsStyle()            { v.ClearDirty(ChildNeedsStyle) }
func (v *RenderView) StyleParent() style.StyleNode     { return nil }
func (v *RenderView) StyleFirstChild() style.StyleNode { return nil }
func (v *RenderView) StyleNextSibling() style.StyleNode {
	return nil
}

// layout.Node implementation
func (v *RenderView) LayoutChildren() iter.Seq[layout.Node] {
	return func(yield func(layout.Node) bool) {}
}
func (v *RenderView) IsDirtyLayout() bool { return v.flags&DirtyLayout != 0 }
func (v *RenderView) ClearDirtyLayout()   { v.ClearDirty(DirtyLayout) }
func (v *RenderView) LogicalNode() any    { return nil }

// LayoutPhase runs the layout process for the given subtree using the LayoutNG-inspired architecture.
func LayoutPhase(root Object, available layout.Size) {
	// 1. Build the constraint space for the viewport.
	// The viewport forces a fixed size.
	space := layout.ConstraintSpace{
		Constraints: layout.Constraints{
			Min: available,
			Max: available,
		},
	}

	// 2. Wrap the root in the formatting context algorithm.
	algo := layout.BlockAlgorithm{
		Node:  root,
		Space: space,
	}

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
