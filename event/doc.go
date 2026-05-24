// Package event defines event types, the Dispatcher, the Synthesizer, and
// KeyStroke helpers for kite.
//
// Dispatch follows the capture → target → bubble model. Listeners return a
// cancel Subscription instead of requiring explicit removal. The Synthesizer
// converts raw backend input into synthetic event (click, focus). High-frequency
// raw inputs (e.g. mouse move, wheel) are typically coalesced by the engine
// before being passed to the Synthesizer to maintain performance. Hit-test
// results are cached on MouseEvent. No IntentEvent — the v1 layering mistake
// is removed.
// # Wheel and Scroll
//
// WheelEvent is dispatched for mouse wheel input. It bubbles until it reaches
// an ancestor that implements Scrollable. The Synthesizer uses a
// ScrollableResolver to identify these targets (typically elements indicating
// scroll containerness via computed style).
//
// ScrollEvent is dispatched by the DOM when an element's scroll offset is
// mutated programmatically or via the default wheel handler. ScrollEvent bubbles.
//
// # Selection (ADR-022)
//
// SelectionChangeEvent is dispatched on the Document whenever the active text
// selection (dom.Selection) or its ranges (dom.Range) are modified.
package event
