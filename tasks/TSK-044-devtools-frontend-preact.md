# TSK-044: Migrate DevTools Frontend to Preact and Vite

## Context
Following ADR 020, the DevTools UI needs to be upgraded from a vanilla JS HTML file to a modern Preact application to support the upcoming Flamechart UI (TSK-043).

## Requirements
1. **Initialize Preact App:** Create a new folder at `devtools/inspector/ui`. Initialize a new Vite project using the Preact template (TypeScript is optional but recommended).
2. **Configure Vite for Single-File Build:** Install and configure a Vite plugin (like `vite-plugin-singlefile`) so that running `npm run build` outputs a *single* `index.html` file containing all CSS and JS inline.
3. **Port Existing UI:** Migrate the existing vanilla JS DOM Inspector code (currently handling the SSE connection and rendering the DOM tree) into Preact components (e.g., `<App>`, `<DOMViewer>`, `<PropertiesPanel>`).
4. **Go Embed Integration:** 
   - Configure the Vite build output directory to be `devtools/inspector/static/`.
   - Update `devtools/inspector/server.go` to use `//go:embed static/index.html` instead of the old vanilla HTML file.
5. **Update Build Scripts:** Ensure there is a clear script or instruction in the `ui` folder on how to rebuild the embedded asset.

## Constraints
- The final output MUST be a single HTML file. We do not want to serve multiple separate `.js` or `.css` files from the Go HTTP server.
- The compiled `index.html` MUST be committed to Git so that standard Go developers do not need Node.js installed to use Kite.

## Documentation Updates
- Add a `README.md` inside the `devtools/inspector/ui` directory explaining the Node.js requirement, how to run the dev server (`npm run dev`), and how to build the final embedded asset (`npm run build`).