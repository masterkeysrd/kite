---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: event-system
  description: Context for dispatching and handling events within the TUI framework's capture, target, and bubble phases.
  displayName: Event System
---

# Event System Skill

The Kite `event` package mimics the browser-based DOM event propagation flow. You must implement user interaction handling using the following rules:

## 1. Propagation Phases
Events flow through three distinct phases:
1. **Capture Phase**: From the document root down to the target element's parent.
2. **Target Phase**: Executing listeners on the target element itself.
3. **Bubble Phase**: Propagating from the target element back up to the document root.

## 2. No IntentEvents
Do not use or introduce "IntentEvents" (e.g., `IntentClick`). This was a v1 layering mistake. Instead, use the `event.Synthesizer` to convert raw terminal inputs (from the backend) into native, semantic DOM events like `click` or `focus`.

## 3. Subscriptions & Cancellation
When attaching an event listener, the method returns a cancel `Subscription` function. You do not need to pass the explicit callback function to a removal method.
* **Example**: `cancel := dispatcher.AddListener(target, event.TypeClick, handler)`

## 4. Keystrokes & Hit-Testing
* Key-related events must leverage the `/key` package for exact code matching and modifiers.
* Mouse events cache their hit-test results. Always query the event's target rather than recalculating the terminal surface layout coordinates.
