# ADR-009 - UA Shadow Subtree for Replaced Elements

## Status
Accepted. Supersedes ADR-007.

## Context
Kite needs interactive widgets — `<input>`, `<textarea>`, and (in the near future) `<checkbox>`, `<radio>`, `<select>`, `<slider>`, `<progress>`. ADR-007 chose a "Replaced Element via Direct Casting" model: each widget brought a bespoke `render.Object` and a bespoke layout algorithm that cast the logical node back via ad-hoc anonymous interfaces. In practice this pattern revealed five flaws:

1. Every new widget invents its own anonymous casting interface, with no shared contract.
2. Each widget reimplements text shaping, line breaking, scroll-shifting, and clipping that the standard Inline Formatting Context (IFC) already provides.
3. UA-forced styles (`inline-block`, `overflow: clip`) live in two places — the DOM default style and the render object — and are not actually enforced against author overrides.
4. Compound widgets (checkbox glyph + label, select value + caret + popup, slider track + thumb) cannot be expressed cleanly because the host element has no children to compose with.
5. The pattern does not scale: every new form control proliferates a `*Element` + `*RenderObject` + `*Algorithm` + custom cast interface.

Browsers themselves do **not** use the full author-facing Shadow DOM for built-in form controls. They use a **closed user-agent shadow tree** — a private, internal DOM subtree the engine owns and the author cannot pierce. That model fits Kite's needs precisely and is far smaller in surface than full Shadow DOM (no slot distribution, no selector scoping, no open-mode encapsulation).

We considered and rejected:
- **Full author-facing Shadow DOM:** would require slot distribution, selector scoping, event retargeting infrastructure, and flattened-tree traversal rewrites. Cost is enormous; Kite has no CSS selectors to scope.
- **Per-widget custom render objects (ADR-007 status quo):** does not scale; see flaws above.
- **Mixing UA children into the public children list with an `IsUA` flag:** every public traversal would need filtering, leaking encapsulation through forgotten code paths.

## Decision
We adopt a **closed UA Shadow Subtree** model for replaced and compound elements.

### 1. UA Subtree Root on Host Elements
- The `dom.Element` interface gains an optional `uaRoot dom.Node` field, accessed internally by the engine.
- The `uaRoot` is a normal `dom.Node` (typically a `dom.Element` or `dom.TextNode`) that the host element constructs **eagerly in its constructor**. It is owned and mutated only by the host's controller code.
- Public traversal APIs (`Children()`, `GetElementByID()`, future `querySelector`) **never** expose `uaRoot` nor its descendants. The encapsulation is closed: author code has no documented path to reach UA-internal nodes.

### 2. Engine-Internal Visibility
- The engine's Sync, Style, Layout, and Paint phases **do** walk `uaRoot` as if it were a direct child of the host. From the layout engine's point of view, the host is a regular block/inline-block whose content is the UA subtree.
- The host element therefore needs **no custom render object and no custom layout algorithm** for the common case. It gets a plain `render.Box` and the standard formatting contexts do the rest.

### 3. Event Dispatch — UA Nodes Are Invisible
- UA-subtree nodes do **not** participate in the dispatch path. Capture, target, and bubble all behave as if `uaRoot` did not exist.
- The host element itself participates normally as the target.
- Identity retargeting reuses the existing `outer` back-pointer (ADR-0036 DOM adoption): when a UA subtree is attached, every node within it has its `outer` set to the host. Any code path that resolves `event.Target()` already honors `outer`, so semantic identity collapses onto the host without any new infrastructure.

### 4. Focus — UA Nodes Are Never Focusable
- UA-subtree nodes never satisfy `dom.Focusable`. Focus always lands on the host.
- Because `focus.Manager` walks the logical DOM via the public `Children()` iterator, UA nodes are naturally excluded from focus order. No focus-engine changes are required.

### 5. Composition Pattern
- `<input>` host owns a UA subtree containing a single internal text node bound to the element's `text.Buffer.Value()`. Standard IFC layout produces line boxes; the host queries those to position the hardware cursor.
- `<textarea>` host owns the same single internal text node, with intrinsic style `white-space: pre-wrap; overflow-wrap: break-word` (see ADR-010). The IFC produces one line box per visual line; 2D navigation is solved by querying the line-box tree, not by re-shaping text.
- Future compound widgets (`<checkbox>`, `<radio>`, `<select>`, `<slider>`, `<progress>`) compose their visuals declaratively inside the UA subtree using existing primitives (`element.Box`, `element.Text`, `element.Flex`).

### 6. Relationship to ADR-007
- ADR-007's "Replaced Element via Direct Casting" pattern is **deprecated** by this ADR. `render.CustomObjectProvider` (TSK-016) remains in the framework — it is useful for elements whose visuals genuinely cannot be expressed as a subtree — but form controls do not use it.

## Consequences
### Positive
- Form controls become **declarative compositions**: no per-widget algorithm, no per-widget render object, no ad-hoc casting interfaces.
- Standard IFC handles text shaping, wrapping, soft/mandatory breaks, and clipping uniformly — one well-tested code path serves all text widgets.
- Encapsulation is closed by construction: there is no `Children()` mode that would expose UA nodes, so authors cannot accidentally depend on internals.
- Scales to compound widgets (checkbox, radio, select, slider) without new architecture.
- Reuses the existing `outer` retargeting mechanism — no new event-system code.

### Negative / Trade-offs
- One new field on `dom.Element` (`uaRoot`) and a small additive contract on the host-element constructor.
- Engine-internal tree walks must take the union of `Children()` and `uaRoot` for Sync/Style/Layout/Paint. This is mechanical but must be applied consistently across the four phases.
- Hosts that need to react to their own UA subtree (e.g., recompute scroll on text change) must do so explicitly; there is no implicit author-side observation channel — which is by design.
