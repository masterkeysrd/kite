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
// # Borders
//
// Borders are configured using a fluent, immutable API on the [style.Border] type.
//
//	element.Box("Content").Style(style.Style{
//	    Border: style.SingleBorder().Color(color.RGBA{255, 0, 0, 255}).Some(),
//	})
//
// # Available components:
//   - Box: A generic container (similar to <div>).
//   - Span: An inline container (similar to <span>).
//   - Text: A leaf node containing text.
//   - UL (UnorderedList): A list with bullet markers (<ul>).
//   - OL (OrderedList): A numbered list (<ol>).
//   - LI (ListItem): An individual item within a list (<li>).
//   - Table, THead, TBody, TFoot, TR, TD: Components for building grid-based layouts.
//   - Input: A single-line text-input widget implemented as a UA shadow host
//     (ADR-009) with intrinsic display:inline-block, overflow:clip, and
//     white-space:nowrap. The host owns an editor.Buffer and a single internal
//     UA text node; the IFC shapes and renders the text. Cursor positioning uses
//     cursor.FromTextFragment (TSK-023). See TSK-024.
//   - TextArea: A multi-line text-input widget implemented as a UA shadow host
//     (ADR-009) with intrinsic display:inline-block, overflow-x:clip,
//     overflow-y:scroll, white-space:pre-wrap, and overflow-wrap:break-word.
//     The host owns an editor.Buffer and a single internal UA text node; the
//     IFC handles shaping, wrapping, and mandatory breaks. Cursor positioning
//     uses cursor.FromTextFragment (TSK-023). Up/Down navigation is
//     implemented by walking the fragment tree. See TSK-025.
package element
