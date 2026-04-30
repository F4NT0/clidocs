package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	snippetsDir := getSnippetsDir()
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
