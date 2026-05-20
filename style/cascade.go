package style

// CascadeOrigin identifies the layer of the style cascade that a value comes
// from. Origins are listed from weakest to strongest; a higher origin wins
// when both provide a value for the same property.
//
// The enum is internal to the style package (ADR-010). It must not leak into
// dom, render, layout, or paint.
type CascadeOrigin uint8

const (
	// OriginInherited represents a value that was inherited from the parent's
	// Computed style. This is the weakest origin.
	OriginInherited CascadeOrigin = iota

	// OriginUADefault represents the element-type default style contributed
	// by DefaultStyle(). Overridable by the author.
	OriginUADefault

	// OriginAuthor represents the author-set sparse style from RawStyle().
	OriginAuthor

	// OriginUserAgent represents the UA-mandated intrinsic style from
	// IntrinsicStyle(). This is the strongest origin; authors cannot override it.
	OriginUserAgent
)
