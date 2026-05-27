package cursor

import (
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/layout/text"
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
	if root == nil {
		return 0, 0, false
	}

	// byteOffset must be non-negative.
	if byteOffset < 0 {
		return 0, 0, false
	}

	if len(root.Children) == 0 && len(root.Text) > 0 {
		cx := resolveX(root, byteOffset)
		return cx, 0, true
	}

	if len(root.Children) == 0 {
		// Empty fragment: offset 0 is at (0,0) if the fragment has height
		// (e.g. an empty line box).
		if byteOffset == 0 && root.Size.Height > 0 {
			return 0, 0, true
		}
		return 0, 0, false
	}

	runningBytes := 0 // total bytes seen across all children so far

	for _, childLink := range root.Children {
		child := childLink.Fragment

		// Skip synthesized text fragments (like list markers) that do not
		// participate in the logical byte-offset model.
		if isSynthesizedAdornment(child) {
			continue
		}

		childBytes := countLineBytes(child)
		childStart := runningBytes
		childEnd := runningBytes + childBytes

		// Does this child enclose byteOffset?
		if byteOffset < childEnd {
			// Found the matching child.
			if len(child.Text) > 0 && len(child.Children) == 0 {
				// Leaf text fragment. Resolve x.
				cx := resolveX(child, byteOffset-childStart)
				return childLink.Offset.X + cx, childLink.Offset.Y, true
			}
			// Recursive call for containers or line boxes.
			cx, cy, ok := FromTextFragment(child, byteOffset-childStart)
			if ok {
				return childLink.Offset.X + cx, childLink.Offset.Y + cy, true
			}
		}

		runningBytes = childEnd
	}

	// Trailing offset case: if byteOffset is exactly at the end of the root,
	// return the end of the last child.
	if byteOffset == runningBytes && len(root.Children) > 0 {
		last := root.Children[len(root.Children)-1]
		if len(last.Fragment.Text) > 0 && len(last.Fragment.Children) == 0 {
			cx := resolveX(last.Fragment, countLineBytes(last.Fragment))
			return last.Offset.X + cx, last.Offset.Y, true
		}
		cx, cy, ok := FromTextFragment(last.Fragment, countLineBytes(last.Fragment))
		if ok {
			return last.Offset.X + cx, last.Offset.Y + cy, true
		}
	}

	// byteOffset exceeds the total byte count.
	return 0, 0, false
}

// countLineBytes returns the total number of UTF-8 bytes contained in all text
// clusters of a line-box fragment and its descendants. Atomic-inline children
// contribute 0 bytes. Synthesized fragments are skipped.
func countLineBytes(lineBox *layout.Fragment) int {
	if lineBox == nil {
		return 0
	}

	// Skip synthesized text fragments (like list markers) that do not
	// participate in the logical byte-offset model.
	if isSynthesizedAdornment(lineBox) {
		return 0
	}

	total := 0
	for _, c := range lineBox.Text {
		total += len(c.Bytes)
	}
	for _, childLink := range lineBox.Children {
		total += countLineBytes(childLink.Fragment)
	}
	return total
}

func isSynthesizedAdornment(f *layout.Fragment) bool {
	return f.Node == nil && f.ParentNode != nil && len(f.Text) > 0
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

	if len(lineBox.Text) > 0 {
		x := 0
		for _, c := range lineBox.Text {
			if bytesSeen >= relOffset {
				return x
			}
			bytesSeen += len(c.Bytes)
			x += clusterWidth(c)
		}
		if bytesSeen >= relOffset {
			return x
		}
	}

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
