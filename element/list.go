package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// UnorderedListElement represents a list of items where the order does not matter (<ul>).
type UnorderedListElement struct {
	elementBase[UnorderedListElement]
}

var _ Element = (*UnorderedListElement)(nil)

var defaultUnorderedListStyle = style.S().
	Display(style.DisplayBlock).
	ListStyleType(style.ListStyleDisc).
	PaddingLeft(2)

// NewUnorderedList creates a new unordered list.
func NewUnorderedList(doc dom.Document) *UnorderedListElement {
	u := &UnorderedListElement{}
	u.initBase(doc.CreateElement("ul", u), u, defaultUnorderedListStyle)
	return u
}

// UL creates a new unordered list with the given children.
func UL(children ...any) *UnorderedListElement {
	u := NewUnorderedList(orphanDocument)
	processChildren(u, children)
	return u
}

// OrderedListElement represents a list of items where the order matters (<ol>).
type OrderedListElement struct {
	elementBase[OrderedListElement]
}

var _ Element = (*OrderedListElement)(nil)

var defaultOrderedListStyle = style.S().
	Display(style.DisplayBlock).
	ListStyleType(style.ListStyleDecimal).
	PaddingLeft(3)

// NewOrderedList creates a new ordered list.
func NewOrderedList(doc dom.Document) *OrderedListElement {
	o := &OrderedListElement{}
	o.initBase(doc.CreateElement("ol", o), o, defaultOrderedListStyle)
	return o
}

// OL creates a new ordered list with the given children.
func OL(children ...any) *OrderedListElement {
	o := NewOrderedList(orphanDocument)
	processChildren(o, children)
	return o
}

// ListItemElement represents an item in a list (<li>).
type ListItemElement struct {
	elementBase[ListItemElement]
}

var _ Element = (*ListItemElement)(nil)

var defaultListItemStyle = style.S().
	Display(style.DisplayListItem)

// NewListItem creates a new list item.
func NewListItem(doc dom.Document) *ListItemElement {
	l := &ListItemElement{}
	l.initBase(doc.CreateElement("li", l), l, defaultListItemStyle)
	return l
}

// LI creates a new list item with the given children.
func LI(children ...any) *ListItemElement {
	l := NewListItem(orphanDocument)
	processChildren(l, children)
	return l
}
