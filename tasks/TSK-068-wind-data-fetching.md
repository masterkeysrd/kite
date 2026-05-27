# TSK-068: Implement `extras/wind` Async Data Fetching

## Objective
Implement `extras/wind`, a data fetching and caching library for Kitex inspired by React Query. It must simplify async state management (`IsLoading`, `Data`, `Error`), handle request deduping, and provide an ergonomic mutation API with cache invalidation capabilities.

## Requirements

### 1. The Client and Context (`extras/wind/client.go`)
- `type Client struct { ... }`
  - Maintains an internal, thread-safe cache mapping keys of type `any` to their respective data and status.
  - Exposes `func (c *Client) InvalidateQueries(key any)` which performs an exact-match invalidation. Any active queries watching this key must immediately refetch.
- `func NewClient() *Client`
- `var Provider = kitex.FC(...)`
  - A Context Provider component that makes the `Client` available to the component tree.
- `func UseClient() *Client`
  - A hook to extract the client from the context.

### 2. The Query Hook (`extras/wind/query.go`)
- `type Result[T any] struct { Data T, IsLoading bool, IsFetching bool, IsError bool, Error error, Refetch func() }`
- `type Options struct { Enabled bool, StaleTime time.Duration }`
- `func Use[K comparable, T any](key K, fetcher func(context.Context) (T, error), opts ...Options) Result[T]`
  - Uses the `K comparable` constraint for the cache key, ensuring zero-reflection map lookups.
  - Automatically manages the execution of the `fetcher` in a background goroutine.
  - If multiple components mount simultaneously with the exact same key, only *one* network request (goroutine) should fire (request deduping). Both components should receive the result.
  - When the component unmounts, the `context.Context` provided to the `fetcher` must be cancelled.

### 3. The Mutation Hook (`extras/wind/mutation.go`)
- `type MutationContext struct { Client *Client }`
- `type MutationOptions[V any, R any] struct { OnSuccess func(R, V, MutationContext), OnError func(error, V, MutationContext) }`
- `type MutationResult[V any] struct { IsPending bool, IsError bool, Error error, Mutate func(V) }`
- `func UseMutation[V any, R any](mutationFn func(context.Context, V) (R, error), opts ...MutationOptions[V, R]) MutationResult[V]`
  - Manages the execution of `mutationFn` when `Mutate()` is called.
  - Injects a `MutationContext` into the `OnSuccess` and `OnError` callbacks, allowing developers to easily call `ctx.Client.InvalidateQueries(...)` without needing to manually use `wind.UseClient()` in their component body.

### 4. Documentation & Examples
- Add a package-level `doc.go` explaining the caching model, `comparable` keys, and the `Client` context setup.
- Create an example application in `examples/wind_demo/main.go`:
  - Initialize the `Provider` at the root.
  - Define a struct to use as a complex cache key (e.g., `type PodKey struct { Namespace, ID string }`).
  - Demonstrate fetching data with `wind.Use`.
  - Demonstrate a button that triggers a `UseMutation`, which upon success, calls `InvalidateQueries` to trigger a refetch of the data.

## Testing Requirements
- Unit tests verifying request deduping (firing multiple `wind.Use` hooks simultaneously with the same key should only invoke the mock fetcher once).
- Unit tests verifying that context cancellation occurs when a component is "unmounted" (simulated via the headless testenv).
- Integration test ensuring `InvalidateQueries` correctly transitions an active `wind.Use` component back into an `IsFetching` state and updates the data.