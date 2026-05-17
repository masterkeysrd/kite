package text_test

import (
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/text"
)

// BenchmarkShape_1kASCII measures the cost of shaping 1 000 ASCII characters
// from scratch (no cache).
func BenchmarkShape_1kASCII(b *testing.B) {
	input := strings.Repeat("a", 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = text.Shape(input)
	}
}

// BenchmarkShape_1kCJK measures the cost of shaping 1 000 CJK ideographs
// from scratch (each is a 3-byte UTF-8 sequence, cell width 2).
func BenchmarkShape_1kCJK(b *testing.B) {
	input := strings.Repeat("中", 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = text.Shape(input)
	}
}

// BenchmarkShape_1kEmoji_ZWJ measures the cost of shaping ~250 family-emoji
// ZWJ sequences (each is one grapheme cluster composed of 5 code points joined
// by ZWJ, totalling ~4 500 UTF-8 bytes).
func BenchmarkShape_1kEmoji_ZWJ(b *testing.B) {
	// "👨‍👩‍👧" = Man + ZWJ + Woman + ZWJ + Girl (18 bytes, 1 cluster)
	const familyEmoji = "👨\u200D👩\u200D👧"
	input := strings.Repeat(familyEmoji, 250)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = text.Shape(input)
	}
}

// BenchmarkShaper_CacheHit measures the throughput of cache-hit paths in
// Shaper.Shape. It must be significantly faster than BenchmarkShape_1kASCII.
func BenchmarkShaper_CacheHit(b *testing.B) {
	s := text.NewShaper(0)
	const input = "the quick brown fox jumps over the lazy dog"
	s.Shape(input) // warm the cache

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = s.Shape(input)
	}
}

// BenchmarkMeasureRun_LongLatinParagraph measures MeasureRun on a realistic
// prose string, exercising the shaper cache for repeated sub-strings.
func BenchmarkMeasureRun_LongLatinParagraph(b *testing.B) {
	paragraph := strings.Repeat(
		"The quick brown fox jumps over the lazy dog. "+
			"Pack my box with five dozen liquor jugs. ",
		20, // ~1 800 characters
	)
	s := text.NewShaper(0)
	s.MeasureRun(paragraph) // warm the cache

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = s.MeasureRun(paragraph)
	}
}
