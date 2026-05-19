---
apiVersion: warp/v1alpha1
kind: Skill
metadata:
  name: styling-engine
  description: Guidelines for applying sparse styling using the Optional pattern and bridging to computed layout constraints.
  displayName: Styling Engine
---

# Styling Engine Skill

Kite utilizes a sparse styling model, heavily inspired by CSS and Flexbox. When assigning or modifying styles in the `style` package, follow these conventions:

## 1. The `Optional[T]` Paradigm
The `style.Style` struct uses an `Optional[T]` wrapper for every field. This allows users to compose styles without overwriting fields they did not explicitly set (distinguishing between a zero-value and an unset value).
* **Always use `style.Some(value)`** to assign a property.
* Example: `Display: style.Some(style.DisplayFlex)`

## 2. Style vs. Computed
* **`style.Style`**: Represents the developer-authored, sparse configuration.
* **`style.Computed`**: Represents the fully resolved styles. It does not use `Optional[T]`. 
* The engine resolves `Style` into `Computed` using `Style.Apply(base Computed)`, which is called by the style resolver after applying inheritance.
* **Invalidation:** Changing `style.Style` triggers `DirtyStyle`. The engine will calculate the new `style.Computed` and perform a diff against the old `Computed` to determine if `DirtyLayout` or `DirtyPaint` needs to be flagged. Never force `DirtyLayout` directly when just applying styles.

## 3. Flexbox Layout Engine
Kite layout relies heavily on Flexbox primitives. Use these standard fields:
* `Display`: Typically `style.DisplayFlex` or `style.DisplayNone`.
* `FlexDirection`: `style.FlexRow` or `style.FlexColumn`.
* `JustifyContent` and `AlignItems` for axis alignment.

## Code Snippet Example
```go
import "github.com/masterkeysrd/kite/style"

myStyle := style.Style{
    Display:       style.Some(style.DisplayFlex),
    FlexDirection: style.Some(style.FlexColumn),
    Width:         style.Some(style.DimensionPercent(100)),
    Background:    style.Some(color.RGBA{R: 255, A: 255}),
}
```
