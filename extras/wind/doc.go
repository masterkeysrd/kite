// Package wind implements an asynchronous data fetching and caching library for Kitex applications,
// inspired by React Query.
//
// It provides automated cache management, request deduping, background updates, and cache
// invalidation.
//
// # Cache Architecture
//
// Cache state is managed by the Client. The Client maintains a thread-safe map of query keys.
// When a component initiates a query using Use, it retrieves the client instance from the
// context Provider. If the query does not exist in the cache, or is considered stale, a background
// goroutine is spawned to execute the fetcher. Callers can also read, seed, and atomically update
// cached values imperatively via GetQueryData, SetQueryData, and UpdateQueryData, or cancel
// in-flight queries and streams via Client.CancelQueries. Call Client.Close on shutdown to cancel
// active queries and streams, stop pending timers, and wait for background goroutines to finish.
//
// # Streaming & Accumulation
//
// UseStream subscribes to declarative streams configured by Stream[E, S] on dedicated goroutines
// outside the bounded promise worker pool so long-lived streams never starve short-lived queries.
// A Stream can optionally load an initial snapshot via Fetch before opening a state-aware Source
// iter.Seq2[E, error] iterator (which receives the current state so cursors such as LastEventID can
// be forwarded across connects and reconnects). Incoming events are folded into state S via Reduce
// (or Latest when replacing state directly) before UI notification throttling, ensuring rapid event
// bursts are never dropped. Stream supports Optional[bool] Enabled, Optional[time.Duration] Throttle,
// and RefetchMode (RefetchReplace, RefetchReset, RefetchAppend).
//
// # Request Deduping
//
// If multiple components mount simultaneously with the exact same cache key, they subscribe to
// the same cache entry. Only a single fetcher goroutine is fired. Once the fetch completes,
// all subscribed components are notified and re-render with the fetched data.
//
// # Comparable Keys
//
// The Use hook enforces a comparable constraint on cache keys (K comparable). This enables
// zero-reflection, fast O(1) map lookups, while internally storing keys under type-erased "any"
// values in the Client.
//
// # Mutations and Invalidation
//
// The UseMutation hook manages mutations and side-effects. It receives a MutationContext containing
// the Client, which allows mutation callbacks (e.g. OnSuccess) to invalidate cached queries
// directly via Client.InvalidateQueries.
//
// Unmounting cancels the contexts of outstanding mutations and suppresses later
// callbacks. Query and mutation functions must pass Wind's context to their
// underlying work; promise.Wrap and promise.WrapWithProps preserve that context,
// whereas promise.New supplies only the scheduler's context.
package wind
