package main

import (
	"os"
	"path/filepath"
)

func getSnippetsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "C:\\Users"
	}
	return filepath.Join(home, "clidocs_snippets")
}
