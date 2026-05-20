# Task: Implement Border Style Metadata & Junction Precedence

## 1. Objective
Refactor the `resolveBorders` post-processing pass in the `paint` engine. Replace the current boolean `FlagIsBorder` bitmask approach with a dedicated `BorderStyle` enum on `paint.Cell`. This allows the intersection resolver to handle borders of mixed styles (e.g., a thick border crossing a single border) using a strict "Heaviest Style Wins" precedence rule, completely removing the fragile "string sniffing" logic.

## 2. Design & Requirements

### The `BorderStyle` Enum (`paint/types.go`)
- Introduce a new type: `type BorderStyle uint8`.
- Define the constants:
  - `BorderNone BorderStyle = 0`
  - `BorderAscii BorderStyle = 1`
  - `BorderRounded BorderStyle = 2`
  - `BorderSingle BorderStyle = 3`
  - `BorderDouble BorderStyle = 4`
  - `BorderThick BorderStyle = 5`
- Add this `BorderStyle` field to the `paint.Cell` struct (or allocate bits within `CellAttrs` if space optimization is paramount, but a direct field is acceptable for now).
- Remove `FlagIsBorder` from `CellAttrs` as it is now redundant; a cell is a border if `c.BorderStyle != BorderNone`.

### Updating the Painter (`paint/engine.go`)
- Update `drawBorder` (or wherever borders are painted) to set `c.BorderStyle` directly based on the computed style (`s.Border.Style`), instead of just setting `FlagIsBorder`.

### Neighbor-Aware Resolver (`paint/engine.go`)
- Update `resolveBorders(surface Surface)` to fetch the `BorderStyle` for the center cell and its 4 neighbors:
  ```go
  c := surface.CellAt(x, y)
  if c.BorderStyle == BorderNone {
      continue
  }
  up := surface.CellAt(x, y-1).BorderStyle
  down := surface.CellAt(x, y+1).BorderStyle
  left := surface.CellAt(x-1, y).BorderStyle
  right := surface.CellAt(x+1, y).BorderStyle
  ```
- **Precedence Rule:** Determine the `dominantStyle` among the intersecting borders. A simple `max()` function across the 5 values (center, up, down, left, right) will enforce the "Heaviest Style Wins" rule (since Thick=5 > Double=4, etc.).
- **Junction Mapping:** Create a unified `getJunctionGlyph` function that takes the `dominantStyle` and the mask of presence (which neighbors are `!= BorderNone`).
- **Rounded Corners Edge Case:** If the `dominantStyle` is `BorderRounded`, and the junction is a strict corner (mask 5, 6, 9, or 10), return the rounded glyph (`╭`, `╮`, `╰`, `╯`). If it is an intersection (e.g., a T-junction or cross), gracefully fall back to `BorderSingle` junctions, since standard rounded intersecting glyphs do not exist.

## 3. Implementation Steps
1. Define the `BorderStyle` enum in `paint/types.go`.
2. Update `paint.Cell` to include the `BorderStyle` field (and remove `FlagIsBorder` if applicable).
3. Modify the paint functions that plot border lines to attach the correct `BorderStyle` enum to the cell.
4. Rewrite `resolveBorders` in `paint/engine.go` to use the new neighbor-aware logic and compute the `dominantStyle`.
5. Rewrite `getJunctionGlyph` to use the `dominantStyle` to select the dictionary, rather than checking the string content of the center cell.

## 4. Testing Requirements
### 4.1. Unit Tests
- Create a test verifying that when a `BorderThick` line intersects a `BorderSingle` line, the resulting junction uses the `BorderThick` glyph dictionary.
- Verify that rounded corners correctly fallback to single-line glyphs when intersecting another border to form a T-junction.
- Verify that standard junctions (Single vs Single) continue to work as expected.

### 4.2. Documentation
- Update `paint/README.md` to explain the new `BorderStyle` metadata and the "Heaviest Style Wins" precedence rule.
