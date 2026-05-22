# Developing Kite

Project layout:

- Top-level packages: `dom`, `layout`, `paint`, `render`, `engine`, `element`, `style`, `text`, etc.
- See `SOURCE_MAP.md` for a concise map of concerns → files.

Common development workflow:

1. Run unit tests for the package you change:

```bash
go test -v -timeout 30s ./package/path
```

2. Run the full test suite before pushing changes:

```bash
gofmt -w .
go vet ./...
go test -timeout 30s ./...
```

3. Benchmarks: add `*_test.go` `BenchmarkX` functions and run:

```bash
go test -bench . ./layout -run ^$ -benchmem
```

4. Performance-sensitive changes to `layout`, `style`, or `paint` should include benchmarks.

5. **Visual Debugging**: Use the **Terminal X-Ray Mode** to debug layout issues in real-time. Call `devtools.Install(eng, devtools.Options{...})` and press the configured hotkey (default `Ctrl+D`) in your application to see the margin, padding, and content boxes of all elements.

Coding notes:

- Follow package `doc.go` design rules and ADR references.
- Keep `style` package dependency-free.
- Use `render` objects as the bridge between `dom` and layout/paint.
