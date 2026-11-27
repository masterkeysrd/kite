# 🌀 Wind

Wind is an async data fetching, caching, and state management library for Kitex inspired by React Query. It simplifies async state management (`IsLoading`, `IsFetching`, `IsError`), automates request deduping, cancels outstanding fetchers when components unmount, and provides an ergonomic mutation API with built-in query invalidation.

## ✨ Features

- 🌀 **Declarative Queries**: Fetch async data easily inside functional components with `wind.Use`.
- 📦 **Thread-Safe Caching**: Cache query results with strict `K comparable` keys for zero-reflection map lookups.
- 🏎️ **Request Deduping**: If multiple components mount simultaneously and request the same key, only a single background request is dispatched.
- 🛑 **Automatic Unmount Cancellation**: Instantly cancels the fetcher's `context.Context` when the number of subscribing components for a key drops to zero.
- ⚡ **Cache Invalidation**: Force active queries to refetch in the background via `Client.InvalidateQueries`.
- 🪓 **Ergonomic Mutations**: Execute mutations with state tracking (`IsPending`, `Error`) and context-based invalidation callbacks.
- 🔗 **Bounded Async**: Leverages the `promise` package and the engine's background worker pool for all async operations, ensuring thread safety and main-thread callback routing.

## 🚀 Getting Started

### 1. Setup the Client Provider

Initialize the `wind.Client` and wrap your application root inside `wind.Provider`:

```go
package main

import (
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/wind"
)

func main() {
	client := wind.NewClient()

	app := kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
		return wind.Provider(wind.ProviderProps{Client: client}, /* Your Views */)
	})
	
	// ... Render app ...
}
```

### 2. Fetch Data with `wind.Use`

Use the query hook by passing a `comparable` cache key and an async fetch function:

```go
type PodKey struct {
	Namespace string
	PodName   string
}

type PodInfo struct {
	Status string
	CPU    string
}

var PodViewer = kitex.FC("PodViewer", func(props PodKey) kitex.Node {
	// res holds Data, IsLoading, IsFetching, IsError, Error, and Refetch()
	res := wind.Use(props, promise.WrapWithProps(
		func(ctx context.Context, key PodKey) (PodInfo, error) {
			return fetchPodDetails(ctx, key.Namespace, key.PodName)
		},
	))

	if res.IsLoading {
		return kitex.Text("Loading pod details...")
	}
	if res.IsError {
		return kitex.Text("Error: " + res.Error.Error())
	}

	return kitex.Box.Style(
		style.S().Padding(1),
		kitex.Span.Style(style.S().Bold(true), kitex.Text("Status: " + res.Data.Status)),
		kitex.Text("CPU Usage: " + res.Data.CPU),
		kitex.Button(kitex.ButtonProps{
			OnClick: func(e *event.MouseEvent) { res.Refetch() },
		}, kitex.Text("Manual Refresh")),
	)
})
```

### 3. Mutate and Invalidate with `wind.UseMutation`

Mutate remote data and invalidate cached queries to trigger automated background refetches:

```go
type DeletePodVariables struct {
	Namespace string
	PodName   string
}

var DeletePodButton = kitex.FC("DeletePodButton", func(props PodKey) kitex.Node {
	mutation := wind.UseMutation(
		promise.WrapWithProps(
			func(ctx context.Context, vars DeletePodVariables) (string, error) {
				return apiDeletePod(ctx, vars.Namespace, vars.PodName)
			},
		),
		wind.MutationOptions[DeletePodVariables, string]{
			OnSuccess: func(result string, vars DeletePodVariables, ctx wind.MutationContext) {
				// Invalidate the query key to auto-trigger a background refetch
				ctx.Client.InvalidateQueries(PodKey{
					Namespace: vars.Namespace,
					PodName:   vars.PodName,
				})
			},
		},
	)

	text := "Delete Pod"
	if mutation.IsPending {
		text = "Deleting..."
	}

	return kitex.Button(kitex.ButtonProps{
		OnClick: func(e *event.MouseEvent) {
			mutation.Mutate(DeletePodVariables{
				Namespace: props.Namespace,
				PodName:   props.PodName,
			})
		},
	}, kitex.Text(text))
})
```

## 🛠 API Reference

### Caching Client & Imperative Cache Helpers

- **`NewClient() *Client`**: Instantiates a thread-safe cache store.
- **`GetQueryData[K comparable, T any](client *Client, key K) (T, bool)`**: Reads cached data for `key` if present, populated, and assignable to `T` without creating a cache entry on a miss.
- **`SetQueryData[K comparable, T any](client *Client, key K, data T)`**: Seeds or replaces cached data for `key`, arming the GC timer when unsubscribed and coalescing subscriber notifications on the main thread.
- **`UpdateQueryData[K comparable, T any](client *Client, key K, updater func(prev T, exists bool) T)`**: Atomically reads and updates cached data under the entry lock (ideal for optimistic updates and concurrent mutations) and coalesces subscriber notifications.
- **`Client.InvalidateQueries(key any)`**: Invalidates matching queries. Subscribed components immediately begin background refetches.
- **`Client.CancelQueries(key any)`**: Cancels in-flight queries and streams matching `key` (exact, prefix, or predicate), stops pending throttle timers, clears `IsFetching`, and notifies subscribers.
- **`Client.Close()`**: Idempotently cancels all active queries and streams, stops pending GC/throttle timers, clears the cache, and waits for background work to drain.

### Options & Helpers

- **`Optional[T]` (`Some`, `None`)**: Zero-allocation wrapper used for optional configuration fields (`Options.Enabled`, `Stream.Enabled`, `Stream.Throttle`) so leaving a field unset preserves its default (`Enabled` defaults to `true`, `Throttle` defaults to `16ms`).
- **`Latest[T](_ T, next T) T`**: Convenience reducer for `Stream.Reduce` that replaces previous state with the latest event.

### Hooks

Query and mutation functions must propagate the context supplied by Wind to the
underlying request. `promise.Wrap` and `promise.WrapWithProps` do this while also
honoring scheduler shutdown. Using `promise.New` and only its executor context
does not preserve Wind's cancellation.

- **`Use[K comparable, T any](key K, fetcher func(context.Context, K) *promise.Promise[T], opts ...Options) Result[T]`**:
  - `Options.Enabled` (`Optional[bool]`): Defaults to `true`; set to `wind.Some(false)` to disable automatic fetching on mount.
  - `Options.StaleTime`: Max age before the cache entry is considered stale and re-fetched on mount/dependency change.
- **`UseStream[K comparable, E, S any](key K, spec Stream[E, S]) StreamResult[S]`**:
  - Subscribes to a declarative streaming query via `promise.Spawn` (scheduler-managed lifecycle outside the bounded worker pool) and exposes `Status` (`"connecting"`, `"open"`, `"closed"`, `"error"`). Stopping the engine scheduler or calling `Client.Close()` cancels active streams.
  - `Stream[E, S].Initial`: Seeds state `S` before the first `Fetch` or `Source` emission.
  - `Stream[E, S].Fetch`: Optional snapshot loader executed on connect/reconnect before `Source` opens. If `Fetch` fails, `Source` is not invoked.
  - `Stream[E, S].Source`: State-aware `func(ctx context.Context, state S) iter.Seq2[E, error]` stream producer receiving the current state (seeded from `Initial`, cached `RefetchAppend` state, or `Fetch`) so cursors like `LastEventID` can be forwarded across reconnects.
  - `Stream[E, S].Reduce`: Folds every yielded event `E` into accumulated state `S` synchronously so throttled UI updates never drop events (when `nil` and `E` is assignable to `S`, events replace state directly).
  - `Stream[E, S].Enabled` (`Optional[bool]`): Defaults to `true`; toggling `Some(true)` -> `Some(false)` cancels when no other enabled consumer remains, and `Some(false)` -> `Some(true)` reconnects.
  - `Stream[E, S].Throttle` (`Optional[time.Duration]`): Defaults to `16ms`; values `<= 0` disable timer throttling and coalesce via microtasks.
  - `Stream[E, S].RefetchMode`: `RefetchReplace` (default, keeps cached data visible until first event), `RefetchReset` (resets to `Initial` immediately), or `RefetchAppend` (folds onto existing cached data).
- **`UseMutation[V, R any](mutationFn func(context.Context, V) *promise.Promise[R], opts ...MutationOptions[V, R]) MutationResult[V]`**:
  - State fields: `IsPending`, `IsError`, `Error`.
  - `Mutate(vars V)`: Triggers mutation function.
  - `OnSuccess(result R, variables V, ctx MutationContext)`: Fired when mutationFn succeeds.
  - `OnError(err error, variables V, ctx MutationContext)`: Fired when mutationFn fails.
  - Unmounting cancels all outstanding mutation contexts and suppresses later
    callbacks. This is cooperative cancellation, not a rollback of server-side
    changes. A retained `Mutate` function does not start work after unmount.
