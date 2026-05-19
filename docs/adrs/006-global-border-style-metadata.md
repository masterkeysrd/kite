# ADR 006: Global Border Style Metadata

## Status
Accepted

## Context
Kite uses a screen-space post-processing pass (`resolveBorders`) at the end of the paint pipeline to automatically merge adjacent structural borders into the correct Unicode junctions (e.g., `┼`, `├`). Currently, the pipeline tags border cells using a simple boolean bitmask (`FlagIsBorder`). When resolving junctions, the algorithm relies on "string sniffing" (comparing the literal string content of the center cell) to guess the border style (`single`, `double`, `thick`). 

This approach fails when borders of different styles intersect (e.g., a thick border crossing a single border). The fixed logic loses the context of what kind of border is arriving from each cardinal direction, making it impossible to apply correct precedence rules or render mixed junctions.

## Decision
We will replace the simple `FlagIsBorder` bitmask with a dedicated `BorderStyle` enum stored directly in the `paint.Cell` (or as a specific field in the `CellAttrs` bitmask). 

1. **Metadata Encoding:** The paint engine will encode the exact style of the border being drawn (`None`, `Single`, `Double`, `Thick`, `Rounded`, `Ascii`) into the framebuffer cell.
2. **Neighbor-Aware Resolution:** The `resolveBorders` pass will read this enum for all cardinal neighbors, preserving the style context from every direction.
3. **Precedence Rules:** When resolving an intersection of different border styles, we will use a "Heaviest Style Wins" precedence rule (e.g., `Thick > Double > Single > Rounded > Ascii`). The junction will be rendered using the dominant style of the participating edges.

## Consequences
### Positive
- **Accurate Precedence:** Ensures that a heavy border (like a modal dialog) cleanly overrides or intersects with lighter background borders without visually breaking.
- **Removes String Sniffing:** Decouples the engine's logical styling metadata from the literal string representations of Unicode characters, making the code much cleaner and less prone to edge-case bugs.
- **Future Extensibility:** Paves the way for mapping out complex mixed-style junctions (e.g., `╢`) if desired in the future, since the raw style data for all 4 directions is now available to the resolver.

### Negative / Trade-offs
- **Memory/Bitmask Usage:** Requires allocating bits in `CellAttrs` or adding a new `uint8` field to the `paint.Cell` struct, slightly increasing memory overhead for the framebuffer.
- **Slightly Slower Resolver:** The junction resolver logic becomes slightly more complex, as it must evaluate weights and dominant styles rather than indexing directly into a fixed 16-element array.
