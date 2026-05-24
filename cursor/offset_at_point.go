package cursor

import (
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

// ByteOffsetAtPoint translates a terminal-cell coordinate (targetX, targetY)
// relative to the IFC fragment tree's origin into a byte offset.
func ByteOffsetAtPoint(root *layout.Fragment, targetX, targetY int) int {
	if root == nil {
		return 0
	}

	if len(root.Children) == 0 && len(root.Text) > 0 {
		return resolveXOffset(root, targetX)
	}

	if len(root.Children) == 0 {
		return 0
	}

	runningBytes := 0

	scrollX, scrollY := 0, 0
	if root.Node != nil && root.Node.Style() != nil {
		s := root.Node.Style()
		if s.OverflowX != style.OverflowVisible || s.OverflowY != style.OverflowVisible {
			if ln := root.Node.LogicalNode(); ln != nil {
				if el, ok := ln.(interface{ Scroll() (x, y int) }); ok {
					rawX, rawY := el.Scroll()
					maxSX, maxSY := layout.MaxScroll(root)
					scrollX = max(0, min(rawX, maxSX))
					scrollY = max(0, min(rawY, maxSY))
				}
			}
		}
	}

	for i, lineLink := range root.Children {
		lineBox := lineLink.Fragment

		// Skip synthesized text fragments (like list markers) that do not
		// participate in the logical byte-offset model.
		if isSynthesizedAdornment(lineBox) {
			continue
		}

		childX := lineLink.Offset.X - scrollX
		childY := lineLink.Offset.Y - scrollY

		// Does this child contain targetY?
		if targetY >= childY && targetY < childY+lineBox.Size.Height {
			// This child is on the correct line.

			// If targetX is within this child, or if it's the LAST child on this line,
			// we recurse into it.
			isLastOnLine := true
			if i+1 < len(root.Children) {
				nextLink := root.Children[i+1]
				nextY := nextLink.Offset.Y - scrollY
				if nextY >= childY && nextY < childY+lineBox.Size.Height {
					isLastOnLine = false
				}
			}

			if targetX <= childX+lineBox.Size.Width || isLastOnLine {
				return runningBytes + ByteOffsetAtPoint(lineBox, targetX-childX, targetY-childY)
			}
			// TargetX is past this child and there are more siblings on the same line.
		} else if childY > targetY {
			// If we passed targetY.
			if runningBytes == 0 {
				return 0
			}
			// Otherwise return end of previous line.
			return runningBytes
		}

		runningBytes += countLineBytes(lineBox)
	}

	return runningBytes
}

func resolveXOffset(lineBox *layout.Fragment, targetX int) int {
	if lineBox == nil {
		return 0
	}
	bytesSeen := 0

	if len(lineBox.Text) > 0 {
		x := 0
		for _, c := range lineBox.Text {
			if c.BreakClass == text.BreakMandatory {
				return bytesSeen
			}
			if x >= targetX {
				return bytesSeen
			}
			cw := clusterWidth(c)
			if cw > 0 && x+cw > targetX {
				return bytesSeen
			}
			bytesSeen += len(c.Bytes)
			x += cw
		}
		return bytesSeen
	}

	// If it has children but no text, it's likely a container.
	// But resolveXOffset is specifically for lines (IFC).
	// If it's called on a container, ByteOffsetAtPoint should have recursed.

	for _, childLink := range lineBox.Children {
		child := childLink.Fragment
		childX := childLink.Offset.X // relative to lineBox
		xInChild := 0

		if len(child.Text) > 0 {
			for _, c := range child.Text {
				// Stop before the mandatory break character (e.g. \n) because its
				// logical position is on this line, but its visual position
				// is effectively "after" the line. Offset-wise, the byte
				// after \n belongs to the next line.
				if c.BreakClass == text.BreakMandatory {
					return bytesSeen
				}

				// If we are already at or past the target visual column, return
				// the bytes accumulated so far.
				if childX+xInChild >= targetX {
					return bytesSeen
				}

				cw := clusterWidth(c)
				// If adding this cluster would take us past targetX, stop here.
				if cw > 0 && childX+xInChild+cw > targetX {
					return bytesSeen
				}
				bytesSeen += len(c.Bytes)
				xInChild += cw
			}
		} else {
			// Atomic inline: check if we should stop before or after it.
			if childX+child.Size.Width > targetX {
				return bytesSeen
			}
		}
	}
	return bytesSeen
}
