package main

import (
	"github.com/charmbracelet/lipgloss"
)

func RenderGitSetupModal(modalInputView, modalInput2View, modalInput3View string, modalStep int) string {
	return renderGitFormModal(" Connect to GitHub", false, modalInputView, modalInput2View, modalInput3View, modalStep)
}

func RenderGitConfigModal(modalInputView, modalInput2View, modalInput3View string, modalStep int) string {
	return renderGitFormModal(" GitHub Configuration", true, modalInputView, modalInput2View, modalInput3View, modalStep)
}

func renderGitFormModal(title string, isEdit bool, modalInputView, modalInput2View, modalInput3View string, modalStep int) string {
	titleStr := modalTitleStyle.Render(title)

	fields := []struct {
		label string
		input string
	}{
		{"Repository URL:", modalInputView},
		{"Username:", modalInput2View},
		{"Email:", modalInput3View},
	}

	var rows []string
	rows = append(rows, titleStr)
	rows = append(rows, "")
	for i, f := range fields {
		label := mutedStyle.Render(f.label)
		if i == modalStep {
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
	if modalStep < 2 {
		action = "Enter/Tab: next field"
	}
	rows = append(rows, helpStyle.Render(action+"  Shift+Tab: prev  Esc: cancel"))

	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func RenderGitSuccessModal(modalError string) string {
	title := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(" Sync Complete")
	msg := lipgloss.NewStyle().Foreground(colorFg).Render(modalError)
	help := helpStyle.Render("Enter / Esc: close")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", msg, "", help),
	)
}

func RenderGitSyncingModal() string {
	title := mutedStyle.Render(" Syncing to GitHub...")
	spinner := accentStyle.Render("Please wait")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", spinner),
	)
}
