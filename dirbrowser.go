package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// dirBrowser is an in-TUI filesystem navigator used to pick a directory.
// It is embedded in the model and driven by handleDirBrowserKey.
type dirBrowser struct {
	active  bool
	cwd     string   // current directory being listed
	entries []string // subdirectory names in cwd
	cursor  int
	scroll  int
}

// newDirBrowser initialises the browser rooted at startDir.
func newDirBrowser(startDir string) dirBrowser {
	db := dirBrowser{active: true, cwd: startDir}
	db.reload()
	return db
}

// reload reads subdirectories of db.cwd into db.entries.
func (db *dirBrowser) reload() {
	db.entries = nil
	db.cursor = 0
	db.scroll = 0

	entries, err := os.ReadDir(db.cwd)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			db.entries = append(db.entries, e.Name())
		}
	}
	sort.Strings(db.entries)
}

// selectedPath returns the full path of the highlighted entry.
func (db dirBrowser) selectedPath() string {
	if len(db.entries) == 0 {
		return db.cwd
	}
	return filepath.Join(db.cwd, db.entries[db.cursor])
}

// enter descends into the highlighted entry.
func (db *dirBrowser) enter() {
	if len(db.entries) == 0 {
		return
	}
	db.cwd = db.selectedPath()
	db.reload()
}

// goUp navigates to the parent directory.
func (db *dirBrowser) goUp() {
	parent := filepath.Dir(db.cwd)
	if parent == db.cwd {
		return // already at root
	}
	db.cwd = parent
	db.reload()
}

// moveUp moves the cursor up.
func (db *dirBrowser) moveUp() {
	if db.cursor > 0 {
		db.cursor--
	}
}

// moveDown moves the cursor down.
func (db *dirBrowser) moveDown() {
	if db.cursor < len(db.entries)-1 {
		db.cursor++
	}
}

// adjustScroll keeps cursor visible in a window of visibleLines.
func (db *dirBrowser) adjustScroll(visibleLines int) {
	if db.cursor < db.scroll {
		db.scroll = db.cursor
	}
	if db.cursor >= db.scroll+visibleLines {
		db.scroll = db.cursor - visibleLines + 1
	}
}

// ---------------------------------------------------------------------------
// splashDirBrowserModel — standalone tea.Model used from main.go when the
// user picks "Browse for a directory..." on the splash screen.
// ---------------------------------------------------------------------------

type splashDirBrowserModel struct {
	width     int
	height    int
	db        dirBrowser
	chosen    string
	cancelled bool
}

func newSplashDirBrowser(startDir string) splashDirBrowserModel {
	return splashDirBrowserModel{db: newDirBrowser(startDir)}
}

func (s splashDirBrowserModel) Init() tea.Cmd { return nil }

func (s splashDirBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			s.cancelled = true
			return s, tea.Quit
		case "up", "k":
			s.db.moveUp()
		case "down", "j":
			s.db.moveDown()
		case "left", "backspace":
			s.db.goUp()
		case "right":
			s.db.enter()
		case "enter":
			if len(s.db.entries) > 0 {
				s.chosen = s.db.selectedPath()
			} else {
				s.chosen = s.db.cwd
			}
			return s, tea.Quit
		}
	}
	return s, nil
}

func RenderDirBrowserModal(db dirBrowser, width, height int) string {
	blue := lipgloss.Color("#58a6ff")
	orange := lipgloss.Color("#e8912d")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#6e7681"))

	title := lipgloss.NewStyle().Foreground(blue).Bold(true).
		Render(" Browse — Select Directory")
	sep := muted.Render(strings.Repeat("─", 58))
	crumb := db.cwd
	if len(crumb) > 56 {
		crumb = "…" + crumb[len(crumb)-55:]
	}
	breadcrumb := lipgloss.NewStyle().Foreground(orange).Render(" " + crumb)

	const visibleLines = 16
	db.adjustScroll(visibleLines)

	var rows []string
	if len(db.entries) == 0 {
		rows = append(rows, muted.Render("  (no subdirectories)"))
	} else {
		end := db.scroll + visibleLines
		if end > len(db.entries) {
			end = len(db.entries)
		}
		for i, name := range db.entries[db.scroll:end] {
			absIdx := db.scroll + i
			label := name
			if len(label) > 50 {
				label = label[:50] + "…"
			}
			if absIdx == db.cursor {
				rows = append(rows,
					lipgloss.NewStyle().Foreground(orange).Render("> ")+
						lipgloss.NewStyle().Foreground(blue).Bold(true).Render(" "+label))
			} else {
				rows = append(rows,
					"   "+lipgloss.NewStyle().Foreground(lipgloss.Color("#c9d1d9")).Render(" "+label))
			}
		}
	}

	footer := muted.Render("Enter: confirm  ←/Bksp: parent  →: open  Esc: cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, breadcrumb, sep}, append(rows, sep, footer)...)...,
	)

	box := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D0D0D")).
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(inner)
	return box
}

func (s splashDirBrowserModel) View() string {
	return RenderDirBrowserModal(s.db, s.width, s.height)
}
