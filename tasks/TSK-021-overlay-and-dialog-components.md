# TSK-021: Overlay and Dialog Components

## Feature Design & Requirements
Implement the high-level UI components that utilize the new Document Overlay API.

1. **`element.Overlay` (Anchored Positioning & Smart Flipping):**
   - Signature: `func Overlay(content dom.Node, config OverlayConfig) *OverlayElement`
   - `OverlayConfig` includes: `Anchor dom.Element`, `ZIndex int`, `Placement OverlayPlacement`, and `Flip bool`.
   - `OverlayPlacement` is an enum defining `PlacementTop`, `PlacementBottom`, `PlacementLeft`, `PlacementRight`.
   - It acts as a wrapper. Upon `OnConnected`, it calls `document.ShowOverlay(self, config.ZIndex)`.
   - **Custom Render Object (`render.Overlay`):** It must use the `CustomObjectProvider` hook (from TSK-016).
   - During `Layout()`, `render.Overlay` measures its content, calls `anchor.GetBoundingClientRect()`, and calculates physical `X,Y` offsets based on the placement.
   - **Smart Flipping:** If `Flip` is true, the layout algorithm must check if the calculated position exceeds the `ConstraintSpace.AvailableSize` (the viewport). If it overflows, it flips to the opposite placement (e.g., Bottom -> Top) and recalculates. If it overflows both, it defaults to the side with the most available space.
   - It outputs a `FragmentLink` whose physical `Offset` is forced to the calculated `X, Y`.

2. **`element.Dialog` (Modal Positioning):**
   - Signature: `func Dialog(content dom.Node, zIndex int) *DialogElement`
   - Uses `ShowOverlay`.
   - Applies styles: `Width: 100%`, `Height: 100%`, `Display: Flex`, `JustifyContent: Center`, `AlignItems: Center`.
   - In `OnConnected`, calls `focusManager.PushScope(&focus.Scope{Root: self})`.
   - In `OnDisconnected`, calls `focusManager.PopScope()`.

## Tests Required
- Rendering test proving `Overlay` fragments appear at exactly the `Anchor` bounds based on `Placement`.
- Collision testing verifying that `Flip` logic correctly swaps `Top` to `Bottom` when placed at the edge of the terminal constraints.
- Lifecycle test proving `Dialog` correctly pushes and pops the focus scope.
