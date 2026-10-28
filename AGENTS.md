# Agent notes

## Adding images to the README

1. Screenshots go in `images/` (`.png`, `.jpg` or `.jpeg`).
2. Round the corners with the script (output is a PNG with transparent corners):

   ```powershell
   .\scripts\round-images.ps1                  # every image in images/
   .\scripts\round-images.ps1 -Radius 24       # custom corner radius
   .\scripts\round-images.ps1 -Snippet         # also print centered HTML tags
   ```

3. In `README.md`, uncomment the matching `<img ...>` placeholder (for example `images/folders-panel.png`). Each image sits inside `<p align="center">`, so it is centered on GitHub. Do not round `images/logo.png` (the README logo).
4. Keep images small. Use these widths in the `<img>` tag: `550` for the start screen and installer/uninstaller/help screens, `700` for full three-panel screenshots, `400` for single-panel screenshots. The logo stays at `450`.

## Colored titles

GitHub Markdown cannot color text, so green section titles are PNG images: `.\scripts\make-title.ps1 -Text "Title" -Out images\name.png` (transparent background, `#00ff88`), centered in `<p align="center">` with `width="200"`.

## Verification

- Build: `go build -o NUL . ; go build -o NUL ./cmd/clidocs-installer`
- Vet: `go vet ./...`
