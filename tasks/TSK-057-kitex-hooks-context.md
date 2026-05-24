# TSK-057: Kitex Reactive Hooks and Execution Context

## Description
Implement the core generic Functional Component wrappers (`kitex.FC`, `kitex.FCC`) and the internal hook execution stack for the `kitex` package.

## Requirements
- **Directory Structure:** Create `extras/kitex` (or place directly in `extras/kitex/`).
- **Component Node:** Define `ComponentNode[T]` that implements the VDOM `Node` interface. It must store its `Props` and a `RenderFn`.
- **FC Wrappers:** Implement `FC[P any](name string, render func(P) Node) func(P) Node`.
- **FCC Wrapper:** Implement `FCC[P any](name string, render func(P) Node) func(P, ...Node) Node` for components that accept children.
- **Execution Stack:** Implement a thread-safe (or clearly documented single-threaded) global/package-level stack (`[]*ComponentNode`) that tracks the currently rendering component.
- **Hook Primitive (`UseState`):** Implement `UseState[T any](initial T) (get func() T, set func(T))`. 
  - `UseState` must look at the top of the execution stack to find its owning `ComponentNode`.
  - It must initialize the state on the first render and retrieve it on subsequent renders based on call order (React hook rules apply).
- **Update Trigger:** The `set` function returned by `UseState` must trigger a Reconciler pass (which will be fully implemented in TSK-058) by flagging the component as dirty.

## Testing
- Unit tests verifying that `FC` returns the correct `ComponentNode` wrapper.
- Unit tests verifying that `UseState` correctly persists values across simulated render cycles.
- Verify that `UseState` panics or returns an error if called outside of a component render phase.
