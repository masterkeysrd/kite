# Task: Move Styling State to Logical DOM (Stateless Render Objects)

## 1. Objective
Refactor the architecture to strictly enforce that the Logical DOM is the single source of truth for both author-set styles (`RawStyle`) and natural element defaults (`ElementDefaultStyle`). Render objects must become stateless proxies for these properties.

## 2. Design & Requirements
- **Feature Design:**
  - `render.Object` currently stores `rawStyle` and `elementStyle`. This duplicates state and blurs the lines of ownership between logical and rendering layers.
  - The `dom.Element` (and its underlying implementations like `elementBase`) already hold `rawStyle` and `defaultStyle`.
  - The goal is to remove style storage from `render.BaseRender` and have its `RawStyle()` and `ElementDefaultStyle()` methods cast its underlying `LogicalNode()` to interfaces (`Stylable`) to retrieve the values dynamically.
- **Rules:**
  - **No Style Storage in Render:** Remove `rawStyle` and `elementStyle` fields from `render.BaseRender`.
  - **Remove Mutators:** Remove `SetRawStyle(style.Style)` and `SetElementDefaultStyle(style.Style)` from the `render.Object` interface and the `BaseRender` implementation.
  - **Interface Check:** Use interface casting inside `BaseRender.RawStyle()` and `BaseRender.ElementDefaultStyle()` to fetch styles safely from `LogicalNode()`.
  - **Preserve Cascade:** Keep `RawStyle` and `ElementDefaultStyle` as distinct layers in the interface to ensure `style.Resolver` correctly applies the CSS inheritance cascade.
  - **Engine Sync Update:** Remove the explicit syncing of styles (`SetRawStyle`, `SetElementDefaultStyle`) inside `engine/engine.go` during render object creation.
  - **Element Update:** When a user calls `Style(s)` on an element, the element must only update its own `rawStyle` and call `ro.MarkDirty(render.DirtyStyle)` (instead of pushing the state to the render object).
  - Rename `ElementDefaultStyle()` to `DefaultStyle()` across the codebase (`style.StyleNode`, `elementBase`, `render.Object`, etc.). Note: Take care not to confuse this with the package-level `style.DefaultStyle()` which returns the global root baseline.

## 3. Implementation Steps
1. **Rename Interface Methods:** Rename `ElementDefaultStyle()` to `DefaultStyle()` in `style.StyleNode`, `render.Object`, and all elements (`elementBase`, `fakeNode` in tests, etc.).
2. **Clean up `render.Object` & `BaseRender`:**
   - Delete `rawStyle` and `elementStyle` fields from `BaseRender`.
   - Delete `SetRawStyle` and `SetElementDefaultStyle` from the `render.Object` interface and `BaseRender`.
   - Update `BaseRender.RawStyle()` and `BaseRender.DefaultStyle()` to proxy to `b.LogicalNode()`.
3. **Clean up `engine/engine.go`:**
   - In the sync phase (`OnRenderObjectCreated` logic), remove calls pushing styles to the new render object.
4. **Update `elementBase.Style(s)`:**
   - Simply assign `b.rawStyle = s` and call `ro.MarkDirty(render.DirtyStyle)` if the render object exists.
5. **Fix Tests:** Update all `render/` and `engine/` tests that relied on `SetRawStyle`. Use the logical DOM elements to configure styles instead.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Render Object Proxy: Verify `BaseRender.RawStyle()` correctly returns the style from an attached logical node.
- [ ] Cascade Preservation: Ensure `style.Resolver` tests still pass, proving that the separation between `DefaultStyle` and `RawStyle` still enforces proper inheritance rules.

### 4.2. Integration Tests
- [ ] Complete render cycle: Ensure that modifying a style on a logical DOM node successfully propagates the `DirtyStyle` flag and forces a re-layout/re-paint during the engine loop.

### 4.5. Documentation
- [ ] Update `render/doc.go` to explicitly mention that render objects do not own style state.
