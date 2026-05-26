package main

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) renderGitSetupModal() string {
	return m.renderGitFormModal(" Connect to GitHub", false)
}

func (m model) renderGitConfigModal() string {
	return m.renderGitFormModal(" GitHub Configuration", true)
}

func (m model) renderGitFormModal(title string, isEdit bool) string {
	titleStr := modalTitleStyle.Render(title)

	fields := []struct {
		label string
		input string
	}{
		{"Repository URL:", m.modalInput.View()},
		{"Username:", m.modalInput2.View()},
		{"Email:", m.modalInput3.View()},
	}

	var rows []string
	rows = append(rows, titleStr)
	rows = append(rows, "")
	for i, f := range fields {
		label := mutedStyle.Render(f.label)
		if i == m.modalStep {
			label = accentStyle.Render("▶ "+f.label)
		}
		rows = append(rows, label)
		rows = append(rows, inputStyle.Width(46).Render(f.input))
		rows = append(rows, "")
	}

	action := "Enter"
	if isEdit {
		action = "Enter to save & sync"
	} else {
		action = "Enter to connect & sync"
	}
	if m.modalStep < 2 {
		action = "Enter/Tab: next field"
	}
	rows = append(rows, helpStyle.Render(action+"  Shift+Tab: prev  Esc: cancel"))

	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderGitSuccessModal() string {
	title := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(" Sync Complete")
	msg := lipgloss.NewStyle().Foreground(colorFg).Render(m.modalError)
	help := helpStyle.Render("Enter / Esc: close")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", help),
	)
}

func (m model) renderGitSyncingModal() string {
	title := mutedStyle.Render(" Syncing to GitHub...")
	spinner := accentStyle.Render("Please wait")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", spinner),
	)
}
