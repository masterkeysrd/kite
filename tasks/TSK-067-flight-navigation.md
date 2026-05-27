# TSK-067: Implement `extras/flight` Stack Navigation

## Objective
Implement `extras/flight`, a stack-based navigation package for Kitex applications. It avoids web-style URL routing in favor of a type-safe stack navigator (push/pop) that naturally fits TUI paradigms and ensures correct focus scoping for active screens.

## Requirements

### 1. Core Types (`extras/flight/flight.go`)
- `type Route interface{}`
  - An empty interface. Developers will use concrete structs to satisfy this, allowing type-safe parameter passing.
- `type Navigator struct { ... }`
  - Needs fields/methods for:
    - `Push(r Route)`
    - `Pop()`
    - `Replace(r Route)` (replaces the top route)
    - `Reset(r Route)` (clears the stack and sets `r` as the only route)

### 2. Context & Hooks (`extras/flight/hooks.go`)
- Setup a Kitex context to provide the `Navigator` to descendants.
- `func UseNavigation() Navigator`
  - A hook that retrieves the `Navigator` from the Kitex context. Panics with a helpful message if called outside of a `flight.Stack`.

### 3. The Stack Component (`extras/flight/stack.go`)
- `type StackProps struct`
  - `InitialRoute Route`
  - `RenderRoute func(Route) kitex.Node`
- `func Stack(props StackProps) kitex.Node`
  - Uses `kitex.UseState` to maintain a `[]Route` representing the history stack.
  - Implements the `Navigator` interface methods to mutate the state slice.
  - Retrieves the active route: `active := stack[len(stack)-1]`.
  - **Crucial:** Wraps the output of `RenderRoute(active)` in a `kitex.Provider` (for the Navigator context) AND a `<focus.Scope>` (to trap keyboard navigation to the currently active screen, preventing interaction with hidden screens in the stack).
  - *Note:* Do not register any implicit keyboard shortcuts (like `Esc`).

### 4. Documentation & Examples
- Add a package-level `doc.go` explaining the "Type-Safe Stack Navigator" pattern and how to use struct type-switches.
- Create an example application in `examples/flight_demo/main.go` that demonstrates:
  - Defining two routes: `HomeRoute{}` and `DetailsRoute{ID string}`.
  - Using a `flight.Stack` at the root.
  - In `HomeView`, explicitly registering a `UseKeyboard` hook to navigate to `DetailsRoute` on 'Enter' (or a button click).
  - In `DetailsView`, extracting the `ID` safely and registering a `UseKeyboard` hook to `nav.Pop()` on 'Esc'.

## Testing Requirements
- Unit tests for the `Stack` component logic (pushing, popping, bounds checking on pop).
- Golden tests / Integration tests ensuring that focus scoping works correctly (i.e., you cannot tab into a button that exists on a route lower in the stack).

## Related Documentation Updates
- Update `README.md` to introduce `flight` as the official navigation solution.