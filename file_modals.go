package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderNewFileModal() string {
	title := modalTitleStyle.Render(" New File")

	label1 := normalItemStyle.Render("File name:")
	input1 := inputStyle.Width(42).Render(m.modalInput.View())
	if m.modalStep == 0 {
		label1 = accentStyle.Render("▶ File name:")
	}

	label2 := normalItemStyle.Render("Extension:")
	input2 := inputStyle.Width(42).Render(m.modalInput2.View())
	if m.modalStep == 1 {
		label2 = accentStyle.Render("▶ Extension:")
	}

	help := helpStyle.Render("Tab/Enter: next field  Esc: cancel")
	if m.modalStep == 1 {
		help = helpStyle.Render("Enter: create file  Tab: back  Esc: cancel")
	}

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			label1,
			input1,
			label2,
			input2,
			help,
		),
	)
}

func (m model) renderRenameFileModal() string {
	name := m.currentFileName()
	title := modalTitleStyle.Render(" Rename Snippet")
	current := mutedStyle.Render("Current: ") + lipgloss.NewStyle().Foreground(colorGreen).Render(name)
	inputRendered := inputStyle.Width(42).Render(m.modalInput.View())
	help := helpStyle.Render("Enter: confirm  Esc: cancel")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			current,
			"",
			inputRendered,
			help,
		),
	)
}

func (m model) renderDeleteConfirmModal() string {
	ext, extColor := getFileIcon(m.currentFileName())
	badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
	name := lipgloss.NewStyle().Foreground(colorFg).Render(m.currentFileName())
	fileStr := badge + "  " + name

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete File")
	sep := mutedStyle.Render(strings.Repeat("─", 44))
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Are you sure you want to delete:")
	help := helpStyle.Render("Enter / y: delete    Esc / n: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", warn, fileStr, sep, help,
		),
	)
}

func (m model) renderCopyFileModal() string {
	title := modalTitleStyle.Render(" Import File")
	folderName := m.currentFolderName()
	dest := lipgloss.NewStyle().Foreground(colorFgSelected).Render(folderName)
	info := lipgloss.NewStyle().Foreground(colorFg).Render("Opening file picker...")
	sub := mutedStyle.Render("Select one or more files to copy into " + dest)
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", info, sub),
	)
}

func (m model) renderMoveFileModal() string {
	destinations := m.moveDestinationsAll()

	ext, extColor := getFileIcon(m.currentFileName())
	badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
	name := lipgloss.NewStyle().Foreground(colorFg).Render(m.currentFileName())
	title := modalTitleStyle.Render(" Move File")
	filesep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, "", badge+"  "+name, filesep, "")

	const maxVisible = 12
	scrollStart := 0
	totalItems := len(destinations) + 1 // +1 for Browse external
	if m.moveCursor >= scrollStart+maxVisible {
		scrollStart = m.moveCursor - maxVisible + 1
	}

	for i := scrollStart; i < totalItems && i < scrollStart+maxVisible; i++ {
		folderIcon := mutedStyle.Render(" ")
		if i == len(destinations) {
			// Browse external entry
			if i == m.moveCursor {
				arrow := lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
				rows = append(rows, arrow+lipgloss.NewStyle().Foreground(colorAccentBlue).Render(" Browse external folder..."))
			} else {
				rows = append(rows, "   "+mutedStyle.Render(" Browse external folder..."))
			}
			continue
		}
		dest := destinations[i]
		indent := strings.Repeat("  ", dest.depth)
		label := truncate(dest.label, 44-len(indent))
		if i == m.moveCursor {
			arrow := lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
			nameStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(indent+label)
			rows = append(rows, arrow+folderIcon+nameStr)
		} else {
			nameStr := lipgloss.NewStyle().Foreground(colorFg).Render(indent+label)
			rows = append(rows, "   "+folderIcon+nameStr)
		}
	}

	if len(destinations) == 0 {
		rows = append(rows, mutedStyle.Render("No subfolders available."))
	}

	rows = append(rows, "", helpStyle.Render("↑↓: select folder   Enter: move   Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderMoveFileBrowseModal() string {
	return m.renderDirBrowserModal()
}

func (m model) renderDirInfoModal() string {
	title := modalTitleStyle.Render(" Snippets Directory")
	sep := mutedStyle.Render(strings.Repeat("─", 44))
	dirStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(m.snippetsDir)
	help := helpStyle.Render("Enter: open in Explorer   s: change directory   Esc: close")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", dirStr, sep, help,
		),
	)
}
