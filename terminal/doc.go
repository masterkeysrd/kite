// Package terminal defines the interfaces for terminal-specific features
// used by the Kite engine. It provides clipboard access, scheduler control,
// and basic terminal operations like title setting and bell.
//
// # Terminal
//
// The Terminal interface exposes clipboard access, a Scheduler for
// background/microtask/macrotask execution, and simple operations like
// SetTitle and Bell. It is accessible from any DOM element via
// el.Document().Terminal().
//
// # Scheduler
//
// Scheduler mirrors the promise package's Scheduler interface, providing
// RunBackground for worker-pool tasks, QueueMicrotask for main-thread
// microtasks, and QueueMacrotask for main-thread macrotasks.
//
// # Clipboard
//
// Clipboard provides asynchronous text and binary read/write operations
// backed by Promise. ReadText and WriteText operate on plain text; Read
// and Write operate on raw bytes with a MIME type. All operations return
// *Promise to integrate with the engine's async model.
//
// # ProgressBarState
//
// ProgressBarState enumerates the visual states of the terminal progress
// bar: hide, normal, error, indeterminate, and paused.
//
// Example:
//
//	clippy := el.Document().Terminal().Clipboard()
//	clippy.WriteText("hello").Then(func(_ struct{}) {
//	     fmt.Println("written")
//	})
package terminal
