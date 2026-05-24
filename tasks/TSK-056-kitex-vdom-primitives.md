# TSK-056: Kitex VDOM Primitive Wrappers

## Description
Implement the lightweight, fully-typed Virtual DOM primitive wrappers in the `extras/kitex` package. These VDOM nodes act as the declarative, type-safe API for developers and map 1:1 to the real, heavy DOM nodes in the root `element` package during reconciliation.

## Requirements
- **VDOM Node Interface:** Define the base `Node` interface for all kitex VDOM elements in `extras/kitex`.
- **Primitive Definitions:** For every UI component in the root `element` package (e.g., `Button`, `Box`/`Div`, `Text`, `Input`), create a corresponding VDOM representation in `kitex`.
  - Example: `type ButtonProps struct { ID string; Class string; Disabled bool; OnClick func(event.Event) }`
  - Example Factory: `func Button(props ButtonProps, children ...Node) Node`
- **Type Safety:** The property structs (e.g., `ButtonProps`) must provide strict, compile-time typing for the attributes and listeners supported by their corresponding real `element` type.
- **Reconciler Mapping Hook:** Each VDOM primitive must expose an internal mechanism (e.g., `Instantiate(doc dom.Document) element.Element` and `Update(el element.Element, oldProps, newProps Props)`) that allows the Reconciler (TSK-058) to easily map the VDOM properties onto the real `element` package instances.

## Testing
- Unit tests verifying that the factory functions (`kitex.Button(...)`) correctly construct the lightweight VDOM structs.
- Unit tests verifying that the VDOM `Update` mechanism correctly translates VDOM prop changes into mutations on a mock `element.Button`.