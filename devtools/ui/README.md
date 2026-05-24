# Kite DevTools UI

This directory contains the Preact-based frontend for the Kite DOM Inspector.

## Development

1. Run `npm install` to install dependencies.
2. Run `npm run dev` to start the local development server.

## Building

Run `npm run build` to generate the final embedded HTML file in `devtools/ui/dist/index.html`. 
The Go `inspector` package embeds this file.
