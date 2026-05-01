package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".ico":  true,
	".tiff": true,
	".tif":  true,
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return imageExts[ext]
}

// renderImagePreview returns a string to show in the preview panel for an image file.
// It tries chafa first (rich ASCII art), then falls back to image metadata.
func renderImagePreview(path string, width int) string {
	if out, ok := tryChafa(path, width); ok {
		return out
	}
	return imageMetadata(path)
}

// tryChafa runs chafa to render the image as colored Unicode art.
// Returns (output, true) if chafa is available and succeeds.
func tryChafa(path string, width int) (string, bool) {
	chafa, err := exec.LookPath("chafa")
	if err != nil {
		return "", false
	}
	cols := fmt.Sprintf("%d", max(20, width-6))
	cmd := exec.Command(chafa,
		"--colors=256",
		"--symbols=block+border+space",
		"--size", cols+"x30",
		path,
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", false
	}
	return buf.String(), true
}

// imageMetadata decodes image dimensions and returns a formatted info string.
func imageMetadata(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("Cannot open image: %v", err)
	}
	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)

	fi, _ := os.Stat(path)
	var size string
	if fi != nil {
		b := fi.Size()
		switch {
		case b >= 1024*1024:
			size = fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
		case b >= 1024:
			size = fmt.Sprintf("%.1f KB", float64(b)/1024)
		default:
			size = fmt.Sprintf("%d B", b)
		}
	}

	var sb strings.Builder
	sb.WriteString("\n")
	if err != nil {
		sb.WriteString(fmt.Sprintf("  Format : %s\n", strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))))
		sb.WriteString(fmt.Sprintf("  Size   : %s\n", size))
		sb.WriteString("\n  Could not decode image dimensions.\n")
	} else {
		sb.WriteString(fmt.Sprintf("  Format : %s\n", strings.ToUpper(format)))
		sb.WriteString(fmt.Sprintf("  Size   : %s\n", size))
		sb.WriteString(fmt.Sprintf("  Width  : %d px\n", cfg.Width))
		sb.WriteString(fmt.Sprintf("  Height : %d px\n", cfg.Height))
	}
	sb.WriteString("\n")
	sb.WriteString("  Install chafa for image preview:\n")
	sb.WriteString("  https://hpjansson.org/chafa/download/\n")
	return sb.String()
}
