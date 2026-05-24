package engine

import (
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
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
