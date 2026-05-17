package style

// DimensionKind identifies which variant of [Dimension] is active.
type DimensionKind uint8

const (
	// KindCells is a fixed number of terminal cells.
	KindCells DimensionKind = iota
	// KindPercent is a percentage of the parent's dimension.
	KindPercent
	// KindAuto lets the layout engine determine the dimension.
	KindAuto
	// KindContent sizes the element to fit its content.
	KindContent
	// KindFr is a fractional unit used in flex/grid contexts.
	KindFr
)

// Dimension is a tagged union representing a layout dimension. Use the
// constructor functions [Cells], [Percent], [Fr] and the package variables
// [Auto] and [Content] to create values; do not construct Dimension literals
// directly.
type Dimension struct {
	kind    DimensionKind
	cells   int
	percent float32
}

// Kind returns which variant is active.
func (d Dimension) Kind() DimensionKind { return d.kind }

// Cells returns a Dimension measured in a fixed number of terminal cells.
func Cells(n int) Dimension { return Dimension{kind: KindCells, cells: n} }

// Percent returns a Dimension measured as a percentage of the parent dimension.
// For example Percent(50) means 50 %.
func Percent(pct float32) Dimension { return Dimension{kind: KindPercent, percent: pct} }

// Fr returns a Dimension measured in fractional units (for flex/grid layouts).
func Fr(n int) Dimension { return Dimension{kind: KindFr, cells: n} }

// Auto is a Dimension that lets the layout engine choose the size.
var Auto = Dimension{kind: KindAuto}

// Content is a Dimension that sizes the element to fit its content.
var Content = Dimension{kind: KindContent}

// CellsValue returns the fixed cell count. Only valid when Kind() == KindCells
// or KindFr; returns 0 otherwise.
func (d Dimension) CellsValue() int { return d.cells }

// PercentValue returns the percentage. Only valid when Kind() == KindPercent;
// returns 0 otherwise.
func (d Dimension) PercentValue() float32 { return d.percent }
