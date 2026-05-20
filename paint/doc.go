// Package paint implements the paint phase of the Kite rendering pipeline.
//
// The paint engine performs a depth-first traversal of the immutable
// [layout.Fragment] tree produced by the layout phase and rasterises each
// fragment into a 2-D grid of terminal [Cell] values held by a [FrameBuffer].
//
// # Per-fragment clipping (ADR-011)
//
// When a fragment's computed style declares OverflowX or OverflowY as any value
// other than OverflowVisible (i.e., Hidden, Clip, Scroll, or Auto), the engine
// creates a clipped sub-[Surface] whose drawable area equals the fragment's
// content box (origin inset by border + padding, size reduced by the same
// amounts). All descendant paint calls are routed through this clipped surface,
// so any cell written outside the content box is silently dropped.
//
// The fragment's own background fill and border decoration are painted onto the
// unclipped surface before the clipped surface is created, ensuring that a
// node's border-box decoration is never eaten by its own overflow property.
//
// For asymmetric overflow (e.g., OverflowX: Hidden, OverflowY: Visible), the
// clip rect spans the full surface extent on the visible axis and the
// content-box inset on the clipping axis. Composition for nested overflow
// containers is automatic: [Surface.Clip] intersects the new rect with the
// already-active clip.
//
// # Border resolution invariant
//
// [PaintEngine.resolveBorders] is invoked exactly once, on the root
// [Surface], after the full fragment tree has been painted. It must never be
// called on a clipped sub-surface because the junction resolver must inspect
// the complete set of border cells across the entire viewport.
package paint
