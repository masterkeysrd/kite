// Package render provides the core engine for maintaining the render tree.
// It bridges the logical DOM tree with the Layout and Style engines via
// render objects that track dirty state and computed styles.
//
// Render objects do not own sparse style state (author styles or element
// defaults); they act as stateless proxies that query their underlying logical
// DOM node dynamically.
//
// # RenderView and Overlays
//
// The RenderView is the root of the render tree and represents the viewport.
// It manages the main document flow and a list of overlay render roots that
// are painted on top of the main flow. LayoutPhase handles both the main tree
// and all active overlays in a single pass.
package render
