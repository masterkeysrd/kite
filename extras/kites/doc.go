// Package kites implements a lightweight, thread-safe external state store
// (similar to Zustand or Redux) for global state management outside of the VDOM tree.
// It integrates with the kitex framework through the Use hook to provide reactive,
// selector-based re-rendering with optimization bailouts.
package kites
