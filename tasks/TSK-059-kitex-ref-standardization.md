# TSK-059: Kitex Ref Standardization

## Objective
Standardize the terminology for references in the `kitex` package and implement the `UseRef` and `CreateRef` public hooks. The internal concept of a component reference will be renamed to `componentRef`, while the `liveNode` associated with VDOM nodes will be correctly mapped to the term `ref`. 

## Requirements

### 1. Internal Terminology Cleanup
In `extras/kitex/kitex.go` and related files:
- Find the `ComponentNode[P]` struct.
- Rename the unexported field `ref` (of type `*componentRef`) to `componentRef`. 
- Rename the unexported field `liveNode` to `ref` (of type `dom.Node`) inside `ComponentNode[P]`, `elementNode[P]`, and `textNode`.
- Update all method usages across the VDOM constructors, component lifecycle, and reconciler to reflect these renames.

### 2. Ref API
In `extras/kitex/hooks.go` and `extras/kitex/kitex.go`:
- Define a generic `Ref[T any]` interface or type (e.g. `*RefObject[T]`) that holds a mutable `Current T` value.
- Introduce `CreateRef[T any]() Ref[T]` for creating refs outside the render cycle.
- Introduce `UseRef[T any](initial T) Ref[T]` which returns a persistent `Ref[T]` using the component hook state mechanism. Modifying the `Ref` should *not* trigger `MarkDirty()`.
- Introduce an unexported `refSetter` interface (e.g., `type refSetter interface { set(dom.Node) }`). `Ref[T]` where `T` is `dom.Node` or a specific element type should implement this.

### 3. Wiring up the DOM Nodes
- Modify `ElementProps` (and derived props structs if they do not embed `ElementProps`) to include an unexported or safely typed mechanism for taking a `Ref` (e.g., a `Ref refSetter` field in `ElementProps`).
- In `elementNode.Instantiate()` and `elementNode.Update()`, if a `refSetter` is provided in the node's properties, call its `set()` method with the instantiated or updated real DOM node (which is now assigned to the VDOM node's internal `ref` field).

### 4. Testing
- Implement table-driven unit tests in `extras/kitex/hooks_test.go` and `extras/kitex/kitex_test.go`.
- Write tests validating that `UseRef` correctly persists state across `Update()` calls without triggering `OnComponentDirty`.
- Write tests confirming that passing a `Ref` to an element (like `Box` or `Button`) correctly populates the `Current` value with the actual instantiated `dom.Element` when rendered via `Instantiate()` or `reconciler.Render()`.

## Rollout
- As this is a pre-release internal cleanup, no backwards-compatibility wrappers are required for `liveNode` or `componentRef`. Update the code natively.
- DO NOT mark this task as Done until all tests pass and a user confirms completion.
