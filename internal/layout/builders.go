package layout

import (
	"slices"
	"sync"

	geometry "github.com/masterkeysrd/kite/geom"
)

// ConstraintSpaceBuilder is a helper to construct a ConstraintSpace.
type ConstraintSpaceBuilder struct {
	space ConstraintSpace
}

// NewConstraintSpaceBuilder creates a new builder initialized with the available size.
// ContainingSpace and ContainerSpace default to zero and must be set explicitly by the caller.
func NewConstraintSpaceBuilder(availableSize geometry.Size) *ConstraintSpaceBuilder {
	return &ConstraintSpaceBuilder{
		space: ConstraintSpace{
			AvailableSize: availableSize,
		},
	}
}

// SetContainingSpace sets the parent's border-box size (ADR-018).
// KindPercent resolution uses ContainerSpace (content-box), not this field.
func (b *ConstraintSpaceBuilder) SetContainingSpace(size geometry.Size) *ConstraintSpaceBuilder {
	b.space.ContainingSpace = size
	return b
}

// SetContainerSpace sets the parent's content-box size (the available space for children
// before per-child margins are subtracted) (ADR-018).
func (b *ConstraintSpaceBuilder) SetContainerSpace(size geometry.Size) *ConstraintSpaceBuilder {
	b.space.ContainerSpace = size
	return b
}

// SetIsFixedInlineSize sets whether the inline size is fixed.
func (b *ConstraintSpaceBuilder) SetIsFixedInlineSize(fixed bool) *ConstraintSpaceBuilder {
	b.space.IsFixedInlineSize = fixed
	return b
}

// SetIsFixedBlockSize sets whether the block size is fixed.
func (b *ConstraintSpaceBuilder) SetIsFixedBlockSize(fixed bool) *ConstraintSpaceBuilder {
	b.space.IsFixedBlockSize = fixed
	return b
}

// ToConstraintSpace returns the constructed ConstraintSpace.
func (b *ConstraintSpaceBuilder) ToConstraintSpace() ConstraintSpace {
	return b.space
}

// BoxFragmentBuilder manages the state of a box fragment being built.
type BoxFragmentBuilder struct {
	node               Node
	space              ConstraintSpace
	size               geometry.Size
	children           []FragmentLink
	currentBlockOffset int
	breakToken         *BreakToken
	hasScrollbarX      bool
	hasScrollbarY      bool
}

var boxBuilderPool = sync.Pool{
	New: func() any {
		return &BoxFragmentBuilder{
			children: make([]FragmentLink, 0, 16),
		}
	},
}

// AcquireBoxFragmentBuilder gets a builder from the pool and initializes it.
func AcquireBoxFragmentBuilder(node Node, space ConstraintSpace) *BoxFragmentBuilder {
	comp := node.Style()
	b := boxBuilderPool.Get().(*BoxFragmentBuilder)
	b.node = node
	b.space = space
	b.size = geometry.Size{}
	b.children = b.children[:0]
	b.currentBlockOffset = comp.Border.Widths().Top + comp.Padding.Top
	b.breakToken = nil
	b.hasScrollbarX = false
	b.hasScrollbarY = false
	return b
}

// NewBoxFragmentBuilder creates a new builder for the given node and constraint space.
func NewBoxFragmentBuilder(node Node, space ConstraintSpace) *BoxFragmentBuilder {
	return AcquireBoxFragmentBuilder(node, space)
}

// SetBreakToken sets the break token for the fragment.
func (b *BoxFragmentBuilder) SetBreakToken(token *BreakToken) {
	b.breakToken = token
}

// SetInlineSize sets the final inline size (width) of the fragment.
func (b *BoxFragmentBuilder) SetInlineSize(width int) {
	b.size.Width = b.clampInlineSize(width)
}

// SetBlockSize sets the final block size (height) of the fragment.
func (b *BoxFragmentBuilder) SetBlockSize(height int) {
	b.size.Height = b.clampBlockSize(height)
}

func isAnonymous(node Node) bool {
	if node == nil {
		return false
	}
	switch node.(type) {
	case *AnonymousBlock, *anonymousTableSection, *anonymousTableRow:
		return true
	default:
		return false
	}
}

func (b *BoxFragmentBuilder) clampInlineSize(width int) int {
	if b.node == nil || isAnonymous(b.node) {
		return width
	}
	comp := b.node.Style()
	if comp == nil {
		return width
	}
	return ClampWidth(b.node, width, b.space)
}

func (b *BoxFragmentBuilder) clampBlockSize(height int) int {
	if b.node == nil || isAnonymous(b.node) {
		return height
	}
	comp := b.node.Style()
	if comp == nil {
		return height
	}
	return ClampHeight(b.node, height, b.space)
}

// SetHasScrollbarX sets whether the fragment has a horizontal scrollbar.
func (b *BoxFragmentBuilder) SetHasScrollbarX(v bool) {
	b.hasScrollbarX = v
}

// SetHasScrollbarY sets whether the fragment has a vertical scrollbar.
func (b *BoxFragmentBuilder) SetHasScrollbarY(v bool) {
	b.hasScrollbarY = v
}

// CurrentBlockOffset returns the current block-direction offset (Y).
func (b *BoxFragmentBuilder) CurrentBlockOffset() int {
	return b.currentBlockOffset
}

// SetBlockOffset sets the current block-direction offset (Y).
func (b *BoxFragmentBuilder) SetBlockOffset(offset int) {
	b.currentBlockOffset = offset
}

// AdvanceBlockOffset increases the block-direction offset (Y).
func (b *BoxFragmentBuilder) AdvanceBlockOffset(delta int) {
	b.currentBlockOffset += delta
}

// AddChild adds a child fragment at the specified offset.
func (b *BoxFragmentBuilder) AddChild(frag *Fragment, offset geometry.Point) {
	b.children = append(b.children, FragmentLink{
		Offset:   offset,
		Fragment: frag,
	})
}

// ToFragment finalizes the builder and returns an immutable Fragment.
// It also returns the builder to the pool.
func (b *BoxFragmentBuilder) ToFragment() *Fragment {
	frag := &Fragment{
		Node:          b.node,
		Size:          b.size,
		Children:      slices.Clone(b.children),
		BreakToken:    b.breakToken,
		HasScrollbarX: b.hasScrollbarX,
		HasScrollbarY: b.hasScrollbarY,
	}
	boxBuilderPool.Put(b)
	return frag
}
