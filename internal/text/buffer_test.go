package text

import (
	"testing"
)

func TestBuffer_InsertDelete(t *testing.T) {
	b := NewBuffer("Hello")
	b.Insert(" 🌍") // Emojis are multiple runes

	if b.Value() != "Hello 🌍" {
		t.Errorf("Expected 'Hello 🌍', got '%s'", b.Value())
	}

	// Backspace 1 grapheme cluster (the emoji)
	b.Backspace(1)

	if b.Value() != "Hello " {
		t.Errorf("Expected 'Hello ', got '%s'", b.Value())
	}

	// Backspace another grapheme cluster (the space)
	b.Backspace(1)
	if b.Value() != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", b.Value())
	}
}

func TestBuffer_DeleteForward(t *testing.T) {
	b := NewBuffer("Hello 🌍")
	b.MoveLeft() // move before the emoji
	b.MoveLeft() // move before the space
	b.MoveLeft() // move before 'o'

	// Value is "Hello 🌍", gap is before 'o'
	// Delete 1 grapheme cluster (the 'o')
	b.Delete(1)

	if b.Value() != "Hell 🌍" {
		t.Errorf("Expected 'Hell 🌍', got '%s'", b.Value())
	}
}

func TestBuffer_ByteOffsets(t *testing.T) {
	b := NewBuffer("A 🌍 B")
	// Value: "A 🌍 B"
	// Bytes: 1 ('A') + 1 (' ') + 4 ('🌍') + 1 (' ') + 1 ('B') = 8 bytes

	// Initial position is at the end
	if b.ByteOffset() != 8 {
		t.Errorf("Expected byte offset 8, got %d", b.ByteOffset())
	}

	// Set to 0
	b.SetByteOffset(0)
	if b.ByteOffset() != 0 {
		t.Errorf("Expected byte offset 0, got %d", b.ByteOffset())
	}

	// Set to just before the emoji (offset 2)
	b.SetByteOffset(2)
	if b.ByteOffset() != 2 {
		t.Errorf("Expected byte offset 2, got %d", b.ByteOffset())
	}

	// Set to just after the emoji (offset 6)
	b.SetByteOffset(6)
	if b.ByteOffset() != 6 {
		t.Errorf("Expected byte offset 6, got %d", b.ByteOffset())
	}
}

func TestBuffer_VersionAndLen(t *testing.T) {
	b := NewBuffer("Hello")
	if b.Version() != 0 {
		t.Errorf("Expected version 0, got %d", b.Version())
	}
	if b.Len() != 5 {
		t.Errorf("Expected len 5, got %d", b.Len())
	}
	if b.ByteLen() != 5 {
		t.Errorf("Expected byte len 5, got %d", b.ByteLen())
	}

	b.Insert("!")
	if b.Version() != 1 {
		t.Errorf("Expected version 1, got %d", b.Version())
	}
	if b.Len() != 6 {
		t.Errorf("Expected len 6, got %d", b.Len())
	}

	b.Backspace(1)
	if b.Version() != 2 {
		t.Errorf("Expected version 2, got %d", b.Version())
	}

	b.Reset("Hi 🌍")
	if b.Version() != 3 {
		t.Errorf("Expected version 3, got %d", b.Version())
	}
	if b.Len() != 4 { // 'H', 'i', ' ', '🌍'
		t.Errorf("Expected len 4, got %d", b.Len())
	}
	if b.ByteLen() != 7 { // "Hi " (3) + "🌍" (4) = 7
		t.Errorf("Expected byte len 7, got %d", b.ByteLen())
	}
}

func TestBuffer_DeleteRange(t *testing.T) {
	b := NewBuffer("Hello 🌍 World")
	// "Hello " (6) + "🌍" (4) + " World" (6) = 16 bytes

	// Delete the emoji (bytes 6 to 10)
	b.DeleteRange(6, 10)
	if b.Value() != "Hello  World" {
		t.Errorf("Expected 'Hello  World', got '%s'", b.Value())
	}

	// Delete "Hello " (0 to 6)
	b.DeleteRange(0, 6)
	if b.Value() != " World" {
		t.Errorf("Expected ' World', got '%s'", b.Value())
	}
}

func TestBuffer_Unicode_Old(t *testing.T) {
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

func TestBuffer_Words_Old(t *testing.T) {
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

func TestBuffer_Delete_Old(t *testing.T) {
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

func TestBuffer_DeleteWords_Old(t *testing.T) {
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
