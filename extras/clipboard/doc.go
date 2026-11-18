// Package clipboard provides an optional OS-native [terminal.Clipboard]
// implementation with multi-MIME support (plain text, HTML, images, and URI
// lists) for local terminal applications.
//
// By default, Kite uses terminal OSC 52 via [engine.ClipboardProxy], which is
// SSH-safe but limited to "text/plain". Local applications that need richer
// clipboard formats (such as "image/png" or "text/html") can install
// [New]() via doc.SetClipboardProvider(clipboard.New()).
package clipboard
