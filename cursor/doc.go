// Package cursor provides a unified abstraction for terminal hardware cursor
// management. It decouples the engine's focus management from the render
// objects' local coordinate systems.
//
// # Core types
//
// [State] carries the cursor's visibility, screen-space position (X, Y), and
// visual [Shape] (block, bar, or underline — each in blinking or steady
// variants). Render objects that want to control the hardware cursor implement
// the [Provider] interface; the engine queries this interface when the owning
// node receives focus.
//
// # Text-fragment helper (TSK-023)
//
// [FromTextFragment] translates a byte offset into a terminal-cell (x, y)
// coordinate by walking a standard IFC fragment tree produced by the inline
// layout algorithm. This is the canonical cursor-positioning helper and
// replaces bespoke per-widget arithmetic that was previously duplicated in
// individual render objects.
//
// The helper is a pure function with no side effects. It depends only on the
// [github.com/masterkeysrd/kite/internal/layout] and
// [github.com/masterkeysrd/kite/text] packages.
package cursor
