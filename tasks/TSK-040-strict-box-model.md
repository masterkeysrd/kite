# TSK-040: Audit and Enforce Strict Border-Box Sizing

## Description
Ensure that all layout algorithms strictly follow the `border-box` model as defined in ADR-017. Currently, some algorithms may be implicitly doing `content-box` math or inconsistent border subtractions.

## Requirements
1. **Audit `BlockAlgorithm`**: Ensure that when `Space.IsFixedInlineSize` is true, the `AvailableSize.Width` is strictly treated as the outer `border-box`. The width given to children MUST be `ParentWidth - BorderX - PaddingX`.
2. **Audit `FlexAlgorithm`**: Ensure that flex basis and fraction math deducts borders and padding before distributing space to flex items.
3. **Audit `TableAlgorithm`**: Ensure grid cells deduct their borders/padding before running the inner IFC or Block algorithms.
4. **Remove Margin Collapsing**: Ensure there is no residual margin collapsing logic in `BlockAlgorithm` (if any existed).

## Tests
- Add a new suite `layout/box_model_test.go` with a `TestStrictBorderBox` test.
- Mount an element with `Width: 10`, `Border: Single` (1 cell each side), and `Padding: 1` (each side).
- Verify the internal child fragment receives exactly `6` cells of available width `(10 - 2 (border) - 2 (padding))`.