package dom

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/masterkeysrd/kite/event"
)

type rangeImpl struct {
	doc            Document
	startContainer Node
	startOffset    int
	endContainer   Node
	endOffset      int
}

var _ Range = (*rangeImpl)(nil)

func (r *rangeImpl) StartContainer() Node { return r.startContainer }
func (r *rangeImpl) StartOffset() int     { return r.startOffset }
func (r *rangeImpl) EndContainer() Node   { return r.endContainer }
func (r *rangeImpl) EndOffset() int       { return r.endOffset }

func (r *rangeImpl) SetStart(node Node, offset int) {
	r.validate(node, offset)
	r.startContainer = node
	r.startOffset = offset
	// Ensure start <= end if containers are the same
	if r.startContainer == r.endContainer && r.startOffset > r.endOffset {
		r.endOffset = r.startOffset
	}
	r.notifyChange()
}

func (r *rangeImpl) SetEnd(node Node, offset int) {
	r.validate(node, offset)
	r.endContainer = node
	r.endOffset = offset
	// Ensure start <= end if containers are the same
	if r.startContainer == r.endContainer && r.endOffset < r.startOffset {
		r.startOffset = r.endOffset
	}
	r.notifyChange()
}

func (r *rangeImpl) Collapse(toStart bool) {
	if toStart {
		r.endContainer = r.startContainer
		r.endOffset = r.startOffset
	} else {
		r.startContainer = r.endContainer
		r.startOffset = r.endOffset
	}
	r.notifyChange()
}

func (r *rangeImpl) IsCollapsed() bool {
	return r.startContainer == r.endContainer && r.startOffset == r.endOffset
}

func (r *rangeImpl) validate(node Node, offset int) {
	if node == nil {
		panic("dom: range node cannot be nil")
	}
	if node.OwnerDocument() != r.doc {
		panic("dom: range node must belong to the same document")
	}
	if offset < 0 {
		panic("dom: range offset cannot be negative")
	}

	if t, ok := node.(TextNode); ok {
		count := utf8.RuneCountInString(t.Data())
		if offset > count {
			panic(fmt.Sprintf("dom: range offset %d exceeds text length %d", offset, count))
		}
	} else {
		// For non-text nodes, offset is child index.
		count := 0
		for range node.ChildNodes() {
			count++
		}
		if offset > count {
			panic(fmt.Sprintf("dom: range offset %d exceeds child count %d", offset, count))
		}
	}
}

func (r *rangeImpl) notifyChange() {
	if r.doc == nil {
		return
	}
	if s, ok := r.doc.Selection().(*selectionImpl); ok {
		s.changed()
	}
}

func (r *rangeImpl) String() string {
	if r.startContainer == nil || r.endContainer == nil {
		return ""
	}

	if r.startContainer == r.endContainer {
		if t, ok := r.startContainer.(TextNode); ok {
			runes := []rune(t.Data())
			start := r.startOffset
			end := r.endOffset
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
			var sb strings.Builder
			idx := 0
			for child := range r.startContainer.ChildNodes() {
				if idx >= r.startOffset && idx < r.endOffset {
					sb.WriteString(child.TextContent())
				}
				idx++
			}
			return sb.String()
		}
	}

	var sb strings.Builder
	started := false
	var walk func(Node) bool
	walk = func(n Node) bool {
		if !started {
			if n == r.startContainer {
				started = true
				if t, ok := n.(TextNode); ok {
					runes := []rune(t.Data())
					if r.startOffset < len(runes) {
						sb.WriteString(string(runes[r.startOffset:]))
					}
				} else {
					idx := 0
					for child := range n.ChildNodes() {
						if idx == r.startOffset {
							started = true
						}
						if started {
							if !walk(child) {
								return false
							}
						}
						idx++
					}
				}
				return true
			} else {
				for child := range n.ChildNodes() {
					if !walk(child) {
						return false
					}
				}
				return true
			}
		}

		// started == true
		if n == r.endContainer {
			if t, ok := n.(TextNode); ok {
				runes := []rune(t.Data())
				end := r.endOffset
				if end > len(runes) {
					end = len(runes)
				}
				if end > 0 {
					sb.WriteString(string(runes[:end]))
				}
			} else {
				idx := 0
				for child := range n.ChildNodes() {
					if idx >= r.endOffset {
						break
					}
					if !walk(child) {
						return false
					}
					idx++
				}
			}
			return false
		}

		if t, ok := n.(TextNode); ok {
			sb.WriteString(t.Data())
		}
		for child := range n.ChildNodes() {
			if !walk(child) {
				return false
			}
		}
		return true
	}

	walk(r.doc)
	return sb.String()
}

type selectionImpl struct {
	doc    Document
	ranges []Range
}

var _ Selection = (*selectionImpl)(nil)

func (s *selectionImpl) RangeCount() int {
	return len(s.ranges)
}

func (s *selectionImpl) GetRangeAt(index int) Range {
	if index < 0 || index >= len(s.ranges) {
		return nil
	}
	return s.ranges[index]
}

func (s *selectionImpl) AddRange(r Range) {
	if r == nil {
		return
	}
	// In standard DOM, it usually supports only one range.
	// Requirement says "Can hold at least one dom.Range".
	s.ranges = append(s.ranges, r)
	s.changed()
}

func (s *selectionImpl) RemoveAllRanges() {
	if len(s.ranges) == 0 {
		return
	}
	s.ranges = nil
	s.changed()
}

func (s *selectionImpl) String() string {
	var sb strings.Builder
	for _, r := range s.ranges {
		sb.WriteString(r.String())
	}
	return sb.String()
}

func (s *selectionImpl) changed() {
	if s.doc == nil {
		return
	}
	// Dispatch selectionchange on the document.
	s.doc.DispatchToTarget(event.NewBaseEvent(event.EventSelectionChange, s.doc, false))
}

func (s *selectionImpl) NewRange() Range {
	return &rangeImpl{doc: s.doc}
}

// Internal helper for document to create selection
func newSelection(doc Document) *selectionImpl {
	return &selectionImpl{doc: doc}
}
