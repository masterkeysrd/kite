# TSK-031: Implement Golden Testing & Visual Dumps

## Overview
Extend the `devtools/testenv` package with tools for visually snapshotting and comparing `paint.FrameBuffer` outputs to catch layout regressions.

## Requirements

1. **Golden Testing Engine:**
   - Implement `env.MatchGolden(t *testing.T, filename string)`.
   - If the file does not exist, or if an `-update` flag is passed to `go test`, it should serialize the current `backend/mock` framebuffer and save it.
   - If the file exists, it should compare the current framebuffer against the stored snapshot. Fail the test if they do not match.

2. **Visual Dump Utilities:**
   - Implement `env.DumpANSI()`: Translates the current `FrameBuffer` (including colors and styles) into a raw string of ANSI escape codes that prints perfectly to standard output.
   - Implement `env.DumpHTML()`: Translates the current `FrameBuffer` into a standalone HTML file with `<span>` tags for styles. Highly useful for debugging CI failures in a browser.
   - Implement `env.DumpText()`: Translates the current `FrameBuffer` into a plain text representation, ignoring styles. Useful for quick diffs.

3. **Serialization Format:**
   - The `.golden` file format should be human-readable. A JSON representation of the cell grid, or a structured text format is acceptable.

## Testing & Verifications
- Create a regression test inside `devtools/testenv` that intentionally breaks a layout and verifies `MatchGolden` fails.
- Verify `DumpANSI()` outputs valid escape sequences using a dummy framebuffer.
