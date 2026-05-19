// Package text provides grapheme cluster segmentation, cell-width measurement,
// and line-break classification for the kite layout engine.
//
// The primary API is [Shape], which segments a string into [Cluster] values.
// For repeated shaping of the same strings, use [Shaper], which caches results
// under a byte-budget LRU policy.
//
// Package text never imports dom, render, paint, events, focus, or backend.
// It may import style.
package text

// BreakClass describes the line-break opportunity BEFORE a grapheme cluster.
// Values mirror the simplified UAX.
type BreakClass int

const (
	// BreakNone means there is no line-break opportunity before this cluster.
	BreakNone BreakClass = iota

	// BreakSoft is a discretionary break: at word boundaries (between a
	// whitespace cluster and the following non-whitespace cluster) and at
	// U+00AD (SOFT HYPHEN). When a soft-hyphen break is taken, a "-" is
	// rendered at the end of the line.
	BreakSoft

	// BreakMandatory is an unconditional break: LF, CR, CRLF, FF, U+0085
	// (NEL), U+2028 (LINE SEPARATOR), U+2029 (PARAGRAPH SEPARATOR).
	BreakMandatory

	// BreakAnywhere marks every cluster boundary as a break opportunity.
	// Assigned to CJK Unified Ideographs, Hangul, Hiragana, Katakana, and
	// emoji clusters.
	BreakAnywhere
)

// Cluster is a shaped extended grapheme cluster the atomic unit of
// text rendering and line-breaking in kite.
//
// Bytes references the source string's backing memory without copying. Callers
// must not modify the slice and must ensure the source string outlives the
// returned clusters.
type Cluster struct {
	Bytes      []byte     // UTF-8 grapheme bytes (no copy; refs source)
	CellWidth  int        // 0, 1, or 2 per East Asian Width + emoji presentation
	BreakClass BreakClass // line-break opportunity before this cluster
}
