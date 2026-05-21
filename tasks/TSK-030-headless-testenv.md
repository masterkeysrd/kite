# TSK-030: Implement Headless Test Environment (`devtools/testenv`)

## Overview
Create a high-level testing harness that wraps the existing `backend/mock` to provide developers with ergonomic tools for testing Kite components without a physical terminal.

## Requirements

1. **`testenv.Environment` Wrapper:**
   - Create the `devtools/testenv` package.
   - Implement `New(eng *engine.Engine)` which wraps the Kite engine using `backend/mock` or takes an already configured engine.
   - Provide a `.Teardown()` or `.Close()` method to gracefully stop the engine.

2. **DOM Query API:**
   - Expose methods to inspect the logical DOM:
     - `GetNodeByID(id string) dom.Element`
     - `QuerySelector(selector string) dom.Element` (can start with simple class/tag matching).
   - Ensure these queries can pierce the UI correctly.

3. **Event Simulation API:**
   - `SendKey(k event.Key)`
   - `Type(text string)`
   - `Click(x, y int)`

4. **Frame Rendering Hooks:**
   - Expose a method like `Flush()` or `RenderFrame()` that blocks until the Kite engine completes a frame, allowing assertions on the newly painted state.

## Testing & Verifications
- Write tests in `testenv_test.go` demonstrating a simple input field application.
- Verify that `Type("hello")` successfully updates the input's DOM state and schedules a new frame.

## Documentation
- Add a new section to `README.md` introducing `devtools/testenv` for headless testing.