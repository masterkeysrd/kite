# TSK-018: UA Shadow Subtree Primitive

> **Note:** This task supersedes the previous TSK-018 ("Input and TextArea Components"). The "Replaced Element via Direct Casting" pattern from ADR-007 has been deprecated in favor of ADR-009 (UA Shadow Subtree) and ADR-010 (Intrinsic Style Layer).

## 1. Objective
Add a closed UA shadow subtree primitive to `dom.Element` so that replaced and compound widgets (`<input>`, `<textarea>`, future `<checkbox>`, `<radio>`, `<select>`, `<slider>`, `<progress>`) can compose their visuals as a private DOM subtree that author code cannot reach but the engine walks normally.

## 2. Design & Requirements

### 2.1 Feature Design
- Add an internal `uaRoot dom.Node` to the element implementation in `/dom`. It is `nil` by default. Hosts that need a shadow subtree set it eagerly in their constructor via a new method `Element.AttachUARoot(root dom.Node)`. The root is typically a `dom.Element` or `dom.TextNode` created against the same `Document` as the host.
- When `AttachUARoot` is called the engine sets the host's `outer` back-pointer on every node in the attached subtree (recursively), so any code that resolves `event.Target()` collapses to the host. Reuses the existing `outer` mechanism (ADR-0036) — no new event-system code.
- A node belongs to a UA subtree if any ancestor (walking parent pointers) holds it via `uaRoot`. Expose this internally as `dom.IsUANode(n) bool` for engine/focus/dispatch checks.

### 2.2 Visibility Rules
| API | Sees UA nodes? |
|---|---|
| Public `Element.Children()` iterator | **No** |
| Public `Element.FirstChild()` / `LastChild()` / `ChildNodes()` | **No** |
| `Document.GetElementByID()` | **No** |
| `focus.Manager` traversal | **No** (it uses public `Children()`) |
| Engine Sync / Style / Layout / Paint walks | **Yes** — engine uses a separate `dom.LayoutChildren()` (or equivalent) that unions public children with `uaRoot`'s subtree |
| Event dispatch (capture/target/bubble) | **No** — UA nodes never appear in the dispatch path; the host is the bubble entry point |

### 2.3 Rules
- **Strict package isolation:** the `uaRoot` field lives on the `dom.Element` implementation. The `style`, `layout`, `paint`, and `event` packages must not need to know that a node is "UA" beyond the unified `LayoutChildren()` iterator.
- **Eager construction:** hosts attach their UA root in the element constructor, before the host is connected to a document. The `Document` passed to the host's constructor is the document used for UA children.
- **Closed encapsulation:** there is no public accessor for `uaRoot`. The only way to mutate it is via the host's own internal controller code.
- **Adoption symmetry:** if the host is moved between documents, its UA subtree is adopted along with it. Detach of the host detaches the UA subtree symmetrically.
- **No focus traversal:** UA-subtree nodes must never satisfy `dom.Focusable`. The `focus.Manager` walks the public `Children()` iterator and therefore needs no changes.

### 2.4 Out of Scope
- Slot distribution / author-facing shadow roots — explicitly not implemented (we adopted the closed UA-only model).
- Per-cell event retargeting beyond reusing `outer`. UA nodes never appear in capture/bubble paths in the first place.
- Reusing this primitive for non-element hosts.

## 3. Implementation Steps
1. Add `uaRoot dom.Node` to the element implementation (`dom/element.go` or equivalent).
2. Implement `Element.AttachUARoot(root dom.Node)`:
   - Sets `uaRoot`.
   - Recursively sets the `outer` back-pointer on every node in `root`'s subtree to the host.
   - Marks the host as `NeedsSync` / `ChildNeedsSync` so the engine picks up the new subtree on the next Sync pass.
3. Add `dom.IsUANode(n dom.Node) bool` helper. Implementation walks parent pointers and checks whether any ancestor's `uaRoot` is an ancestor-or-equal of `n`. Optimize with a cached boolean flag if profiling demands.
4. Introduce (or reuse) an engine-side iterator `dom.LayoutChildren(n) iter.Seq[dom.Node]` that yields public children **followed by** the children of `uaRoot` if present. Document clearly that this is the engine-internal walker and must never be exposed to author code.
5. Audit the engine: Sync (`engine.syncRenderTree`), Style resolver traversal, layout `LayoutChildren()` calls, paint walker. Each must use the new internal walker so UA subtrees become part of the render tree.
6. Audit public-traversal APIs (`Children()`, `FirstChild()`, `LastChild()`, `ChildNodes()`, `GetElementByID()`) and confirm they iterate only the public child list, not `uaRoot`.
7. Audit event dispatch: confirm `event.dispatch` builds the capture/bubble path from the host's ancestors (it already does; document this guarantee here so future changes do not regress it).
8. Audit `focus.Manager`: confirm it uses public `Children()` (no change expected).
9. Document the primitive in `dom/doc.go` with a reference to ADR-009.

## 4. Testing Requirements
### 4.1 Unit Tests
- [ ] `AttachUARoot` sets the `outer` pointer on every descendant of the attached root, including grandchildren.
- [ ] After `AttachUARoot(root)`, `host.Children()` iterator does **not** yield any node from `root`.
- [ ] After `AttachUARoot(root)`, `dom.LayoutChildren(host)` yields all public children **then** all `root` children, in order.
- [ ] `Document.GetElementByID()` cannot find an element placed inside the UA subtree.
- [ ] `dom.IsUANode` returns `true` for UA descendants and `false` for public descendants.
- [ ] An event dispatched on a UA-subtree node (synthetically constructed for the test) has `event.Target() == host` and the bubble path consists of the host's public ancestors only.
- [ ] A focusable element placed inside a UA subtree is **not** discovered by `focus.Manager.NextFocusable()`.
- [ ] Adopting the host into another document carries the UA subtree with it.
- [ ] Detaching the host preserves the UA subtree (does not nil `uaRoot`).

### 4.2 Integration Tests
- [ ] A test host element with a UA subtree of `Box(Text("hi"))` produces a render tree where the inner text is laid out and painted by the standard IFC.
- [ ] Calling `host.RawStyle().Width(20)` and `host.RawStyle().Height(3)` is reflected in the rendered output; the UA subtree inherits inherited properties (e.g., `Color`) from the host through the standard cascade.

### 4.3 Regression Tests (at `./tests/regressions/`)
- [ ] Add `ua_subtree_test.go` with a host element whose UA subtree contains a button-like child. Verify that:
  - Public `querySelector`-style traversal does not find the inner child.
  - A keyboard event targeted at the host is dispatched on the host (not retargeted to UA child).
  - Focus navigation skips the UA subtree entirely.

### 4.4 Benchmarks
- [ ] `BenchmarkLayoutChildren_NoUA`: ensure the unioned iterator imposes < 5 % overhead vs the prior public iterator for nodes without a UA subtree (zero allocation path).
- [ ] `BenchmarkLayoutChildren_WithUA`: with a small UA subtree (1–3 nodes), ensure overhead is < 15 %.

### 4.5 Documentation
- [ ] Update `AGENT.md` to add a "UA Shadow Subtree" rule under Architectural Rules, referencing ADR-009. State: "Public traversal APIs must never expose UA-subtree nodes; engine phases use `dom.LayoutChildren()`."
- [ ] Update `README.md` if it lists notable DOM features.
- [ ] Update `dom/doc.go` with the UA subtree concept.
