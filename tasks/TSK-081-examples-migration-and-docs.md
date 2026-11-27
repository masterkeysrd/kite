# Task: TSK-081 — Examples Migration & Documentation Update

**ADR:** 035-core-and-kitex-ergonomics
**Depends on:** TSK-078, TSK-079, TSK-080

## 1. Objective
Migrate `examples/` to showcase the new ergonomic APIs and compile cleanly with the typed event breaking changes, and update `AGENTS.md` and `README.md` documentation.

## 2. Design & Requirements
- **Examples Migration (`examples/**`):**
  - Update all examples under `examples/` for the typed `ElementProps` event signatures (`*event.MouseEvent`, `*event.KeyEvent`, etc.).
  - Modernize flagship examples (`examples/app1`, `examples/kitex_demo`, `examples/kitex_autocomplete_demo`, `examples/form_demo`, `examples/flight_demo`, `examples/wind_demo`) to demonstrate `style.Hex`/`RGB`, `:hover`/`:focus` pseudo-states (e.g. replacing manual `HoverButton` hover state in `examples/kitex_demo`), `OnValueChange`, `OnPress`, and `kitex.Textf`.
- **Documentation Maintenance:**
  - Update `AGENTS.md` (Concern → File Map) to reflect the split `extras/kitex` files and pseudo-state cascade.
  - Update `README.md`, `extras/kitex/README.md`, and `extras/flight/README.md` to replace outdated `style.Style{...}` struct literals and `func(e event.Event)` signatures with `style.S()`, `style.Hex`, and typed event handlers.

## 3. Implementation Steps
1. Migrate all `examples/**/*.go` and example tests so `go build ./examples/...` and `go test -timeout 30s ./...` pass cleanly.
2. Update `AGENTS.md`, `README.md`, `extras/kitex/README.md`, and `extras/flight/README.md`.

## 4. Testing Requirements
### 4.1. Unit & Integration Tests
- [ ] Full workspace test run `go test -timeout 30s ./...` and `go build ./examples/...` passing with zero errors.
