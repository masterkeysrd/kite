// Package event defines event types, the Dispatcher, the Synthesizer, and
// KeyStroke helpers for kite.
//
// Dispatch follows the capture → target → bubble model. Listeners return a
// cancel Subscription instead of requiring explicit removal. The Synthesizer
// converts raw backend input into synthetic event (click, focus). Hit-test
// results are cached on MouseEvent. No IntentEvent — the v1 layering mistake
// is removed.
package event
