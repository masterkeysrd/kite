// Package style defines the Style and Computed value types and the Resolver
// that performs inheritance, default application, and computed-value
// resolution for kitex (kite v2).
//
// # Style vs Computed
//
// Style fields are Optional[T] to distinguish unset from zero. Computed
// mirrors Style without Optional and is what the render layer reads.
// The package has no dependencies on other kitex packages.
//
// # Four-Layer Cascade (ADR-010)
//
// The Resolver applies style contributions in the following order, from
// weakest to strongest precedence:
//
//  1. Inherited values (OriginInherited): inheritable properties (Foreground,
//     Bold, Italic, Underline, Strikethrough, TextWrap, TextOverflow,
//     WhiteSpace, WordBreak, OverflowWrap, ListStyleType, CursorShape,
//     CursorColor) flow from the parent's Computed style into the child when
//     the child does not supply its own value.
//
//  2. Element-type defaults (OriginUADefault): each element type may return a
//     sparse Style from DefaultStyle() to override the root baseline for its
//     specific tag. For example, a <span> returns Display:Inline. Author
//     styles can override these.
//
//  3. Author styles (OriginAuthor): the author-set sparse Style returned by
//     RawStyle() is overlaid on top of inherited and default values. These
//     are the styles the application developer writes (e.g. via element.Style()).
//
//  4. UA-intrinsic styles (OriginUserAgent): the highest-precedence layer,
//     returned by IntrinsicStyle(). Replaced and compound elements (e.g.
//     <input>, <textarea>) use this layer to force UA-mandated properties
//     (e.g. Display:InlineBlock, OverflowX:Clip, WhiteSpace:PreWrap) that
//     the author cannot override. Most elements return an empty Style{} from
//     IntrinsicStyle(), paying zero additional cost.
//
// The cascade origin enum (CascadeOrigin) is internal to this package and
// must not leak into dom, render, layout, or paint.
// # Fluent Helpers
//
// Style provides fluent shorthands for common property combinations. For
// example, Style.Overflow(v) sets both OverflowX and OverflowY to the same
// value.
package style
