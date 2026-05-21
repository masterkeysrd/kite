package layout

import (
	"strings"
	"unicode"

	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

var defaultShaper = text.NewShaper(0)

// LineBox represents a single horizontal line of positioned inline fragments.
type LineBox struct {
	Size     Size
	Children []FragmentLink
}

func (lb *LineBox) ToFragment() *Fragment {
	return &Fragment{
		Size:     lb.Size,
		Children: lb.Children,
	}
}

func isSpaceCluster(c text.Cluster) bool {
	if len(c.Bytes) != 1 {
		return false
	}
	r := rune(c.Bytes[0])
	return unicode.IsSpace(r)
}

// InlineItemType identifies the role of an item in the inline flow.
type InlineItemType uint8

const (
	// InlineText represents a contiguous run of shaped text.
	InlineText InlineItemType = iota
	// InlineAtomic represents an opaque block element (inline-block).
	InlineAtomic
	// InlineOpenTag marks the start of a styled range.
	InlineOpenTag
	// InlineCloseTag marks the end of a styled range.
	InlineCloseTag
)

// InlineItem is the atomic unit of the flattened inline representation.
type InlineItem struct {
	Type InlineItemType

	// Text contains the shaped clusters for InlineText items.
	Text []text.Cluster

	// Node is the layout node that generated this item.
	// For InlineAtomic, this is the element itself.
	// For Tags, this is the element providing the styles.
	Node Node

	// ParentNode is the containing inline element for text items.
	ParentNode Node
}

// InlineItemsBuilder flattens a tree of layout nodes into a 1D array of InlineItems.
type InlineItemsBuilder struct {
	shaper *text.Shaper
	items  []InlineItem

	// Current state for whitespace collapsing
	lastWasSpace bool

	// Stack of inline parents to associate text with correct styles
	parentStack []Node

	// The block container that established this IFC
	blockContainer Node
}

func NewInlineItemsBuilder(shaper *text.Shaper, block Node) *InlineItemsBuilder {
	return &InlineItemsBuilder{
		shaper:         shaper,
		blockContainer: block,
	}
}

type textSource interface {
	Data() string
}

func (b *InlineItemsBuilder) currentParent() Node {
	if len(b.parentStack) == 0 {
		return nil
	}
	return b.parentStack[len(b.parentStack)-1]
}

func (b *InlineItemsBuilder) Build(root Node) []InlineItem {
	b.items = nil
	b.lastWasSpace = true // Start by assuming a space to collapse leading whitespace
	b.parentStack = nil
	b.collect(root)
	return b.items
}

func (b *InlineItemsBuilder) collect(node Node) {
	// If it's a text node, handle it.
	if ts, ok := node.LogicalNode().(textSource); ok {
		b.collectText(ts.Data(), node)
		return
	}

	comp := node.Style()
	if comp == nil {
		return
	}

	// Determine if it's an atomic inline or a container.
	if comp.Display == style.DisplayInlineBlock || comp.Display == style.DisplayInlineFlex {
		b.items = append(b.items, InlineItem{
			Type:       InlineAtomic,
			Node:       node,
			ParentNode: b.currentParent(), // Back to actual inline parent
		})
		b.lastWasSpace = false
		return
	}

	// For DisplayInline, it's a container.
	b.items = append(b.items, InlineItem{
		Type: InlineOpenTag,
		Node: node,
	})
	b.parentStack = append(b.parentStack, node)

	for child := range node.LayoutChildren() {
		b.collect(child)
	}

	b.parentStack = b.parentStack[:len(b.parentStack)-1]
	b.items = append(b.items, InlineItem{
		Type: InlineCloseTag,
		Node: node,
	})
}

func (b *InlineItemsBuilder) collectText(data string, node Node) {
	comp := node.Style()
	ws := style.WhiteSpaceNormal
	if comp != nil {
		ws = comp.WhiteSpace
	}

	var collapsed strings.Builder
	for _, r := range data {
		if ws == style.WhiteSpacePre || ws == style.WhiteSpacePreWrap {
			collapsed.WriteRune(r)
			continue
		}

		// Collapsing logic (Normal, NoWrap)
		if unicode.IsSpace(r) {
			if !b.lastWasSpace {
				collapsed.WriteRune(' ')
				b.lastWasSpace = true
			}
		} else {
			collapsed.WriteRune(r)
			b.lastWasSpace = false
		}
	}

	textStr := collapsed.String()
	if textStr == "" {
		return
	}

	clusters := b.shaper.Shape(textStr)
	b.items = append(b.items, InlineItem{
		Type:       InlineText,
		Text:       clusters,
		Node:       node,
		ParentNode: b.currentParent(), // Back to actual inline parent
	})
}

// LineBreaker packs InlineItems into physical lines.
type LineBreaker struct {
	items         []InlineItem
	width         int // Available inline size
	textAlign     style.TextAlign
	verticalAlign style.Align

	currentIndex int // current item index
	clusterIndex int // current cluster index within InlineText

	hadForcedBreakAtEnd bool
}

func NewLineBreaker(items []InlineItem, width int, textAlign style.TextAlign, verticalAlign style.Align) *LineBreaker {
	return &LineBreaker{
		items:         items,
		width:         width,
		textAlign:     textAlign,
		verticalAlign: verticalAlign,
	}
}

func (l *LineBreaker) NextLine() (*LineBox, bool) {
	if l.currentIndex >= len(l.items) {
		if l.hadForcedBreakAtEnd {
			l.hadForcedBreakAtEnd = false
			return &LineBox{Size: Size{Width: 0, Height: 1}}, true
		}
		return nil, false
	}

	line := &LineBox{}
	currentX := 0
	lineHeight := 1 // Minimum height of a line

	// Temporary storage for items on this line before building FragmentLinks
	type lineItem struct {
		node   Node
		parent Node // To inherit styles for text
		text   []text.Cluster
		frag   *Fragment
		width  int
		height int
	}
	var lineItems []lineItem

	for l.currentIndex < len(l.items) {
		item := l.items[l.currentIndex]

		switch item.Type {
		case InlineOpenTag, InlineCloseTag:
			l.currentIndex++
			continue

		case InlineAtomic:
			childAlgo := NewAlgorithm(item.Node, ConstraintSpace{
				AvailableSize: Size{Width: l.width, Height: 1000},
			})
			frag := childAlgo.Layout()
			itemWidth := frag.Size.Width
			itemHeight := frag.Size.Height

			if currentX > 0 && currentX+itemWidth > l.width {
				goto lineEnded
			}

			lineItems = append(lineItems, lineItem{
				node:   item.Node,
				parent: item.ParentNode,
				frag:   frag,
				width:  itemWidth,
				height: itemHeight,
			})
			currentX += itemWidth
			lineHeight = max(lineHeight, itemHeight)
			l.currentIndex++

		case InlineText:
			remainingClusters := item.Text[l.clusterIndex:]

			// Collapse leading spaces at start of line
			if currentX == 0 {
				comp := item.Node.Style()
				ws := style.WhiteSpaceNormal
				if comp != nil {
					ws = comp.WhiteSpace
				}
				if ws == style.WhiteSpaceNormal || ws == style.WhiteSpaceNoWrap {
					for len(remainingClusters) > 0 && isSpaceCluster(remainingClusters[0]) {
						remainingClusters = remainingClusters[1:]
						l.clusterIndex++
					}
					if len(remainingClusters) == 0 {
						l.currentIndex++
						l.clusterIndex = 0
						continue
					}
				}
			}

			count, tookWidth, forceBreak := l.findFittingClusters(item, remainingClusters, l.width-currentX)

			if count > 0 {
				lineItems = append(lineItems, lineItem{
					node:   item.Node,
					parent: item.ParentNode,
					text:   remainingClusters[:count],
					width:  tookWidth,
					height: 1,
				})
				currentX += tookWidth
				l.clusterIndex += count

				if l.clusterIndex >= len(item.Text) {
					l.currentIndex++
					l.clusterIndex = 0
				}

				if forceBreak {
					if l.currentIndex >= len(l.items) && l.clusterIndex == 0 {
						l.hadForcedBreakAtEnd = true
					}
					goto lineEnded
				}
			} else {
				if currentX > 0 {
					goto lineEnded
				} else {
					if len(remainingClusters) > 0 {
						cWidth := remainingClusters[0].CellWidth
						lineItems = append(lineItems, lineItem{
							node:   item.Node,
							parent: item.ParentNode,
							text:   remainingClusters[:1],
							width:  cWidth,
							height: 1,
						})
						currentX += cWidth
						l.clusterIndex++
						if l.clusterIndex >= len(item.Text) {
							l.currentIndex++
							l.clusterIndex = 0
						}
					} else {
						l.currentIndex++
						l.clusterIndex = 0
					}
					goto lineEnded
				}
			}
		}
	}

lineEnded:
	line.Size = Size{Width: currentX, Height: lineHeight}

	// Horizontal Alignment (text-align)
	var startX int
	remainingSpace := l.width - currentX
	switch l.textAlign {
	case style.TextAlignRight:
		startX = remainingSpace
	case style.TextAlignCenter:
		startX = remainingSpace / 2
	default:
		startX = 0
	}

	offsetX := startX
	for _, li := range lineItems {
		var frag *Fragment
		if li.frag != nil {
			frag = li.frag
		} else {
			frag = &Fragment{
				Size:       Size{Width: li.width, Height: li.height},
				Node:       li.node,
				Text:       li.text,
				ParentNode: li.parent,
			}
		}

		// Vertical Alignment (vertical-align)
		// We use the container's verticalAlign unless the item has AlignSelf.
		offsetY := 0
		itemAlign := l.verticalAlign

		// Check the item itself, then its parent inline (for text fragments)
		if li.node != nil && li.node.Style() != nil && li.node.Style().AlignSelf != style.AlignStart {
			itemAlign = li.node.Style().AlignSelf
		} else if li.parent != nil && li.parent.Style() != nil && li.parent.Style().AlignSelf != style.AlignStart {
			itemAlign = li.parent.Style().AlignSelf
		}

		switch itemAlign {
		case style.AlignCenter, style.AlignStretch:
			offsetY = (lineHeight - li.height) / 2
		case style.AlignEnd:
			offsetY = lineHeight - li.height
		case style.AlignStart:
			offsetY = 0
		default:
			offsetY = 0
		}

		line.Children = append(line.Children, FragmentLink{
			Offset:   Point{X: offsetX, Y: offsetY},
			Fragment: frag,
		})
		offsetX += li.width
	}

	return line, true
}

func (l *LineBreaker) findFittingClusters(item InlineItem, clusters []text.Cluster, availableWidth int) (count int, width int, forceBreak bool) {
	comp := item.Node.Style()
	ws := style.WhiteSpaceNormal
	if comp != nil {
		ws = comp.WhiteSpace
	}

	canWrap := ws == style.WhiteSpaceNormal || ws == style.WhiteSpacePreWrap

	currentWidth := 0
	lastBreakOp := -1
	lastBreakWidth := 0

	for i, c := range clusters {
		if canWrap && (c.BreakClass == text.BreakSoft || c.BreakClass == text.BreakAnywhere) {
			lastBreakOp = i
			lastBreakWidth = currentWidth
		}

		if canWrap && currentWidth+c.CellWidth > availableWidth {
			if lastBreakOp != -1 {
				return lastBreakOp, lastBreakWidth, false
			}
			// No break opportunity found, we must overflow or break-word.
			// If we've already taken some clusters, break here (emergency break).
			if i > 0 {
				return i, currentWidth, false
			}
			// Even the first cluster doesn't fit — return 0 to trigger
			// at-least-one-cluster break in caller.
			return 0, 0, false
		}

		currentWidth += c.CellWidth

		if c.BreakClass == text.BreakMandatory {
			return i + 1, currentWidth, true
		}
	}

	return len(clusters), currentWidth, false
}

func ComputeInlineMinMaxSizes(items []InlineItem) MinMaxSizes {
	var result MinMaxSizes
	currentLineMax := 0

	for _, item := range items {
		switch item.Type {
		case InlineText:
			comp := item.Node.Style()
			ws := style.WhiteSpaceNormal
			if comp != nil {
				ws = comp.WhiteSpace
			}
			canWrap := ws == style.WhiteSpaceNormal || ws == style.WhiteSpacePreWrap

			// For max-content, we just sum everything up (ignoring soft wraps).
			// For min-content, we find the longest unbreakable run.
			unbreakableRun := 0
			for _, c := range item.Text {
				if canWrap && (c.BreakClass == text.BreakSoft || c.BreakClass == text.BreakMandatory || c.BreakClass == text.BreakAnywhere) {
					result.Min = max(result.Min, unbreakableRun)
					unbreakableRun = 0
				}
				if c.BreakClass != text.BreakMandatory {
					unbreakableRun += c.CellWidth
					currentLineMax += c.CellWidth
				} else {
					result.Max = max(result.Max, currentLineMax)
					currentLineMax = 0
					if !canWrap {
						// Even if we can't soft wrap, mandatory breaks still reset.
						result.Min = max(result.Min, unbreakableRun)
						unbreakableRun = 0
					}
				}
			}
			result.Min = max(result.Min, unbreakableRun)

		case InlineAtomic:
			// Call intrinsic helper on the atomic node.
			childMinMax := IntrinsicMinMaxSizes(item.Node)
			result.Min = max(result.Min, childMinMax.Min)
			currentLineMax += childMinMax.Max
		}
	}

	result.Max = max(result.Max, currentLineMax)
	return result
}
