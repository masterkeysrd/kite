package engine

import "time"

// Clock is an injectable time source used by the engine for frame scheduling.
// The real implementation delegates to the standard library; tests supply a
// fake implementation to exercise time-dependent behaviour deterministically.
//
// Clock implementations must be safe for concurrent use.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// After returns a channel that receives the current time after duration d
	// has elapsed. The channel is never closed.
	After(d time.Duration) <-chan time.Time
}

// realClock is the default Clock backed by the standard library.
type realClock struct{}

// RealClock returns a Clock that delegates to time.Now and time.After.
func RealClock() Clock { return realClock{} }

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
