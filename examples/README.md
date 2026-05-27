# 🚀 Kite Examples

This directory contains a collection of examples demonstrating various features and components of the Kite terminal UI framework.

## 🏃 Running Examples

To run an example, navigate to its directory and use `go run`:

```bash
cd examples/button
go run main.go
```

## 📋 Available Examples

### Core Components
- **[button](button/)**: Basic and styled buttons with click handlers.
- **[input](input/)**: Single-line text input fields.
- **[textarea](textarea/)**: Multi-line text areas with scrolling and selection.
- **[checkbox_radio](checkbox_radio/)**: Interactive checkboxes and radio buttons.
- **[select](select/)**: Dropdown-style selection menus.
- **[list](list/)**: Vertical list layouts.
- **[table](table/)**: Data tables with borders and alignment.

### Layouts
- **[flex](flex/)**: Demonstrates the Flexbox layout engine (rows, columns, alignment).
- **[grid](grid/)**: Demonstrates the CSS-style Grid layout engine.

### Advanced Features
- **[animation](animation/)**: Property interpolation and easing.
- **[grid_animation](grid_animation/)**: Animated grid layouts.
- **[overlay](overlay/)**: Using the Top Layer Overlay API for tooltips and floating elements.
- **[dialog](dialog/)**: Modal dialogs with focus management.
- **[clipboard](clipboard/)**: Integration with the system clipboard.
- **[app1](app1/)**: A comprehensive application example combining multiple components.

### ⚛️ Kitex (Reactive)
- **[kitex_demo](kitex_demo/)**: demonstrates the VDOM reconciler, functional components, and hooks (`UseState`).
- **[kitex_ref_demo](kitex_ref_demo/)**: demonstrates using `UseRef` and `CreateRef` within functional components.
- **[kitex_hooks](kitex_hooks/)**: demonstrates terminal-specific hooks (`UseFocus` and `UseKeyboard`).
- **[form_demo](form_demo/)**: demonstrates form elements (Inputs, Checkbox, RadioGroup, Select) and VDOM submission handler via `kitex.Form`.


---

For more information on how to build your own applications, check out the [Quickstart Guide](../docs/guides/QUICKSTART.md).
