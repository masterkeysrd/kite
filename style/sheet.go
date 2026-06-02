package style

import "fmt"

// ---------------------------------------------------------------------------
// StyleSheet
// ---------------------------------------------------------------------------

// Sheet is an immutable, named collection of Style values. It mirrors the
// React Native StyleSheet.create pattern: styles are declared once, validated
// eagerly, and looked up by name in O(1).
//
// Construct via [NewSheet]; the returned Sheet is immutable — its entries
// cannot be changed after creation. Returned styles are value copies so
// callers cannot mutate the registry.
//
// Example:
//
//	sheet := style.NewSheet(map[string]style.Style{
//	    "button":    {display: style.Some(style.DisplayFlex)},
//	    "buttonHot": {foreground: style.Some(hotColor)},
//	})
//	box.Style(sheet.Get("button"))
type Sheet struct {
	entries map[string]Style
}

// NewSheet creates a Sheet from the given map of named styles. It validates
// each entry eagerly and returns an error if any value is malformed. The
// provided map is copied; later mutations to the caller's map do not affect
// the sheet.
//
// Validation checks performed at creation time:
//   - Dimension values that are set must have a valid Kind (non-zero tag).
//   - EdgeValues with negative cell widths are rejected.
//   - No entry may have an empty key.
func NewSheet(entries map[string]Style) (*Sheet, error) {
	s := &Sheet{
		entries: make(map[string]Style, len(entries)),
	}
	for name, st := range entries {
		if name == "" {
			return nil, fmt.Errorf("style.NewSheet: entry with empty key")
		}
		if err := validateStyle(st); err != nil {
			return nil, fmt.Errorf("style.NewSheet: entry %q: %w", name, err)
		}
		s.entries[name] = st
	}
	return s, nil
}

// Get returns the Style registered under name. If no entry exists for name,
// Get returns a zero Style (all fields unset). The returned value is a copy;
// mutations do not affect the sheet.
func (s *Sheet) Get(name string) Style {
	return s.entries[name]
}

// Has reports whether name is registered in the sheet.
func (s *Sheet) Has(name string) bool {
	_, ok := s.entries[name]
	return ok
}

// Len returns the number of entries in the sheet.
func (s *Sheet) Len() int { return len(s.entries) }

// ---------------------------------------------------------------------------
// validateStyle
// ---------------------------------------------------------------------------

// validateStyle returns a non-nil error when any set Optional field in s
// contains a value that is structurally invalid (e.g. a negative cell
// width, an out-of-range enum). The zero Style is always valid.
func validateStyle(s Style) error {
	// Dimension fields: if set, the kind must be one of the known tags.
	for _, pair := range []struct {
		name string
		d    Optional[Dimension]
	}{
		{"Width", s.width},
		{"Height", s.height},
		{"MinWidth", s.minWidth},
		{"MaxWidth", s.maxWidth},
		{"MinHeight", s.minHeight},
		{"MaxHeight", s.maxHeight},
	} {
		if pair.d.IsSet() {
			if err := validateDimension(pair.name, pair.d.Value()); err != nil {
				return err
			}
		}
	}

	// EdgeValues[int]: negative cell widths are not meaningful.
	if s.padding.IsSet() {
		if err := validateEdgeInt("Padding", s.padding.Value()); err != nil {
			return err
		}
	}
	if s.margin.IsSet() {
		if err := validateEdgeInt("Margin", s.margin.Value()); err != nil {
			return err
		}
	}

	return nil
}

// validateDimension returns an error when d has an unrecognised Kind.
func validateDimension(field string, d Dimension) error {
	switch d.Kind() {
	case KindCells, KindPercent, KindAuto, KindContent, KindFr:
		return nil
	default:
		return fmt.Errorf("%s: unknown Dimension kind %d", field, d.Kind())
	}
}

// validateEdgeInt returns an error when any edge of ev is negative.
func validateEdgeInt(field string, ev EdgeValues[int]) error {
	if ev.Top < 0 || ev.Right < 0 || ev.Bottom < 0 || ev.Left < 0 {
		return fmt.Errorf("%s: negative edge values are not allowed (got top=%d right=%d bottom=%d left=%d)",
			field, ev.Top, ev.Right, ev.Bottom, ev.Left)
	}
	return nil
}
