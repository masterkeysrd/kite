package editor

import (
	"testing"
)

func TestBuffer_Basic(t *testing.T) {
	b := NewBuffer("")
	b.Insert("Hello")
	if b.Value() != "Hello" {
		t.Errorf("expected Hello, got %s", b.Value())
	}
	if b.ByteOffset() != 5 {
		t.Errorf("expected offset 5, got %d", b.ByteOffset())
	}

	b.MoveToStart()
	if b.ByteOffset() != 0 {
		t.Errorf("expected offset 0, got %d", b.ByteOffset())
	}

	b.Insert("Hi ")
	if b.Value() != "Hi Hello" {
		t.Errorf("expected Hi Hello, got %s", b.Value())
	}
	if b.ByteOffset() != 3 {
		t.Errorf("expected offset 3, got %d", b.ByteOffset())
	}

	b.MoveToEnd()
	if b.ByteOffset() != 8 {
		t.Errorf("expected offset 8, got %d", b.ByteOffset())
	}
}

func TestBuffer_Unicode(t *testing.T) {
	// 👨‍👩‍👧‍👦 is a ZWJ sequence: 👨 + ZWJ + 👩 + ZWJ + 👧 + ZWJ + 👦
	// It should be treated as one grapheme cluster.
	emoji := "👨‍👩‍👧‍👦"
	b := NewBuffer(emoji)
	if b.ByteOffset() != len(emoji) {
		t.Errorf("expected offset %d, got %d", len(emoji), b.ByteOffset())
	}

	b.MoveLeft()
	if b.ByteOffset() != 0 {
		t.Errorf("expected offset 0 after MoveLeft, got %d", b.ByteOffset())
	}

	b.MoveRight()
	if b.ByteOffset() != len(emoji) {
		t.Errorf("expected offset %d after MoveRight, got %d", len(emoji), b.ByteOffset())
	}

	b.Insert("!")
	if b.Value() != emoji+"!" {
		t.Errorf("expected %s!, got %s", emoji, b.Value())
	}

	b.MoveLeft() // Move before !
	b.MoveLeft() // Move before emoji
	if b.ByteOffset() != 0 {
		t.Errorf("expected offset 0, got %d", b.ByteOffset())
	}

	b.DeleteNext()
	if b.Value() != "!" {
		t.Errorf("expected !, got %s", b.Value())
	}

	// CJK Test
	b = NewBuffer("你好世界")
	if b.ByteOffset() != 12 { // 4 * 3 bytes
		t.Errorf("expected offset 12, got %d", b.ByteOffset())
	}
	b.MoveLeft() // After 世
	if b.ByteOffset() != 9 {
		t.Errorf("expected offset 9, got %d", b.ByteOffset())
	}
	b.MoveLeft() // After 好
	if b.ByteOffset() != 6 {
		t.Errorf("expected offset 6, got %d", b.ByteOffset())
	}
}

func TestBuffer_Delete(t *testing.T) {
	b := NewBuffer("ABC")
	b.MoveLeft() // Cursor between B and C
	b.DeletePrevious()
	if b.Value() != "AC" {
		t.Errorf("expected AC, got %s", b.Value())
	}
	if b.ByteOffset() != 1 {
		t.Errorf("expected offset 1, got %d", b.ByteOffset())
	}

	b.DeleteNext()
	if b.Value() != "A" {
		t.Errorf("expected A, got %s", b.Value())
	}
	if b.ByteOffset() != 1 {
		t.Errorf("expected offset 1, got %d", b.ByteOffset())
	}
}

func TestBuffer_Words(t *testing.T) {
	b := NewBuffer("Hello world kite")
	b.MoveToStart()

	// MoveWordRight: |Hello world kite -> Hello| world kite
	b.MoveWordRight()
	if b.ByteOffset() != 5 {
		t.Errorf("expected offset 5, got %d", b.ByteOffset())
	}

	// MoveWordRight: Hello| world kite -> Hello |world kite (stops at space)
	b.MoveWordRight()
	if b.ByteOffset() != 6 {
		t.Errorf("expected offset 6, got %d", b.ByteOffset())
	}

	// MoveWordRight: Hello |world kite -> Hello world| kite
	b.MoveWordRight()
	if b.ByteOffset() != 11 {
		t.Errorf("expected offset 11, got %d", b.ByteOffset())
	}

	// MoveWordLeft: Hello world| kite -> Hello |world kite
	b.MoveWordLeft()
	if b.ByteOffset() != 6 {
		t.Errorf("expected offset 6, got %d", b.ByteOffset())
	}

	// MoveWordLeft: Hello |world kite -> Hello| world kite
	b.MoveWordLeft()
	if b.ByteOffset() != 5 {
		t.Errorf("expected offset 5, got %d", b.ByteOffset())
	}

	// MoveWordLeft: Hello| world kite -> |Hello world kite
	b.MoveWordLeft()
	if b.ByteOffset() != 0 {
		t.Errorf("expected offset 0, got %d", b.ByteOffset())
	}
}

func TestBuffer_DeleteWords(t *testing.T) {
	b := NewBuffer("Hello world kite")
	b.MoveToEnd()

	// DeleteWordPrevious: Hello world kite| -> Hello world | (deletes 'kite')
	b.DeleteWordPrevious()
	if b.Value() != "Hello world " {
		t.Errorf("expected 'Hello world ', got '%s'", b.Value())
	}

	// DeleteWordPrevious: Hello world | -> Hello world| (deletes ' ')
	b.DeleteWordPrevious()
	if b.Value() != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", b.Value())
	}

	b.MoveToStart()
	b.MoveWordRight()  // Hello|
	b.DeleteWordNext() // Delete ' ' -> Hello|world
	if b.Value() != "Helloworld" {
		t.Errorf("expected Helloworld, got '%s'", b.Value())
	}
}
