package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	lsTreeDepth = 3
	lsTreeMax   = 500
)

// consoleTree returns the lines of a tree view of the directories inside dir.
func consoleTree(dir string) []string {
	var lines []string
	var walk func(d, prefix string, depth int)
	walk = func(d, prefix string, depth int) {
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		var dirs []string
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				dirs = append(dirs, e.Name())
			}
		}
		for i, name := range dirs {
			if len(lines) >= lsTreeMax {
				return
			}
			branch, next := "├── ", "│   "
			if i == len(dirs)-1 {
				branch, next = "└── ", "    "
			}
			lines = append(lines, prefix+branch+name)
			if depth < lsTreeDepth {
				walk(filepath.Join(d, name), prefix+next, depth+1)
			}
		}
	}
	walk(dir, "", 1)
	if len(lines) == 0 {
		lines = []string{"(no directories)"}
	}
	return lines
}

// lsViewRows is the number of tree rows visible in the ls panel.
func lsViewRows(termH int) int { return max(min(termH-14, 20), 4) }

// RenderLsModal draws the directory tree panel opened by the "ls" console command.
func RenderLsModal(dir string, tree []string, scroll int, input, msg string, termH int) string {
	blue := colorAccentBlue
	title := lipgloss.NewStyle().Foreground(blue).Bold(true).Render("Directory tree")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	prompt := lipgloss.NewStyle().Foreground(blue).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(input) +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(blue).Padding(0, 1).Width(48).Render(prompt)

	rows := lsViewRows(termH)
	end := min(scroll+rows, len(tree))
	orange := lipgloss.NewStyle().Foreground(colorOrange)
	lines := []string{title, sep, inputLine, "", sep, orange.Render(dir)}
	for _, l := range tree[scroll:end] {
		lines = append(lines, orange.Render(l))
	}
	for i := end - scroll; i < rows; i++ {
		lines = append(lines, "")
	}
	if len(tree) > rows {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("(%d-%d of %d)", scroll+1, end, len(tree))))
	}
	if msg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149")).Render(msg))
	}
	lines = append(lines, "", sep, mutedStyle.Render("cd <name>: open  ↑/↓: scroll  Esc: back"))
	style := lipgloss.NewStyle().Background(colorBg).Border(lipgloss.RoundedBorder()).BorderForeground(blue).Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// changeDir makes target the directory shown by clidocs.
func (m *model) changeDir(target string) {
	if m.origSnippetsDir == "" {
		m.origSnippetsDir = m.snippetsDir
	}
	m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
	m.snippetsDir = target
	m.inParentView = false
	m.parentViewDir = ""
	m.folderCursor, m.folderScroll = 0, 0
	m.fileCursor, m.fileScroll = 0, 0
	m.inFavSection = false
	m.loadFolders()
	m.loadFavorites()
	m.loadFiles()
	m.loadPreview()
}

// consoleResolveCd resolves the argument of "cd" relative to cur and checks it is a directory.
func consoleResolveCd(cur, arg string) (string, error) {
	arg = strings.Trim(strings.TrimSpace(arg), `"'`)
	if arg == "" {
		return "", fmt.Errorf("usage: cd <directory>")
	}
	target := arg
	if !filepath.IsAbs(target) {
		target = filepath.Join(cur, target)
	}
	target = filepath.Clean(target)
	if _, err := os.Stat(target); err != nil {
		// case-insensitive match of a direct child name
		entries, rerr := os.ReadDir(cur)
		if rerr != nil {
			return "", fmt.Errorf("cd: cannot find '%s'", arg)
		}
		want := strings.ToLower(strings.Trim(filepath.ToSlash(arg), "./"))
		found := ""
		for _, e := range entries {
			if e.IsDir() && strings.ToLower(e.Name()) == want {
				found = filepath.Join(cur, e.Name())
				break
			}
		}
		if found == "" {
			return "", fmt.Errorf("cd: cannot find '%s'", arg)
		}
		target = found
	}
	if fi, err := os.Stat(target); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("cd: '%s' is not a directory", arg)
	}
	return target, nil
}

// These constants are defined in the config package
const (
	whoamiText = `
Name    : Gabriel Stundner
Role    : Software Developer
GitHub  : github.com/F4NT0
Editor  : Neovim
Shell   : PowerShell / pwsh
OS      : Windows 11
Quote   : "Rise and Rise again, until lambs become lions!"
`
)

func RenderConsoleModal(consoleInput, consoleOutput string, width int) string {
	green := colorAccentBlue
	title := lipgloss.NewStyle().Foreground(green).Bold(true).Render("Cmdline")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	prompt := lipgloss.NewStyle().Foreground(green).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(consoleInput) +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1).
		Width(48).
		Render(prompt)
	rows := []string{title, sep, inputLine}
	if consoleOutput != "" {
		rows = append(rows, "", sep)
		for _, l := range strings.Split(consoleOutput, "\n") {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorOrange).Render(l))
		}
	}
	rows = append(rows, "", sep,
		mutedStyle.Render("cmd · ls · cd · time · whoami · nvim · exit  |  Esc: close"))
	style := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func RenderTimeCalcModal(timeResult string, modalInputView string) string {
	title := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(" Work Hours Calculator")
	sep := mutedStyle.Render(strings.Repeat("─", 52))
	var rows []string
	rows = append(rows, title, sep)
	if timeResult != "" {
		rows = append(rows, "")
		for _, l := range strings.Split(timeResult, "\n") {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorFg).Render(l))
		}
		rows = append(rows, "", sep,
			mutedStyle.Render("Enter: back to console  Esc: back"))
	} else {
		rows = append(rows,
			mutedStyle.Render("Enter your start time (HH:MM):"),
			"",
			" "+modalInputView,
			"",
			sep,
			mutedStyle.Render("Enter: calculate  Esc: back"))
	}
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func RenderWhoamiModal() string {
	green := colorAccentBlue
	title := lipgloss.NewStyle().Foreground(green).Bold(true).Render("Cmdline")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	prompt := lipgloss.NewStyle().Foreground(green).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render("whoami") +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1).
		Width(48).
		Render(prompt)
	rows := []string{title, sep, inputLine, "", sep}
	for _, l := range strings.Split(strings.TrimSpace(whoamiText), "\n") {
		rows = append(rows, lipgloss.NewStyle().Foreground(colorOrange).Render(l))
	}
	rows = append(rows, "", sep,
		mutedStyle.Render("Enter: open fantodocs.github.io  Esc: back to console"))
	style := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
