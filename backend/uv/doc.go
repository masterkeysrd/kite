// Package uv is the ultraviolet terminal backend for kitex (kite v2).
//
// It implements the backend.Backend interface using the charmbracelet/ultraviolet
// library. The async render goroutine model from kite v1 is preserved: EndFrame
// hands the painted surface to a background goroutine that diffs against shadow
// state and writes to the device.
package uv
