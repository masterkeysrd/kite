package dom

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
)

type Range struct {
	doc            dom.Document
	startContainer dom.Node
	startOffset    int
	endContainer   dom.Node
	endOffset      int
}

var _ dom.Range = (*Range)(nil)

func (r *Range) StartContainer() dom.Node { return r.startContainer }
func (r *Range) StartOffset() int         { return r.startOffset }
func (r *Range) EndContainer() dom.Node   { return r.endContainer }
func (r *Range) EndOffset() int           { return r.endOffset }

func (r *Range) SetStart(node dom.Node, offset int) {
	r.validate(node, offset)
	r.startContainer = node
	r.startOffset = offset
	// Ensure start <= end if containers are the same.
	// For different containers, we rely on the caller to maintain order or
	// a future implementation of document-order comparison.
	if r.startContainer == r.endContainer && r.startOffset > r.endOffset {
		r.endOffset = r.startOffset
	}
	r.notifyChange()
}

func (r *Range) SetEnd(node dom.Node, offset int) {
	r.validate(node, offset)
	r.endContainer = node
	r.endOffset = offset
	if r.startContainer == r.endContainer && r.endOffset < r.startOffset {
		r.startOffset = r.endOffset
	}
	r.notifyChange()
}

func (r *Range) Collapse(toStart bool) {
	if toStart {
		r.endContainer = r.startContainer
		r.endOffset = r.startOffset
	} else {
		r.startContainer = r.endContainer
		r.startOffset = r.endOffset
	}
	r.notifyChange()
}

func (r *Range) IsCollapsed() bool {
	return r.startContainer == r.endContainer && r.startOffset == r.endOffset
}

func (r *Range) validate(node dom.Node, offset int) {
	if node == nil {
		panic("dom: range node cannot be nil")
	}
	if node.OwnerDocument() != r.doc {
		panic("dom: range node must belong to the same document")
	}
	if offset < 0 {
		panic("dom: range offset cannot be negative")
	}

	if t, ok := node.(dom.TextNode); ok {
		count := utf8.RuneCountInString(t.Data())
		if offset > count {
			panic(fmt.Sprintf("dom: range offset %d exceeds text length %d", offset, count))
		}
	} else {
		// ADR-009: Use LayoutChildren for validation so that UA shadow nodes
		// (used by text controls for selection) are correctly counted.
		count := 0
		for range LayoutChildren(node) {
			count++
		}
		if offset > count {
			panic(fmt.Sprintf("dom: range offset %d exceeds child count %d", offset, count))
		}
	}
}

func (r *Range) notifyChange() {
	if r.doc == nil {
		return
	}
	if s, ok := r.doc.Selection().(*Selection); ok {
		s.changed()
	}
}

func (r *Range) String() string {
	if r.startContainer == nil || r.endContainer == nil {
		return ""
	}

	var sb strings.Builder

	// Helper to write text including \n for <br>.
	var writeText func(dom.Node)
	writeText = func(n dom.Node) {
		if t, ok := n.(dom.TextNode); ok {
			sb.WriteString(t.Data())
		} else if el, ok := n.(interface{ IsBr() bool }); ok && el.IsBr() {
			sb.WriteString("\n")
		}
		for child := range LayoutChildren(n) {
			writeText(child)
		}
	}

	// 1. Same-container fast path.
	if r.startContainer == r.endContainer {
		if t, ok := r.startContainer.(dom.TextNode); ok {
			runes := []rune(t.Data())
			start, end := r.startOffset, r.endOffset
			if start < 0 {
				start = 0
			}
			if end > len(runes) {
				end = len(runes)
			}
			if start >= end {
				return ""
			}
			return string(runes[start:end])
		} else {
			// Element container.
			idx := 0
			for child := range LayoutChildren(r.startContainer) {
				if idx >= r.startOffset && idx < r.endOffset {
					writeText(child)
				}
				idx++
			}
			return sb.String()
		}
	}

	// 2. Slow path: different containers.
	// We use a strictly one-pass, pre-order walk to accumulate text between
	// the two boundary points.
	started := false
	var walk func(dom.Node) bool
	walk = func(n dom.Node) bool {
		isStart := n == r.startContainer
		isEnd := n == r.endContainer

		// 2a. Handle Start Container.
		if !started && isStart {
			started = true
			if t, ok := n.(dom.TextNode); ok {
				runes := []rune(t.Data())
				if r.startOffset < len(runes) {
					sb.WriteString(string(runes[r.startOffset:]))
				}
				// Handled start; walk() will return true and continue.
			} else {
				idx := 0
				for child := range LayoutChildren(n) {
					if idx >= r.startOffset {
						if !walk(child) {
							return false
						}
					}
					idx++
				}
				return true // Done with this subtree.
			}
		}

		// 2b. Handle End Container.
		if isEnd {
			if t, ok := n.(dom.TextNode); ok {
				if started {
					runes := []rune(t.Data())
					end := r.endOffset
					if end > len(runes) {
						end = len(runes)
					}
					if end > 0 {
						sb.WriteString(string(runes[:end]))
					}
				}
				return false // Stop walking altogether.
			} else {
				idx := 0
				for child := range LayoutChildren(n) {
					if started && idx >= r.endOffset {
						return false // Stop.
					}
					if !walk(child) {
						return false
					}
					idx++
				}
				return false // Finished end container.
			}
		}

		// 2c. Contribution of nodes fully inside the range.
		if started && !isStart {
			if t, ok := n.(dom.TextNode); ok {
				sb.WriteString(t.Data())
			} else if el, ok := n.(interface{ IsBr() bool }); ok && el.IsBr() {
				sb.WriteString("\n")
			}
		}

		// 2d. Recurse (unless already handled by specific logic above).
		if !isStart && !isEnd {
			for child := range LayoutChildren(n) {
				if !walk(child) {
					return false
				}
			}
		}

		return true
	}

	walk(r.doc)
	return sb.String()
}

type Selection struct {
	doc    dom.Document
	ranges []*Range
}

var _ dom.Selection = (*Selection)(nil)

func (s *Selection) RangeCount() int {
	return len(s.ranges)
}

func (s *Selection) GetRangeAt(index int) dom.Range {
	if index < 0 || index >= len(s.ranges) {
		return nil
	}
	return s.ranges[index]
}

func (s *Selection) AddRange(r dom.Range) {
	if r == nil {
		return
	}
	if rng, ok := r.(*Range); ok {
		s.ranges = append(s.ranges, rng)
	}
	s.changed()
}

func (s *Selection) RemoveAllRanges() {
	if len(s.ranges) == 0 {
		return
	}
	s.ranges = nil
	s.changed()
}

func (s *Selection) String() string {
	var sb strings.Builder
	for _, r := range s.ranges {
		sb.WriteString(r.String())
	}
	return sb.String()
}

func (s *Selection) changed() {
	if s.doc == nil {
		return
	}
	if d := AsDirty(s.doc); d != nil {
		d.MarkNeedsSync()
	}

	s.doc.DispatchToTarget(event.NewBaseEvent(event.EventSelectionChange, s.doc, false))
}

func (s *Selection) NewRange() *Range {
	return &Range{doc: s.doc}
}

// Internal helper for document to create selection
func newSelection(doc dom.Document) *Selection {
	return &Selection{doc: doc}
}
