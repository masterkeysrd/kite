package text

import (
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/rivo/uniseg"
)

// Shape segments text into extended grapheme clusters, assigns
// East-Asian-Width cell widths, and classifies line-break opportunities.
//
// The Bytes field of each returned Cluster aliases text's backing memory
// without copying. text must outlive the returned slice.
//
// Returns nil for an empty string.
func Shape(text string) []Cluster {
	if text == "" {
		return nil
	}

	// Fast path: for short strings, build on stack first to avoid capacity overallocation.
	if len(text) <= 64 {
		var localBuf [64]Cluster
		count := 0
		remaining := text
		state := -1
		prevWasBreakableSpace := false

		for len(remaining) > 0 {
			var clusterStr string
			var width int
			clusterStr, remaining, width, state = uniseg.FirstGraphemeClusterInString(remaining, state)

			r, _ := utf8.DecodeRuneInString(clusterStr)
			bc := classifyBreak(r, prevWasBreakableSpace)

			localBuf[count] = Cluster{
				Bytes:      unsafeStringBytes(clusterStr),
				CellWidth:  width,
				BreakClass: bc,
			}
			count++
			prevWasBreakableSpace = isBreakableSpace(r)
		}

		// Allocate exactly count items.
		clusters := make([]Cluster, count)
		copy(clusters, localBuf[:count])
		return clusters
	}

	// For long strings, estimate capacity using rune count instead of byte length.
	runeCount := utf8.RuneCountInString(text)
	clusters := make([]Cluster, 0, runeCount)

	remaining := text
	state := -1
	prevWasBreakableSpace := false

	for len(remaining) > 0 {
		var clusterStr string
		var width int
		clusterStr, remaining, width, state = uniseg.FirstGraphemeClusterInString(remaining, state)

		r, _ := utf8.DecodeRuneInString(clusterStr)
		bc := classifyBreak(r, prevWasBreakableSpace)

		clusters = append(clusters, Cluster{
			Bytes:      unsafeStringBytes(clusterStr),
			CellWidth:  width,
			BreakClass: bc,
		})

		prevWasBreakableSpace = isBreakableSpace(r)
	}

	// Shrink capacity to match exact length to prevent cache memory bloat.
	if cap(clusters) > len(clusters) {
		exactClusters := make([]Cluster, len(clusters))
		copy(exactClusters, clusters)
		return exactClusters
	}

	return clusters
}

// unsafeStringBytes returns a []byte that aliases s's backing memory without
// copying. The caller must not write through the returned slice.
func unsafeStringBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// classifyBreak returns the BreakClass for a cluster whose first rune is r,
// given whether the immediately-preceding cluster was a breakable space.
func classifyBreak(r rune, prevWasBreakableSpace bool) BreakClass {
	// Mandatory line-break characters.
	switch r {
	case '\n', '\r', '\f', '\u0085', '\u2028', '\u2029':
		return BreakMandatory
	}

	// Soft hyphen: discretionary break point.
	if r == '\u00AD' {
		return BreakSoft
	}

	// CJK / emoji: every cluster boundary is a break opportunity.
	if isCJKOrEmoji(r) {
		return BreakAnywhere
	}

	// Word boundary: non-space cluster that immediately follows a breakable
	// space cluster. This models the "between whitespace and non-whitespace".
	if prevWasBreakableSpace && !isBreakableSpace(r) {
		return BreakSoft
	}

	return BreakNone
}

// isBreakableSpace reports whether r is a whitespace character that introduces
// a word-boundary break opportunity before the NEXT non-space cluster.
//
// Non-breaking space (U+00A0) is intentionally excluded.
func isBreakableSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\v':
		return true
	}
	// Guard against non-ASCII range: Zs category characters are always U+0080 or above.
	if r < 128 {
		return false
	}
	// Unicode general category Zs (space separators) except NBSP.
	return r != '\u00A0' && unicode.Is(unicode.Zs, r)
}

// isCJKOrEmoji reports whether r belongs to a CJK ideograph, CJK-adjacent
// syllabary, or emoji range that warrants BreakAnywhere treatment.
func isCJKOrEmoji(r rune) bool {
	// Guard: the lowest CJK/emoji range block is Miscellaneous Symbols (U+2600).
	if r < 0x2600 {
		return false
	}
	switch {
	// Hiragana (U+3040–U+309F) and Katakana (U+30A0–U+30FF)
	case r >= 0x3040 && r <= 0x30FF:
		return true
	// CJK Extension A (U+3400–U+4DBF)
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	// CJK Unified Ideographs — main block (U+4E00–U+9FFF)
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	// CJK Compatibility Ideographs (U+F900–U+FAFF)
	case r >= 0xF900 && r <= 0xFAFF:
		return true
	// Hangul Syllables (U+AC00–U+D7AF)
	case r >= 0xAC00 && r <= 0xD7AF:
		return true
	// CJK Extension B (U+20000–U+2A6DF)
	case r >= 0x20000 && r <= 0x2A6DF:
		return true
	// Regional Indicator Symbols / flags (U+1F1E0–U+1F1FF)
	case r >= 0x1F1E0 && r <= 0x1F1FF:
		return true
	// Miscellaneous Symbols and Dingbats (U+2600–U+27BF)
	case r >= 0x2600 && r <= 0x27BF:
		return true
	// Emoji: Miscellaneous Symbols and Pictographs through
	// Symbols and Pictographs Extended-A (U+1F300–U+1FAFF)
	case r >= 0x1F300 && r <= 0x1FAFF:
		return true
	default:
		return false
	}
}
