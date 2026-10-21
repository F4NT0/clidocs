package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderNewFileModal(modalInputView, modalInput2View string, modalStep int) string {
	title := modalTitleStyle.Render(" New File")

	label1 := normalItemStyle.Render("File name:")
	input1 := inputStyle.Width(42).Render(modalInputView)
	if modalStep == 0 {
		label1 = accentStyle.Render("▶ File name:")
	}

	label2 := normalItemStyle.Render("Extension:")
	input2 := inputStyle.Width(42).Render(modalInput2View)
	if modalStep == 1 {
		label2 = accentStyle.Render("▶ Extension:")
	}

	help := helpStyle.Render("Tab/Enter: next field  Esc: cancel")
	if modalStep == 1 {
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

func RenderRenameFileModal(currentFileName, modalInputView string) string {
	title := modalTitleStyle.Render(" Rename Snippet")
	current := mutedStyle.Render("Current: ") + lipgloss.NewStyle().Foreground(colorGreen).Render(currentFileName)
	inputRendered := inputStyle.Width(42).Render(modalInputView)
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

func RenderDeleteConfirmModal(currentFileName string) string {
	ext, extColor := getFileIcon(currentFileName)
	badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
	name := lipgloss.NewStyle().Foreground(colorFg).Render(currentFileName)
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

func RenderCopyFileModal(folderName string) string {
	title := modalTitleStyle.Render(" Import File")
	dest := lipgloss.NewStyle().Foreground(colorFgSelected).Render(folderName)
	info := lipgloss.NewStyle().Foreground(colorFg).Render("Opening file picker...")
	sub := mutedStyle.Render("Select one or more files to copy into " + dest)
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", info, sub),
	)
}

func RenderMoveFileModal(currentFileName string, destinations []moveDestEntry, moveCursor int) string {
	ext, extColor := getFileIcon(currentFileName)
	badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
	name := lipgloss.NewStyle().Foreground(colorFg).Render(currentFileName)
	title := modalTitleStyle.Render(" Move File")
	filesep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, "", badge+"  "+name, filesep, "")

	const maxVisible = 12
	scrollStart := 0
	totalItems := len(destinations) + 1 // +1 for Browse external
	if moveCursor >= scrollStart+maxVisible {
		scrollStart = moveCursor - maxVisible + 1
	}

	for i := scrollStart; i < totalItems && i < scrollStart+maxVisible; i++ {
		folderIcon := mutedStyle.Render(" ")
		if i == len(destinations) {
			// Browse external entry
			if i == moveCursor {
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
		if i == moveCursor {
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

func RenderDirInfoModal(snippetsDir string) string {
	title := modalTitleStyle.Render(" Snippets Directory")
	sep := mutedStyle.Render(strings.Repeat("─", 44))
	dirStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(snippetsDir)
	help := helpStyle.Render("Enter: open in Explorer   s: change directory   Esc: close")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", dirStr, sep, help,
		),
	)
}

// Helper types and functions
type moveDestination struct {
	label string
	depth int
}
