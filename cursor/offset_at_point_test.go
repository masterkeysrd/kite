package cursor

import (
	"testing"

	"github.com/masterkeysrd/kite/text"
)

// TestByteOffsetAtPoint_NilRoot verifies that a nil root returns 0.
func TestByteOffsetAtPoint_NilRoot(t *testing.T) {
	if off := ByteOffsetAtPoint(nil, 0, 0); off != 0 {
		t.Errorf("nil root: want 0, got %d", off)
	}
}

// TestByteOffsetAtPoint_EmptyRoot verifies that a root with no children returns 0.
func TestByteOffsetAtPoint_EmptyRoot(t *testing.T) {
	root := ifcRoot()
	if off := ByteOffsetAtPoint(root, 0, 0); off != 0 {
		t.Errorf("empty root: want 0, got %d", off)
	}
}

// TestByteOffsetAtPoint_SingleLine_ASCII verifies offset mapping for a single line.
func TestByteOffsetAtPoint_SingleLine_ASCII(t *testing.T) {
	// "hello" — 5 bytes, 5 cells, single line.
	root := ifcRoot(lineFrag(textFrag(shapedASCII("hello"))))

	tests := []struct {
		x, y int
		want int
	}{
		{0, 0, 0},  // before 'h'
		{1, 0, 1},  // before 'e'
		{2, 0, 2},  // before 'l'
		{4, 0, 4},  // before 'o'
		{5, 0, 5},  // past last char
		{50, 0, 5}, // far past last char
	}

	for _, tc := range tests {
		if off := ByteOffsetAtPoint(root, tc.x, tc.y); off != tc.want {
			t.Errorf("Point (%d,%d): want offset %d, got %d", tc.x, tc.y, tc.want, off)
		}
	}
}

// TestByteOffsetAtPoint_TwoLines_HardBreak verifies point mapping for two lines
// split by a mandatory newline.
func TestByteOffsetAtPoint_TwoLines_HardBreak(t *testing.T) {
	// Line 0: "hi\n"  → 3 bytes. The "\n" cluster has CellWidth 0 and BreakMandatory.
	line0Clusters := []text.Cluster{
		{Bytes: []byte{'h'}, CellWidth: 1},
		{Bytes: []byte{'i'}, CellWidth: 1},
		{Bytes: []byte{'\n'}, CellWidth: 0, BreakClass: text.BreakMandatory},
	}
	// Line 1: "go" → 2 bytes, 2 cells.
	line1Clusters := shapedASCII("go")

	root := ifcRoot(
		lineFrag(textFrag(line0Clusters)),
		lineFrag(textFrag(line1Clusters)),
	)

	tests := []struct {
		x, y int
		want int
	}{
		{0, 0, 0}, // 'h'
		{1, 0, 1}, // 'i'
		{2, 0, 2}, // past 'i', before '\n'
		{5, 0, 2}, // far past line 0
		{0, 1, 3}, // 'g' on line 1
		{1, 1, 4}, // 'o'
		{2, 1, 5}, // past 'o'
		{5, 1, 5}, // far past line 1
		{0, 2, 5}, // past last line
	}

	for _, tc := range tests {
		if off := ByteOffsetAtPoint(root, tc.x, tc.y); off != tc.want {
			t.Errorf("Point (%d,%d): want offset %d, got %d", tc.x, tc.y, tc.want, off)
		}
	}
}

// TestByteOffsetAtPoint_AtomicInlines verifies mapping when atomic inlines are present.
func TestByteOffsetAtPoint_AtomicInlines(t *testing.T) {
	// [atomic(3)] "hello" [atomic(2)]
	root := ifcRoot(lineFrag(
		atomicFrag(3),
		textFrag(shapedASCII("hello")),
		atomicFrag(2),
	))

	tests := []struct {
		x, y int
		want int
	}{
		{0, 0, 0},  // inside first atomic
		{2, 0, 0},  // still inside first atomic
		{3, 0, 0},  // start of "hello"
		{4, 0, 1},  // 'e'
		{8, 0, 5},  // start of second atomic
		{10, 0, 5}, // end of second atomic
	}

	for _, tc := range tests {
		if off := ByteOffsetAtPoint(root, tc.x, tc.y); off != tc.want {
			t.Errorf("Point (%d,%d): want offset %d, got %d", tc.x, tc.y, tc.want, off)
		}
	}
}
