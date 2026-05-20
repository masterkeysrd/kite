package editor

import (
	"github.com/rivo/uniseg"
)

// Buffer manages 1-dimensional string edits with Unicode safety.
type Buffer struct {
	text       string
	byteOffset int
}

// NewBuffer creates a new Buffer with the given initial text.
func NewBuffer(text string) *Buffer {
	return &Buffer{
		text:       text,
		byteOffset: len(text),
	}
}

// Value returns the current string value of the buffer.
func (b *Buffer) Value() string {
	return b.text
}

// ByteOffset returns the current cursor position in bytes.
func (b *Buffer) ByteOffset() int {
	return b.byteOffset
}

// Insert adds the given string at the current cursor position.
func (b *Buffer) Insert(s string) {
	b.text = b.text[:b.byteOffset] + s + b.text[b.byteOffset:]
	b.byteOffset += len(s)
}

// MoveToStart moves the cursor to the beginning of the buffer.
func (b *Buffer) MoveToStart() {
	b.byteOffset = 0
}

// MoveToEnd moves the cursor to the end of the buffer.
func (b *Buffer) MoveToEnd() {
	b.byteOffset = len(b.text)
}

// MoveRight moves the cursor one grapheme cluster to the right.
func (b *Buffer) MoveRight() {
	if b.byteOffset >= len(b.text) {
		return
	}
	_, rest, _, _ := uniseg.StepString(b.text[b.byteOffset:], -1)
	clusterLen := len(b.text[b.byteOffset:]) - len(rest)
	b.byteOffset += clusterLen
}

// MoveLeft moves the cursor one grapheme cluster to the left.
func (b *Buffer) MoveLeft() {
	if b.byteOffset <= 0 {
		return
	}

	// We iterate from the beginning of the string to find the grapheme
	// cluster that starts before the current byteOffset.
	g := uniseg.NewGraphemes(b.text[:b.byteOffset])
	newOffset := 0
	for g.Next() {
		start, _ := g.Positions()
		newOffset = start
	}
	b.byteOffset = newOffset
}

// DeletePrevious removes the grapheme cluster before the cursor (Backspace).
func (b *Buffer) DeletePrevious() {
	if b.byteOffset <= 0 {
		return
	}
	oldOffset := b.byteOffset
	b.MoveLeft()
	b.text = b.text[:b.byteOffset] + b.text[oldOffset:]
}

// DeleteNext removes the grapheme cluster after the cursor (Delete).
func (b *Buffer) DeleteNext() {
	if b.byteOffset >= len(b.text) {
		return
	}
	_, rest, _, _ := uniseg.StepString(b.text[b.byteOffset:], -1)
	clusterLen := len(b.text[b.byteOffset:]) - len(rest)
	b.text = b.text[:b.byteOffset] + b.text[b.byteOffset+clusterLen:]
}

// MoveWordRight moves the cursor to the end of the current or next word.
func (b *Buffer) MoveWordRight() {
	if b.byteOffset >= len(b.text) {
		return
	}

	remaining := b.text[b.byteOffset:]
	state := -1
	for len(remaining) > 0 {
		var rest string
		var boundaries int
		_, rest, boundaries, state = uniseg.StepString(remaining, state)
		clusterLen := len(remaining) - len(rest)
		b.byteOffset += clusterLen
		remaining = rest
		if boundaries&uniseg.MaskWord != 0 {
			break
		}
	}
}

// MoveWordLeft moves the cursor to the start of the current or previous word.
func (b *Buffer) MoveWordLeft() {
	if b.byteOffset <= 0 {
		return
	}

	// Iterate from the start to find all word boundaries before the current offset.
	// The new offset will be the last boundary found that is less than the current offset.
	remaining := b.text[:b.byteOffset]
	state := -1
	lastBoundary := 0
	currentPos := 0
	for len(remaining) > 0 {
		var rest string
		var boundaries int
		_, rest, boundaries, state = uniseg.StepString(remaining, state)
		clusterLen := len(remaining) - len(rest)
		currentPos += clusterLen
		remaining = rest
		if boundaries&uniseg.MaskWord != 0 && currentPos < b.byteOffset {
			lastBoundary = currentPos
		}
	}
	b.byteOffset = lastBoundary
}

// DeleteWordPrevious removes the word before the cursor.
func (b *Buffer) DeleteWordPrevious() {
	if b.byteOffset <= 0 {
		return
	}
	oldOffset := b.byteOffset
	b.MoveWordLeft()
	b.text = b.text[:b.byteOffset] + b.text[oldOffset:]
}

// DeleteWordNext removes the word after the cursor.
func (b *Buffer) DeleteWordNext() {
	if b.byteOffset >= len(b.text) {
		return
	}
	oldOffset := b.byteOffset
	b.MoveWordRight()
	b.text = b.text[:oldOffset] + b.text[b.byteOffset:]
	b.byteOffset = oldOffset
}
