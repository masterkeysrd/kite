# Kite Quickstart

Prerequisites:

- Go 1.26+ installed
- A terminal with UTF-8 and a compatible terminal emulator (for examples)

Build and run the project locally:

```bash
# Run unit tests (with timeout to avoid hangs)
go test -timeout 30s ./...

# Build all packages (no main binary by default)
go list ./... >/dev/null
```

Running examples:

The `examples/` directory contains sample apps. If an example contains a `main` package you can run it directly:

```bash
go run ./examples/app1
```

If an example is a test-driven demo, run:

```bash
go test ./examples/...
```

Reading the source:

- Start with `SOURCE_MAP.md` for a high-level map of packages and key files.
- Read package `doc.go` files (e.g. `dom/doc.go`, `layout/doc.go`) for design context.
