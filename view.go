package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.modal != modalNone {
		return m.renderWithModal()
	}

	return m.renderMain()
}

func (m model) renderMain() string {
	totalH := m.height - 3 // reserve for header + statusbar

	// Panel widths
	foldersW := 22
	filesW := 32
	previewW := m.width - foldersW - filesW - 6 // 6 for borders
	if previewW < 20 {
		previewW = 20
	}

	folders := m.renderFoldersPanel(foldersW, totalH)
	files := m.renderFilesPanel(filesW, totalH)
	preview := m.renderPreviewPanel(previewW, totalH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, folders, files, preview)

	header := m.renderHeader()
	statusbar := m.renderStatusBar()

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(m.width).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				header,
				body,
				statusbar,
			),
		)
}

func (m model) renderHeader() string {
	title := lipgloss.NewStyle().
		Background(colorBg).
		Foreground(colorAccent).
		Bold(true).
		Padding(0, 1).
		Render("clidocs")

	folderTab := headerInactiveStyle.Render("Folders")
	snippetsTab := headerInactiveStyle.Render("Snippets")
	previewTab := headerInactiveStyle.Render("Preview")

	switch m.activePanel {
	case panelFolders:
		folderTab = headerActiveStyle.Render("Folders")
	case panelFiles:
		snippetsTab = headerActiveStyle.Render("Snippets")
	case panelPreview:
		previewTab = headerActiveStyle.Render("Preview")
	}

	// Breadcrumb
	breadcrumb := ""
	if m.currentFolderName() != "" {
		breadcrumb = mutedStyle.Render(" / ") + blueStyle.Render(m.currentFolderName())
		if m.currentFileName() != "" {
			breadcrumb += mutedStyle.Render(" . ") + mutedStyle.Render(m.currentFileName())
		}
	}

	// Git indicator
	gitIndicator := ""
	if m.gitCfgLoaded {
		gitIndicator = lipgloss.NewStyle().Foreground(colorAccent).Render("  ") +
			mutedStyle.Render(m.gitCfg.Username)
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Center,
		title,
		folderTab,
		snippetsTab,
		previewTab,
		breadcrumb,
		lipgloss.NewStyle().Background(colorBg).Render("  "),
		gitIndicator,
	)

	return lipgloss.NewStyle().
		Background(colorBg).
		Width(m.width).
		Padding(0, 0).
		Render(tabs)
}

func (m model) renderFoldersPanel(w, h int) string {
	isActive := m.activePanel == panelFolders

	innerW := w - 2
	innerH := h - 2

	var sb strings.Builder

	if len(m.folders) == 0 {
		sb.WriteString(mutedStyle.Render("No folders yet"))
		sb.WriteString("\n")
		sb.WriteString(mutedStyle.Render("Press n to create"))
	} else {
		for i, name := range m.folders {
			line := truncate(name, innerW-3)
			if i == m.folderCursor {
				prefix := arrowStyle.Render("→ ")
				row := prefix + selectedItemStyle.Width(innerW - 2).Render(line)
				sb.WriteString(row)
			} else {
				row := "  " + normalItemStyle.Width(innerW - 2).Render(line)
				sb.WriteString(row)
			}
			sb.WriteString("\n")
			if i >= innerH-1 {
				break
			}
		}
	}

	content := sb.String()
	// Pad to fill height
	lines := strings.Count(content, "\n")
	for lines < innerH {
		content += "\n"
		lines++
	}

	style := panelStyle
	if isActive {
		style = activePanelStyle
	}

	return style.
		Width(w).
		Height(h).
		Render(content)
}

func (m model) renderFilesPanel(w, h int) string {
	isActive := m.activePanel == panelFiles

	innerW := w - 2
	innerH := h - 2

	var sb strings.Builder

	if len(m.folders) == 0 {
		sb.WriteString(mutedStyle.Render("Select a folder"))
	} else if len(m.files) == 0 {
		count := fmt.Sprintf("0 snippets")
		sb.WriteString(mutedStyle.Render(count) + "\n\n")
		sb.WriteString(mutedStyle.Render("Press n to create a file"))
	} else {
		count := fmt.Sprintf("%d snippet", len(m.files))
		if len(m.files) != 1 {
			count = fmt.Sprintf("%d snippets", len(m.files))
		}
		sb.WriteString(mutedStyle.Render(count) + "\n\n")

		for i, f := range m.files {
			icon, iconColor := getFileIcon(f.name)
			iconStr := lipgloss.NewStyle().Foreground(iconColor).Render(icon)
			rel := relativeTime(f.modTime)
			folderName := m.currentFolderName()

			maxNameW := innerW - 4
			displayName := truncate(f.name, maxNameW)
			meta := mutedStyle.Render(folderName) + mutedStyle.Render(" • ") + mutedStyle.Render(rel)

			if i == m.fileCursor {
				nameStr := accentStyle.Render(displayName)
				metaStr := accentStyle.Render(folderName) + mutedStyle.Render(" • ") + accentStyle.Render(rel)
				row := iconStr + " " + nameStr + "\n" + "  " + metaStr
				sb.WriteString(selectedItemStyle.Width(innerW).Render(row))
			} else {
				nameStr := normalItemStyle.Render(displayName)
				row := iconStr + " " + nameStr + "\n" + "  " + meta
				sb.WriteString(normalItemStyle.Width(innerW).Render(row))
			}
			sb.WriteString("\n\n")

			// break if we exceed height
			usedLines := 2 + (i+1)*3
			if usedLines >= innerH {
				break
			}
		}
	}

	content := sb.String()
	lines := strings.Count(content, "\n")
	for lines < innerH {
		content += "\n"
		lines++
	}

	style := panelStyle
	if isActive {
		style = activePanelStyle
	}

	return style.
		Width(w).
		Height(h).
		Render(content)
}

func (m model) renderPreviewPanel(w, h int) string {
	isActive := m.activePanel == panelPreview

	innerH := h - 2
	innerW := w - 4

	var content string
	if m.previewHighlight == "" {
		if len(m.files) == 0 {
			content = mutedStyle.Render("No file selected")
		} else {
			content = mutedStyle.Render("Empty file")
		}
	} else {
		lines := strings.Split(m.previewHighlight, "\n")
		start := m.previewScroll
		if start > len(lines)-1 {
			start = max(0, len(lines)-1)
		}
		end := start + innerH
		if end > len(lines) {
			end = len(lines)
		}
		visible := lines[start:end]
		content = strings.Join(visible, "\n")
	}

	_ = innerW

	style := panelStyle
	if isActive {
		style = activePanelStyle
	}

	return style.
		Width(w).
		Height(h).
		Render(content)
}

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

func (m model) renderStatusBar() string {
	help := "Tab: panel  ↑↓: nav  Enter: edit  n: new  g: sync GitHub  G: git config  q: quit"
	if m.statusMsg != "" {
		help = m.statusMsg
	}
	return lipgloss.NewStyle().
		Background(colorBg).
		Foreground(colorFgMuted).
		Width(m.width).
		Padding(0, 1).
		Render(help)
}

func (m model) renderWithModal() string {
	base := m.renderMain()

	var modal string
	switch m.modal {
	case modalNewFolder:
		modal = m.renderNewFolderModal()
	case modalNewFile:
		modal = m.renderNewFileModal()
	case modalError:
		modal = m.renderErrorModal()
	case modalGitSetup:
		modal = m.renderGitSetupModal()
	case modalGitConfig:
		modal = m.renderGitConfigModal()
	case modalGitSuccess:
		modal = m.renderGitSuccessModal()
	case modalGitSyncing:
		modal = m.renderGitSyncingModal()
	}

	return overlayModal(base, modal, m.width, m.height)
}

func (m model) renderNewFolderModal() string {
	title := modalTitleStyle.Render(" New Folder")
	inputRendered := inputStyle.Width(42).Render(m.modalInput.View())
	help := helpStyle.Render("Enter: confirm  Esc: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			inputRendered,
			help,
		),
	)
}

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

func (m model) renderErrorModal() string {
	title := errorStyle.Render("  Error")
	msg := lipgloss.NewStyle().Foreground(colorFg).Render(m.modalError)
	help := helpStyle.Render("Enter / Esc: close")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			msg,
			"",
			help,
		),
	)
}

func overlayModal(base, modal string, width, height int) string {
	modalLines := strings.Split(modal, "\n")
	mh := len(modalLines)
	mw := 0
	for _, l := range modalLines {
		if lw := lipgloss.Width(l); lw > mw {
			mw = lw
		}
	}

	top := (height - mh) / 2
	left := (width - mw) / 2
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}

	baseLines := strings.Split(base, "\n")
	for i, ml := range modalLines {
		row := top + i
		if row >= len(baseLines) {
			break
		}
		bl := baseLines[row]
		blRunes := []rune(stripAnsi(bl))

		var before, after string
		if left > len(blRunes) {
			before = bl + strings.Repeat(" ", left-len(blRunes))
			after = ""
		} else {
			before = string(blRunes[:left])
			endIdx := left + mw
			if endIdx > len(blRunes) {
				after = ""
			} else {
				after = string(blRunes[endIdx:])
			}
		}

		baseLines[row] = before + ml + after
	}

	return strings.Join(baseLines, "\n")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
