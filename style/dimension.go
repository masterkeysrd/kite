package style

// DimensionKind identifies which variant of [Dimension] is active.
type DimensionKind uint8

const (
	// KindAuto lets the layout engine determine the dimension.
	KindAuto DimensionKind = iota
	// KindCells is a fixed number of terminal cells.
	KindCells
	// KindPercent is a percentage of the parent's dimension.
	KindPercent
	// KindContent sizes the element to fit its content, capped at the available width.
	KindContent
	// KindFr is a fractional unit used in flex/grid contexts.
	KindFr
	// KindMaxContent sizes the element to its unconstrained max-content width,
	// ignoring the available width limit. Used internally by UA shadow subtrees
	// (e.g. the inner div of <input>) that must be able to overflow their clip
	// container so that the host's programmatic scroll offset can pan the content.
	KindMaxContent
)

// Dimension is a tagged union representing a layout dimension. Use the
// constructor functions [Cells], [Percent], [Fr] and the package variables
// [Auto] and [Content] to create values; do not construct Dimension literals
// directly.
type Dimension struct {
	kind    DimensionKind
	cells   int
	percent float32
	fr      float32
}

// Kind returns which variant is active.
func (d Dimension) Kind() DimensionKind { return d.kind }

// Cells returns a Dimension measured in a fixed number of terminal cells.
func Cells(n int) Dimension { return Dimension{kind: KindCells, cells: n} }

// Percent returns a Dimension measured as a percentage of the parent dimension.
// For example Percent(50) means 50 %.
func Percent(pct float32) Dimension { return Dimension{kind: KindPercent, percent: pct} }

// Fr returns a Dimension measured in fractional units (for flex/grid layouts).
func Fr(f float32) Dimension { return Dimension{kind: KindFr, fr: f} }

// Auto is a Dimension that lets the layout engine choose the size.
var Auto = Dimension{kind: KindAuto}

// Content is a Dimension that sizes the element to fit its content, capped
// at the available width.
var Content = Dimension{kind: KindContent}

// MaxContent is a Dimension that sizes the element to its unconstrained
// max-content width, not capped by the available width. Used for UA shadow
// subtree inner elements that need to overflow a clip container.
var MaxContent = Dimension{kind: KindMaxContent}

// CellsValue returns the fixed cell count. Only valid when Kind() == KindCells;
// returns 0 otherwise.
func (d Dimension) CellsValue() int { return d.cells }

// PercentValue returns the percentage. Only valid when Kind() == KindPercent;
// returns 0 otherwise.
func (d Dimension) PercentValue() float32 { return d.percent }

// FrValue returns the fractional value. Only valid when Kind() == KindFr;
// returns 0 otherwise.
func (d Dimension) FrValue() float32 { return d.fr }
