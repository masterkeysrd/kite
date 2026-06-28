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
4. **UA-Intrinsic Styles**: Mandatory properties for compound elements. Examples:
    - `<textarea>`: `OverflowY: Auto`, `OverflowX: Clip`.
    - `<button>`: `Display: Flex`, `AlignItems: Center`, `JustifyContent: Center`.

### Simplified Fluent API
The `Style` type provides variadic methods for common layout properties to reduce boilerplate:

- **`Gap(values ...int)`**: Sets row and column gaps. Expects 1 or 2 `int` values.
    - `Gap(1)` sets both row and column gaps to 1.
    - `Gap(1, 2)` sets row gap to 1 and column gap to 2.
- **`Padding(values ...int)`** & **`Margin(values ...int)`**: Uses CSS-like shorthand. Expects 1, 2, or 4 `int` values.
    - `Padding(1)` sets all sides to 1.
    - `Padding(1, 2)` sets top/bottom to 1, left/right to 2.
    - `Padding(1, 2, 3, 4)` sets top, right, bottom, left.
- **`Flex(grow int, rest ...any)`**: Configures flex item properties. Expects a mandatory `grow (int)`, and optional `shrink (int)` and `basis (style.Dimension)`.
    - `Flex(1)` sets grow to 1 (shrink=1, basis=auto).
    - `Flex(1, 0)` sets grow to 1, shrink to 0.
    - `Flex(1, 1, style.Cells(10))` sets grow, shrink, and basis.
- **`GridTemplateColumns(v ...GridTrackSize)`** & **`GridTemplateRows(v ...GridTrackSize)`**: Variadic track definitions. Expects variadic `style.Dimension` values.
    - `GridTemplateColumns(style.Cells(10), style.Fr(1))`
- **`Border(args ...any)`**: Sets border properties globally. Accepts `bool`, `style.BorderStyle`, `color.Color`, `style.BorderGlyphs`, or `style.Border`.
    - `Border(true)` enables all edges with `BorderSingle`.
    - `Border(true, style.BorderDouble)` sets style for all edges.
    - `Border(true, color.Red)` sets color for all edges.
    - `Border(true, style.BorderGlyphs{H: "#"})` sets custom glyphs and switches to `BorderCustom`.
- **`BorderTop`, `BorderRight`, `BorderBottom`, `BorderLeft`, `BorderHorizontal`, `BorderVertical`**: Variadic side-specific settings. Accepts optional `style.BorderStyle`, `color.Color`, or `style.BorderGlyphs`.
    - `BorderTop(true, style.BorderDouble, color.Red)`
    - `BorderLeft(true, style.BorderGlyphs{V: "!"})` // Switches to `BorderCustom` automatically.

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

## 📱 Media Queries & Responsive Layouts

Kite features a declarative, high-performance **Media Queries** system that resolves styles dynamically based on the terminal's viewport size.

### Media Rules API
You can register conditional styles on any base `Style` builder using the `Media()` method:

```go
var CardStyle = style.S().
    Width(style.Cells(30)).
    Background(color.RGBA{B: 255, A: 255}). // Blue background on small screen
    // Merge overrides when the viewport width is >= 80 cells
    Media(style.Query().MinWidth(80), style.S().
        Width(style.Cells(60)).
        Background(color.RGBA{G: 255, A: 255}), // Green background on wide screen
    )
```

Predefined query constraints:
- **`MinWidth(w int)`**: Match when viewport width is $\ge w$ cells.
- **`MaxWidth(w int)`**: Match when viewport width is $\le w$ cells.
- **`MinHeight(h int)`**: Match when viewport height is $\ge h$ cells.
- **`MaxHeight(h int)`**: Match when viewport height is $\le h$ cells.

### Performance & Cache Invalidation
- **0% Overhead Static Matching**: Media queries avoid arbitrary functions or heap allocations during normal render loops.
- **Selective Cache Invalidation**: The style resolver caches resolved styles. When the viewport is resized, the engine walks the render tree and invalidates *only* the subtrees containing media rules. All other nodes remain cached, resulting in zero overhead.
