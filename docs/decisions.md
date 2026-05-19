# Design Decisions Summary

This document logs the consensus-based decisions made during Kite design sessions. Detailed reasoning for major architectural shifts can be found in the `docs/adrs/` directory.

| Date | Decision | Context / Motivation | Associated ADR / Task |
|------|----------|----------------------|-----------------------|
| 2026-05-19 | Established Design Workflow | Need a structured way to run consensus-based design sessions without writing code directly. | N/A |
| 2026-05-19 | Virtual Markers for List Layout | Needed a way to render list bullets/numbers without violating the unified render box rule or causing clipping in the terminal grid. | ADR 001 |
| 2026-05-19 | Logical DOM Components for Lists | Introduced `ul`, `ol`, and `li` components. Decided to use `PaddingLeft` for terminal indentation and explicitly map `ListStyleType` as an inheritable style property. | Task: TSK-002 |
| 2026-05-19 | Table Layout and Fault Tolerance | Designed a two-pass algorithm for tables supporting ColSpan/RowSpan. Resolved to handle malformed tables strictly within the layout algorithm (like IFC anonymous blocks) to keep the engine sync phase pure. | ADR 002 |
