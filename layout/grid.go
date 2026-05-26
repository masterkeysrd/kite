package layout

import (
	"github.com/masterkeysrd/kite/style"
)

// GridAlgorithm implements the CSS Grid formatting context layout.
type GridAlgorithm struct {
	Node  Node
	Space ConstraintSpace

	builder *GridBuilder
}

// Layout executes the grid layout algorithm and returns an immutable Fragment.
func (a *GridAlgorithm) Layout(ctx *Context) *Fragment {
	if cached := a.Node.CachedLayout(a.Space); cached != nil {
		return cached
	}
	defer ctx.Begin("Layout(Grid)")()

	comp := a.Node.Style()
	decor := ResolveDecorations(a.Node, false, false)

	if a.builder == nil {
		a.builder = &GridBuilder{}
		a.builder.Init(a.Node, a.Space)
		a.builder.PlaceItems()
	}
	builder := a.builder

	colGap := comp.GridColumnGap
	rowGap := comp.GridRowGap

	// 1. Determine resolved container size (border-box)
	var minMax MinMaxSizes
	if !a.Space.IsFixedInlineSize || comp.Width.Kind() == style.KindMaxContent || comp.Width.Kind() == style.KindAuto || comp.Width.Kind() == style.KindContent {
		minMax = a.ComputeMinMaxSizes(ctx)
	}

	resolvedWidth := a.Space.AvailableSize.Width
	if !a.Space.IsFixedInlineSize || comp.Width.Kind() == style.KindMaxContent {
		switch comp.Width.Kind() {
		case style.KindPercent:
			resolvedWidth = int(float32(a.Space.ContainerSpace.Width) * comp.Width.PercentValue() / 100.0)
		case style.KindCells:
			resolvedWidth = comp.Width.CellsValue()
		case style.KindAuto:
			// Block-level boxes take full available width by default.
			resolvedWidth = a.Space.AvailableSize.Width
		case style.KindContent:
			resolvedWidth = min(minMax.Max, a.Space.AvailableSize.Width)
		case style.KindMaxContent:
			resolvedWidth = minMax.Max
		}
	}
	resolvedWidth = max(resolvedWidth, decor.Insets.Left+decor.Insets.Right)
	viewportWidth := resolvedWidth - decor.Insets.Left - decor.Insets.Right

	// Resolve columns first to know widths for row height measurement
	colWidths := a.resolveTracks(ctx, builder.items, builder.colTemplate, builder.maxCol, viewportWidth, colGap, true, nil)

	// Resolve rows
	// Initial guess for viewport height if not fixed
	tempViewportHeight := a.Space.AvailableSize.Height
	if !a.Space.IsFixedBlockSize {
		tempViewportHeight = InfiniteBlockSize
	}
	rowHeights := a.resolveTracks(ctx, builder.items, builder.rowTemplate, builder.maxRow, tempViewportHeight, rowGap, false, colWidths)

	contentHeight := sum(rowHeights) + max(0, len(rowHeights)-1)*rowGap
	resolvedHeight := a.Space.AvailableSize.Height
	if !a.Space.IsFixedBlockSize {
		isIndefinitePercent := comp.Height.Kind() == style.KindPercent && a.Space.ContainerSpace.Height >= InfiniteBlockSize
		if comp.Height.Kind() == style.KindAuto || comp.Height.Kind() == style.KindContent || isIndefinitePercent {
			resolvedHeight = contentHeight + decor.Insets.Top + decor.Insets.Bottom
		} else {
			switch comp.Height.Kind() {
			case style.KindPercent:
				resolvedHeight = int(float32(a.Space.ContainerSpace.Height) * comp.Height.PercentValue() / 100.0)
			case style.KindCells:
				resolvedHeight = comp.Height.CellsValue()
			}
		}
	}
	resolvedHeight = max(resolvedHeight, decor.Insets.Top+decor.Insets.Bottom)

	// If height is fixed, we might need to re-resolve rows to distribute remaining space if there are FR tracks
	if a.Space.IsFixedBlockSize || comp.Height.Kind() == style.KindCells || comp.Height.Kind() == style.KindPercent {
		viewportHeight := resolvedHeight - decor.Insets.Top - decor.Insets.Bottom
		rowHeights = a.resolveTracks(ctx, builder.items, builder.rowTemplate, builder.maxRow, viewportHeight, rowGap, false, colWidths)
	}

	// 3. Layout Pass
	boxBuilder := NewBoxFragmentBuilder(a.Node, a.Space)
	boxBuilder.SetInlineSize(resolvedWidth)
	boxBuilder.SetBlockSize(resolvedHeight)

	for i := range builder.items {
		item := &builder.items[i]
		// Calculate available size for the item based on spans
		itemWidth := 0
		for j := 0; j < item.colSpan; j++ {
			c := item.colStart + j
			if c < len(colWidths) {
				itemWidth += colWidths[c]
			}
		}
		itemWidth += max(0, item.colSpan-1) * colGap

		itemHeight := 0
		for j := 0; j < item.rowSpan; j++ {
			r := item.rowStart + j
			if r < len(rowHeights) {
				itemHeight += rowHeights[r]
			}
		}
		itemHeight += max(0, item.rowSpan-1) * rowGap

		childSpace := ConstraintSpace{
			AvailableSize:     Size{Width: itemWidth, Height: itemHeight},
			ContainingSpace:   Size{Width: itemWidth, Height: itemHeight},
			ContainerSpace:    Size{Width: itemWidth, Height: itemHeight},
			IsFixedInlineSize: true,
			IsFixedBlockSize:  true,
		}

		childAlgo := NewAlgorithm(item.node, childSpace)
		frag := childAlgo.Layout(ctx)

		// Offset relative to border-box
		offsetX := decor.Insets.Left
		for j := 0; j < item.colStart; j++ {
			if j < len(colWidths) {
				offsetX += colWidths[j] + colGap
			}
		}

		offsetY := decor.Insets.Top
		for j := 0; j < item.rowStart; j++ {
			if j < len(rowHeights) {
				offsetY += rowHeights[j] + rowGap
			}
		}

		boxBuilder.AddChild(frag, Point{X: offsetX, Y: offsetY})
	}

	fragment := boxBuilder.ToFragment()
	a.Node.SetCachedLayout(a.Space, fragment)
	return fragment
}

func (a *GridAlgorithm) resolveTracks(ctx *Context, items []gridItem, template []style.GridTrackSize, count int, available int, gap int, isCol bool, colWidths []int) []int {
	var tracksBuf [16]style.GridTrackSize
	var tracks []style.GridTrackSize
	if count <= 16 {
		tracks = tracksBuf[:count]
	} else {
		tracks = make([]style.GridTrackSize, count)
	}

	for i := 0; i < count; i++ {
		if i < len(template) {
			tracks[i] = template[i]
		} else {
			tracks[i] = style.Auto
		}
	}

	resolved := ResolveTrackSizes(tracks, available, gap)

	// 1. Measure Pass (Auto AND Fr tracks)
	for i, t := range tracks {
		kind := t.Kind()
		if kind == style.KindAuto || kind == style.KindFr {
			maxSize := 0
			// Find items that span ONLY this track (simplified auto resolution)
			for j := range items {
				item := &items[j]
				start, span := item.rowStart, item.rowSpan
				if isCol {
					start, span = item.colStart, item.colSpan
				}

				if start == i && span == 1 {
					if isCol {
						maxSize = max(maxSize, IntrinsicMinMaxSizes(ctx, item.node).Min)
					} else if colWidths != nil {
						// For rows, we need the resolved width of the item to measure its height
						itemWidth := 0
						for k := 0; k < item.colSpan; k++ {
							c := item.colStart + k
							if c < len(colWidths) {
								itemWidth += colWidths[c]
							}
						}
						itemWidth += max(0, item.colSpan-1) * gap

						maxSize = max(maxSize, IntrinsicBlockSize(ctx, item.node, itemWidth))
					} else {
						// Fallback if colWidths not yet resolved
						maxSize = max(maxSize, IntrinsicBlockSize(ctx, item.node, available))
					}
				}
			}
			resolved[i] = maxSize
		}
	}

	// 2. Fractional Pass
	totalNonFr := 0
	for _, w := range resolved {
		totalNonFr += w
	}
	totalGaps := max(0, count-1) * gap
	remaining := max(0, available-totalNonFr-totalGaps)

	totalFr := float32(0)
	for _, t := range tracks {
		if t.Kind() == style.KindFr {
			totalFr += t.FrValue()
		}
	}

	if totalFr > 0 && available < InfiniteBlockSize {
		sumFr := 0
		lastFrIdx := -1
		for i, t := range tracks {
			if t.Kind() == style.KindFr {
				val := int(float32(remaining) * t.FrValue() / totalFr)
				resolved[i] += val
				sumFr += val
				lastFrIdx = i
			}
		}
		// Distribute rounding remainder to the last fr track
		if lastFrIdx >= 0 && sumFr < remaining {
			resolved[lastFrIdx] += (remaining - sumFr)
		}
	}

	return resolved
}

func (a *GridAlgorithm) ComputeMinMaxSizes(ctx *Context) MinMaxSizes {
	if sizes, ok := a.Node.CachedMinMaxSizes(); ok {
		return sizes
	}

	if a.builder == nil {
		a.builder = &GridBuilder{}
		a.builder.Init(a.Node, a.Space)
		a.builder.PlaceItems()
	}
	builder := a.builder

	comp := a.Node.Style()
	colGap := comp.GridColumnGap

	// Resolve tracks with large available space to get intrinsic max
	colWidths := a.resolveTracks(ctx, builder.items, builder.colTemplate, builder.maxCol, InfiniteBlockSize, colGap, true, nil)

	totalMax := sum(colWidths) + max(0, len(colWidths)-1)*colGap

	sizes := MinMaxSizes{Min: 0, Max: totalMax}
	a.Node.SetCachedMinMaxSizes(sizes)
	return sizes
}

func sum(vals []int) int {
	s := 0
	for _, v := range vals {
		s += v
	}
	return s
}
