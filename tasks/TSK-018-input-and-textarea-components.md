# TSK-018: Input and TextArea Components

## Feature Design & Requirements
Implement the logical and render components for text input using the "Replaced Element via Direct Casting" pattern (similar to `render.Text`).

1. **Logical Components (`element.Input`, `element.TextArea`):**
   - Implement `render.CustomObjectProvider` to return `render.NewInput(self)`.
   - Embed the `editor.Buffer` (TSK-017) to manage the string value and 1D cursor.
   - Maintain `scrollX` and `scrollY` state.
   - Attach `event.EventKeyDown` listeners in `OnConnected` to handle typing, 1D navigation, and emitting `event.EventChange`.
   - For 2D navigation (Up/Down in TextArea), query the `render.TextArea` to determine the target byte offset before updating the `editor.Buffer`.

2. **Render Components (`render.Input`, `render.TextArea`):**
   - Embed `render.Box`.
   - In `Layout()` and `Paint()`, directly cast `LogicalNode()` to the specific element type to read the string, cursor index, and scroll offsets.
   - Perform `text.Shape()` locally.
   - Calculate text measurements, handle line wrapping (for TextArea), and generate the opaque `Fragment`.
   - Implement `cursor.Provider`: Calculate the local physical `(X,Y)` coordinate of the cursor by summing shaped cluster widths (minus scroll offsets) and return the `cursor.State`.

## Tests Required
- Render tests verifying that `render.Input` correctly pulls string data and shapes it into fragments.
- Navigation tests ensuring that typing updates the cursor physical bounds accurately.
