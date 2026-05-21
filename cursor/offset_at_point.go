package cursor

import (
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/text"
)

// ByteOffsetAtPoint translates a terminal-cell coordinate (targetX, targetY)
// relative to the IFC fragment tree's origin into a byte offset.
func ByteOffsetAtPoint(root *layout.Fragment, targetX, targetY int) int {
	if root == nil || len(root.Children) == 0 {
		return 0
	}

	runningBytes := 0
	var lastLineOffset layout.Point

	for _, lineLink := range root.Children {
		lineBox := lineLink.Fragment
		lineBytes := countLineBytes(lineBox)

		if lineLink.Offset.Y == targetY {
			return runningBytes + resolveXOffset(lineBox, targetX-lineLink.Offset.X)
		}

		// If targetY is between lines (shouldn't happen in terminal grid but for completeness),
		// or if we passed targetY.
		if lineLink.Offset.Y > targetY {
			// If we are before the first line, return 0.
			if runningBytes == 0 {
				return 0
			}
			// Otherwise return end of previous line.
			return runningBytes
		}

		runningBytes += lineBytes
		lastLineOffset = lineLink.Offset
	}

	// targetY is past the last line.
	if targetY > lastLineOffset.Y {
		return runningBytes
	}

	// targetY matched the last line but targetX might be past the end.
	// This is already handled by the lineLink.Offset.Y == targetY check above
	// because resolveXOffset handles trailing X.

	return runningBytes
}

func resolveXOffset(lineBox *layout.Fragment, targetX int) int {
	if lineBox == nil {
		return 0
	}
	bytesSeen := 0

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
