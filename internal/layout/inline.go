package layout

import (
	"sync"

	geometry "github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout/text"
	"github.com/masterkeysrd/kite/style"
)

var defaultShaper = text.NewShaper(0)

var inlineBuilderPool = sync.Pool{
	New: func() any {
		return &InlineItemsBuilder{
			items:       make([]InlineItem, 0, 32),
			parentStack: make([]Node, 0, 8),
		}
	},
}

// AcquireInlineItemsBuilder gets a builder from the pool and initializes it.
func AcquireInlineItemsBuilder(shaper *text.Shaper, block Node) *InlineItemsBuilder {
	b := inlineBuilderPool.Get().(*InlineItemsBuilder)
	b.shaper = shaper
	b.blockContainer = block
	b.Reset()
	return b
}

// ReleaseInlineItemsBuilder returns a builder to the pool.
func ReleaseInlineItemsBuilder(b *InlineItemsBuilder) {
	b.shaper = nil
	b.blockContainer = nil
	for i := range b.items {
		b.items[i].Node = nil
		b.items[i].ParentNode = nil
		b.items[i].Text = nil
	}
	b.items = b.items[:0]
	for i := range b.parentStack {
		b.parentStack[i] = nil
	}
	b.parentStack = b.parentStack[:0]
	inlineBuilderPool.Put(b)
}

// LineBox represents a single horizontal line of positioned inline fragments.
type LineBox struct {
	Size     geometry.Size
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
	b := c.Bytes[0]
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\v' || b == '\f'
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
	// InlineBr represents a <br> element that forces a mandatory line break.
	// It contributes zero bytes and zero visual width but terminates the
	// current line immediately, similar to a \n cluster with BreakMandatory.
	InlineBr
	// InlineBrPlaceholder represents a trailing placeholder <br> that forces
	// a line break (so the textarea always has height for an empty last line)
	// but contributes ZERO bytes to the cursor byte-offset model. This maps
	// to the browser model where the placeholder break is not part of the
	// buffer value.
	InlineBrPlaceholder
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
		items:          make([]InlineItem, 0, 32),
		parentStack:    make([]Node, 0, 8),
	}
}

type textSource interface {
	Data() string
}

// brElement is implemented by elements that represent a mandatory line break
// (e.g. <br>). The IFC builder emits an InlineBr item for such elements.
type brElement interface {
	IsBr() bool
}

// brPlaceholderElement is implemented by elements that represent a trailing
// placeholder line break. Like brElement it forces a new line so the textarea
// always has height for an empty last line, but it emits NO byte to the cursor
// offset model — matching the browser's <br id="placeholder"> convention.
type brPlaceholderElement interface {
	IsPlaceholderBr() bool
}

func (b *InlineItemsBuilder) currentParent() Node {
	if len(b.parentStack) == 0 {
		return nil
	}
	return b.parentStack[len(b.parentStack)-1]
}

func (b *InlineItemsBuilder) Reset() {
	b.items = b.items[:0]
	b.lastWasSpace = true
	b.parentStack = b.parentStack[:0]
}

func (b *InlineItemsBuilder) Build(root Node) []InlineItem {
	b.Reset()
	b.collect(root)
	return b.items
}

func (b *InlineItemsBuilder) collect(node Node) {
	// If it's a text node, handle it.
	if ts, ok := node.LogicalNode().(textSource); ok {
		b.collectText(ts.Data(), node)
		return
	}

	// If it's a placeholder <br>, emit a zero-byte placeholder break item.
	if br, ok := node.LogicalNode().(brPlaceholderElement); ok && br.IsPlaceholderBr() {
		b.items = append(b.items, InlineItem{Type: InlineBrPlaceholder})
		return
	}

	// If it's a content <br>, emit a mandatory line-break item with a '\n' byte.
	if br, ok := node.LogicalNode().(brElement); ok && br.IsBr() {
		b.items = append(b.items, InlineItem{
			Type:       InlineBr,
			Node:       node,
			ParentNode: b.currentParent(),
		})
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

	for child := node.FirstLayoutChild(); child != nil; child = node.NextLayoutSibling(child) {
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

	// For pre / pre-wrap: shape the original bytes with no collapsing.
	if ws == style.WhiteSpacePre || ws == style.WhiteSpacePreWrap {
		clusters := b.shaper.Shape(data)
		if len(clusters) == 0 {
			return
		}
		b.lastWasSpace = isSpaceCluster(clusters[len(clusters)-1])
		b.items = append(b.items, InlineItem{
			Type:       InlineText,
			Text:       clusters,
			Node:       node,
			ParentNode: b.currentParent(),
		})
		return
	}

	// For Normal / NoWrap: shape the ORIGINAL string, then mark clusters that
	// CSS whitespace collapsing would remove or merge as CellWidth=0.
	//
	// This preserves the byte-offset invariant that cursor.FromTextFragment
	// depends on: Cluster.Bytes must reference original string bytes so that
	// summing len(c.Bytes) across all clusters equals len(data), keeping the
	// buffer's byteOffset in sync with the fragment's byte count.
	//
	// Collapsing rules applied here (CSS §white-space: normal / nowrap):
	//   • A run of whitespace is represented as a single visible space; any
	//     additional spaces in that run become CellWidth=0 (invisible).
	//   • Leading whitespace (when b.lastWasSpace is true on entry) is
	//     entirely invisible (every space → CellWidth=0).
	//
	// IMPORTANT: the shaper returns a cached slice shared across all callers.
	// We must COPY the slice before mutating any CellWidth field to avoid
	// poisoning the cache for other callers (e.g. a pre-whitespace text node
	// that renders the same string without collapsing).
	shaped := b.shaper.Shape(data)
	if len(shaped) == 0 {
		return
	}

	// Determine whether any clusters need collapsing before copying.
	// Also determine whether a cross-item word-boundary break opportunity needs
	// to be injected. The shaper operates per-text-node and cannot see across
	// node boundaries, so a space at the end of one text node does not cause the
	// first cluster of the next to be marked BreakSoft. We correct that here
	// using the builder's lastWasSpace flag (copy-on-write to protect the cache).
	needsCopy := false
	lws := b.lastWasSpace

	// crossItemBreakIdx is the index of the first non-space cluster that should
	// be promoted to BreakSoft due to a trailing space on the previous text node.
	// -1 means no injection is needed.
	crossItemBreakIdx := -1
	if b.lastWasSpace && (ws == style.WhiteSpaceNormal || ws == style.WhiteSpaceNoWrap) {
		for i, c := range shaped {
			if !isSpaceCluster(c) {
				if c.BreakClass == text.BreakNone {
					// This cluster is a word-boundary break point because the
					// previous item ended with a space. Record it for fixup.
					crossItemBreakIdx = i
					needsCopy = true
				}
				break
			}
		}
	}

	for _, c := range shaped {
		if isSpaceCluster(c) {
			if lws {
				needsCopy = true
				break
			}
			lws = true
		} else {
			lws = false
		}
	}

	var clusters []text.Cluster
	if needsCopy {
		clusters = make([]text.Cluster, len(shaped))
		copy(clusters, shaped)
	} else {
		clusters = shaped
	}

	// Apply the cross-item word-boundary break injection.
	if crossItemBreakIdx >= 0 {
		clusters[crossItemBreakIdx].BreakClass = text.BreakSoft
	}

	for i := range clusters {
		c := &clusters[i]
		if isSpaceCluster(*c) {
			if b.lastWasSpace {
				// Collapsed: keep bytes but make visually invisible.
				c.CellWidth = 0
			} else {
				// First space in a run: keep as a visible space.
				b.lastWasSpace = true
				// CellWidth remains as shaped (1 for ASCII space).
			}
		} else {
			b.lastWasSpace = false
		}
	}

	b.items = append(b.items, InlineItem{
		Type:       InlineText,
		Text:       clusters,
		Node:       node,
		ParentNode: b.currentParent(),
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

func (l *LineBreaker) NextLine(ctx *Context) (*LineBox, bool) {
	if l.currentIndex >= len(l.items) {
		if l.hadForcedBreakAtEnd {
			l.hadForcedBreakAtEnd = false
			return &LineBox{Size: geometry.Size{Width: 0, Height: 1}}, true
		}
		return nil, false
	}
	defer ctx.Begin("Layout(IFC):NextLine")()

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

		case InlineBr:
			// Content <br>: forces an immediate line break and emits a virtual
			// '\n' cluster so cursor.FromTextFragment counts the byte.
			lineItems = append(lineItems, lineItem{
				text: []text.Cluster{{
					Bytes:      []byte{'\n'},
					CellWidth:  0,
					BreakClass: text.BreakMandatory,
				}},
				width:  0,
				height: 1,
			})
			l.currentIndex++
			goto lineEnded

		case InlineBrPlaceholder:
			// Placeholder <br>: creates exactly one empty trailing line for
			// cursor height without contributing any bytes. It always appears
			// as the last item. Emit the current (empty) line directly via
			// lineEnded; no hadForcedBreakAtEnd is needed.
			l.currentIndex++
			goto lineEnded

		case InlineAtomic:
			childSpace := ConstraintSpace{
				AvailableSize: geometry.Size{Width: l.width, Height: 1000},
			}
			childAlgo := GetAlgorithm(item.Node)
			frag := childAlgo.Layout(ctx, item.Node, childSpace)
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

			// Suppress leading visible spaces at the start of a wrapped line.
			//
			// Spaces that collectText already collapsed have CellWidth=0 and are
			// present in the slice for byte-offset tracking — they need no action.
			// A space with CellWidth>0 that arrives at the start of a new wrapped
			// line must become invisible; we copy it before zeroing to avoid
			// mutating the shaper's cached cluster slice.
			if currentX == 0 {
				comp := item.Node.Style()
				ws := style.WhiteSpaceNormal
				if comp != nil {
					ws = comp.WhiteSpace
				}
				if ws == style.WhiteSpaceNormal || ws == style.WhiteSpaceNoWrap {
					for i := range remainingClusters {
						if !isSpaceCluster(remainingClusters[i]) {
							break
						}
						if remainingClusters[i].CellWidth > 0 {
							// Copy-on-write: replace the shared slice with a fresh
							// copy before mutating, so the shaper cache is not poisoned.
							copied := make([]text.Cluster, len(remainingClusters))
							copy(copied, remainingClusters)
							remainingClusters = copied
							remainingClusters[i].CellWidth = 0
						}
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
					// Even the first cluster doesn't fit in l.width.
					// If OverflowWrap allows it, take one cluster anyway to ensure forward progress.
					ow := style.OverflowWrapNormal
					if item.Node.Style() != nil {
						ow = item.Node.Style().OverflowWrap
					}

					if len(remainingClusters) > 0 && (ow == style.OverflowWrapBreakWord || ow == style.OverflowWrapAnywhere) {
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
						goto lineEnded
					}

					// OverflowWrapNormal or no clusters: take everything and overflow.
					tookWidth := 0
					for _, c := range remainingClusters {
						tookWidth += c.CellWidth
					}
					lineItems = append(lineItems, lineItem{
						node:   item.Node,
						parent: item.ParentNode,
						text:   remainingClusters,
						width:  tookWidth,
						height: 1,
					})
					currentX += tookWidth
					l.currentIndex++
					l.clusterIndex = 0
					goto lineEnded
				}
			}
		}
	}

lineEnded:
	line.Size = geometry.Size{Width: currentX, Height: lineHeight}

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
				Size:       geometry.Size{Width: li.width, Height: li.height},
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
		if li.node != nil && li.node.Style() != nil && li.node.Style().AlignSelf != style.AlignAuto {
			itemAlign = li.node.Style().AlignSelf
		} else if li.parent != nil && li.parent.Style() != nil && li.parent.Style().AlignSelf != style.AlignAuto {
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
			Offset:   geometry.Point{X: offsetX, Y: offsetY},
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

	if canWrap {
		for i, c := range clusters {
			if c.BreakClass == text.BreakSoft || c.BreakClass == text.BreakAnywhere {
				lastBreakOp = i
				lastBreakWidth = currentWidth
			}

			if currentWidth+c.CellWidth > availableWidth {
				if lastBreakOp > 0 || (lastBreakOp == 0 && availableWidth < l.width) {
					return lastBreakOp, lastBreakWidth, false
				}

				// No break opportunity found, check OverflowWrap.
				ow := style.OverflowWrapNormal
				if comp != nil {
					ow = comp.OverflowWrap
				}

				if ow == style.OverflowWrapBreakWord || ow == style.OverflowWrapAnywhere {
					// Emergency break: if we've already taken some clusters, break here.
					if i > 0 {
						return i, currentWidth, false
					}

					// Even the first cluster doesn't fit in availableWidth.
					// If we are at the START of the line, we MUST take one cluster
					// to make progress.
					if availableWidth >= l.width {
						return 1, c.CellWidth, false
					}

					// Not at start of line: return 0 to trigger line-end and retry
					// at start of next line.
					return 0, 0, false
				}

				// OverflowWrapNormal: never emergency-break. Continue until next
				// break opportunity (mandatory or end of run).
			}

			currentWidth += c.CellWidth

			if c.BreakClass == text.BreakMandatory {
				return i + 1, currentWidth, true
			}
		}
	} else {
		for i, c := range clusters {
			currentWidth += c.CellWidth

			if c.BreakClass == text.BreakMandatory {
				return i + 1, currentWidth, true
			}
		}
	}

	return len(clusters), currentWidth, false
}

func ComputeInlineMinMaxSizes(ctx *Context, items []InlineItem) MinMaxSizes {
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
			if canWrap {
				for _, c := range item.Text {
					if c.BreakClass == text.BreakSoft || c.BreakClass == text.BreakMandatory || c.BreakClass == text.BreakAnywhere {
						result.Min = max(result.Min, unbreakableRun)
						unbreakableRun = 0
					}
					if c.BreakClass != text.BreakMandatory {
						unbreakableRun += c.CellWidth
						currentLineMax += c.CellWidth
					} else {
						result.Max = max(result.Max, currentLineMax)
						currentLineMax = 0
					}
				}
			} else {
				for _, c := range item.Text {
					if c.BreakClass != text.BreakMandatory {
						unbreakableRun += c.CellWidth
						currentLineMax += c.CellWidth
					} else {
						result.Max = max(result.Max, currentLineMax)
						currentLineMax = 0
						result.Min = max(result.Min, unbreakableRun)
						unbreakableRun = 0
					}
				}
			}
			result.Min = max(result.Min, unbreakableRun)

		case InlineAtomic:
			// Call intrinsic helper on the atomic node.
			childMinMax := IntrinsicMinMaxSizes(ctx, item.Node)
			result.Min = max(result.Min, childMinMax.Min)
			currentLineMax += childMinMax.Max

		case InlineBr:
			// <br> forces a mandatory break: end the current line.
			result.Max = max(result.Max, currentLineMax)
			currentLineMax = 0

		case InlineBrPlaceholder:
			// Placeholder <br>: same line-reset as a content br for sizing
			// purposes, but contributes no bytes.
			result.Max = max(result.Max, currentLineMax)
			currentLineMax = 0
		}
	}

	result.Max = max(result.Max, currentLineMax)
	return result
}
