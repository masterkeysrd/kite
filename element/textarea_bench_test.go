package element

import (
	"fmt"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/engine"
)

func setupBenchTextArea(lines int) (*engine.Engine, *TextAreaElement) {
	be := mock.New(80, 24)
	eng := engine.New(be, engine.Options{})

	var sb strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&sb, "This is line number %d\n", i)
	}

	txa := NewTextArea(eng.Document(), sb.String())
	eng.Document().AppendChild(txa)
	eng.Frame() // initial layout

	return eng, txa
}

func BenchmarkTextArea_CursorMove(b *testing.B) {
	sizes := []int{50, 500}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Lines-%d", size), func(b *testing.B) {
			_, txa := setupBenchTextArea(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Move right and sync.
				// Before optimization, this triggered rebuildUASubtree.
				// Now it should only mark DirtyPaint.
				txa.Buffer().MoveRight()
				txa.SyncBuffer()
			}
		})
	}
}

func BenchmarkTextArea_Insert(b *testing.B) {
	sizes := []int{50, 500}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Lines-%d", size), func(b *testing.B) {
			_, txa := setupBenchTextArea(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Insert character and sync.
				// This triggers incremental rebuildUASubtree.
				txa.Buffer().Insert("a")
				txa.SyncBuffer()
			}
		})
	}
}

func BenchmarkTextArea_Frame(b *testing.B) {
	sizes := []int{50, 500}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("Lines-%d", size), func(b *testing.B) {
			eng, txa := setupBenchTextArea(size)
			// Ensure txa is focused for cursor math
			eng.Frame()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Force a frame update.
				// This tests ScrollCursorIntoView and updateHardwareCursor caching.
				txa.needsScrollIntoView = true
				eng.Frame()
			}
		})
	}
}
