package text_test

import (
	"testing"

	"github.com/masterkeysrd/kite/text"
)

// ---------------------------------------------------------------------------
// TestShape_ASCII_OneClusterPerByte
// ---------------------------------------------------------------------------

func TestShape_ASCII_OneClusterPerByte(t *testing.T) {
	input := "hello"
	clusters := text.Shape(input)

	if len(clusters) != 5 {
		t.Fatalf("got %d clusters for %q, want 5", len(clusters), input)
	}
	for i, c := range clusters {
		if len(c.Bytes) != 1 {
			t.Errorf("cluster[%d]: bytes len = %d, want 1", i, len(c.Bytes))
		}
		if c.Bytes[0] != input[i] {
			t.Errorf("cluster[%d]: byte = %q, want %q", i, c.Bytes[0], input[i])
		}
		if c.CellWidth != 1 {
			t.Errorf("cluster[%d]: width = %d, want 1", i, c.CellWidth)
		}
	}
}

// ---------------------------------------------------------------------------
// TestShape_CJK_TwoCellsEach
// ---------------------------------------------------------------------------

func TestShape_CJK_TwoCellsEach(t *testing.T) {
	// "你好" — two CJK Unified Ideographs, each occupies 2 terminal cells.
	input := "你好"
	clusters := text.Shape(input)

	if len(clusters) != 2 {
		t.Fatalf("got %d clusters for %q, want 2", len(clusters), input)
	}
	for i, c := range clusters {
		if c.CellWidth != 2 {
			t.Errorf("cluster[%d]: width = %d, want 2", i, c.CellWidth)
		}
		if c.BreakClass != text.BreakAnywhere {
			t.Errorf("cluster[%d]: BreakClass = %v, want BreakAnywhere", i, c.BreakClass)
		}
	}
}

// ---------------------------------------------------------------------------
// TestShape_Emoji_ZWJSequence_OneCluster
// ---------------------------------------------------------------------------

func TestShape_Emoji_ZWJSequence_OneCluster(t *testing.T) {
	// Family emoji: Man + ZWJ + Woman + ZWJ + Girl.
	// UAX #29 ZWJ sequence rules merge the entire run into one grapheme cluster.
	const familyEmoji = "👨\u200D👩\u200D👧"
	clusters := text.Shape(familyEmoji)

	if len(clusters) != 1 {
		t.Fatalf("ZWJ sequence shaped into %d clusters, want 1", len(clusters))
	}
	if clusters[0].CellWidth < 1 {
		t.Errorf("cluster width = %d, want >= 1", clusters[0].CellWidth)
	}
	// First rune U+1F468 (Man) is in the emoji range → BreakAnywhere.
	if clusters[0].BreakClass != text.BreakAnywhere {
		t.Errorf("BreakClass = %v, want BreakAnywhere", clusters[0].BreakClass)
	}
	// Bytes must reference the source string (same length, no allocation).
	if len(clusters[0].Bytes) != len(familyEmoji) {
		t.Errorf("Bytes len = %d, want %d (source length)", len(clusters[0].Bytes), len(familyEmoji))
	}
}

// ---------------------------------------------------------------------------
// TestShape_CombiningMark_ZeroWidth
// ---------------------------------------------------------------------------

func TestShape_CombiningMark_ZeroWidth(t *testing.T) {
	// "e\u0301" (e + combining acute accent) is ONE grapheme cluster with
	// cell width 1 (the combining mark adds no extra width).
	composed := "e\u0301"
	clusters := text.Shape(composed)
	if len(clusters) != 1 {
		t.Fatalf("\"e\\u0301\" shaped into %d clusters, want 1", len(clusters))
	}
	if clusters[0].CellWidth != 1 {
		t.Errorf("\"e\\u0301\" width = %d, want 1", clusters[0].CellWidth)
	}

	// A standalone combining grave accent (U+0300) at the start of a string
	// forms its own cluster with width 0 (no preceding base character).
	standalone := "\u0300"
	single := text.Shape(standalone)
	if len(single) != 1 {
		t.Fatalf("standalone combining mark shaped into %d clusters, want 1", len(single))
	}
	if single[0].CellWidth != 0 {
		t.Errorf("standalone combining mark width = %d, want 0", single[0].CellWidth)
	}
}

// ---------------------------------------------------------------------------
// TestBreakClass_NewlineMandatory
// ---------------------------------------------------------------------------

func TestBreakClass_NewlineMandatory(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"LF", "\n"},
		{"CR", "\r"},
		{"CRLF", "\r\n"},
		{"FF", "\f"},
		{"NEL", "\u0085"},
		{"LS", "\u2028"},
		{"PS", "\u2029"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clusters := text.Shape(tc.input)
			if len(clusters) == 0 {
				t.Fatal("got 0 clusters, want >= 1")
			}
			if clusters[0].BreakClass != text.BreakMandatory {
				t.Errorf("BreakClass = %v, want BreakMandatory", clusters[0].BreakClass)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestBreakClass_SoftHyphenSoft
// ---------------------------------------------------------------------------

func TestBreakClass_SoftHyphenSoft(t *testing.T) {
	clusters := text.Shape("\u00AD")
	if len(clusters) != 1 {
		t.Fatalf("got %d clusters, want 1", len(clusters))
	}
	if clusters[0].BreakClass != text.BreakSoft {
		t.Errorf("BreakClass = %v, want BreakSoft", clusters[0].BreakClass)
	}
}

// ---------------------------------------------------------------------------
// TestBreakClass_CJKAnywhere
// ---------------------------------------------------------------------------

func TestBreakClass_CJKAnywhere(t *testing.T) {
	// "中文" — two CJK ideographs, each gets BreakAnywhere.
	clusters := text.Shape("中文")
	for i, c := range clusters {
		if c.BreakClass != text.BreakAnywhere {
			t.Errorf("cluster[%d]: BreakClass = %v, want BreakAnywhere", i, c.BreakClass)
		}
	}
}

// ---------------------------------------------------------------------------
// TestShaper_CacheHit
// ---------------------------------------------------------------------------

func TestShaper_CacheHit(t *testing.T) {
	s := text.NewShaper(0)

	const input = "hello world"
	first := s.Shape(input)
	second := s.Shape(input)

	if len(first) == 0 {
		t.Fatal("Shape returned empty clusters")
	}
	// A cache hit must return the exact same backing array.
	if &first[0] != &second[0] {
		t.Error("expected cache hit (same backing array), got different slice allocations")
	}
}

// ---------------------------------------------------------------------------
// TestShaper_LRUEviction
// ---------------------------------------------------------------------------

func TestShaper_LRUEviction(t *testing.T) {
	// Each single-ASCII-char entry costs: len("x")=1 + 1*40=40 → 41 bytes.
	// A budget of 100 bytes holds exactly 2 entries (2*41=82 ≤ 100).
	// Inserting a third entry triggers eviction of the LRU entry.
	s := text.NewShaper(100)

	first := s.Shape("a") // entry 1; LRU order: [a]
	_ = s.Shape("b")      // entry 2; LRU order: [b, a]
	_ = s.Shape("c")      // entry 3; triggers eviction of "a"; LRU order: [c, b]

	if len(first) == 0 {
		t.Fatal("Shape(\"a\") returned empty clusters on first call")
	}

	// Re-shape "a": must have been evicted, so a new slice is created.
	second := s.Shape("a")
	if len(second) == 0 {
		t.Fatal("Shape(\"a\") returned empty clusters after eviction")
	}
	if &first[0] == &second[0] {
		t.Error("expected \"a\" to be evicted, but got a cache hit (same backing array)")
	}
}
