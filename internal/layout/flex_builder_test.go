package layout

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestFlexLineBuilder_ResolveFlexibleLengths_Grow(t *testing.T) {
	geom := flexGeometry{direction: style.FlexRow}
	builder := NewFlexLineBuilder(geom, 0, 0)

	// Add 3 items with flex-grow: 1
	for i := 0; i < 3; i++ {
		builder.AddItem(&mockNode{}, 10, 0, 0, 1, 0, 0)
	}

	builder.ComputeLines(60, false)
	builder.ResolveFlexibleLengths(0, 60)

	lines := builder.Lines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	for i, item := range lines[0].Items {
		if item.MainSize != 20 {
			t.Errorf("item %d: expected MainSize 20, got %d", i, item.MainSize)
		}
	}
}

func TestFlexLineBuilder_FreezeAndRestart(t *testing.T) {
	geom := flexGeometry{direction: style.FlexRow}
	builder := NewFlexLineBuilder(geom, 0, 0)

	// Item 1: flex-grow: 1, max-width: 15
	builder.AddItem(&mockNode{}, 10, 0, 15, 1, 0, 0)
	// Item 2: flex-grow: 1
	builder.AddItem(&mockNode{}, 10, 0, 0, 1, 0, 0)

	// Total available: 40. Total hypothetical: 20. Free space: 20.
	// Both want to grow by 10.
	// Item 1 hits max 15. It freezes.
	// Remaining free space for Item 2: 40 - 15 - 10 = 15.
	// Item 2 grows by 15 to reach 25.

	builder.ComputeLines(40, false)
	builder.ResolveFlexibleLengths(0, 40)

	items := builder.Lines()[0].Items
	if items[0].MainSize != 15 {
		t.Errorf("item 0: expected MainSize 15, got %d", items[0].MainSize)
	}
	if items[1].MainSize != 25 {
		t.Errorf("item 1: expected MainSize 25, got %d", items[1].MainSize)
	}
}
