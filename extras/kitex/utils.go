package kitex

import (
	"github.com/masterkeysrd/kite/dom"
)

// --- Rendering Utilities ------------------------------------------------------
//
// These helpers mirror the patterns commonly used in React/JSX to make building
// VDOM trees more ergonomic in Go. They are all pure functions with no hidden
// allocations beyond the returned slice or node.

// Map converts a slice of data into a Node (a Fragment containing the mapped nodes)
// using a mapping function. The mapping function receives each item and its
// zero-based index, allowing the caller to embed both data and position into
// the rendered node.
//
// Example — render a keyed list of items:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.Map(items, func(item Item, i int) kitex.Node {
//	        return ItemCard(ItemCardProps{Key: item.ID, Title: item.Name})
//	    }),
//	)
func Map[D any](items []D, fn func(item D, i int) Node) Node {
	nodes := make([]Node, 0, len(items))
	for i, item := range items {
		if n := fn(item, i); n != nil {
			nodes = append(nodes, n)
		}
	}
	return Fragment(nodes...)
}

// Nodes merges one or more Nodes (which may be Fragments, individual nodes, or nil)
// into a single flat Fragment. Nil entries are filtered out, and nested Fragments
// are flattened into the returned Fragment.
//
// Example:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.Nodes(
//	        kitex.Map(items, renderItem),
//	        footer,
//	    ),
//	)
func Nodes(nodes ...Node) Node {
	var flat []Node
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if frag, ok := n.(*fragmentNode); ok {
			for _, c := range frag.children {
				if c != nil {
					flat = append(flat, c)
				}
			}
		} else {
			flat = append(flat, n)
		}
	}
	return Fragment(flat...)
}

// If renders node returned by fn when cond is true, otherwise returns nil.
// Nil children are safely ignored by all kitex element factories and the
// reconciler, so this can be used inline without wrapping in a slice.
//
// Example:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.If(isLoggedIn, func() kitex.Node { return UserMenu(UserMenuProps{}) }),
//	    kitex.Text("Welcome"),
//	)
func If(cond bool, fn func() Node) Node {
	if cond {
		return fn()
	}
	return nil
}

// IfElse renders thenNode when cond is true, elseNode otherwise.
// Both branches are evaluated eagerly (Go has no lazy evaluation), so avoid
// using IfElse when the branches have expensive side-effects; use a regular
// if-statement instead in those cases.
//
// Example:
//
//	kitex.IfElse(isAdmin,
//	    AdminPanel(AdminPanelProps{}),
//	    kitex.Text("Access denied"),
//	)
func IfElse(cond bool, thenNode, elseNode Node) Node {
	if cond {
		return thenNode
	}
	return elseNode
}

// Fragment returns its children grouped together without introducing any
// wrapper element into the DOM.
//
// Fragment is useful when a component needs to return multiple sibling
// nodes but cannot or should not wrap them in a Box, or when conditionally
// rendering a group of nodes.
//
// Example:
//
//	var MyComp = kitex.SimpleFC("MyComp", func() kitex.Node {
//	    return kitex.Fragment(
//	        kitex.Box(kitex.BoxProps{Style: boldStyle}, kitex.Text("Title")),
//	        kitex.Box(kitex.BoxProps{Style: mutedStyle}, kitex.Text("Subtitle")),
//	    )
//	})
func Fragment(children ...Node) Node {
	return &fragmentNode{children: children}
}

type fragmentNode struct {
	children []Node
	refs     []dom.Node
}

var _ Node = (*fragmentNode)(nil)
var _ nodeInternal = (*fragmentNode)(nil)

func (f *fragmentNode) Instantiate(doc dom.Document) []dom.Node {
	var reals []dom.Node
	for _, child := range f.children {
		if child != nil {
			reals = append(reals, child.Instantiate(doc)...)
		}
	}
	f.refs = reals
	return reals
}

func (f *fragmentNode) Update(els []dom.Node, old Node) {
	f.refs = els
}

func (f *fragmentNode) setRefs(els []dom.Node) {
	f.refs = els
}

func (f *fragmentNode) Children() []Node { return f.children }
func (f *fragmentNode) Props() any       { return nil }
func (f *fragmentNode) TagName() string  { return "#fragment" }
func (f *fragmentNode) Key() string      { return "" }
func (f *fragmentNode) Release() {
	f.refs = nil
}

func (f *fragmentNode) realNodes() []dom.Node {
	return f.refs
}

func (f *fragmentNode) complexity() int {
	score := 1
	for _, child := range f.children {
		if child != nil {
			if ni, ok := child.(nodeInternal); ok {
				score += ni.complexity()
			}
		}
	}
	return score
}

func (f *fragmentNode) containsProvider() bool {
	for _, child := range f.children {
		if child != nil {
			if ni, ok := child.(nodeInternal); ok && ni.containsProvider() {
				return true
			}
		}
	}
	return false
}

func (f *fragmentNode) isProvider() bool        { return false }
func (f *fragmentNode) hasDirectProvider() bool { return false }
