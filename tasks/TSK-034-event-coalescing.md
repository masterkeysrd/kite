# TSK-034: Event Coalescing and Throttling in Engine

## Description
Modify the main `engine.Run` loop to buffer incoming raw events from the backend rather than processing them synchronously upon arrival. Coalesce high-frequency inputs immediately before dispatching them to the DOM in the frame ticker.

## Requirements
1. **Event Queueing**: Update `engine.Run` so that the `input := e.backend.Events()` case appends events to an internal slice/buffer, rather than directly calling `e.processRawEvent()`.
2. **Pre-Frame Draining**: Right before `e.Frame()` is invoked (inside the ticker case), drain the buffer and process the events.
3. **Coalescing Logic**:
   - **Mouse Moves**: Iterate over the buffered events. If multiple `event.MouseEvent` of type `MouseMove` are found, discard the older ones and only keep the *latest* coordinate.
   - **Wheel Events**: If multiple `event.WheelEvent` are targeting the same `EventTarget`, combine them. Sum their `DeltaX` and `DeltaY` fields into a single event per target.
4. **Dispatch**: Send the coalesced and non-coalesced events through `e.synthesizer.Process()` and the existing DOM dispatch rules as normal.
5. Ensure the frame budget/timing is respected. If the buffer is empty, do not run dispatch.

## Tests
- Write a unit test `TestEngine_EventCoalescing_MouseMove` simulating multiple mouse moves in quick succession, verifying only 1 event reaches the DOM element.
- Write a unit test `TestEngine_EventCoalescing_WheelEvents` simulating 5 wheel ticks, verifying the DOM target receives a single event with the aggregated deltas.

## Documentation
- Update `engine/README.md` to mention the input coalescing phase before layout.