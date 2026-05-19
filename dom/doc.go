// Package dom implements the logical node tree for kitex (kite v2).
//
// DOM nodes hold a reference to their RenderObject (if any). Structural
// mutations on a DOM node signal the parent's render object; style mutations
// signal the node's own render object. Nodes with no render attachment (e.g.
// fragments) do not participate in rendering.
//
// # Adoption and connection
//
// Every element carries a self back-pointer set by the DOM during the
// attach walk so that event.Target(), GetElementByID(), and
// RenderObject.Node() all return the outermost user-visible wrapper even
// when a widget embeds a native element.
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
package dom
