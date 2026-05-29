package text

import (
	"strings"
	"testing"
)

func BenchmarkBuffer_InsertAtEnd(b *testing.B) {
	buf := NewBuffer("")

	for b.Loop() {
		buf.Insert("a")
	}
}

func BenchmarkBuffer_InsertAtStart(b *testing.B) {
	buf := NewBuffer(strings.Repeat("x", 1000))

	for b.Loop() {
		buf.MoveStart()
		buf.Insert("a")
	}
}

func BenchmarkBuffer_Move(b *testing.B) {
	buf := NewBuffer(strings.Repeat("x", 1000))

	for i := 0; b.Loop(); i++ {
		if i%2 == 0 {
			buf.MoveStart()
		} else {
			buf.MoveEnd()
		}
	}
}

func BenchmarkBuffer_Value(b *testing.B) {
	buf := NewBuffer(strings.Repeat("x", 1000))
	buf.MoveLeft() // ensure gap is in the middle

	for b.Loop() {
		_ = buf.Value()
	}
}

func BenchmarkBuffer_DeleteRange(b *testing.B) {
	initial := strings.Repeat("x", 10000)

	for b.Loop() {
		b.StopTimer()
		buf := NewBuffer(initial)
		b.StartTimer()
		buf.DeleteRange(100, 500)
	}
}

func BenchmarkBuffer_GraphemeOps(b *testing.B) {
	buf := NewBuffer(strings.Repeat("🌍", 100))

	for i := 0; b.Loop(); i++ {
		buf.MoveLeft()
		if i%10 == 0 {
			buf.MoveEnd()
		}
	}
}

func BenchmarkBuffer_Backspace(b *testing.B) {
	initial := strings.Repeat("🌍", 1000)
	buf := NewBuffer(initial)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Backspace(1)
		if buf.Len() == 0 {
			b.StopTimer()
			buf.Reset(initial)
			b.StartTimer()
		}
	}
}

func BenchmarkBuffer_MoveWord(b *testing.B) {
	buf := NewBuffer(strings.Repeat("Hello world kite ", 100))

	for i := 0; b.Loop(); i++ {
		buf.MoveWordLeft()
		if i%20 == 0 {
			buf.MoveEnd()
		}
	}
}
