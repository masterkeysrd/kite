# Walkthrough: TSK-049 System Clipboard Integration

Implemented a robust, extensible system clipboard integration for Kite (v2) that supports global text selection, multi-MIME rich data, macOS hotkeys, and terminal-specific extensions like Kitty's secure clipboard transfer.

## Key Changes

### 1. Rich Clipboard Events (`event` package)
- Refactored `ClipboardEvent` to support multi-MIME data using an `Items map[string][]byte`.
- Added a `Clipboard` field to the event for direct `ClipboardBridge` access.
- Enabled bubbling by default for `ClipboardEvent`.
- Updated `Synthesizer` to support both **macOS (Cmd+C/V)** and **Linux/Windows (Ctrl+C/V)** shortcuts.
- Introduced `RawOscEvent` to surface raw terminal sequences to extensions.

### 2. Global Document Handlers (`dom` package)
- Added global `handleCopy` and `handlePaste` to `dom.Document`.
- **Copy:** Captures document selection and updates the system clipboard via OSC 52.
- **Paste:** Provides a fallback to pull from the system clipboard if the terminal doesn't provide data.

### 3. Backend & Engine Improvements
- **Direct TTY Writing:** The `uv` backend now writes escape sequences directly to the terminal device, bypassing stdout buffering.
- **OSC 52 Support:** Implemented `SetClipboard` in the `uv` backend using the universal OSC 52 protocol.
- **Extension Registry:** Added support for `TerminalExtension`s in the `Engine`, allowing protocol-specific handling like Kitty's OSC 5522.
- **Persistent Handshake:** Extensions can now persistently probe the terminal until a handshake is established.

### 4. Kitty OSC 5522 Integration (`internal/term/kitty`)
- Implemented the Kitty secure clipboard transfer protocol.
- Handles the 4-step handshake (Enable → Password → MIME Request → Chunked Delivery).

## Sample Application
A new example application in `example/clipboard` demonstrates:
- **Global Selection:** Copy text from non-input elements.
- **macOS Support:** Standard `Cmd+C` / `Cmd+V` shortcuts.
- **Image Detection:** Specifically identifies when an image *path* is dragged-and-dropped or when rich image *data* is received via Kitty.

## Verification Results
- **Integrated Tests:** `dom/clipboard_test.go` and `internal/term/kitty/kitty_test.go` verify the core logic.
- **Full Suite:** `go test ./...` passed.
- **Manual Test:** Verified on macOS with Kitty terminal (Drag & Drop and Cmd+V).
