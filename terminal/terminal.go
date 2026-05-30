package terminal

import (
	"context"

	"github.com/masterkeysrd/kite/promise"
)

// Terminal provides access to terminal-specific features like the
// clipboard and layout engine.
//
// This interface is implement by the engine and is accessible for the
// dom via el.Document().Terminal().
type Terminal interface {
	Clipboard() Clipboard
	Scheduler() Scheduler
}

type Scheduler interface {
	// RunBackground executes a task on a background worker pool.
	// The provided context is managed by the scheduler.
	RunBackground(task func(ctx context.Context))
	// QueueMicrotask schedules a task to run as a microtask on the main thread.
	QueueMicrotask(task func())
	// QueueMacrotask schedules a task to run as a macrotask on the main thread.
	QueueMacrotask(task func())
}

type Clipboard interface {
	// ReadText returns the current text content of the clipboard. If the clipboard is
	// empty or unavailable, it returns an empty string and an error.
	//
	// This call blocks until the clipboard content is available and fails
	// after 100ms if the clipboard is unavailable or access is denied.
	ReadText() *promise.Promise[string]
	WriteText(text string) *promise.Promise[struct{}]

	// ListFormats returns a list of MIME types representing the formats currently available in
	// the clipboard. If the clipboard is unavailable or access is denied, it returns an error.
	// ListFormats() ([]string, error)

	// Read returns the current content of the clipboard as a byte slice. If the clipboard is
	// empty or unavailable, it returns an empty byte slice and an error.
	//
	// This call blocks until the clipboard content is available and fails
	// after 100ms if the clipboard is unavailable or access is denied.
	//
	// E.g. Read("text/plain") returns the same content as ReadText() but as bytes.
	// Read("image/png") would return PNG-encoded image data if the clipboard contains an image.
	// Read("text/html") would return HTML content if the clipboard contains HTML.
	// Read(mime string) ([]byte, error)
	Read(mime string) *promise.Promise[[]byte]

	// Write writes the given data to the clipboard with the specified MIME type. If the clipboard is unavailable or access is denied, it returns an error.
	Write(mime string, data []byte) *promise.Promise[struct{}]
}
