# API Overview

This overview points to the primary packages and where to find their authoritative files. For a full, navigable map see `SOURCE_MAP.md`.

- `dom` — logical node tree and lifecycle. See `dom/doc.go`, `dom/node.go`, `dom/document.go`.
- `render` — render-object interfaces and dirty tracking. See `render/object.go`, `render/box.go`.
- `layout` — layout algorithms (block, flex, inline, list, table). See `layout/doc.go`, `layout/flex_builder.go`, `layout/table_builder.go`.
- `paint` — paint pipeline and framebuffer. See `paint/doc.go`, `paint/framebuffer.go`.
- `engine` — frame loop and task queues. See `engine/doc.go`, `engine/engine.go`.
- `element` — declarative components and builders. See `element/doc.go`, `element/element.go`.

When making API-level changes:

- Update `README.md` and `AGENT.md` as required.
- Add or update package `doc.go` describing the contract and invariants.
