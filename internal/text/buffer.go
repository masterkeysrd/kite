package text

import (
	"sync"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

var byteBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func getByteBuf() *[]byte {
	return byteBufPool.Get().(*[]byte)
}

func putByteBuf(b *[]byte) {
	if cap(*b) > 4096 {
		*b = make([]byte, 0, 1024)
	}
	*b = (*b)[:0]
	byteBufPool.Put(b)
}

// Buffer manages 1-dimensional string edits with Unicode safety using a gap buffer.
type Buffer struct {
	content  []rune
	gapStart int
	gapEnd   int
	version  int64
}

// NewBuffer creates a new Buffer with the given initial text.
func NewBuffer(text string) *Buffer {
	runes := []rune(text)
	// Initial capacity is the length of text plus some extra gap
	capacity := len(runes) + 16
	content := make([]rune, capacity)
	copy(content, runes)
	return &Buffer{
		content:  content,
		gapStart: len(runes),
		gapEnd:   capacity,
		version:  0,
	}
}

// Version returns a monotonically increasing number that increments after
// every text-mutating operation. Moving the cursor does not increment the
// version.
func (b *Buffer) Version() int64 {
	return b.version
}

// Reset replaces the entire buffer content with the given text and moves
// the cursor to the end.
func (b *Buffer) Reset(text string) {
	runes := []rune(text)
	capacity := len(runes) + 16
	content := make([]rune, capacity)
	copy(content, runes)
	b.content = content
	b.gapStart = len(runes)
	b.gapEnd = capacity
	b.version++
}

// Len returns the number of runes in the buffer.
func (b *Buffer) Len() int {
	return b.gapStart + (len(b.content) - b.gapEnd)
}

// ByteLen returns the number of bytes in the buffer.
func (b *Buffer) ByteLen() int {
	// This is still slightly expensive due to string conversion if we don't count
	// manually, but ByteLen is rarely called in hot loops.
	return len(b.Value())
}

// Insert adds the given text at the current gap position.
func (b *Buffer) Insert(text string) {
	runes := []rune(text)
	needed := len(runes)
	b.ensureSpace(needed)

	copy(b.content[b.gapStart:], runes)
	b.gapStart += needed
	b.version++
}

// Backspace removes the specified number of grapheme clusters before the gap.
func (b *Buffer) Backspace(graphemeCount int) {
	if graphemeCount <= 0 || b.gapStart <= 0 {
		return
	}

	// For a small number of graphemes, we can use a windowed scan to avoid
	// scanning the entire buffer from the start.
	windowRunes := 64
	if graphemeCount > 10 {
		windowRunes = graphemeCount * 4
	}

	start := b.gapStart - windowRunes
	if start < 0 {
		start = 0
	}

	runes := b.content[start:b.gapStart]

	// Convert runes to bytes using pooled buffer to avoid allocations.
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf
	for _, r := range runes {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], r)
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	// Track rune offsets of grapheme boundaries.
	var stackOffsets [64]int
	var offsets []int
	if graphemeCount <= 64 {
		offsets = stackOffsets[:0]
	} else {
		offsets = make([]int, 0, 128)
	}

	state := -1
	currentRune := start
	remaining := byteBuf
	for len(remaining) > 0 {
		var cluster []byte
		cluster, remaining, _, state = uniseg.Step(remaining, state)
		offsets = append(offsets, currentRune)
		currentRune += utf8.RuneCount(cluster)
	}

	if graphemeCount >= len(offsets) {
		b.gapStart = 0
	} else {
		b.gapStart = offsets[len(offsets)-graphemeCount]
	}
	b.version++
	*pBuf = byteBuf // Update pooled buffer header in case it grew
}

// Delete removes the specified number of grapheme clusters after the gap.
func (b *Buffer) Delete(graphemeCount int) {
	if graphemeCount <= 0 || b.gapEnd >= len(b.content) {
		return
	}

	runes := b.content[b.gapEnd:]
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf

	// For correctness with complex emojis, we MUST scan enough runes to see
	// the full grapheme clusters.
	// Let's scan everything for now if it's within a reasonable limit (e.g. 1000 runes),
	// otherwise use a larger heuristic.
	scanLimit := 1024
	if graphemeCount > 200 {
		scanLimit = graphemeCount * 8
	}
	if scanLimit > len(runes) {
		scanLimit = len(runes)
	}

	for i := 0; i < scanLimit; i++ {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], runes[i])
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	state := -1
	runesToDelete := 0
	remaining := byteBuf
	count := 0
	for count < graphemeCount && len(remaining) > 0 {
		var cluster []byte
		cluster, remaining, _, state = uniseg.Step(remaining, state)
		runesToDelete += utf8.RuneCount(cluster)
		count++
	}

	b.gapEnd += runesToDelete
	b.version++
	*pBuf = byteBuf
}

// MoveLeft moves the gap one grapheme cluster to the left.
func (b *Buffer) MoveLeft() {
	if b.gapStart <= 0 {
		return
	}

	window := 32
	start := b.gapStart - window
	if start < 0 {
		start = 0
	}

	runes := b.content[start:b.gapStart]
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf
	for _, r := range runes {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], r)
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	state := -1
	newStartRune := start
	remaining := byteBuf
	for len(remaining) > 0 {
		var cluster []byte
		cluster, remaining, _, state = uniseg.Step(remaining, state)
		if len(remaining) == 0 {
			break
		}
		newStartRune += utf8.RuneCount(cluster)
	}

	b.moveGap(newStartRune)
	*pBuf = byteBuf
}

// MoveRight moves the gap one grapheme cluster to the right.
func (b *Buffer) MoveRight() {
	if b.gapEnd >= len(b.content) {
		return
	}

	runes := b.content[b.gapEnd:]
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf

	// Only need the first grapheme cluster
	for i := 0; i < 8 && i < len(runes); i++ {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], runes[i])
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	_, _, _, _ = uniseg.Step(byteBuf, -1)
	// We need the rune count of the first cluster.
	// Step doesn't give it directly, but we can use FirstGraphemeClusterRunes
	cluster, _, _, _ := uniseg.FirstGraphemeCluster(byteBuf, -1)
	runeLen := utf8.RuneCount(cluster)

	b.moveGap(b.gapStart + runeLen)
	*pBuf = byteBuf
}

// MoveStart moves the gap to the beginning of the buffer.
func (b *Buffer) MoveStart() {
	b.moveGap(0)
}

// MoveEnd moves the gap to the end of the buffer.
func (b *Buffer) MoveEnd() {
	b.moveGap(len(b.content) - (b.gapEnd - b.gapStart))
}

// ByteOffset returns the current gap (cursor) position as a byte offset.
func (b *Buffer) ByteOffset() int {
	return len(string(b.content[:b.gapStart]))
}

// SetByteOffset moves the gap to the specified byte offset.
func (b *Buffer) SetByteOffset(offset int) {
	if offset <= 0 {
		b.moveGap(0)
		return
	}

	text := b.Value()
	if offset >= len(text) {
		b.MoveEnd()
		return
	}

	// Find the rune index corresponding to this byte offset.
	runeIndex := 0
	byteCount := 0
	for _, r := range text {
		if byteCount >= offset {
			break
		}
		byteCount += utf8.RuneLen(r)
		runeIndex++
	}
	b.moveGap(runeIndex)
}

// DeleteRange removes the text between the start and end byte offsets.
func (b *Buffer) DeleteRange(start, end int) {
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}

	text := b.Value()
	if end > len(text) {
		end = len(text)
	}
	if start == end {
		return
	}

	startRune := len([]rune(text[:start]))
	endRune := len([]rune(text[:end]))

	b.moveGap(startRune)
	diff := endRune - startRune
	b.gapEnd += diff
	b.version++
}

// MoveWordRight moves the gap to the end of the next word.
func (b *Buffer) MoveWordRight() {
	if b.gapEnd >= len(b.content) {
		return
	}

	runes := b.content[b.gapEnd:]
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf

	// We need to scan forward for word boundaries.
	// Word boundaries can be far, so we might need more runes.
	for _, r := range runes {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], r)
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	state := -1
	runeShift := 0
	remaining := byteBuf
	for len(remaining) > 0 {
		var cluster []byte
		var boundaries int
		cluster, remaining, boundaries, state = uniseg.Step(remaining, state)
		runeShift += utf8.RuneCount(cluster)
		if boundaries&uniseg.MaskWord != 0 {
			break
		}
	}

	b.moveGap(b.gapStart + runeShift)
	*pBuf = byteBuf
}

// MoveWordLeft moves the gap to the start of the previous word.
func (b *Buffer) MoveWordLeft() {
	if b.gapStart <= 0 {
		return
	}

	runes := b.content[:b.gapStart]
	pBuf := getByteBuf()
	defer putByteBuf(pBuf)
	byteBuf := *pBuf
	for _, r := range runes {
		var tmp [4]byte
		n := utf8.EncodeRune(tmp[:], r)
		byteBuf = append(byteBuf, tmp[:n]...)
	}

	state := -1
	lastBoundaryRune := 0
	currentRunePos := 0
	remaining := byteBuf
	for len(remaining) > 0 {
		var cluster []byte
		var boundaries int
		cluster, remaining, boundaries, state = uniseg.Step(remaining, state)
		runeLen := utf8.RuneCount(cluster)
		currentRunePos += runeLen
		if boundaries&uniseg.MaskWord != 0 && currentRunePos < b.gapStart {
			lastBoundaryRune = currentRunePos
		}
	}

	b.moveGap(lastBoundaryRune)
	*pBuf = byteBuf
}

// DeleteWordPrevious removes the word before the gap.
func (b *Buffer) DeleteWordPrevious() {
	if b.gapStart <= 0 {
		return
	}
	oldGapStart := b.gapStart
	b.MoveWordLeft()

	diff := oldGapStart - b.gapStart
	b.gapEnd += diff
	b.version++
}

// DeleteWordNext removes the word after the gap.
func (b *Buffer) DeleteWordNext() {
	if b.gapEnd >= len(b.content) {
		return
	}

	oldGapStart := b.gapStart
	b.MoveWordRight()

	b.gapStart = oldGapStart
	b.version++
}

// SetOffset is an alias for SetByteOffset for compatibility.
func (b *Buffer) SetOffset(offset int) {
	b.SetByteOffset(offset)
}

// MoveToStart is an alias for MoveStart for compatibility.
func (b *Buffer) MoveToStart() {
	b.MoveStart()
}

// MoveToEnd is an alias for MoveEnd for compatibility.
func (b *Buffer) MoveToEnd() {
	b.MoveEnd()
}

// DeletePrevious is an alias for Backspace(1) for compatibility.
func (b *Buffer) DeletePrevious() {
	b.Backspace(1)
}

// DeleteNext is an alias for Delete(1) for compatibility.
func (b *Buffer) DeleteNext() {
	b.Delete(1)
}

// moveGap moves the gap to a new start position.
func (b *Buffer) moveGap(newStart int) {
	if newStart == b.gapStart {
		return
	}

	if newStart > b.gapStart {
		// Moving right: copy characters from right side of gap to left side
		charsToMove := newStart - b.gapStart
		copy(b.content[b.gapStart:], b.content[b.gapEnd:b.gapEnd+charsToMove])
		b.gapStart += charsToMove
		b.gapEnd += charsToMove
	} else {
		// Moving left: copy characters from left side of gap to right side
		charsToMove := b.gapStart - newStart
		copy(b.content[b.gapEnd-charsToMove:], b.content[newStart:b.gapStart])
		b.gapEnd -= charsToMove
		b.gapStart -= charsToMove
	}
}

// Chunks returns the two parts of the buffer content around the gap.
func (b *Buffer) Chunks() ([]rune, []rune) {
	return b.content[:b.gapStart], b.content[b.gapEnd:]
}

// Value returns the current string value of the buffer.
func (b *Buffer) Value() string {
	return string(b.content[:b.gapStart]) + string(b.content[b.gapEnd:])
}

func (b *Buffer) ensureSpace(n int) {
	if b.gapEnd-b.gapStart >= n {
		return
	}

	// Resize
	newCapacity := len(b.content) + n + 16
	newContent := make([]rune, newCapacity)

	// Copy left side
	copy(newContent, b.content[:b.gapStart])

	// Copy right side to the end of new content
	rightLen := len(b.content) - b.gapEnd
	newGapEnd := newCapacity - rightLen
	copy(newContent[newGapEnd:], b.content[b.gapEnd:])

	b.content = newContent
	b.gapEnd = newGapEnd
}
