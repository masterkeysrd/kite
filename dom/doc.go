// Package dom implements the logical node tree for kitex (kite v2).
//
// DOM nodes hold a reference to their RenderObject (if any). Structural
// mutations on a DOM node signal the parent's render object; style mutations
// signal the node's own render object. Nodes with no render attachment (e.g.
// fragments) do not participate in rendering.
//
// # Adoption and connection
//
// Implicit Document Adoption: When a detached node (or subtree) created by one
// Document is appended to or inserted into another Document, it is implicitly
// adopted by the destination document. All nodes in the subtree have their
// OwnerDocument updated recursively. Attempting to append or insert a node
// that is already connected to another document tree will panic.
//
// Element Identity Adoption: Every element carries a self back-pointer set by
// the DOM during the attach walk so that event.Target(), GetElementByID(), and
// RenderObject.Node() all return the outermost user-visible wrapper even when
// a widget embeds a native element.
//
// A node is "connected" when it is reachable from the Document root.
// IsConnected() returns this state in O(1). The attach walk (pre-order) sets
// the flag and fires OnConnected; the detach walk (post-order) fires
// OnDisconnected and then clears the flag.
//
// # Lifecycle
//
// Types that implement the Lifecycle interface receive OnConnected and
// OnDisconnected callbacks synchronously inside the mutation call that
// triggered the walk. Self- and descendant-mutations inside a callback are
// permitted; ancestor-mutations panic.
//
// # UA Shadow Subtree (ADR-009)
//
// Host elements that need private internal DOM structure (replaced elements
// such as <input>, <textarea>, or compound widgets like <checkbox>) can attach
// a closed UA shadow subtree via Element.AttachUARoot(root).
//
// Key invariants:
//   - Public traversal APIs (ChildNodes, FirstChild, LastChild, Children,
//     GetElementByID) never expose UA-subtree nodes. Author code has no
//     documented path to reach UA internals.
//   - Engine-internal phases (Sync, Style, Layout, Paint) use
//     dom.LayoutChildren(n) which yields the public child list followed by the
//     UA root's children. This function is the single authoritative walker for
//     all engine phases; it must never be called from author code.
//   - Every node inside the UA subtree has its outer back-pointer set to the
//     host element (ADR-0036), so event.Target() and identity queries collapse
//     to the host regardless of which UA node triggered a hit-test.
//   - UA nodes never satisfy dom.Focusable; focus.Manager uses the public
//     Children() iterator and therefore sees no UA nodes.
//   - dom.IsUANode(n) reports whether n belongs to a UA subtree. Engine and
//     focus code may use this predicate for additional guard checks.
//
// # Scroll (ADR-012)
//
// Every Element exposes Scroll, ScrollTo, and ScrollBy. These methods manage
// a lazy scroll-offset state (X, Y) in terminal cells. The stored offset represents
// raw author intent and is not clamped at the DOM level. Clamping and coordinate
// translation happen at paint time if the element's computed overflow style
// indicates it is a scroll container.
//
// Mutating the scroll offset marks the element's render object for a paint update
// (DirtyScroll) and dispatches a bubbling event.ScrollEvent.
//
// # Overlays (ADR-008)
//
// The document manages a top layer for out-of-flow elements such as dialogs,
// tooltips, and context menus via ShowOverlay(el, zIndex) and HideOverlay(el).
//
// Overlays are rendered above the document body and are sorted primarily by
// zIndex (ascending) and secondarily by insertion order. Because overlays are
// out-of-flow, they do not affect the layout of the main document body.
//
// # Selection (ADR-022)
//
// The Document maintains a global Selection state representing the range of
// text selected by the user or the current caret position. The Selection holds
// one or more Range objects, each defining a segment of the document between
// two boundary points (StartContainer/StartOffset and EndContainer/EndOffset).
//
// Mutating the selection or its ranges dispatches an event.EventSelectionChange
// on the Document. The selection state is logical and refers to text nodes;
// the render and paint engines resolve these logical boundaries into physical
// screen-space highlights during the rendering pipeline.
package dom
