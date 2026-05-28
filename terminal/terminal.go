package terminal

// Terminal provides access to terminal-specific features like the
// clipboard and layout engine.
//
// This interface is implement by the engine and is accessible for the
// dom via el.Document().Terminal().
type Terminal interface {
	Clipboard() Clipboard
}

type Clipboard interface {
	// ReadText returns the current text content of the clipboard. If the clipboard is
	// empty or unavailable, it returns an empty string and an error.
	//
	// This call blocks until the clipboard content is available and fails
	// after 100ms if the clipboard is unavailable or access is denied.
	ReadText() (string, error)
	WriteText(text string) error

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
