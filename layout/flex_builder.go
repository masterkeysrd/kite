package layout

import (
	"sort"

	"github.com/masterkeysrd/kite/style"
)

// flexGeometry provides logical axis helpers.
type flexGeometry struct {
	direction style.FlexDirection
}

func (g flexGeometry) MainSize(s Size) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return s.Height
	}
	return s.Width
}

func (g flexGeometry) CrossSize(s Size) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return s.Width
	}
	return s.Height
}

func (g flexGeometry) MakeSize(main, cross int) Size {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return Size{Width: cross, Height: main}
	}
	return Size{Width: main, Height: cross}
}

func (g flexGeometry) MainAxis(p Point) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return p.Y
	}
	return p.X
}

func (g flexGeometry) CrossAxis(p Point) int {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return p.X
	}
	return p.Y
}

func (g flexGeometry) MakePoint(main, cross int) Point {
	if g.direction == style.FlexColumn || g.direction == style.FlexColumnReverse {
		return Point{X: cross, Y: main}
	}
	return Point{X: main, Y: cross}
}

// FlexItem represents a transient layout state for a flex child.
type FlexItem struct {
	Node Node

	// Flex base size (initial main size before growing/shrinking).
	BaseSize int

	// Hypothetical main size (base size clamped by min/max constraints).
	HypotheticalMainSize int

	// Final resolved sizes.
	MainSize  int
	CrossSize int

	// Frozen indicates the item's main size has been fixed.
	Frozen bool

	// Style shortcuts.
	Grow   int
	Shrink int
	Order  int

	// Min/Max constraints in main axis.
	MinMainSize int
	MaxMainSize int

	// Cached measurement result.
	Fragment *Fragment

	// Physical offset relative to container content box.
	Offset Point
}

// FlexLine represents a group of flex items that fit on a single line.
type FlexLine struct {
	Items     []*FlexItem
	MainSize  int
	CrossSize int
}

// FlexLineBuilder encapsulates the mutable state required for resolving flex lines.
// It acts as the state machine for the flex layout algorithm, handling item collection,
// line breaking, space distribution (growing/shrinking), and alignment.
type FlexLineBuilder struct {
	geom     flexGeometry
	mainGap  int
	crossGap int
	items    []*FlexItem
	lines    []*FlexLine
}

func NewFlexLineBuilder(geom flexGeometry, mainGap, crossGap int) *FlexLineBuilder {
	return &FlexLineBuilder{
		geom:     geom,
		mainGap:  mainGap,
		crossGap: crossGap,
	}
}

func (b *FlexLineBuilder) AddItem(node Node, baseSize, minSize, maxSize, grow, shrink, order int) {
	hypothetical := baseSize
	if minSize > 0 && hypothetical < minSize {
		hypothetical = minSize
	}
	if maxSize > 0 && hypothetical > maxSize {
		hypothetical = maxSize
	}

	b.items = append(b.items, &FlexItem{
		Node:                 node,
		BaseSize:             baseSize,
		HypotheticalMainSize: hypothetical,
		MinMainSize:          minSize,
		MaxMainSize:          maxSize,
		Grow:                 grow,
		Shrink:               shrink,
		Order:                order,
	})
}

// SortItems sorts items by their Order property.
func (b *FlexLineBuilder) SortItems() {
	sort.SliceStable(b.items, func(i, j int) bool {
		return b.items[i].Order < b.items[j].Order
	})
}

// ReverseItems reverses the order of items (used for row-reverse/column-reverse).
func (b *FlexLineBuilder) ReverseItems() {
	for i, j := 0, len(b.items)-1; i < j; i, j = i+1, j-1 {
		b.items[i], b.items[j] = b.items[j], b.items[i]
	}
}

// ComputeLines groups the items into lines based on the available main size.
func (b *FlexLineBuilder) ComputeLines(availableMainSize int, wrap bool) {
	if len(b.items) == 0 {
		return
	}

	currentLine := &FlexLine{}
	b.lines = append(b.lines, currentLine)

	for _, item := range b.items {
		if wrap && len(currentLine.Items) > 0 && currentLine.MainSize+b.mainGap+item.HypotheticalMainSize > availableMainSize {
			currentLine = &FlexLine{}
			b.lines = append(b.lines, currentLine)
		}

		if len(currentLine.Items) > 0 {
			currentLine.MainSize += b.mainGap
		}
		currentLine.Items = append(currentLine.Items, item)
		currentLine.MainSize += item.HypotheticalMainSize
	}
}

// ResolveFlexibleLengths distributes free space among items in a line.
func (b *FlexLineBuilder) ResolveFlexibleLengths(lineIndex int, availableMain int) {
	line := b.lines[lineIndex]

	// 1. Determine used flex factor.
	totalHypotheticalMainSize := 0
	for _, item := range line.Items {
		totalHypotheticalMainSize += item.HypotheticalMainSize
	}
	// Add gaps to the total hypothetical main size.
	if len(line.Items) > 1 {
		totalHypotheticalMainSize += b.mainGap * (len(line.Items) - 1)
	}

	freeSpace := availableMain - totalHypotheticalMainSize
	useGrow := freeSpace > 0

	// 2. Size inflexible items.
	for _, item := range line.Items {
		item.Frozen = false
		if (useGrow && item.Grow == 0) || (!useGrow && item.Shrink == 0) {
			item.MainSize = item.HypotheticalMainSize
			item.Frozen = true
		}
	}

	// 3. Loop until all items are frozen.
	for {
		totalFlexFactor := 0
		totalBaseSizeFactor := 0 // for shrinking
		remainingFreeSpace := availableMain
		if len(line.Items) > 1 {
			remainingFreeSpace -= b.mainGap * (len(line.Items) - 1)
		}

		for _, item := range line.Items {
			if item.Frozen {
				remainingFreeSpace -= item.MainSize
			} else {
				remainingFreeSpace -= item.HypotheticalMainSize
				if useGrow {
					totalFlexFactor += item.Grow
				} else {
					totalFlexFactor += item.Shrink
					totalBaseSizeFactor += item.Shrink * item.BaseSize
				}
			}
		}

		if totalFlexFactor == 0 {
			for _, item := range line.Items {
				if !item.Frozen {
					item.MainSize = item.HypotheticalMainSize
					item.Frozen = true
				}
			}
			break
		}

		// Distribute free space.
		var violationCount int

		for _, item := range line.Items {
			if item.Frozen {
				continue
			}

			if useGrow {
				item.MainSize = item.HypotheticalMainSize + (remainingFreeSpace * item.Grow / totalFlexFactor)
			} else if totalBaseSizeFactor > 0 {
				shrinkAmount := ((-remainingFreeSpace) * item.Shrink * item.BaseSize) / totalBaseSizeFactor
				item.MainSize = item.HypotheticalMainSize - shrinkAmount
			} else {
				item.MainSize = item.HypotheticalMainSize
			}

			// Respect min/max bounds
			if item.MainSize < item.MinMainSize {
				item.MainSize = item.MinMainSize
				item.Frozen = true
				violationCount++
			} else if item.MaxMainSize > 0 && item.MainSize > item.MaxMainSize {
				item.MainSize = item.MaxMainSize
				item.Frozen = true
				violationCount++
			}
		}

		if violationCount == 0 {
			break
		}
	}
}

// Lines returns the computed lines.
func (b *FlexLineBuilder) Lines() []*FlexLine {
	return b.lines
}

// AlignLine handles main-axis alignment (justify-content) for a single line.
func (b *FlexLineBuilder) AlignLine(lineIndex int, containerMainSize int, justifyContent style.Justify, isReverse bool) {
	line := b.lines[lineIndex]
	remainingMain := containerMainSize - line.MainSize
	var startMainOffset int
	var itemSpacing = b.mainGap

	switch justifyContent {
	case style.JustifyStart:
		if isReverse {
			startMainOffset = remainingMain
		} else {
			startMainOffset = 0
		}
	case style.JustifyEnd:
		if isReverse {
			startMainOffset = 0
		} else {
			startMainOffset = remainingMain
		}
	case style.JustifyCenter:
		startMainOffset = remainingMain / 2
	case style.JustifyBetween:
		if remainingMain < 0 {
			startMainOffset = 0
			itemSpacing = b.mainGap
		} else if len(line.Items) > 1 {
			itemSpacing = b.mainGap + remainingMain/(len(line.Items)-1)
		}
	case style.JustifyAround:
		if remainingMain < 0 {
			startMainOffset = 0
			itemSpacing = b.mainGap
		} else if len(line.Items) > 0 {
			itemSpacing = b.mainGap + remainingMain/len(line.Items)
			startMainOffset = (itemSpacing - b.mainGap) / 2
		}
	case style.JustifyEvenly:
		if remainingMain < 0 {
			startMainOffset = 0
			itemSpacing = b.mainGap
		} else if len(line.Items) > 0 {
			itemSpacing = b.mainGap + remainingMain/(len(line.Items)+1)
			startMainOffset = itemSpacing - b.mainGap
		}
	}

	currentMainOffset := startMainOffset
	for i, item := range line.Items {
		childMargin := item.Node.Style().Margin

		// Add the "before" margin on the main axis.
		if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
			currentMainOffset += childMargin.Left
		} else {
			currentMainOffset += childMargin.Top
		}

		// Store main-axis offset in item.Offset
		if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
			item.Offset.X = currentMainOffset
		} else {
			item.Offset.Y = currentMainOffset
		}

		currentMainOffset += b.geom.MainSize(item.Fragment.Size)

		// Add the "after" margin on the main axis.
		if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
			currentMainOffset += childMargin.Right
		} else {
			currentMainOffset += childMargin.Bottom
		}

		if i < len(line.Items)-1 {
			currentMainOffset += itemSpacing
		}
	}
}

// AlignCrossAxis handles cross-axis alignment (align-content and align-items).
func (b *FlexLineBuilder) AlignCrossAxis(containerCrossSize int, alignContent style.Align, alignItems style.Align) {
	// 1. Calculate totalSumLineCross
	totalSumLineCross := 0
	for i, line := range b.lines {
		totalSumLineCross += line.CrossSize
		if i > 0 {
			totalSumLineCross += b.crossGap
		}
	}

	// 2. align-content: stretch
	extraCross := containerCrossSize - totalSumLineCross
	if extraCross > 0 && len(b.lines) > 0 && alignContent == style.AlignStretch {
		perLineExtra := extraCross / len(b.lines)
		for _, line := range b.lines {
			line.CrossSize += perLineExtra
		}
	}

	currentCrossOffset := 0
	for _, line := range b.lines {
		for _, item := range line.Items {
			childMargin := item.Node.Style().Margin

			// align-items math
			itemAvailableCross := line.CrossSize
			if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
				itemAvailableCross -= childMargin.Top + childMargin.Bottom
			} else {
				itemAvailableCross -= childMargin.Left + childMargin.Right
			}

			itemActualCross := b.geom.CrossSize(item.Fragment.Size)
			crossOffset := 0
			switch alignItems {
			case style.AlignStart:
				crossOffset = 0
			case style.AlignEnd:
				crossOffset = itemAvailableCross - itemActualCross
			case style.AlignCenter:
				crossOffset = (itemAvailableCross - itemActualCross) / 2
			case style.AlignStretch:
				crossOffset = 0
			}

			// Add the "before" margin on the cross axis.
			if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
				crossOffset += childMargin.Top
			} else {
				crossOffset += childMargin.Left
			}

			// Store cross-axis offset in item.Offset
			if b.geom.direction == style.FlexRow || b.geom.direction == style.FlexRowReverse {
				item.Offset.Y = currentCrossOffset + crossOffset
			} else {
				item.Offset.X = currentCrossOffset + crossOffset
			}
		}
		currentCrossOffset += line.CrossSize + b.crossGap
	}
}
