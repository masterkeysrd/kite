// Package mock provides a recording backend for kite tests.
//
// The mock backend implements backend.Backend and captures all Surface
// draw calls so tests can assert on rendered output without a real terminal.
package mock
