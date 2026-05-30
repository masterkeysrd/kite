package terminal

// import "github.com/masterkeysrd/kite/backend"

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
	RunBackground(task func())
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
	ReadText() (string, error)
	WriteText(text string) error

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
	Read(mime string) ([]byte, error)

	// Write writes the given data to the clipboard with the specified MIME type. If the clipboard is unavailable or access is denied, it returns an error.
	Write(mime string, data []byte) error
}

type InteractiveClipboard interface {
	Clipboard

	// Sequences returns a channel that emits byte slices of scape sequences to be sent to
	// the terminal for clipboard interactions (init, read, write, list formats).
	Sequences() <-chan []byte

	// IsInitialized returns true if the clipboard has been initialized and is ready for
	// read/write operations.
	IsInitialized() bool

	// IsWaiting returns true if the terminal is currently waiting for to the terminal to complete
	// a read/write operation after requesting it to the terminal.
	IsWaiting() bool

	// Cancel cancels any pending read/write operation and unblocks
	// any waiting calls to ReadText, Read, or ListFormats.
	Cancel()

	// HandleEvent processes events related to clipboard like Osc, key and clipboard events.
	// This should be called by the terminal's event loop to ensure proper handling
	// of clipboard interactions.
	//
	// The event is returned unmodified if the clipboard does not handle it,
	// it can transform the event or return a different event if it handles it. The terminal
	// should use the returned event for further processing.
	// HandleEvent(event backend.RawEvent) backend.RawEvent
}
