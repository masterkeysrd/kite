package element_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/testenv"
)

func BenchmarkCaretMovement(b *testing.B) {
	env := testenv.New(b, 80, 25)

	// Create a cursor-navigable button
	btn := element.Button("  Hello World  ").WithID("btn").CursorNavigable(true)
	env.Mount(btn)
	env.Flush()

	// Focus the button initially
	btn.Focus()
	env.Flush()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Move caret right
		env.Engine.MoveCaret(dom.DirectionRight)
		// Move caret left
		env.Engine.MoveCaret(dom.DirectionLeft)
	}
}
