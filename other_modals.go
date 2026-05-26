package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderErrorModal() string {
	title := errorStyle.Render("  Error")
	help := helpStyle.Render("Enter / Esc: close")

	// wrap long messages so they don't overflow the terminal width
	maxW := m.width - 12
	if maxW < 30 {
		maxW = 30
	}
	var msgLines []string
	for _, line := range strings.Split(m.modalError, "\n") {
		for len(line) > maxW {
			msgLines = append(msgLines, line[:maxW])
			line = line[maxW:]
		}
		msgLines = append(msgLines, line)
	}
	var rows []string
	rows = append(rows, title, "")
	for _, l := range msgLines {
		rows = append(rows, lipgloss.NewStyle().Foreground(colorFg).Render(l))
	}
	rows = append(rows, "", help)

	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderEditorReadyModal() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Neovim not found")
	sep := mutedStyle.Render(strings.Repeat("─", 52))

	warn := lipgloss.NewStyle().Foreground(colorFg).Render(
		"nvim was not found in your PATH.")

	hl := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true)
	mut := mutedStyle

	win1 := mut.Render("  Winget (recommended):")
	win2 := hl.Render("    winget install Neovim.Neovim")
	win3 := mut.Render("  Scoop:")
	win4 := hl.Render("    scoop install neovim")
	win5 := mut.Render("  Chocolatey:")
	win6 := hl.Render("    choco install neovim")
	win7 := mut.Render("  Manual: https://github.com/neovim/neovim/releases")

	alternate := mut.Render("Or open files with another editor:")
	vsCodeHint := hl.Render("    code <file>") + mut.Render("  (VS Code, if installed)")

	help := helpStyle.Render("Esc / Enter: close")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			warn, sep,
			mut.Render("Install Neovim for Windows:"), "",
			win1, win2, "",
			win3, win4, "",
			win5, win6, "",
			win7, sep,
			alternate, vsCodeHint, "",
			help,
		),
	)
}
