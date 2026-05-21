package cursor

import (
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/text"
)

// FromTextFragment translates a byte offset within an IFC fragment tree into a
// terminal-cell coordinate (x, y) suitable for positioning the hardware cursor.
//
// # Fragment tree structure
//
// root is the block-level fragment produced by the inline formatting context
// (IFC). Its direct children (root.Children) are line-box fragments — one per
// visual line. Each line-box fragment may contain one or more text-run child
// fragments (Fragment.Text != nil) and/or atomic-inline children (non-text
// fragments whose Size.Width contributes to the visual column offset but whose
// byte count is zero).
//
// # Algorithm
//
// Y-axis: the function iterates root.Children (line boxes) and accumulates the
// total byte length of each line by summing len(cluster.Bytes) for every text
// cluster in every text-fragment descendant. The first line whose cumulative
// byte total causes the running byte sum to reach or exceed byteOffset yields y.
//
// X-axis: within the matching line box, text-fragment children are walked in
// order. For each text.Cluster, CellWidth is accumulated until the cluster
// whose byte range encloses byteOffset is reached. The accumulated cell-width
// before that cluster is x.
//
// Atomic inlines: non-text child fragments contribute 0 bytes and their
// Size.Width cells to the visual X. This handles inline-block elements that
// appear before or between text runs.
//
// Trailing offset: when byteOffset equals the total byte length of the
// fragment tree, the function returns the cell position immediately after the
// last cluster on the last line (x past the last glyph, y at the last line).
//
// # Return values
//
// Returns (x, y, true) on success. Returns (0, 0, false) if root is nil, the
// tree contains no children, or byteOffset exceeds the total byte length of the
// fragment tree.
//
// # TSK-023
//
// This is the canonical cursor-positioning helper introduced in TSK-023. It
// replaces bespoke per-widget cursor math that was duplicated across render
// objects such as render.Input and render.TextArea.
func FromTextFragment(root *layout.Fragment, byteOffset int) (x, y int, ok bool) {
	if root == nil || len(root.Children) == 0 {
		return 0, 0, false
	}

	// byteOffset must be non-negative.
	if byteOffset < 0 {
		return 0, 0, false
	}

	runningBytes := 0 // total bytes seen across all lines so far

	for lineIdx, lineLink := range root.Children {
		lineBox := lineLink.Fragment
		lineBytes := countLineBytes(lineBox)

		// Does this line box enclose byteOffset?
		//
		// The condition is satisfied when byteOffset falls within the byte
		// range [runningBytes, runningBytes+lineBytes), OR when byteOffset
		// equals runningBytes+lineBytes and we are on the LAST line (trailing
		// offset case).
		lineStart := runningBytes
		lineEnd := runningBytes + lineBytes
		isLastLine := lineIdx == len(root.Children)-1

		if byteOffset < lineEnd || (isLastLine && byteOffset == lineEnd) {
			// Found the matching line. Now resolve x within this line.
			cx := resolveX(lineBox, byteOffset-lineStart)
			return lineLink.Offset.X + cx, lineLink.Offset.Y, true
		}

		runningBytes = lineEnd
	}

	// byteOffset exceeds the total byte count.
	return 0, 0, false
}

// countLineBytes returns the total number of UTF-8 bytes contained in all text
// clusters of a line-box fragment. Atomic-inline children contribute 0 bytes.
func countLineBytes(lineBox *layout.Fragment) int {
	if lineBox == nil {
		return 0
	}
	total := 0
	for _, childLink := range lineBox.Children {
		child := childLink.Fragment
		for _, c := range child.Text {
			total += len(c.Bytes)
		}
	}
	return total
}

// resolveX computes the terminal-cell column for a byte offset that is relative
// to the start of lineBox.
//
// The function walks child fragments of lineBox in order:
//   - Text fragments: accumulate cell widths cluster-by-cluster until the
//     target byte is reached.
//   - Atomic-inline fragments (no Text): their full Size.Width is added to the
//     cell cursor since they occupy visual space but contain no bytes.
func resolveX(lineBox *layout.Fragment, relOffset int) int {
	if lineBox == nil {
		return 0
	}
	bytesSeen := 0

	for _, childLink := range lineBox.Children {
		child := childLink.Fragment

		if len(child.Text) > 0 {
			// Text fragment: walk cluster by cluster.
			xInChild := 0
			for _, c := range child.Text {
				if bytesSeen >= relOffset {
					return childLink.Offset.X + xInChild
				}
				bytesSeen += len(c.Bytes)
				xInChild += clusterWidth(c)
			}

			// Check if target is exactly at the end of this text fragment.
			if bytesSeen >= relOffset {
				return childLink.Offset.X + xInChild
			}
		} else {
			// Atomic inline: contributes 0 bytes. We continue to see if a
			// subsequent text fragment or the trailing case handles this offset.
			continue
		}
	}

	// relOffset is past the end of all text fragments, or the line only
	// contains atomic inlines — return trailing position of the last child.
	if len(lineBox.Children) > 0 {
		last := lineBox.Children[len(lineBox.Children)-1]
		return last.Offset.X + last.Fragment.Size.Width
	}

	return 0
}

// clusterWidth returns the display cell width for a single text cluster,
// clamped to a minimum of 0.
func clusterWidth(c text.Cluster) int {
	if c.CellWidth < 0 {
		return 0
	}
	return c.CellWidth
}
