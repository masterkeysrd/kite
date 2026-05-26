# Task: Screen-Space Border Intersection Resolver

## 1. Objective
Replace the hardcoded `+` corner logic with a global post-processing pass. This pass will scan the painted `Surface` at the end of every frame, find all framework-drawn borders, and intelligently replace them with the correct Unicode box-drawing junctions (e.g., `├`, `┴`, `┼`) based on their neighbors. This allows sidebars, tables, and cards to magically connect without explicit configuration.

## 2. Design & Requirements

### The `FlagIsBorder` Internal Attribute (`paint/types.go`)
- We have added `FlagIsBorder` to `CellAttrs`. 
- **Requirement:** Any time the layout engine or `paint.Engine` explicitly draws a structural border line (`│` or `─`), it must set the `FlagIsBorder` bitmask on that cell.
- **Strict Rule:** Text nodes and user content must *never* have this flag set, even if the user types a `|`. This prevents the engine from mangling ASCII art or user text.

### Surface Interface Updates (`paint/types.go` & `paint/framebuffer.go`)
- Add `CellAt(x, y int) Cell` to the `Surface` interface.
- Implement `CellAt` on `FrameBuffer`. Return a zero-value `Cell` (with `FlagIsBorder == 0`) if the coordinates are out of bounds.
- Implement `CellAt` on `clippedSurface`. If out of bounds, delegate to `cs.fb.CellAt` but ensure you still check the global boundaries.

### The Resolver Pass (`paint/engine.go`)
- Create a new private method: `func (p *PaintEngine) resolveBorders(surface Surface)`
- Call this method at the very end of `PaintEngine.Paint()`, after all fragments have been recursively painted.
- **Algorithm:**
  1. Iterate over every `(x, y)` coordinate in `surface.Bounds()`.
  2. Retrieve the cell: `c := surface.CellAt(x, y)`.
  3. If `c.Attrs & FlagIsBorder == 0`, `continue`.
  4. Check the 4 cardinal neighbors using `CellAt`. Determine if they are *also* borders (i.e., they have `FlagIsBorder`).
  5. Pass the 4 boolean results (up, down, left, right) to a mapping function that returns the correct Unicode character (e.g., `Up && Left && Right` returns `┴`).
  6. **Style Matching:** To determine *which* style the junction should use (Single, Double, Rounded, Thick), read the `c.Content` or extract the `BorderStyle` if you attached it to the cell. *Simplest approach:* Use a standard Single-line junction for intersections, or use the style of the current cell `c` to map to the correct junction dictionary.
  7. Update the cell: `c.Content = newJunction; surface.Set(x, y, c)`.

## 3. Implementation Steps
1. Update `paint/types.go` and `paint/framebuffer.go` to add and implement `CellAt(x, y int) Cell`.
2. Update `TSK-009`'s implementation of `drawBorder` so that every border cell painted includes `Attrs: FlagIsBorder`.
3. Build the intersection lookup table (a function that maps 4 booleans to a Unicode character).
4. Implement `resolveBorders` in `paint/engine.go`.
5. Call `resolveBorders` at the end of `Paint`.

## 4. Testing Requirements
### 4.1. Unit Tests
- [x] Test `resolveBorders` against a simulated `FrameBuffer` where a vertical line and a horizontal line cross. Verify the center cell is replaced with `┼`.
- [x] Verify that a `FlagIsBorder` cell next to a normal text `|` cell does *not* form a junction.

### 4.2. Integration Tests
- [x] Run the examples (like `examples/app1`) and verify that sibling boxes placed flush against each other form `┬` and `┴` junctions seamlessly.

### 4.5. Documentation
- [x] Update `paint/README.md` to explain the two-phase pipeline (Recursive Fragment Paint -> Screen-Space Border Resolution).
