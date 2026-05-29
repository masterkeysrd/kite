# TSK-029: Unified Text Control Base

## 1. Objective
Extract the duplicated scrolling, caret positioning, and core event handling logic from `InputElement` and `TextAreaElement` into a shared, embedded `textControlBase[T dom.Element]` struct. This will centralize terminal coordinate math, ensure scroll state perfectly tracks the cursor across both single and multi-line controls, and dramatically simplify the implementation of future text-based UI components.

## 2. Design & Requirements

### 2.1 Context
Currently, `element/input.go` contains both `InputElement` and `TextAreaElement`. Both elements manage their own internal `text.Buffer`, their own internal UA `uaDiv` block, and their own nearly identical copies of:
- `CursorState()` (calculating `insetLeft + cx` math)
- `ScrollCursorIntoView()` (calculating scroll containment boxes and updating `ScrollTo()`)
- `handleMouseDown()` (calculating hit-test offsets with `scrollX/Y` offsets)
- `handleKeyDown()` (wiring basic editor buffer movement)

This duplication is error-prone. A fix to scroll translation in one control must be manually ported to the other.

### 2.2 Feature Design

#### `textControlBase[T dom.Element]` Struct
Create a new generic struct in `element/text_control.go` to serve as the unified editor host.
```go
type textControlBase[T dom.Element] struct {
    elementBase[T] // inherits base DOM Element capabilities
    
    buf *text.Buffer
    uaDiv dom.Element // The internal block wrapper containing the text nodes
    doc dom.Document
    
    needsScrollIntoView bool
    isMultiline bool // true for TextArea, false for Input
    
    // syncCallback is provided by the concrete element to define how its
    // UA Subtree should be rebuilt/synced when the buffer changes.
    syncCallback func()
}
```

#### Shared Interface: `cursor.Provider`
Move `CursorState()` entirely into `textControlBase`.
- Use `base.uaDiv.RenderObject().Fragment()` to query the physical line-boxes.
- Add `base.host.ComputedStyle()` inset offsets to convert to the expected terminal border-box origin.

#### Shared Scroll Engine
Move `ScrollCursorIntoView()` into `textControlBase`.
- Query `cx, cy` relative to `uaDiv`.
- Fetch `base.host.Scroll()` offsets.
- Calculate `contentW` and `contentH` based on the host's padded fragment size.
- Clamp `scrollX` and `scrollY` independently so that the cursor `cx, cy` remains visible within the viewport.
- If `isMultiline == false` (e.g., `<input>`), `contentH` will simply equal 1, making Y-scrolling a natural no-op without needing specialized code.

#### Shared Event Bindings
Move `handleMouseDown` and `handleKeyDown` to `textControlBase`.
- Mouse clicks will identically map screen coordinates to buffer offsets by applying `scrollX` and `scrollY`.
- Keyboard bindings (Left, Right, Home, End, Backspace, Delete) will route to the buffer.
- Use `base.isMultiline` to guard operations like `Up/Down` and `Enter`. If false, ignore/prevent them or pass them back to the engine's focus/spatial navigation.

#### Concrete Simplification
Refactor `InputElement` and `TextAreaElement` to embed `textControlBase`.
- They will only define their `DefaultStyle` and `IntrinsicStyle`.
- They will construct their specific UA Subtree (e.g., `TextArea` building `<br>` elements).
- They will pass their specific `syncCallback` to the base to re-render when text changes.

### 2.3 Rules
- The base MUST NOT assume the structure of the text nodes inside `uaDiv`. It simply uses `uaDiv.RenderObject().Fragment()` to ask the layout engine for the resulting text fragment geometry.
- `textControlBase` must rely entirely on the TSK-028 `Scroll()` and `ScrollTo()` DOM methods for scroll state. It must not store `scrollX/Y` internally.
- `textControlBase` must continue to dispatch the `syncCallback` after any buffer mutation (e.g., after `handleKeyDown`).

## 3. Implementation Steps

1. Create `element/text_control.go`.
2. Define the generic `textControlBase[T dom.Element]` struct.
3. Migrate `CursorState()`, `ScrollCursorIntoView()`, `handleMouseDown()`, and `handleKeyDown()` from `element/input.go` into `textControlBase`.
4. Refactor `InputElement` (in `input.go`) to embed `textControlBase[InputElement]`. Remove its duplicate methods.
5. Refactor `TextAreaElement` (in `textarea.go`, extract it from `input.go` if needed) to embed `textControlBase[TextAreaElement]`. Remove its duplicate methods.
6. Ensure that both components correctly initialize the base with their specific `isMultiline` flag and `syncCallback`.
7. Verify all internal imports and interface assertions (`cursor.Provider`, `dom.Focusable`) are met by the base or concrete types as appropriate.

## 4. Testing Requirements

### 4.1 Unit Tests
- [ ] Existing `input_test.go` and `textarea_test.go` must pass without modification (regression guard).
- [ ] Add explicit unit tests verifying that `ScrollCursorIntoView` correctly updates Y-scroll state when `isMultiline == true`, and ignores Y-scroll when `isMultiline == false`.

### 4.2 Integration Tests
- [ ] Ensure that clicking inside a heavily scrolled `<textarea>` correctly maps the mouse click to the correct buffer byte offset using the generic `textControlBase` hit-testing logic.

### 4.3 Documentation
- [ ] Update `element/doc.go` to document the new `textControlBase` pattern for building custom text editors.
