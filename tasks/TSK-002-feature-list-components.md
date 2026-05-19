# Task: Implement List and ListItem DOM Components

## 1. Objective
Create the logical DOM wrappers (`UnorderedList`, `OrderedList`, and `ListItem`) in the `element` package to provide easy-to-use list components with correct default styles and inheritance behavior.

## 2. Design & Requirements

### Feature Design
- **`element.UnorderedList` (`ul`)**:
  - Sets default tag `"ul"`.
  - Exposes an `ElementDefaultStyle` with `Display: Block`, `ListStyleType: Disc`, and `Padding.Left: 2` (to allow space for markers in the terminal grid).
- **`element.OrderedList` (`ol`)**:
  - Sets default tag `"ol"`.
  - Exposes an `ElementDefaultStyle` with `Display: Block`, `ListStyleType: Decimal`, and `Padding.Left: 3`.
- **`element.ListItem` (`li`)**:
  - Sets default tag `"li"`.
  - Exposes an `ElementDefaultStyle` with `Display: ListItem`.
  
### Rules
- **Style Inheritance (`style/resolver.go`)**: 
  - Update **Layer 3** in `resolver.go` to explicitly inherit `ListStyleType`. This ensures the `<ol>` or `<ul>` correctly passes its style type down to the `<li>` children without manual assignment.
- **Component Base**: 
  - All new elements must embed `elementBase[Self]` properly, mirroring the pattern in `element/box.go`.

## 3. Implementation Steps
1. **Style Inheritance**: Modify `style/resolver.go` around Layer 3 (`if parent != nil { ... }`) to copy `c.ListStyleType = parent.ListStyleType`.
2. **`ElementDefaultStyle` Plumbing**: Ensure that `elementBase` or the concrete elements expose an `ElementDefaultStyle()` method so that the `render` tree can pull these tag-specific defaults during the Style resolution phase (Layer 2).
3. **Create `element/list.go`**: Implement `UnorderedList`, `OrderedList`, and `ListItem` structs. 
4. **Implement Constructors**: Write `NewUnorderedList`, `NewOrderedList`, and `NewListItem`.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: `NewUnorderedList` sets the tag to `ul` and its default style contains `Padding.Left == 2` and `ListStyleType == Disc`.
- [ ] Test case 2: `NewOrderedList` sets the tag to `ol` and its default style contains `Padding.Left == 3` and `ListStyleType == Decimal`.
- [ ] Test case 3: `NewListItem` sets the tag to `li` and its default style contains `Display == DisplayListItem`.
- [ ] Test case 4: (Style Resolver) Verify that a child `ListItem` inside an `OrderedList` resolves its computed `ListStyleType` to `Decimal` via inheritance.

### 4.2. Integration Tests
- [ ] Construct a tree with an `<ol>` containing two `<li>` elements. Run a full `engine` style resolution pass and assert that the computed styles match expectations.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Ensure that changing `ListStyleType` dynamically on a `<ol>` triggers `DirtyStyle` on its `<li>` children and updates their computed values correctly.

### 4.4. Benchmarks
- [ ] N/A. (DOM creation is cheap; Style resolving is already benchmarked).

### 4.5. Documentation
- [ ] Update `README.md` to show a short code snippet of how to create `ul`, `ol`, and `li` components.
- [ ] Update `element/doc.go` to list the new list components.
