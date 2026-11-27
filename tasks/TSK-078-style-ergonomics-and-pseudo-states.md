# Task: TSK-078 — Core Style Ergonomics & Pseudo-State Cascade

**ADR:** 035-core-and-kitex-ergonomics

## 1. Objective
Add CSS-equivalent color constructors (`RGB`, `RGBA`, `Hex`, `HexInt`), conditional style composition (`When`), and native pseudo-state cascade support (`:hover`, `:focus`, `:active`, `:disabled`) across `style`, `internal/dom`, `internal/styler`, and `engine`.

## 2. Design & Requirements
- **Color Helpers (`style/color.go`):**
  - `RGB(r, g, b uint8) color.Color` → `color.RGBA{R: r, G: g, B: b, A: 255}`.
  - `RGBA(r, g, b, a uint8) color.Color` → `color.RGBA{R: r, G: g, B: b, A: a}`.
  - `ParseHex(hex string) (color.Color, error)` and `Hex(hex string) color.Color` supporting `#RGB`, `RGB`, `#RRGGBB`, `RRGGBB`, `#RRGGBBAA`, `RRGGBBAA` (case-insensitive). On invalid input, `ParseHex` returns an error and `Hex` returns opaque black (`color.RGBA{A: 255}`) without panicking.
  - `HexInt(v uint32) color.Color` interpreting `0xRRGGBB` with `A: 255`.
- **Conditional Style Merging (`style/style.go`):**
  - `(s Style) When(cond bool, other Style) Style` — returns `s.Merge(other)` when `cond` is true, or `s` unchanged when false.
- **Pseudo-State Cascade (`style`, `internal/dom`, `internal/styler`, `engine`):**
  - Define `type PseudoState uint8` bitmask constants: `PseudoNone = 0`, `PseudoHover = 1 << iota`, `PseudoFocus`, `PseudoActive`, `PseudoDisabled`.
  - Define `type PseudoRule struct { State PseudoState; Style Style }` stored on `style.Style` (`pseudoRules []PseudoRule`, `hasPseudoRules bool`) symmetrically to `MediaRule`.
  - Add fluent builders on `style.Style`: `Hover(Style) Style`, `Focus(Style) Style`, `Active(Style) Style`, `Disabled(Style) Style`, `Pseudo(PseudoState, Style) Style`, `HasPseudoRules() bool`, and `EvaluatePseudo(active PseudoState) Style` (merges rules where `active&rule.State == rule.State` in order, clearing `pseudoRules`/`hasPseudoRules` on the returned `Style`).
  - Update `Style.Merge`, `Style.Equal`, and `validateStyle` (`style/sheet.go`) to propagate, compare, and validate `pseudoRules`.
  - Track transient `hovered` and `active` state on logical DOM elements (`internal/dom`) and expose read/write helpers via `internal/dom` so `styler.Resolver` can compute the element's current `PseudoState` (`PseudoHover` if hovered, `PseudoFocus` if `doc.CurrentFocus()` matches the element or its outer wrapper, `PseudoActive` if active, `PseudoDisabled` if `dom.Disableable` reports `IsDisabled()`).
  - In `internal/styler/resolver.go`: evaluate `authorStyle = authorStyle.EvaluateMedia(viewportSize).EvaluatePseudo(pseudoState)` at Layer 4 (`OriginAuthor`).
  - In `engine` / `focus`: call `MarkStyleDirty()` whenever an element's `hovered` (`EventMouseEnter`/`EventMouseLeave`), `active` (left `EventMouseDown` until next `EventMouseUp`, even if released off-target), or `focus` (`EventFocus`/`EventBlur` / `focus.Manager`) changes.
- **Rules:**
  - Strict package isolation: `/style` must remain zero-dependency on other Kite packages (`dom`, `render`, `event`).
  - Update outdated struct-literal doc comments in `style/style.go` and `style/sheet.go` to use `style.S()`.

## 3. Implementation Steps
1. Implement `RGB`, `RGBA`, `ParseHex`, `Hex`, and `HexInt` in `style/color.go`.
2. Implement `When`, `PseudoState`, `PseudoRule`, `Hover`, `Focus`, `Active`, `Disabled`, `Pseudo`, `HasPseudoRules`, and `EvaluatePseudo` in `style/`, updating `Merge`, `Equal`, and `validateStyle`.
3. Add `hovered` and `active` state accessors on `internal/dom` elements (respecting `ADR-0036` outer element identity).
4. Wire `EvaluatePseudo` into `internal/styler/resolver.go`.
5. Wire hover, active, and focus transitions in `engine` (and `internal/focus` if needed) to update state and call `MarkStyleDirty()`.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test `RGB`, `RGBA`, `Hex` (`#rgb`, `#rrggbb`, `#rrggbbaa`, no `#`, invalid strings), `ParseHex`, and `HexInt` in `style/style_test.go` or `style/color_test.go`.
- [ ] Test `Style.When`, `Hover`/`Focus`/`Active`/`Disabled`, `EvaluatePseudo`, `Merge`, `Equal`, and `NewSheet` validation.
- [ ] Test `styler.Resolver` pseudo-state resolution and cache invalidation on state changes in `internal/styler/resolver_test.go`.

### 4.2. Integration Tests
- [ ] Verify end-to-end hover, focus, active, and disabled style updates through `testenv` / `engine` without requiring manual event listeners in user code.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test (`// Regression tests for Style PseudoStates — covers TSK-078.`) verifying `:hover`, `:focus`, and `:active` cleanup when pointer leaves or releases outside the target.

### 4.4. Benchmarks
- [ ] Add `testing.B` benchmarks in `style/` and `internal/styler/resolver_bench_test.go` verifying zero regressions on styles without pseudo-rules and fast resolution with pseudo-rules.
