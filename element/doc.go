// Package element provides the high-level logical DOM components for Kite.
//
// These components wrap the base DOM elements and provide default styles,
// fluent builder-style APIs, and tag-specific behaviors.
//
// # Declarative API
//
// Kite elements favor a declarative, functional approach to tree construction.
// Constructors like Box(), Span(), and UL() accept variadic children and
// automatically box strings into text nodes.
//
//	ui := element.Box(
//	    element.Span("Hello "),
//	    element.Span("World"),
//	)
//
// # Adoption
//
// Elements created via declarative constructors are implicitly adopted by
// the engine's main document when the tree is mounted via engine.Mount().
//
// # Available components:
//   - Box: A generic container (similar to <div>).
//   - Span: An inline container (similar to <span>).
//   - Text: A leaf node containing text.
//   - UL (UnorderedList): A list with bullet markers (<ul>).
//   - OL (OrderedList): A numbered list (<ol>).
//   - LI (ListItem): An individual item within a list (<li>).
//   - Table, THead, TBody, TFoot, TR, TD: Components for building grid-based layouts.
package element
