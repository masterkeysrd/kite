# Examples

The `examples/` directory contains small demo applications and usage patterns. Each subdirectory focuses on a particular component or layout.

Common actions:

- List examples:

```bash
ls -1 examples
```

- Run an example with `main`:

```bash
go run ./examples/<example-name>
```

- Run example tests or demos:

```bash
go test ./examples/...
```

Inspect typical examples:

- `examples/app1` — app scaffold demonstrating mounting and basic widgets.
- `examples/flex` — flexbox layout usage.
- `examples/input` — input widget example.
- `examples/checkbox_radio` — Checkbox and RadioGroup demonstration.
- `examples/select` — Select (Dropdown) component demonstration.
- `examples/overlay` — basic anchored overlay demo.
- `examples/dialog` — modal dialog demonstration.
- `examples/overlay_tweaks` — interactive configuration of overlay smart-flipping.
- `examples/animation` — demonstration of property interpolation and tweening.
- `examples/kitex_demo` — functional VDOM components, reactive hooks, list reconciliation, and state.
- `examples/kitex_ref_demo` — VDOM hook refs, persistent non-rendering states, and DOM element ref wiring.
- `examples/form_demo` — comprehensive form submission showing inputs, select, checkbox, and radio controls within a `kitex.Form`.
- `examples/responsive` — responsive layout and terminal-specific hook actions (`UseViewportSize`, `UseTitle`, `UseBell`, `UseWindowFocus`, `UseProgressBar`).

If an example lacks a `main` package, it may be structured as tests or packages; read the `README.md` inside the example subfolder (when present) for details.
