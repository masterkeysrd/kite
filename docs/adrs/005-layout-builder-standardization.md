# ADR 005: Layout Builder Standardization

## Status
Accepted

## Context
Our Layout Engine follows a LayoutNG-inspired architecture, taking input nodes and yielding immutable `*layout.Fragment` trees. 
For simple formatting contexts like Block, the generic `BoxFragmentBuilder` handles state tracking (current Y offset) effortlessly. However, complex 1D and 2D formatting contexts (`display: flex`, `display: table`) require significant mutable state during calculation:
- **Flex:** Iterating children to determine base sizes, chunking into wrapped lines, calculating remaining free space, growing/shrinking items based on fractions, and running alignment algorithms across cross/main axes.

Currently, `FlexAlgorithm` manages all these array mutations, geometry conversions, and distribution loops directly inside its implementation file. This blurs the line between the *Layout Strategy* and the *State Machine*.

## Decision
We will standardize the Builder Pattern across all complex Layout Formatting Contexts. Following the precedent set by `InlineItemsBuilder` and `TableFragmentBuilder` (ADR 004), the `FlexAlgorithm` will be refactored to delegate state mutation to a dedicated `FlexLineBuilder`.

### Architecture Rule:
- **Algorithm (Coordinator):** Responsible for traversing the DOM, spawning child layout passes, and routing inputs. Must avoid deep array mutations or complex math loops.
- **Builder (State Machine):** Encapsulates the specific algorithm's internal math (e.g., flex-grow distribution, grid sizing overlaps). Exposes highly semantic methods (e.g., `builder.AddFlexItem(node, minMax)`).

## Consequences

### Positive
- **Maintainability:** `FlexAlgorithm.Layout()` will shrink significantly and read like a clear set of procedural steps.
- **Testability:** Complex constraint math (like Flex basis resolution and freeze/restart strategies for min-max bounds) can be tested cleanly in isolation by instantiating the Builder.

### Negative / Trade-offs
- Refactoring `layout/flex.go` will touch core layout code, which always carries a minor risk of regressions. Strict reliance on the existing `flex_test.go` integration tests is required.
