# Task: Flex Layout Builder Refactor

## 1. Objective
Refactor `layout/flex.go` to extract the complex math, item chunking, and space distribution logic out of `FlexAlgorithm` and into a dedicated `FlexLineBuilder`. This standardizes the layout engine architecture as dictated by ADR 005.

## 2. Design & Requirements

### The `FlexLineBuilder` (`layout/flex_builder.go`)
Create a new builder that encapsulates the mutable state required for resolving flex lines:
1. **Item Collection:** Maintain the `[]FlexItem` slice internally. Expose `AddItem(node Node, intrinsicSizes MinMaxSizes, margin MarginStrut)`.
2. **Line Chunking:** Implement `ComputeLines(availableMainSize int, wrap bool)` which groups the items into `[]flexLine` slices based on their flex base size.
3. **Space Distribution:** Move the "freeze and restart" algorithm (used to distribute `flex-grow` and `flex-shrink` while respecting `min-width`/`max-width` bounds) entirely into the builder. 
   - Expose: `ResolveFlexibleLengths(lineIndex int)`.
4. **Alignment:** Move the math for `JustifyContent`, `AlignItems`, and `AlignContent` into the builder.
   - Expose: `AlignLine(lineIndex int)` and `AlignCrossAxis(containerHeight int)`.
5. **Output:** Provide an iterator or a `ToFragments()` method that hands the resolved physical offsets back to the main Algorithm so it can construct the final Fragment tree.

### `FlexAlgorithm` Cleanup
- The core algorithm will simply:
  1. Instantiate `FlexLineBuilder`.
  2. Iterate over `Node.Children()`, measure their intrinsic bounds, and call `builder.AddItem()`.
  3. Call `builder.ComputeLines()`, `builder.ResolveFlexibleLengths()`, and the alignment passes.
  4. Collect the fully resolved items from the builder and construct the `BoxFragmentBuilder` to yield the final `*Fragment`.

## 3. Implementation Steps
1. Create `layout/flex_builder.go`.
2. Migrate the unexported structs (`flexItem`, `flexLine`) from `flex.go` into the builder file, making them public to the package if necessary.
3. Migrate the space distribution logic (growing/shrinking items) into the builder methods.
4. Refactor `FlexAlgorithm.Layout()` to use the new builder.

## 4. Testing Requirements
### 4.1. Unit Tests
- [ ] Test `FlexLineBuilder` in isolation: Add 3 items with `flex-grow: 1`. Assert that `ResolveFlexibleLengths` correctly assigns exactly 1/3rd of the available space to each.
- [ ] Test the "Freeze and Restart" logic within the builder: Ensure an item with `max-width` correctly freezes at its maximum and the remaining space is redistributed to its siblings.

### 4.2. Integration Tests
- [ ] Run the existing tests in `layout/flex_test.go` and `examples/flex/`. They must pass without modification, as the external observable behavior of Flexbox cannot change.

### 4.5. Documentation
- [ ] Add `doc.go` comments to the `FlexLineBuilder` explaining its role as the state machine for the flex algorithm.
