# `extras/clipboard`

Optional OS-native [`terminal.Clipboard`](../../terminal/terminal.go) provider for Kite with multi-MIME support (`text/plain`, `text/html`, `image/png`, `image/jpeg`, `image/gif`, `image/webp`, `image/tiff`, and `text/uri-list`).

By default, Kite uses terminal OSC 52 via [`engine.ClipboardProxy`](../../engine/terminal.go), which is SSH-safe but limited to `text/plain`. Local terminal applications that need rich clipboard formats (such as pasted images or HTML) can install `clipboard.New()` on the document.

## Usage

```go
package main

import (
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/clipboard"
)

func setup(doc dom.Document, textarea *element.TextAreaElement) {
	doc.SetClipboardProvider(clipboard.New())

	textarea.AddEventListener(event.EventPaste, func(e event.Event) {
		ce, ok := e.(*event.ClipboardEvent)
		if !ok {
			return
		}

		for mime, data := range ce.Items {
			if mime == "image/png" || mime == "image/jpeg" || mime == "image/gif" || mime == "image/webp" {
				ce.PreventDefault() // cancels textarea's default text insertion
				attachImage(mime, data)
				return
			}
		}
		// No PreventDefault(): textarea inserts ce.Text() normally.
	})
}
```

## How It Works with Kite's Paste Pipeline

1. **`KeyDown` cancellation:** Calling `PreventDefault()` on a `Ctrl+V` / `Cmd+V` `KeyDown` event aborts `EventPaste`.
2. **Pre-dispatch multi-MIME loading:** Before dispatching `EventPaste` to the focused element, the engine calls `ListFormats()` and `Read(mime)` to populate `ce.Items` (and enriches `RawBracketedPaste` with any non-`text/plain` formats available).
3. **Cancelable default action:** `<input>` and `<textarea>` insert `ce.Text()` as a post-dispatch default action only when `!ce.DefaultPrevented()`.

## Platform Support

| OS | Mechanism | Supported Formats |
|---|---|---|
| **macOS** | `pbpaste` / `pbcopy` + `osascript` (JXA `NSPasteboard`) | `text/plain`, `text/html`, `image/png` (including `public.tiff` → PNG transcoding), `image/jpeg`, `image/gif`, `image/webp`, `image/tiff`, `text/uri-list` |
| **Linux (Wayland)** | `wl-paste` / `wl-copy` | All MIME types reported by `wl-paste --list-types` |
| **Linux (X11)** | `xclip` (or `xsel` for plain text) | All MIME types reported by `xclip -t TARGETS` |
| **Windows** | PowerShell (`System.Windows.Forms.Clipboard`) | `text/plain`, `image/png` (read), `text/uri-list` (listed) |

Other platforms compile cleanly and return `clipboard.ErrUnsupported`.

## Direct API

`SystemClipboard` implements `terminal.Clipboard`:

- `ListFormats() *promise.Promise[[]string]`
- `Read(mime string) *promise.Promise[[]byte]`
- `ReadText() *promise.Promise[string]`
- `Write(mime string, data []byte) *promise.Promise[struct{}]`
- `WriteText(text string) *promise.Promise[struct{}]`
