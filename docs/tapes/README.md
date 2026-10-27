# VHS Demo Tapes

This directory contains [VHS](https://github.com/charmbracelet/vhs) `.tape` scripts used to generate demo GIFs/MP4s for Kite's documentation.

Generated `.gif`, `.mp4`, and `.webm` files are ignored by Git (`.gitignore`) to avoid bloating repository history. Instead, upload generated recordings to GitHub `user-attachments` (by dragging and dropping them into the GitHub web editor for `README.md` or an Issue/PR comment) and embed the resulting CDN URL.

## Generating Recordings

Install VHS (if not already installed):

```bash
brew install vhs
```

Run from the repository root:

```bash
vhs docs/tapes/trail_demo.tape
vhs docs/tapes/stage.tape
vhs docs/tapes/kitex_demo.tape
vhs docs/tapes/grid_animation.tape
```

Tip: Change `Output docs/tapes/<name>.gif` to `.mp4` in any `.tape` file if you want to upload native 60FPS MP4 video to GitHub `user-attachments`.
