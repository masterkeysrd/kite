# Task: Segregate and Simplify `render.Object`

## Objective
Reduce the bloated `render.Object` interface by removing proxy methods that simply defer to the logical `dom.Node`. 

## Requirements
1. **Remove Style Proxies from `render.Object`:**
   - Remove `RawStyle()`, `DefaultStyle()`, and `IntrinsicStyle()` from the `render.Object` interface.
   - Note: Keep `ComputedStyle()` and `SetComputedStyle(*style.Computed)` on the `render.Object`, as it remains the owner of the resolved mathematical state.

2. **Refactor the Style Engine (`style.Resolver`):**
   - The Style Resolver currently accepts a `style.StyleNode` (which was implemented by `render.Object`).
   - Refactor the resolver/cascade to accept the `dom.Node` directly, or have `dom.Node` implement the required `style.StyleNode` interface natively.
   - Update `engine.Engine`'s style phase to pass the `dom.Node` into the resolver, take the resulting `*style.Computed`, and apply it to the corresponding `render.Object` (using the map introduced in TSK-071).

3. **Cleanup:**
   - Remove the `StyleNode` proxy implementations from `render.Box` and `render.Text`.

## Tests to Verify
- Run `go test ./style/...` to ensure the cascade algorithm works directly against the DOM elements.
- Run `go test ./engine/...` to verify the Style Phase pipeline correctly applies the computed output to the render objects.

## Documentation Updates
- None required.