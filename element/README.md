# element

Package `element` provides high-level logical DOM components for Kite. These components wrap base `dom.Element` and `dom.TextNode` implementations to provide default styles, a declarative API, and fluent builder-style modifiers.

## Declarative API

Kite elements use a functional, declarative API for tree construction. This allows you to build complex UI structures with minimal boilerplate.

### Functional Constructors

Each element provides a global functional constructor that accepts variadic `...any` children.

```go
ui := element.Box(
    element.Box("Header").Style(headerStyle),
    element.UL(
        element.LI("First Item"),
        element.LI("Second Item"),
    ),
)
```

### Automatic String Boxing

When a `string` is passed as a child to a functional constructor, it is automatically converted into a `TextElement`.

```go
// These are equivalent:
element.Box("Hello")
element.Box(element.Text("Hello"))
```

### Slice Flattening

Constructors automatically flatten slices of elements or `any` values, making it easy to generate children dynamically.

```go
items := []string{"A", "B", "C"}
list := element.UL(items) // Automatically creates 3 TextElements
```

## Fluent Modifiers

Elements support fluent modifier methods that return the element itself for chaining. These methods allow you to set styles, IDs, classes, and event listeners.

```go
btn := element.Box("Click Me").
    SetID("submit-btn").
    WithClass("primary").
    Style(style.Style{
        Background: style.Some(style.ColorBlue),
    }).
    OnEvent(event.TypeClick, func(ev event.Event) {
        fmt.Println("Clicked!")
    })
```

## Available Elements

| Function | Tag Name | Description |
|----------|----------|-------------|
| `Box` | `box` | A generic container (similar to `<div>`). |
| `Span` | `span` | An inline container (similar to `<span>`). |
| `Text` | `#text` | A leaf node containing text. |
| `Button` | `button` | A clickable button with centered content and interactive states. |
| `Input` | `input` | A single-line text input field. |
| `TextArea` | `textarea` | A multi-line scrollable text editor. |
| `Select` | `select` | A dropdown selection component with overlay. |
| `Option` | `option` | A data element for Select options. |
| `Checkbox` | `checkbox` | A toggleable checkbox with UA glyphs. |
| `RadioGroup`| `radiogroup`| A container that manages a set of Radio buttons. |
| `Radio` | `radio` | A single radio button within a group. |
| `UL` | `ul` | An unordered list with markers. |
| `OL` | `ol` | An ordered list with numbers. |
| `LI` | `li` | A list item. |
| `Table` | `table` | A table container. |
| `TR` | `tr` | A table row. |
| `TD` | `td` | A table cell. |
| `Overlay` | `overlay` | An anchored overlay with smart flipping. |
| `Dialog` | `dialog` | A full-screen modal container. |

## Implicit Adoption

Elements created via functional constructors are initially owned by a global "orphan" document. When you mount the root of your tree to the Kite `Engine` using `eng.Mount(root)`, the entire tree is recursively adopted by the engine's main document. This eliminates the need to pass a document reference through every component constructor.
