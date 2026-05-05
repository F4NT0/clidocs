package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) >= 2 {
		// Directory argument provided — skip splash and go directly to app
		arg := os.Args[1]
		var snippetsDir string
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
		runApp(snippetsDir)
		return
	}

	// No argument — show splash screen to choose directory
	sp := newSplashModel()
	p := tea.NewProgram(sp, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	sm, ok := finalModel.(splashModel)
	if !ok || sm.quit {
		return
	}

	chosenDir := sm.chosenDir
	if sm.browsePick {
		// Run the in-TUI directory browser as a separate program phase.
		startDir := getSnippetsDir()
		// Try to use the parent of the default snippets dir as a good root.
		if parent := filepath.Dir(startDir); parent != startDir {
			startDir = parent
		}
		db := newSplashDirBrowser(startDir)
		p2 := tea.NewProgram(db, tea.WithAltScreen())
		result, err := p2.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		res, ok := result.(splashDirBrowserModel)
		if !ok || res.cancelled || res.chosen == "" {
			// cancelled — go back to splash
			main()
			return
		}
		chosenDir = res.chosen
	}
	runApp(chosenDir)
}

func runApp(snippetsDir string) {
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
