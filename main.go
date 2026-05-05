package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var snippetsDir string
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "." {
			var err error
			snippetsDir, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
				os.Exit(1)
			}
		} else {
			abs, err := filepath.Abs(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid path: %v\n", err)
				os.Exit(1)
			}
			snippetsDir = abs
		}
	} else {
		snippetsDir = getSnippetsDir()
	}

	if err := os.MkdirAll(snippetsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating snippets directory: %v\n", err)
		os.Exit(1)
	}

	m := newModel(snippetsDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
