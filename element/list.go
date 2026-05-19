package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// UnorderedList represents a list of items where the order does not matter (<ul>).
type UnorderedList struct {
	elementBase[UnorderedList]
}

var _ Element = (*UnorderedList)(nil)

// NewUnorderedList creates a new unordered list.
func NewUnorderedList(doc dom.Document) *UnorderedList {
	u := &UnorderedList{}
	u.initBase(doc.CreateElement("ul", u), u, style.Style{
		Display:       style.Some(style.DisplayBlock),
		ListStyleType: style.Some(style.ListStyleDisc),
		Padding:       style.Some(style.EdgeValues[int]{Left: 2}),
	})
	return u
}

// OrderedList represents a list of items where the order matters (<ol>).
type OrderedList struct {
	elementBase[OrderedList]
}

var _ Element = (*OrderedList)(nil)

// NewOrderedList creates a new ordered list.
func NewOrderedList(doc dom.Document) *OrderedList {
	o := &OrderedList{}
	o.initBase(doc.CreateElement("ol", o), o, style.Style{
		Display:       style.Some(style.DisplayBlock),
		ListStyleType: style.Some(style.ListStyleDecimal),
		Padding:       style.Some(style.EdgeValues[int]{Left: 3}),
	})
	return o
}

// ListItem represents an item in a list (<li>).
type ListItem struct {
	elementBase[ListItem]
}

var _ Element = (*ListItem)(nil)

// NewListItem creates a new list item.
func NewListItem(doc dom.Document) *ListItem {
	l := &ListItem{}
	l.initBase(doc.CreateElement("li", l), l, style.Style{
		Display: style.Some(style.DisplayListItem),
	})
	return l
}
