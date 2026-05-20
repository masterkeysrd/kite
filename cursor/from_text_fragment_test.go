package cursor

import (
	"testing"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/text"
)

// ---------------------------------------------------------------------------
// Fragment construction helpers
// ---------------------------------------------------------------------------

// textFrag builds a text-only Fragment from pre-shaped clusters.
func textFrag(clusters []text.Cluster) *layout.Fragment {
	w := 0
	for _, c := range clusters {
		w += c.CellWidth
	}
	return &layout.Fragment{
		Size: layout.Size{Width: w, Height: 1},
		Text: clusters,
	}
}

// atomicFrag builds an atomic-inline Fragment (no text) with an explicit width.
func atomicFrag(width int) *layout.Fragment {
	return &layout.Fragment{
		Size: layout.Size{Width: width, Height: 1},
	}
}

// lineFrag builds a line-box Fragment whose children are positioned fragments.
func lineFrag(children ...*layout.Fragment) *layout.Fragment {
	links := make([]layout.FragmentLink, len(children))
	offsetX := 0
	for i, c := range children {
		links[i] = layout.FragmentLink{
			Offset:   layout.Point{X: offsetX, Y: 0},
			Fragment: c,
		}
		offsetX += c.Size.Width
	}
	return &layout.Fragment{
		Size:     layout.Size{Width: offsetX, Height: 1},
		Children: links,
	}
}

// ifcRoot builds a root IFC Fragment whose children are line-box fragments.
func ifcRoot(lines ...*layout.Fragment) *layout.Fragment {
	links := make([]layout.FragmentLink, len(lines))
	offsetY := 0
	for i, l := range lines {
		links[i] = layout.FragmentLink{
			Offset:   layout.Point{X: 0, Y: offsetY},
			Fragment: l,
		}
		offsetY += l.Size.Height
	}
	return &layout.Fragment{
		Size:     layout.Size{Width: 80, Height: offsetY},
		Children: links,
	}
}

// shapedASCII shapes a plain ASCII string into clusters (1 byte, 1 cell each).
func shapedASCII(s string) []text.Cluster {
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
// Unit tests — FromTextFragment
// ---------------------------------------------------------------------------

// TestFromTextFragment_NilRoot verifies that a nil root returns (0,0,false).
func TestFromTextFragment_NilRoot(t *testing.T) {
	x, y, ok := FromTextFragment(nil, 0)
	if ok || x != 0 || y != 0 {
		t.Errorf("nil root: want (0,0,false), got (%d,%d,%v)", x, y, ok)
	}
}

// TestFromTextFragment_EmptyRoot verifies that a root with no children returns
// (0,0,false).
func TestFromTextFragment_EmptyRoot(t *testing.T) {
	root := &layout.Fragment{}
	x, y, ok := FromTextFragment(root, 0)
	if ok || x != 0 || y != 0 {
		t.Errorf("empty root: want (0,0,false), got (%d,%d,%v)", x, y, ok)
	}
}

// TestFromTextFragment_OffsetOutOfRange verifies that an offset past the total
// byte count returns (0,0,false).
func TestFromTextFragment_OffsetOutOfRange(t *testing.T) {
	root := ifcRoot(lineFrag(textFrag(shapedASCII("hello"))))
	x, y, ok := FromTextFragment(root, 100)
	if ok || x != 0 || y != 0 {
		t.Errorf("out-of-range offset: want (0,0,false), got (%d,%d,%v)", x, y, ok)
	}
}

// TestFromTextFragment_SingleLine_ASCII verifies offset 0, mid-string, and
// end-of-string for a single ASCII line.
func TestFromTextFragment_SingleLine_ASCII(t *testing.T) {
	// "hello" — 5 bytes, 5 cells, single line.
	root := ifcRoot(lineFrag(textFrag(shapedASCII("hello"))))

	tests := []struct {
		offset int
		wantX  int
		wantY  int
	}{
		{0, 0, 0}, // before 'h'
		{1, 1, 0}, // before 'e'
		{2, 2, 0}, // before 'l'
		{3, 3, 0}, // before second 'l'
		{4, 4, 0}, // before 'o'
		{5, 5, 0}, // trailing: one past last char
	}

	for _, tc := range tests {
		x, y, ok := FromTextFragment(root, tc.offset)
		if !ok || x != tc.wantX || y != tc.wantY {
			t.Errorf("offset %d: want (%d,%d,true), got (%d,%d,%v)", tc.offset, tc.wantX, tc.wantY, x, y, ok)
		}
	}
}

// TestFromTextFragment_TwoLines_HardBreak verifies that a two-line root split
// by a mandatory newline routes offsets to the correct y value.
//
// Simulates "hi\ngo" where the newline cluster contributes 1 byte (len("\n")==1)
// and the IFC emits it inside the first line box.
func TestFromTextFragment_TwoLines_HardBreak(t *testing.T) {
	// Line 0: "hi\n"  → 3 bytes. The "\n" cluster has CellWidth 0.
	line0Clusters := []text.Cluster{
		{Bytes: []byte{'h'}, CellWidth: 1},
		{Bytes: []byte{'i'}, CellWidth: 1},
		{Bytes: []byte{'\n'}, CellWidth: 0}, // mandatory break cluster
	}
	// Line 1: "go" → 2 bytes, 2 cells.
	line1Clusters := shapedASCII("go")

	root := ifcRoot(
		lineFrag(textFrag(line0Clusters)),
		lineFrag(textFrag(line1Clusters)),
	)

	tests := []struct {
		offset int
		wantX  int
		wantY  int
		wantOk bool
	}{
		{0, 0, 0, true}, // 'h'
		{1, 1, 0, true}, // 'i'
		{2, 2, 0, true}, // '\n' — cursor before the break cluster
		{3, 0, 1, true}, // 'g' on second line
		{4, 1, 1, true}, // 'o'
		{5, 2, 1, true}, // trailing on last line
		{6, 0, 0, false}, // out of range
	}

	for _, tc := range tests {
		x, y, ok := FromTextFragment(root, tc.offset)
		if ok != tc.wantOk || x != tc.wantX || y != tc.wantY {
			t.Errorf("offset %d: want (%d,%d,%v), got (%d,%d,%v)", tc.offset, tc.wantX, tc.wantY, tc.wantOk, x, y, ok)
		}
	}
}

// TestFromTextFragment_SoftWrap verifies that a soft-wrapped two-line fragment
// (no explicit newline) correctly places the wrap-boundary offset at y==1, x==0.
func TestFromTextFragment_SoftWrap(t *testing.T) {
	// Soft-wrap: "hello" on line 0, "world" on line 1.
	// Byte boundary between lines is at offset 5.
	root := ifcRoot(
		lineFrag(textFrag(shapedASCII("hello"))),
		lineFrag(textFrag(shapedASCII("world"))),
	)

	// Offset 5 is the *start* of line 1 (byte 0 on that line), because line 0
	// has bytes 0-4. The condition byteOffset < lineEnd fails for 5 (5 < 5 == false),
	// so the algorithm advances to line 1.
	table := []struct {
		offset int
		wantX  int
		wantY  int
		wantOk bool
	}{
		{0, 0, 0, true},   // start of line 0
		{4, 4, 0, true},   // last char on line 0
		{5, 0, 1, true},   // wrap boundary → first pos on line 1
		{6, 1, 1, true},   // second char of "world"
		{10, 5, 1, true},  // trailing on last line
		{11, 0, 0, false}, // out of range
	}

	for _, tc := range table {
		x, y, ok := FromTextFragment(root, tc.offset)
		if ok != tc.wantOk || x != tc.wantX || y != tc.wantY {
			t.Errorf("soft-wrap offset %d: want (%d,%d,%v), got (%d,%d,%v)", tc.offset, tc.wantX, tc.wantY, tc.wantOk, x, y, ok)
		}
	}
}

// TestFromTextFragment_WideCJK verifies that wide CJK characters (CellWidth==2)
// accumulate correctly so that byte-offset→cell-offset math is accurate.
func TestFromTextFragment_WideCJK(t *testing.T) {
	// Three CJK characters: each is 3 UTF-8 bytes, 2 cells wide.
	// Total: 9 bytes, 6 cells.
	cjk := func(r rune) text.Cluster {
		b := []byte(string(r))
		return text.Cluster{Bytes: b, CellWidth: 2}
	}
	clusters := []text.Cluster{
		cjk('中'), // bytes 0-2
		cjk('文'), // bytes 3-5
		cjk('字'), // bytes 6-8
	}
	root := ifcRoot(lineFrag(textFrag(clusters)))

	tests := []struct {
		offset int
		wantX  int
		wantOk bool
	}{
		{0, 0, true}, // before '中' → cell 0
		{3, 2, true}, // before '文' → cell 2
		{6, 4, true}, // before '字' → cell 4
		{9, 6, true}, // trailing: one past last → cell 6
	}

	for _, tc := range tests {
		x, y, ok := FromTextFragment(root, tc.offset)
		if ok != tc.wantOk || x != tc.wantX || y != 0 {
			t.Errorf("CJK offset %d: want (%d,0,%v), got (%d,%d,%v)", tc.offset, tc.wantX, tc.wantOk, x, y, ok)
		}
	}
}

// TestFromTextFragment_MultiClusterGrapheme verifies that a ZWJ emoji sequence
// (multiple bytes, single CellWidth) contributes its single cell width once.
func TestFromTextFragment_MultiClusterGrapheme(t *testing.T) {
	// Family emoji (👨‍👩‍👧‍👦): one grapheme cluster, 25 bytes, 2 cells wide.
	emoji := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	emojiBytes := []byte(emoji)
	clusters := []text.Cluster{
		{Bytes: emojiBytes, CellWidth: 2},
		{Bytes: []byte{'!'}, CellWidth: 1},
	}
	root := ifcRoot(lineFrag(textFrag(clusters)))

	x, y, ok := FromTextFragment(root, 0)
	if !ok || x != 0 || y != 0 {
		t.Errorf("emoji offset 0: want (0,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// offset == len(emoji) → just after the emoji, before '!'
	x, y, ok = FromTextFragment(root, len(emojiBytes))
	if !ok || x != 2 || y != 0 {
		t.Errorf("emoji offset len: want (2,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// trailing offset
	total := len(emojiBytes) + 1
	x, y, ok = FromTextFragment(root, total)
	if !ok || x != 3 || y != 0 {
		t.Errorf("trailing offset %d: want (3,0,true), got (%d,%d,%v)", total, x, y, ok)
	}
}

// TestFromTextFragment_MultiFragmentLine verifies that a line box containing
// multiple text-run child fragments (e.g., styled spans) is walked in order.
func TestFromTextFragment_MultiFragmentLine(t *testing.T) {
	// Line: [hello][world] — two separate text fragments on one line.
	frag1 := textFrag(shapedASCII("hello"))
	frag2 := textFrag(shapedASCII("world"))
	root := ifcRoot(lineFrag(frag1, frag2))

	tests := []struct {
		offset int
		wantX  int
	}{
		{0, 0},
		{5, 5},  // start of "world"
		{9, 9},  // last char of "world"
		{10, 10}, // trailing
	}

	for _, tc := range tests {
		x, y, ok := FromTextFragment(root, tc.offset)
		if !ok || x != tc.wantX || y != 0 {
			t.Errorf("multi-frag offset %d: want (%d,0,true), got (%d,%d,%v)", tc.offset, tc.wantX, x, y, ok)
		}
	}
}

// TestFromTextFragment_AtomicInline verifies that atomic-inline children
// contribute 0 bytes and their Size.Width to the cell cursor.
func TestFromTextFragment_AtomicInline(t *testing.T) {
	// Line: [atomic 3 cells][hello 5 bytes]
	atom := atomicFrag(3)
	txt := textFrag(shapedASCII("hello"))
	root := ifcRoot(lineFrag(atom, txt))

	// Byte offset 0 refers to the first byte of "hello" (atom has 0 bytes).
	// Visual x must skip the 3-cell atomic inline.
	x, y, ok := FromTextFragment(root, 0)
	if !ok || x != 3 || y != 0 {
		t.Errorf("atomic offset 0: want (3,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// Byte offset 3 → before the 'd' in "hello"
	x, y, ok = FromTextFragment(root, 3)
	if !ok || x != 6 || y != 0 {
		t.Errorf("atomic offset 3: want (6,0,true), got (%d,%d,%v)", x, y, ok)
	}

	// Trailing
	x, y, ok = FromTextFragment(root, 5)
	if !ok || x != 8 || y != 0 {
		t.Errorf("atomic trailing: want (8,0,true), got (%d,%d,%v)", x, y, ok)
	}
}

// TestFromTextFragment_TrailingOffset verifies the "byteOffset == total" case
// returns one past the last glyph on the last line.
func TestFromTextFragment_TrailingOffset(t *testing.T) {
	root := ifcRoot(
		lineFrag(textFrag(shapedASCII("abc"))),
		lineFrag(textFrag(shapedASCII("de"))),
	)
	// Total bytes = 3 + 2 = 5. Trailing offset == 5.
	x, y, ok := FromTextFragment(root, 5)
	if !ok || x != 2 || y != 1 {
		t.Errorf("trailing offset 5: want (2,1,true), got (%d,%d,%v)", x, y, ok)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// BenchmarkFromTextFragment_ShortLine benchmarks a 40-char ASCII line at
// offset 20 (mid-string). Must stay within the 60 FPS frame budget.
func BenchmarkFromTextFragment_ShortLine(b *testing.B) {
	const text40 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN"
	root := ifcRoot(lineFrag(textFrag(shapedASCII(text40))))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = FromTextFragment(root, 20)
	}
}

// BenchmarkFromTextFragment_LongWrapped benchmarks a 200-character soft-wrapped
// text across 5 line boxes with the cursor in the last line (offset 180).
// Per the task spec: must complete in under 1 µs at 60 FPS.
func BenchmarkFromTextFragment_LongWrapped(b *testing.B) {
	const lineText = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnop" // 42 chars per line × 5 = 210 chars
	lines := make([]*layout.Fragment, 5)
	for i := range lines {
		lines[i] = lineFrag(textFrag(shapedASCII(lineText)))
	}
	root := ifcRoot(lines...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = FromTextFragment(root, 180)
	}
}
