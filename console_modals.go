package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderConsoleModal() string {
	green := lipgloss.Color("#228B22")
	title := lipgloss.NewStyle().Foreground(green).Bold(true).Render("Cmdline")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	prompt := lipgloss.NewStyle().Foreground(green).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(m.consoleInput) +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1).
		Width(48).
		Render(prompt)
	rows := []string{title, sep, inputLine}
	if m.consoleOutput != "" {
		rows = append(rows, "", sep)
		for _, l := range strings.Split(m.consoleOutput, "\n") {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorOrange).Render(l))
		}
	}
	rows = append(rows, "", sep,
		mutedStyle.Render("time · whoami · nvim · help · exit  |  Esc: close"))
	style := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green).
		Padding(0, 1)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderTimeCalcModal() string {
	title := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(" Work Hours Calculator")
	sep := mutedStyle.Render(strings.Repeat("─", 52))
	var rows []string
	rows = append(rows, title, sep)
	if m.timeResult != "" {
		rows = append(rows, "")
		for _, l := range strings.Split(m.timeResult, "\n") {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorFg).Render(l))
		}
		rows = append(rows, "", sep,
			mutedStyle.Render("Enter: back to console  Esc: back"))
	} else {
		rows = append(rows,
			mutedStyle.Render("Enter your start time (HH:MM):"),
			"",
			" "+m.modalInput.View(),
			"",
			sep,
			mutedStyle.Render("Enter: calculate  Esc: back"))
	}
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderWhoamiModal() string {
	title := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(" whoami")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	var rows []string
	rows = append(rows, title, sep)
	for _, l := range strings.Split(whoamiText, "\n") {
		rows = append(rows, lipgloss.NewStyle().Foreground(colorFg).Render(l))
	}
	rows = append(rows, sep, mutedStyle.Render("Enter / Esc: back to console"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderHelpConsoleModal() string {
	white := lipgloss.Color("white")
	colStyle := lipgloss.NewStyle().
		Foreground(colorFg).
		Background(colorBg).
		Border(lipgloss.NormalBorder()).
		BorderForeground(white).
		Padding(0, 1).
		Width(34)

	leftCol := colStyle.Render(
		lipgloss.NewStyle().Foreground(colorFg).Render(helpLeft),
	)
	rightCol := colStyle.Render(
		lipgloss.NewStyle().Foreground(colorFg).Render(helpRight),
	)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)

	title := lipgloss.NewStyle().Foreground(white).Bold(true).Render(" Help — Keyboard Shortcuts & Commands")
	footer := mutedStyle.Render("Enter / Esc: back to console")

	outer := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(white).
		Padding(0, 1)
	return outer.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", columns, "", footer))
}

func (m model) renderNvimGuideModal() string {
	green := lipgloss.Color("#57a143")
	cyan := lipgloss.Color("#00e5ff")
	orange := colorOrange
	blue := colorAccentBlue

	h := func(s string) string { return lipgloss.NewStyle().Foreground(green).Bold(true).Render(s) }
	k := func(s string) string {
		return lipgloss.NewStyle().Foreground(orange).Background(lipgloss.Color("#1e2a1e")).Padding(0, 1).Render(s)
	}
	dim := func(s string) string { return mutedStyle.Render(s) }

	colW := 46

	left := lipgloss.NewStyle().
		Foreground(colorFg).Background(colorBg).
		Border(lipgloss.NormalBorder()).BorderForeground(blue).
		Padding(0, 1).Width(colW).Render(
		h(" Navigation") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			k("h j k l") + dim("  ← ↓ ↑ →  (or arrow keys)") + "\n" +
			k("gg") + dim("       go to top of file") + "\n" +
			k("G") + dim("        go to bottom of file") + "\n" +
			k("Ctrl+d") + dim("  scroll half page down") + "\n" +
			k("Ctrl+u") + dim("  scroll half page up") + "\n" +
			k("w") + dim(" / ") + k("b") + dim("    next / previous word") + "\n" +
			k("0") + dim(" / ") + k("$") + dim("    start / end of line") + "\n" +
			"\n" +
			h(" Editing") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			k("i") + dim("        insert before cursor") + "\n" +
			k("a") + dim("        insert after cursor") + "\n" +
			k("o") + dim("        new line below, insert") + "\n" +
			k("O") + dim("        new line above, insert") + "\n" +
			k("Esc") + dim("      return to Normal mode") + "\n" +
			k("u") + dim("        undo") + "\n" +
			k("Ctrl+r") + dim("  redo") + "\n" +
			k("dd") + dim("       delete (cut) line") + "\n" +
			k("yy") + dim("       yank (copy) line") + "\n" +
			k("p") + dim("        paste after cursor") + "\n" +
			k("x") + dim("        delete character"),
	)

	right := lipgloss.NewStyle().
		Foreground(colorFg).Background(colorBg).
		Border(lipgloss.NormalBorder()).BorderForeground(blue).
		Padding(0, 1).Width(colW).Render(
		h(" Save & Quit") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			k(":w") + dim("       save file") + "\n" +
			k(":q") + dim("       quit (no unsaved changes)") + "\n" +
			k(":wq") + dim("  / ") + k(":x") + dim("  save and quit") + "\n" +
			k(":q!") + dim("      quit without saving") + "\n" +
			"\n" +
			h(" Search") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			k("/word") + dim("    search forward") + "\n" +
			k("n") + dim(" / ") + k("N") + dim("     next / previous match") + "\n" +
			"\n" +
			h(" Comment multiple lines") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			dim("1. ") + k("Ctrl+V") + dim(" → select lines (block mode)") + "\n" +
			dim("2. ") + k(":") + dim(" → opens command line") + "\n" +
			dim("3. type ") + k("'<,'>s/^/#") + dim("  → adds # at line start") + "\n" +
			"\n" +
			h(" Uncomment multiple lines") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			dim("1. ") + k("Ctrl+V") + dim(" → select lines (block mode)") + "\n" +
			dim("2. ") + k(":") + dim(" → opens command line") + "\n" +
			dim("3. type ") + k(`'<,'>s/^#//`) + dim("  → removes # from start") + "\n" +
			"\n" +
			h(" Visual mode") + "\n" + dim(strings.Repeat("─", colW-2)) + "\n" +
			k("v") + dim("  char select  ") + k("V") + dim("  line select") + "\n" +
			k("Ctrl+V") + dim("  block select"),
	)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	title := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render(" Neovim Quick Reference")
	footer := mutedStyle.Render("Enter / Esc: back to console")

	outer := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Padding(1, 2)
	return outer.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", columns, "", footer))
}
