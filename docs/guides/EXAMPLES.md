# Examples

The `example/` directory contains small demo applications and usage patterns. Each subdirectory focuses on a particular component or layout.

Common actions:

- List examples:

```bash
ls -1 example
```

- Run an example with `main`:

```bash
go run ./example/<example-name>
```

- Run example tests or demos:

```bash
go test ./example/...
```

Inspect typical examples:

- `example/app1` — app scaffold demonstrating mounting and basic widgets.
- `example/flex` — flexbox layout usage.
- `example/input` — input widget example.
- `example/checkbox_radio` — Checkbox and RadioGroup demonstration.
- `example/select` — Select (Dropdown) component demonstration.
- `example/overlay` — basic anchored overlay demo.
- `example/dialog` — modal dialog demonstration.
- `example/overlay_tweaks` — interactive configuration of overlay smart-flipping.
- `example/animation` — demonstration of property interpolation and tweening.
- `example/kitex_demo` — functional VDOM components, reactive hooks, list reconciliation, and state.
- `example/kitex_ref_demo` — VDOM hook refs, persistent non-rendering states, and DOM element ref wiring.

If an example lacks a `main` package, it may be structured as tests or packages; read the `README.md` inside the example subfolder (when present) for details.
