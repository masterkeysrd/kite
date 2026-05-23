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
// # Text Control Base Pattern
//
// Custom text-editing widgets should embed [textControlBase[T]] (ADR-013,
// TSK-029). This generic base struct centralises the three mechanical
// concerns shared by every text field:
//
//  1. CursorState() — translates the editor.Buffer's byte offset into a
//     terminal-cell (X, Y) position relative to the host's border-box,
//     using cursor.FromTextFragment on the inner uaDiv's IFC fragment.
//
//  2. ScrollCursorIntoView() — ensures the caret stays visible after any
//     buffer mutation or focus acquisition by updating the host's scroll
//     offset via ScrollTo(). It reads content geometry from the layout
//     fragment; it never stores its own scrollX/Y. To prevent lag, this
//     math is deferred to the engine's auto-scroll frame phase.
//
//  3. handleMouseDown / handleKeyDown — generic event handlers that
//     translate screen-space mouse clicks to buffer byte offsets (accounting
//     for host inset and current scroll offset) and route keyboard input to
//     editor.Buffer operations. The isMultiline flag gates Up/Down/Enter.
//
// Concrete elements (InputElement, TextAreaElement) embed textControlBase and
// only need to:
//   - Define IntrinsicStyle() and DefaultStyle().
//   - Build their specific UA shadow subtree (ADR-009).
//   - Provide a syncCallback to rebuild the UA subtree after buffer mutations.
//
// # Available components:
//   - Box: A generic container (similar to <div>).
//   - Span: An inline container (similar to <span>).
//   - Text: A leaf node containing text.
//   - Button: A clickable button with centered content and interactive states.
//   - UL (UnorderedList): A list with bullet markers (<ul>).
//   - OL (OrderedList): A numbered list (<ol>).
//   - LI (ListItem): An individual item within a list (<li>).
//   - Table, THead, TBody, TFoot, TR, TD: Components for building grid-based layouts.
//   - Input: A single-line text-input widget implemented as a UA shadow host
//     (ADR-009) with intrinsic display:inline-block, overflow:clip, and
//     white-space:nowrap. Embeds textControlBase[InputElement] (ADR-013).
//     See TSK-024, TSK-029.
//   - TextArea: A multi-line text-input widget implemented as a UA shadow host
//     (ADR-009) with intrinsic display:inline-block, overflow-y:scroll, and
//     overflow-wrap:break-word. Embeds textControlBase[TextAreaElement]
//     (ADR-013). See TSK-025, TSK-029.
package element
