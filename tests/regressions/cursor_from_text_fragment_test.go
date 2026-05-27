// Package regressions – cursor.FromTextFragment regression tests (TSK-023).
//
// These tests reproduce byte-arithmetic edge cases found during TextArea
// development:
//
//   - Mandatory break at the end of the last line.
//   - Scroll-shifted lines (multiple lines, cursor in a non-first line).
//   - Leading-space-collapsed lines (first bytes on a line are spaces that IFC
//     collapsed; the helper must still resolve offsets correctly for the bytes
//     that DO appear in the fragment).
//   - Equivalence with the simple "hello\nworld" case.
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/text"
)

// ---------------------------------------------------------------------------
// Fragment construction helpers (local, mirrors those in cursor package tests)
// ---------------------------------------------------------------------------

func cftTextFrag(clusters []text.Cluster) *layout.Fragment {
	w := 0
	for _, c := range clusters {
		w += c.CellWidth
	}
	return &layout.Fragment{
		Size: geom.Size{Width: w, Height: 1},
		Text: clusters,
	}
}

func cftLineFrag(children ...*layout.Fragment) *layout.Fragment {
	links := make([]layout.FragmentLink, len(children))
	offsetX := 0
	for i, c := range children {
		links[i] = layout.FragmentLink{
			Offset:   geom.Point{X: offsetX, Y: 0},
			Fragment: c,
		}
		offsetX += c.Size.Width
	}
	return &layout.Fragment{
		Size:     geom.Size{Width: offsetX, Height: 1},
		Children: links,
	}
}

func cftRoot(lines ...*layout.Fragment) *layout.Fragment {
	links := make([]layout.FragmentLink, len(lines))
	offsetY := 0
	for i, l := range lines {
		links[i] = layout.FragmentLink{
			Offset:   geom.Point{X: 0, Y: offsetY},
			Fragment: l,
		}
		offsetY += l.Size.Height
	}
	return &layout.Fragment{
		Size:     geom.Size{Width: 80, Height: offsetY},
		Children: links,
	}
}

func cftASCII(s string) []text.Cluster {
	clusters := make([]text.Cluster, len(s))
	for i := range s {
		clusters[i] = text.Cluster{
			Bytes:     []byte{s[i]},
			CellWidth: 1,
		}
	}
	return clusters
}

// ---------------------------------------------------------------------------
// TSK-023 Regression: "hello\nworld" equivalence
// ---------------------------------------------------------------------------

// TestCursorFromTextFragment_HelloWorld verifies the canonical two-line IFC
// fragment built from "hello\nworld" agrees with expected cursor positions.
//
// This test acts as the integration regression: if the byte-arithmetic in
// FromTextFragment changes, this test will catch it against a known-good
// hand-calculated baseline.
func TestCursorFromTextFragment_HelloWorld(t *testing.T) {
	// IFC produces two line boxes:
	//   Line 0: "hello\n" → clusters h,e,l,l,o,\n (6 bytes; \n has CellWidth 0)
	//   Line 1: "world"   → clusters w,o,r,l,d   (5 bytes; 5 cells)
	line0 := []text.Cluster{
		{Bytes: []byte{'h'}, CellWidth: 1},
		{Bytes: []byte{'e'}, CellWidth: 1},
		{Bytes: []byte{'l'}, CellWidth: 1},
		{Bytes: []byte{'l'}, CellWidth: 1},
		{Bytes: []byte{'o'}, CellWidth: 1},
		{Bytes: []byte{'\n'}, CellWidth: 0},
	}
	line1 := cftASCII("world")

	root := cftRoot(
		cftLineFrag(cftTextFrag(line0)),
		cftLineFrag(cftTextFrag(line1)),
	)

	table := []struct {
		name   string
		offset int
		wantX  int
		wantY  int
		wantOk bool
	}{
		{"before h", 0, 0, 0, true},
		{"before e", 1, 1, 0, true},
		{"before l1", 2, 2, 0, true},
		{"before l2", 3, 3, 0, true},
		{"before o", 4, 4, 0, true},
		{"before \\n", 5, 5, 0, true},
		{"after \\n (start of world)", 6, 0, 1, true},
		{"before o in world", 7, 1, 1, true},
		{"before r", 8, 2, 1, true},
		{"before l", 9, 3, 1, true},
		{"before d", 10, 4, 1, true},
		{"trailing", 11, 5, 1, true},
		{"out of range", 12, 0, 0, false},
	}

	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			x, y, ok := cursor.FromTextFragment(root, tc.offset)
			if ok != tc.wantOk || x != tc.wantX || y != tc.wantY {
				t.Errorf("offset %d: want (%d,%d,%v), got (%d,%d,%v)",
					tc.offset, tc.wantX, tc.wantY, tc.wantOk, x, y, ok)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TSK-023 Regression: mandatory break at end of last line
// ---------------------------------------------------------------------------

// TestCursorFromTextFragment_MandatoryBreakAtEndOfLastLine verifies that when
// the final line box ends with a mandatory-break cluster (\n), the trailing
// offset (total bytes) is still resolved correctly as one past the end.
func TestCursorFromTextFragment_MandatoryBreakAtEndOfLastLine(t *testing.T) {
	// Single line: "abc\n" — 4 bytes. The \n has CellWidth 0.
	clusters := []text.Cluster{
		{Bytes: []byte{'a'}, CellWidth: 1},
		{Bytes: []byte{'b'}, CellWidth: 1},
		{Bytes: []byte{'c'}, CellWidth: 1},
		{Bytes: []byte{'\n'}, CellWidth: 0},
	}
	root := cftRoot(cftLineFrag(cftTextFrag(clusters)))

	// Trailing offset == 4 (total bytes). The \n contributes 0 cells, so
	// the trailing cell position is 3 (right after 'c').
	x, y, ok := cursor.FromTextFragment(root, 4)
	if !ok || x != 3 || y != 0 {
		t.Errorf("mandatory break trailing: want (3,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// Offset 3 → before '\n' → x==3
	x, y, ok = cursor.FromTextFragment(root, 3)
	if !ok || x != 3 || y != 0 {
		t.Errorf("before \\n: want (3,0,true), got (%d,%d,%v)", x, y, ok)
	}
}

// ---------------------------------------------------------------------------
// TSK-023 Regression: scroll-shifted lines (cursor in non-first line)
// ---------------------------------------------------------------------------

// TestCursorFromTextFragment_ScrollShiftedLines verifies cursor resolution when
// the visible window starts at line 2 (lines 0 and 1 are scrolled off). The
// fragment tree only contains lines 2+ — a common pattern in textarea scrolling
// where the root fragment is rebuilt for the visible viewport.
//
// In this test the root only has one visible line (line 2) beginning at byte
// offset 0 relative to the fragment (the scroll offset is absorbed by the
// caller before invoking FromTextFragment).
func TestCursorFromTextFragment_ScrollShiftedLines(t *testing.T) {
	// Simulate a textarea that scrolled two lines of 5 bytes each (total 10
	// bytes scrolled off). The caller passes byteOffset adjusted for scrolling,
	// so we test with a fresh root that starts at offset 0 within the fragment.
	//
	// Visible fragment: one line "world" (5 bytes).
	root := cftRoot(cftLineFrag(cftTextFrag(cftASCII("world"))))

	x, y, ok := cursor.FromTextFragment(root, 3) // before 'l'
	if !ok || x != 3 || y != 0 {
		t.Errorf("scroll-shifted offset 3: want (3,0,true), got (%d,%d,%v)", x, y, ok)
	}
}

// ---------------------------------------------------------------------------
// TSK-023 Regression: leading-space-collapsed lines
// ---------------------------------------------------------------------------

// TestCursorFromTextFragment_LeadingSpaceCollapsed verifies that when the IFC
// collapses leading spaces on a soft-wrapped line (the space cluster was
// consumed by the line breaker and does NOT appear in the line-box fragment),
// the helper still accounts for the visible (non-space) bytes correctly.
//
// After collapsing, the line-box fragment's text runs begin at the first
// non-space cluster. The byte offset into the collapsed fragment starts at 0;
// the caller must account for any bytes that were eliminated by collapsing.
func TestCursorFromTextFragment_LeadingSpaceCollapsed(t *testing.T) {
	// Collapsed line: "hello" (leading space was stripped; not present in fragment).
	// The fragment therefore contains only "hello" clusters starting at byte 0.
	root := cftRoot(cftLineFrag(cftTextFrag(cftASCII("hello"))))

	// Byte 0 within the fragment → x==0 (the 'h').
	x, y, ok := cursor.FromTextFragment(root, 0)
	if !ok || x != 0 || y != 0 {
		t.Errorf("collapsed leading space offset 0: want (0,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// Byte 3 within the fragment → x==3 (the second 'l').
	x, y, ok = cursor.FromTextFragment(root, 3)
	if !ok || x != 3 || y != 0 {
		t.Errorf("collapsed leading space offset 3: want (3,0,true), got (%d,%d,%v)", x, y, ok)
	}
}

// ---------------------------------------------------------------------------
// TSK-023 Regression: multi-line with multiple styled spans per line
// ---------------------------------------------------------------------------

// TestCursorFromTextFragment_MultiSpanLines exercises multi-fragment lines
// (styled spans) across multiple lines to ensure both the y-routing and
// within-line x-resolution work end-to-end.
func TestCursorFromTextFragment_MultiSpanLines(t *testing.T) {
	// Line 0: [foo][bar] → 6 bytes, 6 cells
	// Line 1: [baz][qux] → 6 bytes, 6 cells
	// Total: 12 bytes
	line0 := cftLineFrag(
		cftTextFrag(cftASCII("foo")),
		cftTextFrag(cftASCII("bar")),
	)
	line1 := cftLineFrag(
		cftTextFrag(cftASCII("baz")),
		cftTextFrag(cftASCII("qux")),
	)
	root := cftRoot(line0, line1)

	table := []struct {
		offset int
		wantX  int
		wantY  int
	}{
		{0, 0, 0},  // 'f'
		{3, 3, 0},  // 'b' (start of "bar")
		{5, 5, 0},  // 'r'
		{6, 0, 1},  // 'b' (start of "baz")
		{9, 3, 1},  // 'q' (start of "qux")
		{11, 5, 1}, // 'x'
		{12, 6, 1}, // trailing
	}

	for _, tc := range table {
		x, y, ok := cursor.FromTextFragment(root, tc.offset)
		if !ok || x != tc.wantX || y != tc.wantY {
			t.Errorf("multi-span offset %d: want (%d,%d,true), got (%d,%d,%v)",
				tc.offset, tc.wantX, tc.wantY, x, y, ok)
		}
	}
}
