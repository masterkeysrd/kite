// Package layout provides the core engine and algorithms for computing the layout of a DOM tree.
// It is responsible for calculating the size and position of each element in the tree based on its
// content and styling. The layout engine takes into account factors such as margins, padding,
// borders, and flexbox properties to determine how elements should be arranged on the page.
//
// The output of the layout engine is a set of rectangles that represent the position and size
// of each element, which can then be used by engine in the Layout phase of the rendering process
// to draw the elements on the screen. The layout package is a critical component of the rendering
// pipeline and is designed to be efficient and flexible to handle a wide range of layout scenarios.
// The package implements four primary formatting contexts:
//   - Block Formatting Context (BFC): Stacks elements vertically.
//   - Flex Formatting Context (FFC): Lays out elements in a flexible one-dimensional
//     arrangement (row or column) with support for growing, shrinking, and alignment.
//   - Inline Formatting Context (IFC): Lays out text and atomic inlines horizontally,
//     wrapping them into line boxes.
//   - List Formatting Context (LFC): Formats list items with virtual markers using a
//     two-column row layout.
//
// The layout process follows a LayoutNG-inspired immutable fragment tree model.
package layout
