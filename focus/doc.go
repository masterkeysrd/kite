// Package focus implements focus management for kitex (kite v2).
//
// focus.Manager owns the current focus node, the Reason (programmatic,
// pointer, keyboard, restore), and a Scope stack. IsFocusVisible returns true
// only when the reason is keyboard, matching the web :focus-visible heuristic.
// Tab navigation is DOM-order within the active scope. The spatial sub-package
// provides directional (arrow-key) navigation.
package focus
