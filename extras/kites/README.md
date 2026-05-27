# 📦 Kites

Kites is a lightweight, thread-safe external state store for global state management outside of the Virtual DOM (VDOM) tree, inspired by libraries like Zustand or Redux. It integrates with the Kitex framework via the selector-based `kites.Use` hook, offering optimized, reactive re-rendering.

## ✨ Features

- 🔒 **Fully Thread-Safe**: Safe for concurrent updates and reads using a reader-writer lock (`sync.RWMutex`).
- 🛑 **Deadlock Protection**: Listener notifications are dispatched outside of the write lock to prevent deadlocks when updating state inside subscriber callbacks.
- 🎯 **Selector-Based Re-rendering**: Components only subscribe to and re-render on changes to specific slices of the state. If the selected state doesn't change, the component bails out of re-rendering.
- 🧹 **Automated Lifecycle Cleanup**: Subscriptions are set up on component mount and automatically cleaned up on unmount.

## 🚀 Getting Started

### 1. Create a Global Store

Define your global state struct and initialize the store:

```go
package store

import (
	"github.com/masterkeysrd/kite/extras/kites"
)

type AppState struct {
	Counter int
	Theme   string
}

// Create the store with initial values
var GlobalStore = kites.Create(AppState{
	Counter: 0,
	Theme:   "dark",
})
```

### 2. Connect Store to Components with selectors

Use `kites.Use` inside functional components. The selector determines exactly which piece of the store the component listens to:

```go
package views

import (
	"fmt"

	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kites"
	"github.com/masterkeysrd/kite/store"
)

var CounterDisplay = kitex.SimpleFC("CounterDisplay", func() kitex.Node {
	// Component will ONLY re-render when store.Counter changes.
	// Changes to store.Theme will be ignored.
	count := kites.Use(store.GlobalStore, func(s store.AppState) int {
		return s.Counter
	})

	return kitex.Text(fmt.Sprintf("Current count: %d", count))
})
```

### 3. Update the State

Call `store.Set` with a modifier function to update the store:

```go
func IncrementCounter() {
	store.GlobalStore.Set(func(s store.AppState) store.AppState {
		s.Counter++
		return s
	})
}
```

### 4. Direct Subscription (Optional)

You can subscribe to changes outside of components (e.g., for logging or persisting state):

```go
unsubscribe := store.GlobalStore.Subscribe(func(newVal, oldVal store.AppState) {
	fmt.Printf("State updated from %+v to %+v\n", oldVal, newVal)
})

// Call unsubscribe when done
defer unsubscribe()
```

## 🛠 API Reference

### Store

- **`Create[T](initial T) *Store[T]`**: Instantiates a new store containing type `T`.
- **`Store.Get() T`**: Returns the current state.
- **`Store.Set(updater func(T) T)`**: Updates the state thread-safely. Calls the updater and dispatches listener notifications after unlocking.
- **`Store.Subscribe(listener func(newVal T, oldVal T)) func()`**: Subscribes to store changes. Returns an unsubscribe function.

### Hooks

- **`Use[T any, U comparable](s *Store[T], selector func(T) U) U`**:
  - Connects the store to a functional component.
  - Generically bound to any `U comparable` slice for zero-cost comparison checks during updates.
