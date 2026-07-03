// Package dom defines the logical DOM tree, its node types, and the lifecycle
// of adoption. It models structure only — no layout algorithms, computed styles,
// or drawing logic belong here.
//
// # Node Types
//
// Three node kinds exist: Document, Element, and Text. Element carries
// identity (tag name, id, attributes) and bridges interactivity state
// (Focusable, Disableable) to the render layer via render.Object.
//
// # UA Shadow Subtrees (ADR-009)
//
// Host elements (replaced or compound widgets) attach a closed UA shadow
// subtree via AttachUARoot. Public traversal APIs (ChildNodes(),
// FirstChild(), LastChild(), Children(), GetElementByID()) never expose
// UA-subtree nodes; engine phases use LayoutChildren() instead.
//
// # Element Identity & Adoption (ADR-0036)
//
// Every Element carries an outer back-pointer. When widgets wrap standard
// elements, functions like event.Target(), GetElementByID(), and
// RenderObject.Node() always return the outermost, user-visible wrapper.
//
// # Scroll Model (ADR-012)
//
// Every Element exposes Scroll(), ScrollTo(x, y), and ScrollBy(dx, dy).
// Scroll state is held in a lazy *scrollState pointer, allocated only when
// needed. Programmatic scroll is valid on any element; paint only applies
// translation if the computed style indicates a scroll container.
//
// Example:
//
//	doc := dom.NewDocument()
//	box := dom.NewElement(doc, "div", nil)
//	box.SetID("main")
//	doc.AppendChild(box)
//
//	text := dom.NewTextNode(doc, "Hello", nil)
//	box.AppendChild(text)
package dom
