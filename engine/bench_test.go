package engine

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
)

func BenchmarkPipeline_Standard(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Frame()
	}
}

func BenchmarkPipeline_Profiling(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{Profiler: true})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Frame()
	}
}

func BenchmarkPipeline_SyncAndLayout_100Nodes(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{})

	// Add 100 child elements to the document
	doc := e.Document()
	for i := 0; i < 100; i++ {
		child := doc.CreateElement("div", nil)
		doc.AppendChild(child)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Mark the document sync-dirty to trigger diffChildren
		if dn := internaldom.AsDirty(doc); dn != nil {
			dn.MarkNeedsSync()
		}
		// Mark children style-dirty
		for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
			if de := internaldom.AsDirtyElement(child); de != nil {
				de.MarkStyleDirty()
			}
		}

		e.pipeline.Sync(e)
		e.pipeline.Style(e)
		e.pipeline.Layout(e)
	}
}
