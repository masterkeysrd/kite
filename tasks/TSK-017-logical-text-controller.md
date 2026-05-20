# TSK-017: Logical Text Controller

## Feature Design & Requirements
Logical input elements need to navigate strings safely across Unicode boundaries without knowing physical cell widths. 

1. **New Package/Utility `editor/buffer.go`:**
   - Create a `Buffer` struct to manage 1-dimensional string edits.
   - It maintains the raw `string` and a logical `byteOffset`.
   - Provide methods for 1D navigation and mutation:
     - `Insert(s string)`
     - `DeletePrevious()` (Backspace)
     - `DeleteNext()` (Delete)
     - `MoveLeft()` / `MoveRight()` (by grapheme cluster using `uniseg`)
     - `MoveWordLeft()` / `MoveWordRight()`
     - `DeleteWordPrevious()` / `DeleteWordNext()`
     - `MoveToStart()` / `MoveToEnd()`
     - `Value() string`
     - `ByteOffset() int`

2. **Implementation Detail:**
   - Use `github.com/rivo/uniseg` to safely step through grapheme clusters to ensure multi-byte characters and ZWJ emoji sequences are treated as single atomic units.

## Tests Required
- Extensive unit tests covering ASCII, wide characters (CJK), and complex emojis (ZWJ sequences).
- Tests verifying word-boundary detection and edge cases (moving past start/end of buffer).
