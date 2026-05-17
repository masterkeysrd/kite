// Package spatial implements directional (arrow-key) spatial navigation for
// kitex (kite v2).
//
// Navigate(m, dir) is a pure function that selects the nearest focusable node
// in the given direction within the active focus scope, using edge-to-edge
// Euclidean distance with an off-axis penalty of 2.0. It does not wrap and
// does not bind any keys — callers wire it to arrow-key events themselves.
package spatial
