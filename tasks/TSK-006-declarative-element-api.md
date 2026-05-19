# Task: Implement Declarative API for Elements

## 1. Objective
Refactor the `element` package to provide a declarative, SwiftUI-like API for constructing UI trees using variadic children and automatic string boxing (ADR 003).

## 2. Design & Requirements

### Feature Design
- **Struct Renaming:** Rename all structs in `/element` to have an `Element` suffix:
  - `Box` -> `BoxElement`
  - `Text` -> `TextElement`
  - `Span` -> `SpanElement`
  - (Include any new components from TSK-002 and TSK-004 if they exist, or refactor them in those tasks).
- **Global Scratch Document:** Create an unexported global `orphanDocument = dom.NewDocument()` inside the `element` package.
- **Variadic Constructors:** Expose functional constructors matching the original bare names:
  - `func Box(children ...any) *BoxElement`
  - `func Text(data string) *TextElement`
- **Child Processing (`processChildren` helper):**
  - Iterate over `...any` arguments.
  - If `string`, create a `TextElement` and append it.
  - If `dom.Node`, append it.
  - If slice/array of `any` or `dom.Node`, recursively flatten and append.
  - If `style.Style` (optional stretch goal), apply it to the parent.

### Rules
- Ensure the variadic constructors use the `orphanDocument` so users do not pass `doc`.
- Rely on TSK-005 (DOM Adoption) so the engine safely adopts the orphan nodes when `engine.Mount()` is called.

## 3. Implementation Steps
1. Rename structs and existing constructors in `element/box.go`, `element/span.go`, `element/text.go`.
2. Add `orphanDocument` to a central file (e.g., `element/element.go`).
3. Write `processChildren(parent dom.Element, children []any)`.
4. Create the new declarative global functions `Box()`, `Span()`, etc.

## 4. Testing Requirements

### 4.1. Unit Tests
- [ ] Test case 1: `Box("Hello", Span("World"))` successfully creates a parent box containing a Text node and a Span node.
- [ ] Test case 2: `Box([]any{"a", "b"})` successfully flattens slices and appends both string nodes.

### 4.2. Integration Tests
- [ ] Build a tree entirely using the declarative syntax, mount it to the Engine, and assert the tree successfully connects and renders.

### 4.3. Regression Tests (at `./tests/regressions/`)
- [ ] Add a regression test to ensure nested variadic slices (e.g., passing a slice of nodes to `Box`) do not crash the `processChildren` reflection loop.

### 4.4. Benchmarks
- [ ] N/A.

### 4.5. Documentation
- [ ] Update `README.md` to showcase the new declarative API in the primary "Usage Example".
- [ ] Update `AGENT.md` to instruct AI assistants to write UI code using the new declarative syntax instead of imperative DOM calls.
