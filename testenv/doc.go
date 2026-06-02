// Package testenv provides an ergonomic testing environment for Kite components.
// It includes a headless mock backend, event dispatchers, and fluent assertion
// helpers to verify DOM state, layout fragments, and rendered output.
//
// Use Default(width, height) to create an environment, and then use the provided
// helpers like Mount, Flush, and various assertion types to test UI components
// in isolation.
package testenv
