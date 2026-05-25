package element_test

import (
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/editor"
	"github.com/masterkeysrd/kite/element"
)

func BenchmarkTextArea_UpdateSelectionRange(b *testing.B) {
	doc := dom.NewDocument()
	// Large textarea content to make mapping expensive.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "This is line number "+strings.Repeat("x", 20))
	}
	content := strings.Join(lines, "\n")
	txa := element.NewTextArea(doc, content)
	doc.AppendChild(txa)

	// Ensure UA subtree is built.
	txa.SyncBuffer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Set selection at the end.
		txa.SetSelectionRange(len(content)-10, len(content))
	}
}

func BenchmarkBuffer_DeleteRange(b *testing.B) {
	content := strings.Repeat("hello world ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := editor.NewBuffer(content)
		buf.DeleteRange(100, 200)
	}
}
