package clipboard

import (
	"context"
	"errors"
	"slices"

	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"
)

// ErrUnsupported is returned when the current platform or clipboard tool does
// not support the requested operation or MIME type.
var ErrUnsupported = errors.New("clipboard: unsupported platform or format")

type platformBackend interface {
	listFormats(ctx context.Context) ([]string, error)
	read(ctx context.Context, mime string) ([]byte, error)
	write(ctx context.Context, mime string, data []byte) error
}

// SystemClipboard implements [terminal.Clipboard] using the host OS
// clipboard, supporting multiple MIME types (e.g. "text/plain", "text/html",
// "image/png", "text/uri-list").
type SystemClipboard struct {
	backend platformBackend
}

var _ terminal.Clipboard = (*SystemClipboard)(nil)

// New creates a new [SystemClipboard] backed by the current operating system's
// native clipboard utilities.
func New() *SystemClipboard {
	return &SystemClipboard{backend: newPlatformBackend()}
}

// ReadText returns the current "text/plain" content from the OS clipboard.
func (c *SystemClipboard) ReadText() *promise.Promise[string] {
	return promise.New(func(ctx context.Context) (string, error) {
		data, err := c.backend.read(ctx, "text/plain")
		if err != nil {
			return "", err
		}
		return string(data), nil
	})
}

// WriteText stores plain text into the OS clipboard.
func (c *SystemClipboard) WriteText(text string) *promise.Promise[struct{}] {
	return c.Write("text/plain", []byte(text))
}

// ListFormats returns the deduplicated list of MIME types currently available
// on the OS clipboard.
func (c *SystemClipboard) ListFormats() *promise.Promise[[]string] {
	return promise.New(func(ctx context.Context) ([]string, error) {
		formats, err := c.backend.listFormats(ctx)
		if err != nil {
			return nil, err
		}
		return dedupeStrings(formats), nil
	})
}

// Read retrieves the payload for the given MIME type from the OS clipboard.
func (c *SystemClipboard) Read(mime string) *promise.Promise[[]byte] {
	return promise.New(func(ctx context.Context) ([]byte, error) {
		return c.backend.read(ctx, mime)
	})
}

// Write stores data for the given MIME type into the OS clipboard.
func (c *SystemClipboard) Write(mime string, data []byte) *promise.Promise[struct{}] {
	return promise.New(func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.backend.write(ctx, mime, data)
	})
}

func dedupeStrings(in []string) []string {
	var out []string
	for _, s := range in {
		if s == "" || slices.Contains(out, s) {
			continue
		}
		out = append(out, s)
	}
	return out
}
