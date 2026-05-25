package dom_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func BenchmarkRange_String_UAContent(b *testing.B) {
	doc := dom.NewDocument()
	host := doc.CreateElement("div", nil)
	uaRoot := doc.CreateElement("ua-root", nil)

	// Create a large amount of content in the UA subtree.
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		text := fmt.Sprintf("Line %d: This is some sample text for benchmarking selection stringification.\n", i)
		uaRoot.AppendChild(doc.CreateTextNode(text, nil))
		sb.WriteString(text)
	}
	host.AttachUARoot(uaRoot)
	doc.AppendChild(host)

	r := doc.CreateRange()
	r.SetStart(uaRoot.FirstChild(), 0)
	r.SetEnd(uaRoot.LastChild(), 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.String()
	}
}
