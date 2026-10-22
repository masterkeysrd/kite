// Package trail provides an Ink-like inline CLI framework for Kite.
//
// Unlike traditional fullscreen TUI applications that enter an alternate screen buffer,
// apps built with trail run inline within the user's terminal scrollback. They occupy
// only as many rows as needed for their content, dynamically grow and shrink, and
// allow completed output to be permanently inserted into the terminal scrollback
// via InsertAbove and the Static component.
//
// trail is built on Kite's engine and the kitex reactive VDOM framework.
package trail
