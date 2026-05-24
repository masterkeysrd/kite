package kitex

// --- Rendering Utilities ------------------------------------------------------
//
// These helpers mirror the patterns commonly used in React/JSX to make building
// VDOM trees more ergonomic in Go. They are all pure functions with no hidden
// allocations beyond the returned slice or node.

// Map converts a slice of data into a slice of Nodes using a mapping function.
// The mapping function receives each item and its zero-based index, allowing
// the caller to embed both data and position into the rendered node.
//
// The returned slice can be spread into any variadic children parameter using
// the "..." spread syntax, or passed to [Nodes] to merge with other nodes.
//
// Example — render a keyed list of items:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.Map(items, func(item Item, i int) kitex.Node {
//	        return ItemCard(ItemCardProps{Key: item.ID, Title: item.Name})
//	    })...,
//	)
func Map[D any](items []D, fn func(item D, i int) Node) []Node {
	nodes := make([]Node, 0, len(items))
	for i, item := range items {
		if n := fn(item, i); n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// Nodes merges one or more Node slices and/or individual Nodes into a single
// flat []Node. Nil entries are filtered out. Use this to combine the output of
// [Map] with other nodes before spreading into a parent element.
//
// Example:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.Nodes(
//	        kitex.Map(items, renderItem),
//	        []kitex.Node{footer},
//	    )...,
//	)
func Nodes(groups ...[]Node) []Node {
	total := 0
	for _, g := range groups {
		total += len(g)
	}
	out := make([]Node, 0, total)
	for _, g := range groups {
		for _, n := range g {
			if n != nil {
				out = append(out, n)
			}
		}
	}
	return out
}

// If renders node when cond is true, otherwise returns nil.
// Nil children are safely ignored by all kitex element factories and the
// reconciler, so this can be used inline without wrapping in a slice.
//
// Example:
//
//	kitex.Box(kitex.BoxProps{},
//	    kitex.If(isLoggedIn, UserMenu(UserMenuProps{})),
//	    kitex.Text("Welcome"),
//	)
func If(cond bool, node Node) Node {
	if cond {
		return node
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

// Fragment returns its children as a flat []Node slice without introducing any
// wrapper element into the DOM. Use the spread syntax ("...") to inline the
// result into a parent's variadic children.
//
// Fragment is useful when a helper function needs to return multiple sibling
// nodes but cannot or should not wrap them in a Box.
//
// Example:
//
//	func renderHeader(title, subtitle string) []kitex.Node {
//	    return kitex.Fragment(
//	        kitex.Box(kitex.BoxProps{Style: boldStyle}, kitex.Text(title)),
//	        kitex.Box(kitex.BoxProps{Style: mutedStyle}, kitex.Text(subtitle)),
//	    )
//	}
//
//	// In the parent:
//	kitex.Box(kitex.BoxProps{}, renderHeader("Hello", "World")...)
func Fragment(children ...Node) []Node {
	out := make([]Node, 0, len(children))
	for _, n := range children {
		if n != nil {
			out = append(out, n)
		}
	}
	return out
}
