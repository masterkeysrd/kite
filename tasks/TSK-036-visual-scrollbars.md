# TSK-036: Customizable Visual Scrollbars

## Description
Implement the visual representation of scrollbars. In a TUI, screen real estate is precious, so scrollbars are an explicit opt-in styling decision configured via `style.Scrollbar`.

## Requirements

### 1. Style Package (`style/scrollbar.go`, `style/style.go`, `style/computed.go`)
- Create a `style.Scrollbar` struct featuring:
  - `X Optional[bool]` and `Y Optional[bool]`
  - `TrackGlyph Optional[rune]` and `TrackColor Optional[color.Color]`
  - `ThumbGlyph Optional[rune]` and `ThumbColor Optional[color.Color]`
- Add `Scrollbar Optional[Scrollbar]` to `style.Style` and `Scrollbar Scrollbar` to `style.Computed`.
- Extend the fluent API on `style.Style` to easily configure these properties (e.g., `ScrollbarY(true)`, `ScrollbarThumb(glyph, color)`).
- Update `style.Resolver` to merge scrollbar configurations and apply sensible TUI defaults for glyphs if explicitly set to true but missing glyphs.

### 2. Layout Engine (`layout/ng.go`, `layout/block.go`)
- Update layout algorithms (starting with `BlockAlgorithm`) to check if the current node is a scroll container (`OverflowY/X == Scroll | Auto`) **and** the computed style requests a scrollbar (`Scrollbar.Y == true`).
- If so, reduce the available width (for Y) or height (for X) passed to children by 1 cell.
- Add `HasScrollbarX` and `HasScrollbarY` boolean flags to `layout.Fragment` so the paint engine knows the space was successfully reserved.

### 3. Paint Engine (`paint/engine.go`)
- In `paintFragment`, after drawing children (and clipping them), check if `frag.HasScrollbarY` (or X) is true.
- If true, query the `scrollX, scrollY` from the element (ensure you pass the logical node up or cast appropriately).
- Calculate the ratio between the Fragment's content-box size and the total scrollable content extent (found via `layout.MaxScroll`).
- Draw the `TrackGlyph` down the reserved column/row.
- Draw the `ThumbGlyph` at the mathematically correct offset to represent the current scroll position.
- Apply `TrackColor` and `ThumbColor`.

### 4. Text Controls Integration (`element/textarea.go`)
- Update the `IntrinsicStyle` of `<textarea>` to set `ScrollbarY(true)`. By default, textareas should have a visual scrollbar if they overflow, matching user expectations for standard forms.

## Tests
- `TestStyle_ScrollbarCascade`: Verify merging of partial scrollbar definitions.
- `TestLayout_ScrollbarSpaceReservation`: Verify that available width is reduced by 1 when `ScrollbarY` is true and `Overflow` is `Auto/Scroll`.
- `TestPaint_ScrollbarRendering`: Use `testenv` to assert that track and thumb glyphs are printed into the framebuffer at the correct offsets when a view is scrolled.

## Documentation
- Document the new fluent API in `style/README.md`.
- Ensure users know that setting `OverflowAuto` without `ScrollbarY(true)` defaults to an invisible, space-saving scroll behavior.