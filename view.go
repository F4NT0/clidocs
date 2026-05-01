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

	// Filter out .git and hidden git dirs
	visible := make([]string, 0, len(m.folders))
	for _, f := range m.folders {
		if f == ".git" || f == ".gitignore" {
			continue
		}
		visible = append(visible, f)
	}

	panelTitle := panelTitleStyle.Render(" Folders")
	var sb strings.Builder
	sb.WriteString(panelTitle + "\n")
	sb.WriteString(mutedStyle.Render(strings.Repeat("─", innerW)) + "\n")

	if len(visible) == 0 {
		sb.WriteString(mutedStyle.Render("No folders yet"))
		sb.WriteString("\n")
		sb.WriteString(mutedStyle.Render("Press n to create"))
	} else {
		// find effective cursor in visible list
		effIdx := 0
		rawName := ""
		if m.folderCursor < len(m.folders) {
			rawName = m.folders[m.folderCursor]
		}
		for vi, vn := range visible {
			if vn == rawName {
				effIdx = vi
				break
			}
		}

		folderIcon := mutedStyle.Render(" ")
		for i, name := range visible {
			line := truncate(name, innerW-5)
			if i == effIdx {
				prefix := arrowStyle.Render("> ")
				nameStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(line)
				sb.WriteString(prefix + folderIcon + nameStr)
			} else {
				nameStr := lipgloss.NewStyle().Foreground(colorFg).Render(line)
				sb.WriteString("   " + folderIcon + nameStr)
			}
			sb.WriteString("\n")
			if i >= innerH-3 {
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

func (m model) renderFilesPanel(w, h int) string {
	isActive := m.activePanel == panelFiles

	innerW := w - 2
	innerH := h - 2

	var sb strings.Builder

	// ── title + search bar ───────────────────────────────────────────────────
	if m.searchActive {
		cursor := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
		searchBar := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(" / ") +
			lipgloss.NewStyle().Foreground(colorFg).Render(m.searchQuery) +
			cursor
		sb.WriteString(searchBar + "\n")
	} else {
		sb.WriteString(panelTitleStyle.Render(" Snippets") + "\n")
	}
	sb.WriteString(mutedStyle.Render(strings.Repeat("─", innerW)) + "\n")

	// ── file list (filtered when searching) ──────────────────────────────────
	filtered := m.filteredFiles()
	folderName := m.currentFolderName()

	if len(m.folders) == 0 {
		sb.WriteString(mutedStyle.Render("Select a folder"))
	} else if len(filtered) == 0 && m.searchActive {
		sb.WriteString(mutedStyle.Render("No matches for: ") +
			lipgloss.NewStyle().Foreground(colorAccentBlue).Render(m.searchQuery))
	} else if len(m.files) == 0 {
		sb.WriteString(mutedStyle.Render("0 snippets") + "\n\n")
		sb.WriteString(mutedStyle.Render("Press n to create a file"))
	} else {
		countN := len(filtered)
		totalN := len(m.files)
		var countStr string
		if m.searchActive && countN < totalN {
			countStr = fmt.Sprintf("%d / %d snippets", countN, totalN)
		} else if totalN == 1 {
			countStr = "1 snippet"
		} else {
			countStr = fmt.Sprintf("%d snippets", totalN)
		}
		sb.WriteString(mutedStyle.Render(countStr) + "\n\n")

		// clamp fileCursor within filtered range
		cur := m.fileCursor
		if cur >= len(filtered) {
			cur = max(0, len(filtered)-1)
		}

		for i, f := range filtered {
			ext, extColor := getFileIcon(f.name)
			badge := lipgloss.NewStyle().
				Foreground(extColor).
				Width(5).
				Align(lipgloss.Right).
				Render(ext)
			rel := relativeTime(f.modTime)
			maxNameW := innerW - 8
			displayName := truncate(f.name, maxNameW)

			if i == cur {
				cursor := fileArrowStyle.Render("> ")
				nameStr := lipgloss.NewStyle().Foreground(colorGreen).Render(displayName)
				metaStr := mutedStyle.Render(folderName + " • " + rel)
				sb.WriteString(cursor + badge + " " + nameStr + "\n")
				sb.WriteString("       " + metaStr)
			} else {
				nameStr := lipgloss.NewStyle().Foreground(colorFg).Render(displayName)
				metaStr := mutedStyle.Render(folderName + " • " + rel)
				sb.WriteString("  " + badge + " " + nameStr + "\n")
				sb.WriteString("       " + metaStr)
			}
			sb.WriteString("\n\n")

			usedLines := 4 + (i+1)*3
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
	return style.Width(w).Height(h).Render(content)
}

func (m model) renderPreviewPanel(w, h int) string {
	isActive := m.activePanel == panelPreview

	// reserve: title(1) + sep(1) + optional search bar(1)
	headerLines := 2
	if m.previewSearchActive {
		headerLines = 3
	}
	availH := h - 2 - headerLines // h-2 for panel border

	fileName := m.currentFileName()
	var panelTitle string
	if fileName != "" {
		ext, extColor := getFileIcon(fileName)
		badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
		name := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Render(fileName)
		// line numbers indicator
		lnTag := ""
		if m.previewLineNumbers {
			lnTag = " " + mutedStyle.Render("[LN]")
		}
		panelTitle = " " + badge + "  " + name + lnTag
	} else {
		panelTitle = panelTitleStyle.Render(" Preview")
	}

	var contentLines []string
	contentLines = append(contentLines, panelTitle)
	contentLines = append(contentLines, mutedStyle.Render(strings.Repeat("─", w-4)))

	// search bar
	if m.previewSearchActive {
		prompt := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(" / ")
		qText := lipgloss.NewStyle().Foreground(colorFg).Render(m.previewSearchQuery)
		cur := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
		hitInfo := ""
		if len(m.previewSearchHits) > 0 {
			hitInfo = " " + mutedStyle.Render(fmt.Sprintf("%d/%d  n: next  N: prev",
				m.previewSearchCursor+1, len(m.previewSearchHits)))
		} else if m.previewSearchQuery != "" && len(m.previewSearchHits) == 0 {
			hitInfo = " " + lipgloss.NewStyle().Foreground(colorOrange).Render("Enter to search")
		}
		contentLines = append(contentLines, prompt+qText+cur+hitInfo)
	}

	// build hit-line set for fast lookup
	hitSet := make(map[int]bool, len(m.previewSearchHits))
	for _, idx := range m.previewSearchHits {
		hitSet[idx] = true
	}
	currentHitLine := -1
	if len(m.previewSearchHits) > 0 {
		currentHitLine = m.previewSearchHits[m.previewSearchCursor]
	}

	if m.previewIsImage {
		lines := strings.Split(m.previewHighlight, "\n")
		start := m.previewScroll
		if start > len(lines)-1 {
			start = max(0, len(lines)-1)
		}
		end := start + availH
		if end > len(lines) {
			end = len(lines)
		}
		contentLines = append(contentLines, lines[start:end]...)
	} else if m.previewHighlight == "" {
		if len(m.files) == 0 {
			contentLines = append(contentLines, mutedStyle.Render("No file selected"))
		} else if m.previewContent == "" && m.currentFileName() != "" {
			contentLines = append(contentLines, mutedStyle.Render("Binary or unreadable file — preview not available"))
		} else {
			contentLines = append(contentLines, mutedStyle.Render("Empty file — press Enter to edit"))
		}
	} else {
		hLines := strings.Split(m.previewHighlight, "\n")
		start := m.previewScroll
		if start > len(hLines)-1 {
			start = max(0, len(hLines)-1)
		}
		end := start + availH
		if end > len(hLines) {
			end = len(hLines)
		}

		// line number gutter width
		totalLines := len(strings.Split(m.previewContent, "\n"))
		gnW := len(fmt.Sprintf("%d", totalLines))

		for i, line := range hLines[start:end] {
			absLine := start + i // 0-based line index
			if m.previewLineNumbers {
				lineNum := fmt.Sprintf("%*d", gnW, absLine+1)
				var gnStyle lipgloss.Style
				if absLine == currentHitLine {
					gnStyle = lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
				} else if hitSet[absLine] {
					gnStyle = lipgloss.NewStyle().Foreground(colorGreen)
				} else {
					gnStyle = mutedStyle
				}
				contentLines = append(contentLines, gnStyle.Render(lineNum)+ mutedStyle.Render(" │ ")+line)
			} else if absLine == currentHitLine {
				// highlight current hit line even without line numbers
				contentLines = append(contentLines,
					lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("▶ ")+line)
			} else if hitSet[absLine] {
				contentLines = append(contentLines,
					lipgloss.NewStyle().Foreground(colorGreen).Render("• ")+line)
			} else {
				contentLines = append(contentLines, line)
			}
		}
	}

	content := strings.Join(contentLines, "\n")

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

func (m model) renderEditorReadyModal() string {
	filename := ""
	if m.editorPath != "" {
		parts := strings.Split(strings.ReplaceAll(m.editorPath, "\\", "/"), "/")
		filename = parts[len(parts)-1]
	}
	icon, iconColor := getFileIcon(filename)
	iconStr := lipgloss.NewStyle().Foreground(iconColor).Render(icon)

	title := modalTitleStyle.Render(" Open in Neovim")
	fileStr := lipgloss.NewStyle().Foreground(colorFg).Render(iconStr + " " + filename)
	sep := mutedStyle.Render(strings.Repeat("─", 46))

	info := lipgloss.NewStyle().Foreground(colorFg).Render(
		" Opens Neovim in a new Windows Terminal window.",
	)
	step1 := mutedStyle.Render("1. Edit your file in Neovim")
	step2 := mutedStyle.Render("2. Save and exit Neovim   " + lipgloss.NewStyle().Foreground(colorFgSelected).Render(":wq"))
	step3 := mutedStyle.Render("3. Close the terminal tab  " + lipgloss.NewStyle().Foreground(colorFgSelected).Bold(true).Render("exit"))
	step4 := mutedStyle.Render("4. Back here, press        " + lipgloss.NewStyle().Foreground(colorFgSelected).Bold(true).Render("r") + mutedStyle.Render("  to reload preview"))

	help := helpStyle.Render("Enter: open editor  Esc: cancel")

	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			fileStr,
			sep,
			info,
			"",
			step1,
			step2,
			step3,
			step4,
			help,
		),
	)
}

func (m model) renderMoveFileModal() string {
	destinations := m.moveDestinations()

	ext, extColor := getFileIcon(m.currentFileName())
	badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
	name := lipgloss.NewStyle().Foreground(colorFg).Render(m.currentFileName())
	title := modalTitleStyle.Render(" Move File")
	filesep := mutedStyle.Render(strings.Repeat("─", 44))

	var rows []string
	rows = append(rows, title, "", badge+"  "+name, filesep, "")

	if len(destinations) == 0 {
		rows = append(rows, mutedStyle.Render("No other folders available."))
	} else {
		for i, f := range destinations {
			folderIcon := mutedStyle.Render(" ")
			if i == m.moveCursor {
				arrow := lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
				nameStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(f)
				rows = append(rows, arrow+folderIcon+nameStr)
			} else {
				nameStr := lipgloss.NewStyle().Foreground(colorFg).Render(f)
				rows = append(rows, "   "+folderIcon+nameStr)
			}
		}
	}

	rows = append(rows, "", helpStyle.Render("↑↓: select folder   Enter: move   Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
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

func (m model) renderChangeDirPickerModal() string {
	title := modalTitleStyle.Render(" Change Directory")
	info := lipgloss.NewStyle().Foreground(colorFg).Render("Opening folder picker...")
	sub := mutedStyle.Render("Select a new directory to use as snippets root.")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", info, sub),
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

func (m model) renderGitSyncingModal() string {
	title := mutedStyle.Render(" Syncing to GitHub...")
	spinner := accentStyle.Render("Please wait")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", spinner),
	)
}

func (m model) renderStatusBar() string {
	var help string
	if m.statusMsg != "" {
		help = m.statusMsg
	} else if m.searchActive {
		help = "Typing: filter files  ↑↓: navigate results  Enter: select  Esc: cancel search"
	} else if m.previewSearchActive {
		if len(m.previewSearchHits) > 0 {
			help = fmt.Sprintf("Enter: search  n: next hit  N: prev hit  (%d matches)  Esc: close",
				len(m.previewSearchHits))
		} else {
			help = "Type word then Enter to search  Esc: cancel"
		}
	} else {
		switch m.activePanel {
		case panelFolders:
			help = "↑↓: folders  Enter/→: open  n: new folder  o: dir info  Tab/→: next panel  q: quit"
		case panelFiles:
			help = "↑↓: files  Enter: edit  /: search  n: new  m: move  c: import  d: delete  r: reload  Tab: next panel"
		case panelPreview:
			help = "↑↓: scroll  /: find word  L: line numbers  Tab: next panel  q: quit"
		}
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
	case modalEditorReady:
		modal = m.renderEditorReadyModal()
	case modalCopyFile:
		modal = m.renderCopyFileModal()
	case modalDeleteConfirm:
		modal = m.renderDeleteConfirmModal()
	case modalMoveFile:
		modal = m.renderMoveFileModal()
	case modalDirInfo:
		modal = m.renderDirInfoModal()
	case modalChangeDirPicker:
		modal = m.renderChangeDirPickerModal()
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