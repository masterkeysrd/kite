package element

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
)

type cachedOffsetInfo struct {
	pos        geom.Point
	textNode   dom.Node
	textOffset int
}

// CaretController tracks cursor state and detects transitions across element boundaries.
type CaretController struct {
	Element dom.Element
	Offset  int

	// lastKnownCX/Y preserve column alignment during vertical movements
	lastKnownCX int
	lastKnownCY int

	// Cache visual coordinates and resolved DOM references to achieve zero-allocation lookup
	cachedVersion uint64
	cachedInfos   []cachedOffsetInfo
}

// Move moves the caret inside the element in the given direction.
// Returns (newOffset, boundaryReached).
func (c *CaretController) Move(dir dom.Direction, view dom.View) (int, bool) {
	if view == nil || c.Element == nil {
		return c.Offset, false
	}

	textLen := len(c.Element.TextContent())

	switch dir {
	case dom.DirectionLeft:
		if c.Offset <= 0 {
			return 0, true // Boundary reached: left edge
		}
		curPos, curOk := c.getParentRelativeCaretPosition(c.Offset, view)
		c.Offset--
		// Skip collapsed characters that share the same visual position
		for c.Offset > 0 {
			pos, ok := c.getParentRelativeCaretPosition(c.Offset, view)
			if ok && curOk && pos == curPos {
				c.Offset--
			} else {
				break
			}
		}
		// Update virtual column tracking
		if pos, ok := c.getParentRelativeCaretPosition(c.Offset, view); ok {
			c.lastKnownCX = pos.X
			c.lastKnownCY = pos.Y
		}
		return c.Offset, false

	case dom.DirectionRight:
		if c.Offset >= textLen {
			return textLen, true // Boundary reached: right edge
		}
		curPos, curOk := c.getParentRelativeCaretPosition(c.Offset, view)
		c.Offset++
		// Skip collapsed characters that share the same visual position
		for c.Offset < textLen {
			pos, ok := c.getParentRelativeCaretPosition(c.Offset, view)
			if ok && curOk && pos == curPos {
				c.Offset++
			} else {
				break
			}
		}
		if pos, ok := c.getParentRelativeCaretPosition(c.Offset, view); ok {
			c.lastKnownCX = pos.X
			c.lastKnownCY = pos.Y
		}
		return c.Offset, false

	case dom.DirectionUp:
		newOffset := view.MoveCursorVertically(c.Element, c.Offset, -1, c.lastKnownCX, c.lastKnownCY)
		if newOffset == c.Offset {
			return c.Offset, true // Boundary reached: top edge of text
		}
		c.Offset = newOffset
		return c.Offset, false

	case dom.DirectionDown:
		newOffset := view.MoveCursorVertically(c.Element, c.Offset, 1, c.lastKnownCX, c.lastKnownCY)
		if newOffset == c.Offset {
			return c.Offset, true // Boundary reached: bottom edge of text
		}
		c.Offset = newOffset
		return c.Offset, false
	}

	return c.Offset, false
}

// getParentRelativeCaretPosition retrieves the caret coordinates relative to the parent element's content-box.
func (c *CaretController) getParentRelativeCaretPosition(offset int, view dom.View) (geom.Point, bool) {
	if c == nil || c.Element == nil {
		return geom.Point{}, false
	}

	doc := c.Element.OwnerDocument()
	if doc == nil {
		return geom.Point{}, false
	}

	// Check if we can reuse the cached coordinates for the current layout version
	version := view.LayoutVersion()
	if c.cachedVersion == version && c.cachedVersion > 0 {
		if offset >= 0 && offset < len(c.cachedInfos) {
			return c.cachedInfos[offset].pos, true
		}
	}

	// Cache miss: pre-calculate visual coordinates and resolve text nodes for all byte offsets
	textLen := len(c.Element.TextContent())
	infos := make([]cachedOffsetInfo, textLen+1)

	for i := 0; i <= textLen; i++ {
		textNode, textOffset := doc.FindNodeAtByteOffset(c.Element, i)
		pos, ok := view.GetCaretPosition(c.Element, i)
		if ok {
			infos[i] = cachedOffsetInfo{
				pos:        pos,
				textNode:   textNode,
				textOffset: textOffset,
			}
		} else if i > 0 {
			infos[i] = cachedOffsetInfo{
				pos:        infos[i-1].pos,
				textNode:   textNode,
				textOffset: textOffset,
			}
		}
	}

	c.cachedVersion = version
	c.cachedInfos = infos

	if offset >= 0 && offset < len(c.cachedInfos) {
		return c.cachedInfos[offset].pos, true
	}
	return geom.Point{}, false
}

// --- elementBase integration ------------------------------------------------

// CursorNavigable enables character-level caret navigation and text selection on this element.
func (b *elementBase[Self]) CursorNavigable(v bool) *Self {
	b.cursorNavigable = v
	if v && b.caret == nil {
		b.caret = &CaretController{Element: b.Element}
	}
	return b.self
}

// IsCursorNavigable reports whether the element is cursor-navigable.
func (b *elementBase[Self]) IsCursorNavigable() bool {
	return b.cursorNavigable
}

// SetCursorNavigable sets whether the element is cursor-navigable.
func (b *elementBase[Self]) SetCursorNavigable(v bool) {
	b.CursorNavigable(v)
}

// MoveCaret satisfies the dom.SpatialCaret interface.
func (b *elementBase[Self]) MoveCaret(dir dom.Direction) (boundaryReached bool) {
	if !b.cursorNavigable || b.caret == nil {
		return true // Default: boundary reached immediately (jump focus)
	}
	view := b.Element.OwnerDocument().DefaultView()
	_, boundaryReached = b.caret.Move(dir, view)
	if !boundaryReached {
		b.updateSelection()
	}
	return boundaryReached
}

// ResetCaret satisfies the dom.SpatialCaret interface.
func (b *elementBase[Self]) ResetCaret(dir dom.Direction) {
	if !b.cursorNavigable || b.caret == nil {
		return
	}
	textLen := len(b.Element.TextContent())
	if dir == dom.DirectionDown || dir == dom.DirectionRight {
		b.caret.Offset = 0 // Entered from top/left -> cursor at start
	} else {
		b.caret.Offset = textLen // Entered from bottom/right -> cursor at end
	}
	b.updateSelection()
}

// updateSelection updates the document selection to match b.caret.Offset
func (b *elementBase[Self]) updateSelection() {
	doc := b.Element.OwnerDocument()
	if doc == nil || b.caret == nil {
		return
	}
	view := doc.DefaultView()
	if view == nil {
		return
	}

	// Ensure cache is populated
	if _, ok := b.caret.getParentRelativeCaretPosition(b.caret.Offset, view); !ok {
		return
	}

	info := b.caret.cachedInfos[b.caret.Offset]
	if info.textNode == nil {
		return
	}

	sel := doc.Selection()
	if sel.RangeCount() > 0 {
		r := sel.GetRangeAt(0)
		r.SetStart(info.textNode, info.textOffset)
		r.SetEnd(info.textNode, info.textOffset)
		return
	}
	r := doc.CreateRange()
	r.SetStart(info.textNode, info.textOffset)
	r.SetEnd(info.textNode, info.textOffset)
	sel.RemoveAllRanges()
	sel.AddRange(r)
}

// CursorState satisfies the cursor.Provider interface.
func (b *elementBase[Self]) CursorState() cursor.State {
	if !b.cursorNavigable || b.caret == nil {
		return cursor.State{}
	}
	view := b.Element.OwnerDocument().DefaultView()
	if view == nil {
		return cursor.State{}
	}
	pos, ok := b.caret.getParentRelativeCaretPosition(b.caret.Offset, view)
	if !ok {
		return cursor.State{}
	}
	return cursor.State{
		Visible: true,
		X:       pos.X,
		Y:       pos.Y,
	}
}
