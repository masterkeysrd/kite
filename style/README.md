# Package style

The `style` package manages the sparse `Style` definitions and the fully-resolved `Computed` styles for the Kite framework. It implements a four-layer cascade resolution system inspired by CSS.

## 🏛 Architecture

### Style vs Computed
- **`Style`**: A sparse struct where every field is `Optional[T]`. This is the "authoring" model used by developers to set properties.
- **`Computed`**: A fully-resolved struct with concrete values for every property. This is what the layout and paint engines read.

### The Four-Layer Cascade (ADR-010)
1. **Inherited Values**: Properties like `Foreground` and `Bold` flow from parent to child.
2. **Element Defaults**: Tag-specific defaults (e.g., `Display: Inline` for `<span>`).
3. **Author Styles**: Properties set explicitly via `Style()`.
4. **UA-Intrinsic Styles**: Mandatory properties for compound elements (e.g., `OverflowY: Auto` for `<textarea>`).

## 📜 Scrollbars (Task 036)

Scrollbars in a TUI consume precious cell space, so they are explicit opt-in decisions configured via the `Scrollbar` property.

### Fluent API
The `Style` and `Element` types provide fluent helpers for scrollbar configuration:

- **`ScrollbarX(bool)`**: Enables or disables the horizontal scrollbar.
- **`ScrollbarY(bool)`**: Enables or disables the vertical scrollbar.
- **`ScrollbarThumb(glyph rune, c color.Color)`**: Customizes the thumb appearance.
- **`ScrollbarTrack(glyph rune, c color.Color)`**: Customizes the track appearance.

### Defaults
When scrollbars are enabled but glyphs are omitted, the resolver applies sensible TUI defaults:

- **Vertical**: Track `│`, Thumb `┃`
- **Horizontal**: Track `─`, Thumb `━`

### Behavior
- **Reservation**: If `Overflow` is `Auto` or `Scroll` AND scrollbars are enabled, the layout engine reserves 1 cell of space along the appropriate edge.
- **Auto-hide**: If `Overflow` is `Auto`, space is only reserved if the content actually overflows the viewport. If `ScrollbarY(true)` is NOT set, `OverflowAuto` still allows scrolling but hides the visual indicator.
- **Rendering**: Scrollbars are painted over the element's content, representing the current `Scroll()` offset of the logical DOM element.
