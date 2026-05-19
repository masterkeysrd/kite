# Task: Border Rendering Engine and Fluent DevX API

## 1. Objective
Rewrite the `paint.Engine`'s border drawing logic to properly utilize Unicode box-drawing characters, per-edge colors, and correct terminal widths (boolean presence). Overhaul the Developer Experience (DevX) by making `style.Border` an immutable fluent API.

## 2. Design & Requirements

### Fluent API & Struct Changes (`style/border.go`)
- **TUI Width Reality:** In a terminal, borders take exactly 1 cell. The `int` width is misleading. Replace `Width EdgeValues[int]` with `Edges EdgeValues[bool]` to represent the on/off visibility of each edge.
- **Naming Collisions:** To support fluent modifier methods like `.Color()` without colliding with struct fields, update the `Border` struct to use pluralized names:
  ```go
  type Border struct {
      Edges  EdgeValues[bool]
      Styles EdgeValues[BorderStyle]
      Colors EdgeValues[color.Color]
      Glyphs BorderGlyphs
  }
  ```
- **Fluent Modifiers:**
  - `func (b Border) Color(c color.Color) Border` (Sets all edges in `Colors`).
  - `func (b Border) Style(s BorderStyle) Border` (Sets all edges in `Styles`).
  - `func (b Border) Top(visible bool) Border` (and Bottom, Left, Right).
  - `func (b Border) CornerOverride(tl, tr, bl, br string) Border`
- **Optional Helper:**
  - `func (b Border) Some() Optional[Border] { return Some(b) }`
- **Static Constructors:** 
  - `func SingleBorder() Border`
  - `func DoubleBorder() Border`
  - `func RoundedBorder() Border`
  - `func EmptyBorder() Border`
  - These should return a `Border` struct with `Edges: EdgeAll(true)` and the corresponding `Styles`.

### Glyph Definitions (`style/border.go`)
- Define the `BorderGlyphsMap` directly in `style/border.go`:
  ```go
  var BorderGlyphsMap = map[BorderStyle]BorderGlyphs{
      BorderSingle:  {H: "─", V: "│", TL: "┌", TR: "┐", BL: "└", BR: "┘"},
      BorderDouble:  {H: "═", V: "║", TL: "╔", TR: "╗", BL: "╚", BR: "╝"},
      BorderRounded: {H: "─", V: "│", TL: "╭", TR: "╮", BL: "╰", BR: "╯"},
      BorderThick:   {H: "━", V: "┃", TL: "┏", TR: "┓", BL: "┗", BR: "┛"},
      BorderASCII:   {H: "-", V: "|", TL: "+", TR: "+", BL: "+", BR: "+"},
  }
  ```

### Layout Engine Updates
- Update `layout/` logic that currently reads `s.Border.Width`. It must now read `s.Border.Edges` and treat `true` as `1` cell and `false` as `0` cells when computing geometry.

### Rendering Engine (`paint/engine.go`)
- Completely rewrite `drawBorder()`.
- **Edge Drawing:**
  - Loop through Top, Right, Bottom, Left.
  - If `border.Edges` is `true` for that side, draw the line using `BorderGlyphsMap[border.Styles.Edge].H` or `.V`.
  - Apply the specific `border.Colors.Edge` to the characters.
- **Corner Drawing:**
  - A corner is drawn *only* if the two intersecting edges are both `true` in `border.Edges`.
  - Check `border.Glyphs.EffectiveTL()`. If it returns a non-empty string, use it. Otherwise, look up the corner glyph from `BorderGlyphsMap` using the Top edge's `BorderStyle`.

## 3. Implementation Steps
1. Update `style.Border` fields to `Edges`, `Styles`, `Colors`, and `Glyphs`.
2. Add `BorderGlyphsMap` and static constructors (`SingleBorder()`, etc.) to `style/border.go`.
3. Implement immutable modifier methods on `style.Border`.
4. Update `examples/` to use the new fluent syntax: `Border: style.SingleBorder().Color(myColor).Some(),`
5. Fix layout algorithms (`layout/block.go`, etc.) to interpret `border.Edges` boolean values as 1 or 0 integers.
6. Rewrite `paint.Engine.drawBorder()` to respect the new structure, per-edge colors, and Unicode characters.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test that `SingleBorder().Color(red)` returns a struct with `Colors` properly set via `EdgeAll()`.
- [ ] Test the `paint` engine manually by asserting the resulting `Surface` cells contain the correct Unicode characters.

### 4.2. Integration Tests
- [ ] Run the UI examples to visually verify that single, double, and rounded borders render flawlessly and connect correctly at the corners.

### 4.5. Documentation
- [ ] Update `element/doc.go` or `README.md` to demonstrate the new fluent Border API.
