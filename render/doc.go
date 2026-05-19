// Package render provides the core engine for maintaining the render tree.
// It bridges the logical DOM tree with the Layout and Style engines via
// render objects that track dirty state and computed styles.
//
// Render objects do not own sparse style state (author styles or element
// defaults); they act as stateless proxies that query their underlying logical
// DOM node dynamically.
package render
