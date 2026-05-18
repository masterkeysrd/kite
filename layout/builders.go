package layout

// ConstraintSpaceBuilder is a helper to construct a ConstraintSpace.
type ConstraintSpaceBuilder struct {
	space ConstraintSpace
}

// NewConstraintSpaceBuilder creates a new builder initialized with the available size.
func NewConstraintSpaceBuilder(availableSize Size) *ConstraintSpaceBuilder {
	return &ConstraintSpaceBuilder{
		space: ConstraintSpace{
			AvailableSize: availableSize,
			// By default, percentage resolution size matches available size.
			PercentageResolutionSize: availableSize,
		},
	}
}

// SetPercentageResolutionSize sets the size used for percentage resolution.
func (b *ConstraintSpaceBuilder) SetPercentageResolutionSize(size Size) *ConstraintSpaceBuilder {
	b.space.PercentageResolutionSize = size
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
	size               Size
	children           []FragmentLink
	currentBlockOffset int
}

// NewBoxFragmentBuilder creates a new builder for the given node and constraint space.
func NewBoxFragmentBuilder(node Node, space ConstraintSpace) *BoxFragmentBuilder {
	comp := node.Style()
	return &BoxFragmentBuilder{
		node:               node,
		space:              space,
		currentBlockOffset: comp.Border.Width.Top + comp.Padding.Top,
	}
}

// SetInlineSize sets the final inline size (width) of the fragment.
func (b *BoxFragmentBuilder) SetInlineSize(width int) {
	b.size.Width = width
}

// SetBlockSize sets the final block size (height) of the fragment.
func (b *BoxFragmentBuilder) SetBlockSize(height int) {
	b.size.Height = height
}

// CurrentBlockOffset returns the current block-direction offset (Y).
func (b *BoxFragmentBuilder) CurrentBlockOffset() int {
	return b.currentBlockOffset
}

// AdvanceBlockOffset increases the block-direction offset (Y).
func (b *BoxFragmentBuilder) AdvanceBlockOffset(delta int) {
	b.currentBlockOffset += delta
}

// AddChild adds a child fragment at the specified offset.
func (b *BoxFragmentBuilder) AddChild(frag *Fragment, offset Point) {
	b.children = append(b.children, FragmentLink{
		Offset:   offset,
		Fragment: frag,
	})
}

// ToFragment finalizes the builder and returns an immutable Fragment.
func (b *BoxFragmentBuilder) ToFragment() *Fragment {
	// TODO: Apply Min/Max constraints from style if needed,
	// though they are often already handled by the algorithm.
	return &Fragment{
		Node:     b.node,
		Size:     b.size,
		Children: b.children,
	}
}
