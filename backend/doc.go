// Package backend defines the Backend interface that decouples the paint
// engine from the terminal (or other output target) in kitex (kite v2).
//
// A Backend provides BeginFrame / EndFrame lifecycle hooks and vends a
// Surface for the PaintEngine to draw onto. Sub-packages provide concrete
// implementations: uv (ultraviolet terminal) and mock (recording, for tests).
package backend
