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
// goroutine is spawned to execute the fetcher.
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
package wind
