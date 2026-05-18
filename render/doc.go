// Package render provides the core engine for maintaining the render tree.
// It bridges the logical DOM tree with the Layout and Style engines via
// render objects (Block, Inline, Flex, Text) that track dirty state and computed
// styles.
package render
