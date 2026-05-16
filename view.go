package main

import (
	"fmt"
	"path/filepath"
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
	innerH := h - 2 // total lines budget inside the panel border

	folderIcon := "󰉋 "
	starIcon := lipgloss.NewStyle().Foreground(colorOrange).Render("★")

	// main section: title(1) + divider(1) + folder rows
	mainListH := innerH - 2
	if mainListH < 1 {
		mainListH = 1
	}

	// collect exactly innerH lines into `lines []string`
	lines := make([]string, 0, innerH)

	// ── Title ─────────────────────────────────────────────────────────
	titleText := " Folders"
	if m.inParentView {
		titleText = " " + filepath.Base(m.parentViewDir) + " " + mutedStyle.Render("← back")
	} else if len(m.folderDirStack) > 0 {
		titleText = " Folders " + mutedStyle.Render("← back")
	} else if m.snippetsDir != m.origSnippetsDir {
		titleText = " Folders " + mutedStyle.Render("H:home")
	}
	if m.folderSearchActive {
		searchBar := lipgloss.NewStyle().Foreground(colorOrange).Render("/ ") +
			lipgloss.NewStyle().Foreground(colorFg).Render(m.folderSearchQuery) +
			lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
		lines = append(lines, searchBar)
	} else {
		lines = append(lines, panelTitleStyle.Render(titleText))
	}
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", innerW-1)))

	if m.inParentView {
		// ── Parent-view list: ~/ row + subfolder rows ─────────────────
		subs := m.subfolderNames(m.parentViewDir)
		// total items = 1 (~/) + len(subs)
		totalItems := 1 + len(subs)
		scroll := clampScroll(m.folderCursor, m.folderScroll, mainListH)
		for row := 0; row < mainListH; row++ {
			idx := scroll + row
			if idx >= totalItems {
				lines = append(lines, "")
				continue
			}
			var rowStr string
			if idx == 0 {
				// ~/ entry — show star if parentViewDir itself is a favorite
				parentIsFav := m.isFavoriteAbs(m.parentViewDir)
				favMark := ""
				if parentIsFav {
					favMark = " " + starIcon
				}
				if m.folderCursor == 0 {
					rowStr = arrowStyle.Render("> ") +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ") +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render("~/") +
						favMark
				} else {
					rowStr = "   " + mutedStyle.Render("󰉋 ") +
						mutedStyle.Render("~/") + favMark
				}
			} else {
				// subfolder entry
				subIdx := idx - 1
				name := subs[subIdx]
				label := truncate(name, innerW-6)
				subAbs := filepath.Join(m.parentViewDir, name)
				hasChildren := len(m.subfolderNames(subAbs)) > 0
				subMark := ""
				if hasChildren {
					subMark = mutedStyle.Render(" ›")
				}
				// Show star if this subfolder is a favorite
				isFav := m.isFavoriteAbs(subAbs)
				favMark := ""
				if isFav {
					favMark = " " + starIcon
				}
				if m.folderCursor == idx {
					rowStr = arrowStyle.Render("> ") +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(folderIcon) +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(label) +
						subMark + favMark
				} else {
					rowStr = "   " + mutedStyle.Render(folderIcon) +
						lipgloss.NewStyle().Foreground(colorFg).Render(label) +
						subMark + favMark
				}
			}
			lines = append(lines, rowStr)
		}
	} else {
		// ── Main folders list (virtual scroll, no scrollbar) ─────────────
		displayFolders := m.filteredFolders()

		// Build virtual list: when hasRootFiles insert ~/ at index 0
		type virtualEntry struct {
			isRoot bool
			name   string
		}
		var virtualList []virtualEntry
		if m.hasRootFiles && !m.folderSearchActive {
			virtualList = append(virtualList, virtualEntry{isRoot: true})
		}
		for _, f := range displayFolders {
			virtualList = append(virtualList, virtualEntry{name: f})
		}

		scroll := clampScroll(m.folderCursor, m.folderScroll, mainListH)

		if len(virtualList) == 0 {
			lines = append(lines, mutedStyle.Render("No folders yet"))
			lines = append(lines, mutedStyle.Render("Press n to create"))
		} else {
			for row := 0; row < mainListH; row++ {
				idx := scroll + row
				if idx >= len(virtualList) {
					lines = append(lines, "")
					continue
				}
				entry := virtualList[idx]
				isSelected := idx == m.folderCursor

				var rowStr string
				if entry.isRoot {
					// ~/  — check if snippetsDir itself is a favorite
					rootIsFav := m.isFavoriteAbs(m.snippetsDir)
					rootFavMark := ""
					if rootIsFav {
						rootFavMark = " " + starIcon
					}
					if isSelected {
						rowStr = arrowStyle.Render("> ") +
							lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ") +
							lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render("~/") +
							rootFavMark
					} else {
						rowStr = "   " + mutedStyle.Render("󰉋 ") + mutedStyle.Render("~/") + rootFavMark
					}
					lines = append(lines, rowStr)
					continue
				}
				name := entry.name
				label := truncate(name, innerW-6)
				isFav := m.isFavorite(name)
				favMark := ""
				if isFav {
					favMark = " " + starIcon
				}
				subMark := ""
				if m.hasSubfolders(name) {
					subMark = mutedStyle.Render(" ›")
				}
				// Determine the real folder index for multi-delete highlighting
				folderIdx := idx
				if m.hasRootFiles {
					folderIdx = idx - 1 // account for the ~/ virtual entry
				}
				isMultiSelected := m.multiDeleteMode && m.multiDeleteSelected[folderIdx]
				if isMultiSelected {
					// highlight in red for multi-delete selection
					redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true)
					rowStr = arrowStyle.Render("> ") +
						redStyle.Render(folderIcon) +
						redStyle.Render(label) +
						subMark +
						favMark
				} else if isSelected {
					rowStr = arrowStyle.Render("> ") +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(folderIcon) +
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(label) +
						subMark +
						favMark
				} else {
					rowStr = "   " +
						mutedStyle.Render(folderIcon) +
						lipgloss.NewStyle().Foreground(colorFg).Render(label) +
						subMark +
						favMark
				}
				lines = append(lines, rowStr)
			}
		}
	}

	// ── Pad to exact innerH ───────────────────────────────────────────
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	content := strings.Join(lines[:innerH], "\n")

	style := panelStyle
	if isActive {
		style = activePanelStyle
	}
	return style.Width(w).Height(h).Render(content)
}


func (m model) renderFilesPanel(w, h int) string {
	isActive := m.activePanel == panelFiles

	innerW := w - 2
	innerH := h - 2

	var sb strings.Builder

	// ── title + search bar ───────────────────────────────────────────────────
	if m.searchActive {
		cur := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
		searchBar := lipgloss.NewStyle().Foreground(colorAccentBlue).Render(" / ") +
			lipgloss.NewStyle().Foreground(colorFg).Render(m.searchQuery) +
			cur
		sb.WriteString(searchBar + "\n")
	} else {
		sb.WriteString(panelTitleStyle.Render(" Snippets") + "\n")
	}
	sb.WriteString(mutedStyle.Render(strings.Repeat("─", innerW)) + "\n")

	// ── file list (filtered when searching) ──────────────────────────────────────
	filtered := m.filteredFiles()
	// Determine the display name for the folder context shown in file metadata
	var folderName string
	if m.inParentView {
		if m.folderCursor == 0 {
			folderName = "~/"
		} else {
			subs := m.subfolderNames(m.parentViewDir)
			idx := m.folderCursor - 1
			if idx < len(subs) {
				folderName = subs[idx]
			}
		}
	} else if m.hasRootFiles && m.folderCursor == 0 {
		folderName = "~/"
	} else {
		folderName = m.currentFolderName()
	}

	noFolders := !m.inParentView && !m.hasRootFiles && len(m.folders) == 0
	if noFolders {
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

		// Each file entry uses 3 lines: name, meta, blank.
		// Available lines after header (title+div+count+blank = 4 lines used above).
		headerUsed := 4
		availLines := innerH - headerUsed
		itemH := 3 // lines per file entry
		visibleItems := max(1, availLines/itemH)

		fileScroll := clampScroll(cur, m.fileScroll, visibleItems)

		maxNameW := innerW - 8 // indent(2) + badge(5) + space(1)
		metaIndent := "       " // 7 chars
		maxMetaW := innerW - len(metaIndent)

		for i := fileScroll; i < len(filtered) && i < fileScroll+visibleItems; i++ {
			f := filtered[i]
			ext, extColor := getFileIcon(f.name)
			badge := lipgloss.NewStyle().
				Foreground(extColor).
				Width(5).
				Align(lipgloss.Right).
				Render(ext)
			rel := relativeTime(f.modTime)
			displayName := truncate(f.name, maxNameW)
			metaText := truncate(folderName+" • "+rel, maxMetaW)

			if i == cur {
				arrow := fileArrowStyle.Render("> ")
				nameStr := lipgloss.NewStyle().Foreground(colorGreen).Render(displayName)
				sb.WriteString(arrow + badge + " " + nameStr + "\n")
			} else {
				nameStr := lipgloss.NewStyle().Foreground(colorFg).Render(displayName)
				sb.WriteString("  " + badge + " " + nameStr + "\n")
			}
			sb.WriteString(metaIndent + mutedStyle.Render(metaText) + "\n")
			sb.WriteString("\n")
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

	// Determine the file name to display from previewFilePath (covers subNav) or currentFileName
	displayName := m.currentFileName()
	if m.previewFilePath != "" {
		displayName = filepath.Base(m.previewFilePath)
	}

	// reserve: title(1) + sep(1) + path line when path is known(1) + optional search bar(1)
	headerLines := 2
	showPath := m.previewFilePath != ""
	if showPath {
		headerLines = 3
	}
	if m.previewSearchActive {
		headerLines++
	}
	availH := h - 2 - headerLines // h-2 for panel border

	var panelTitle string
	if displayName != "" {
		ext, extColor := getFileIcon(displayName)
		badge := lipgloss.NewStyle().Foreground(extColor).Render(ext)
		name := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Render(displayName)
		// indicator tags
		lnTag := ""
		if m.previewLineNumbers {
			lnTag = " " + mutedStyle.Render("[LN]")
		}
		mdTag := ""
		if m.previewIsMarkdown {
			mdTag = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#519aba")).Bold(true).Render("[MD]")
		}
		panelTitle = " " + badge + "  " + name + lnTag + mdTag
	} else {
		panelTitle = panelTitleStyle.Render(" Preview")
	}

	var contentLines []string
	contentLines = append(contentLines, panelTitle)
	contentLines = append(contentLines, mutedStyle.Render(strings.Repeat("─", w-4)))
	if showPath {
		pathStr := truncate(m.previewFilePath, w-6)
		contentLines = append(contentLines,
			" "+lipgloss.NewStyle().Foreground(colorOrange).Render(pathStr))
	}

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

		// visible width budget for content inside the panel borders/padding
		lineMaxW := w - 4
		if lineMaxW < 10 {
			lineMaxW = 10
		}

		if m.previewIsMarkdown {
			// glamour output: render lines directly without gutter or hit markers
			// (its ANSI sequences would be mangled by prefix injection)
			for _, line := range hLines[start:end] {
				contentLines = append(contentLines, truncateAnsiLine(line, lineMaxW))
			}
		} else {
			// line number gutter width
			totalLines := len(strings.Split(m.previewContent, "\n"))
			gnW := len(fmt.Sprintf("%d", totalLines))

			for i, line := range hLines[start:end] {
				absLine := start + i // 0-based line index
				var rendered string
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
					prefix := gnStyle.Render(lineNum) + mutedStyle.Render(" │ ")
					prefixW := gnW + 3
					rendered = prefix + truncateAnsiLine(line, lineMaxW-prefixW)
				} else if absLine == currentHitLine {
					prefix := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("▶ ")
					rendered = prefix + truncateAnsiLine(line, lineMaxW-2)
				} else if hitSet[absLine] {
					prefix := lipgloss.NewStyle().Foreground(colorGreen).Render("• ")
					rendered = prefix + truncateAnsiLine(line, lineMaxW-2)
				} else {
					rendered = truncateAnsiLine(line, lineMaxW)
				}
				contentLines = append(contentLines, rendered)
			}
		}
	}

	// Hard-clamp contentLines to availH so long files never overflow the panel
	if len(contentLines) > headerLines+availH {
		contentLines = contentLines[:headerLines+availH]
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
		if m.statusIsSuccess {
			return lipgloss.NewStyle().
				Background(colorBg).
				Foreground(colorGreen).
				Bold(true).
				Width(m.width).
				Padding(0, 1).
				Render(m.statusMsg)
		}
		help = m.statusMsg
	} else if m.folderSearchActive {
		help = "Typing: filter folders  ↑↓: navigate results  Enter: select  Esc: cancel search"
	} else if m.searchActive {
		help = "Typing: filter files  ↑↓: navigate results  Enter: select  Esc: cancel search"
	} else if m.previewSearchActive {
		if len(m.previewSearchHits) > 0 {
			help = fmt.Sprintf("Enter: search  n: next hit  N: prev hit  (%d matches)  Esc: close",
				len(m.previewSearchHits))
		} else {
			help = "Type word then Enter to search  Esc: cancel"
		}
	} else if m.multiDeleteMode {
		help = "MULTI-DELETE: Space=select/deselect  X=confirm  Esc=cancel  ↑↓: navigate"
	} else {
		switch m.activePanel {
		case panelFolders:
			if len(m.folderDirStack) > 0 {
				help = "↑↓: folders  Enter: open  ←: back  n: new  N: new subfolder  r: rename  x: delete  X: multi-delete  d: fav  D: favs  /: search  q: quit"
			} else {
				help = "↑↓: folders  Enter: open  n: new  N: new subfolder  r: rename  x: delete  X: multi-delete  d: fav  D: favs  /: search  o: location  q: quit"
			}
		case panelFiles:
			help = "↑↓: files  Enter: edit  /: search  n: new  r: rename  m: move  c: import  d: delete  g: sync  G: git config  Tab: next panel"
		case panelPreview:
			help = "↑↓: scroll  /: find  L: line numbers  c: copy  e: edit in nvim  o: open folder  g: sync  Tab: next panel  q: quit"
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
	case modalMoveFileBrowse:
		modal = m.renderMoveFileBrowseModal()
	case modalDirInfo:
		modal = m.renderDirInfoModal()
	case modalDirBrowser:
		modal = m.renderDirBrowserModal()
	case modalFavorites:
		modal = m.renderFavoritesModal()
	case modalRenameFolder:
		modal = m.renderRenameFolderModal()
	case modalDeleteFolder:
		modal = m.renderDeleteFolderModal()
	case modalRenameFile:
		modal = m.renderRenameFileModal()
	case modalSubfolderNav:
		modal = m.renderSubfolderNavModal()
	case modalNewSubfolder:
		modal = m.renderNewSubfolderModal()
	case modalConsole:
		modal = m.renderConsoleModal()
	case modalTimeCalc:
		modal = m.renderTimeCalcModal()
	case modalWhoami:
		modal = m.renderWhoamiModal()
	case modalHelpConsole:
		modal = m.renderHelpConsoleModal()
	case modalNvimGuide:
		modal = m.renderNvimGuideModal()
	case modalSubfolderSelect:
		modal = m.renderSubfolderSelectModal()
	case modalMultiDeleteConfirm:
		modal = m.renderMultiDeleteConfirmModal()
	case modalFolderSearch:
		modal = m.renderFolderSearchModal()
	}

	return overlayModal(base, modal, m.width, m.height)
}

func (m model) renderFavoritesModal() string {
	title := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(" ★ Favorites")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	var rows []string
	rows = append(rows, title, sep)

	if len(m.favorites) == 0 {
		rows = append(rows, mutedStyle.Render("No favorites yet"))
	} else {
		for i, absPath := range m.favorites {
			label := truncate(m.favDisplayLabel(absPath), 40)
			if i == m.favCursor {
				rows = append(rows,
					arrowStyle.Render("> ")+
						lipgloss.NewStyle().Foreground(colorOrange).Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render(label))
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(label))
			}
		}
	}

	rows = append(rows, sep, helpStyle.Render("↑↓: navigate   Enter: go to folder   o: open in Explorer   f: unfavorite   Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
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

func (m model) renderRenameFolderModal() string {
	// Use folderOpTargetAbs for the correct display path
	absPath := m.folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(m.snippetsDir, m.currentFolderName())
	}
	root := m.origSnippetsDir
	if root == "" {
		root = m.snippetsDir
	}
	var displayPath string
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		displayPath = filepath.Base(absPath)
	} else {
		displayPath = filepath.ToSlash(rel) + "/"
	}
	title := modalTitleStyle.Render(" Rename Folder")
	current := mutedStyle.Render("Current: ") + lipgloss.NewStyle().Foreground(colorAccentBlue).Render(displayPath)
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

func (m model) renderDeleteFolderModal() string {
	// Use folderOpTargetAbs (set when modal was opened) for the correct path.
	absPath := m.folderOpTargetAbs
	if absPath == "" {
		absPath = filepath.Join(m.snippetsDir, m.currentFolderName())
	}
	name := filepath.Base(absPath)
	// Build full relative path for display
	var fullPath string
	root := m.origSnippetsDir
	if root == "" {
		root = m.snippetsDir
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		fullPath = name
	} else {
		fullPath = filepath.ToSlash(rel) + "/"
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete Folder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Delete folder and ALL its contents?")
	nameStr := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render("  " + fullPath)
	alert := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Render("This cannot be undone.")
	help := helpStyle.Render("Enter / y: delete    Esc / n: cancel")
	return modalStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title, "", warn, nameStr, "", alert, sep, help,
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

func (m model) renderSubfolderNavModal() string {
	// build breadcrumb path
	crumb := "/"
	if len(m.subNavStack) > 0 {
		crumb = "/" + strings.Join(m.subNavStack, "/")
	}
	title := modalTitleStyle.Render(" Browse Folder")
	breadcrumb := mutedStyle.Render(crumb)
	sep := mutedStyle.Render(strings.Repeat("─", 44))

	var rows []string
	rows = append(rows, title, breadcrumb, sep)

	if len(m.subNavEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(empty)"))
	} else {
		for i, e := range m.subNavEntries {
			var icon string
			if e.isDir {
				icon = "󰉋 "
			} else {
				ext, extColor := getFileIcon(e.name)
				icon = lipgloss.NewStyle().Foreground(extColor).Render(ext) + " "
			}
			label := truncate(e.name, 36)
			if i == m.subNavCursor {
				arrow := arrowStyle.Render("> ")
				if e.isDir {
					rows = append(rows, arrow+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(icon)+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render(label))
				} else {
					rows = append(rows, arrow+icon+
						lipgloss.NewStyle().Foreground(colorGreen).Render(label))
				}
			} else {
				if e.isDir {
					rows = append(rows, "   "+mutedStyle.Render(icon)+
						lipgloss.NewStyle().Foreground(colorFg).Render(label))
				} else {
					rows = append(rows, "   "+icon+
						mutedStyle.Render(label))
				}
			}
		}
	}

	backLabel := "Backspace: up"
	if len(m.subNavStack) <= 1 {
		backLabel = "Backspace: close"
	}
	rows = append(rows, sep,
		helpStyle.Render("↑↓: navigate  Enter: open  "+backLabel+"  Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
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

func (m model) renderNewSubfolderModal() string {
	// Use newSubfolderParentAbs (set when modal opened) to display the correct parent name/path.
	var parentDisplay string
	if m.newSubfolderParentAbs != "" {
		root := m.origSnippetsDir
		if root == "" {
			root = m.snippetsDir
		}
		rel, err := filepath.Rel(root, m.newSubfolderParentAbs)
		if err != nil {
			parentDisplay = filepath.Base(m.newSubfolderParentAbs)
		} else {
			parentDisplay = filepath.ToSlash(rel) + "/"
		}
	} else {
		parentDisplay = m.currentFolderName()
	}
	title := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(" New Subfolder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	info := mutedStyle.Render("Inside: ") + lipgloss.NewStyle().Foreground(colorOrange).Render(parentDisplay)
	rows := []string{title, sep, info, "", " " + m.modalInput.View(), ""}
	rows = append(rows, sep, mutedStyle.Render("Enter: create  Esc: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) renderConsoleModal() string {
	cyan := lipgloss.Color("#00e5ff")
	title := lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("─── Cmdline ───")
	sep := mutedStyle.Render(strings.Repeat("─", 60))
	prompt := lipgloss.NewStyle().Foreground(cyan).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(m.consoleInput) +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Padding(0, 1).
		Width(58).
		Render(prompt)
	rows := []string{"", title, "", inputLine}
	if m.consoleOutput != "" {
		rows = append(rows, "", sep)
		for _, l := range strings.Split(m.consoleOutput, "\n") {
			rows = append(rows, lipgloss.NewStyle().Foreground(colorOrange).Render(l))
		}
	}
	rows = append(rows, "", sep,
		mutedStyle.Render("time · whoami · nvim · help · clear · exit  |  Esc: close"))
	style := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan).
		Padding(1, 2)
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
	blue := colorAccentBlue
	colStyle := lipgloss.NewStyle().
		Foreground(colorFg).
		Background(colorBg).
		Border(lipgloss.NormalBorder()).
		BorderForeground(blue).
		Padding(0, 1).
		Width(34)

	leftCol := colStyle.Render(
		lipgloss.NewStyle().Foreground(blue).Bold(true).Render(" Shortcuts — Left") + "\n" +
			mutedStyle.Render(strings.Repeat("─", 32)) + "\n" +
			lipgloss.NewStyle().Foreground(colorFg).Render(helpLeft),
	)
	rightCol := colStyle.Render(
		lipgloss.NewStyle().Foreground(blue).Bold(true).Render(" Shortcuts — Right") + "\n" +
			mutedStyle.Render(strings.Repeat("─", 32)) + "\n" +
			lipgloss.NewStyle().Foreground(colorFg).Render(helpRight),
	)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)

	title := lipgloss.NewStyle().Foreground(blue).Bold(true).Render(" Help — Keyboard Shortcuts & Commands")
	footer := mutedStyle.Render("Enter / Esc: back to console")

	outer := lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(blue).
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

// truncateAnsiLine truncates a line that may contain ANSI escape codes so that
// its *visible* character count does not exceed maxVisible. The ANSI reset
// sequence is appended when truncation was needed, so colours do not bleed
// into the next line.
func truncateAnsiLine(s string, maxVisible int) string {
	if maxVisible <= 0 {
		return ""
	}
	visible := 0
	inEsc := false
	var out strings.Builder
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			inEsc = true
			out.WriteRune(r)
			continue
		}
		if inEsc {
			out.WriteRune(r)
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if visible >= maxVisible {
			// truncated — close any open colour sequence
			out.WriteString("\x1b[0m")
			return out.String()
		}
		out.WriteRune(r)
		visible++
	}
	return out.String()
}

// renderSubfolderSelectModal renders the "Select the subfolder" modal.
func (m model) renderSubfolderSelectModal() string {
	// Build breadcrumb
	crumb := strings.Join(m.subSelectStack, "/") + "/"
	title := modalTitleStyle.Render(" Select the subfolder to open")
	breadcrumb := mutedStyle.Render("  " + crumb)
	sep := mutedStyle.Render(strings.Repeat("─", 50))

	var rows []string
	rows = append(rows, title, breadcrumb, sep)

	const maxVisible = 10
	start := 0
	if m.subSelectCursor >= start+maxVisible {
		start = m.subSelectCursor - maxVisible + 1
	}

	if len(m.subSelectEntries) == 0 {
		rows = append(rows, mutedStyle.Render("(no subfolders)"))
	} else {
		for i := start; i < len(m.subSelectEntries) && i < start+maxVisible; i++ {
			sub := m.subSelectEntries[i]
			displayPath := m.subSelectDisplayPath(sub)
			label := truncate(displayPath, 44)
			// Check if sub itself has subfolders (to show › indicator)
			subAbsParts := append(append([]string{m.snippetsDir}, m.subSelectStack...), sub)
			subAbs := filepath.Join(subAbsParts...)
			hasSubs := len(m.subfolderNames(subAbs)) > 0
			subMark := ""
			if hasSubs {
				subMark = mutedStyle.Render(" ›")
			}
			if i == m.subSelectCursor {
				rows = append(rows,
					arrowStyle.Render("> ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(label)+
						subMark)
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(label)+
						subMark)
			}
		}
	}

	rows = append(rows, sep,
		helpStyle.Render("Enter: Select  →: Access subfolders  ←: Back to parent  q: Exit"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// renderMultiDeleteConfirmModal renders the confirmation modal for multi-folder deletion.
func (m model) renderMultiDeleteConfirmModal() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render(" Delete Folders")
	sep := mutedStyle.Render(strings.Repeat("─", 48))
	warn := lipgloss.NewStyle().Foreground(colorFg).Render("Delete the following folders and ALL their contents?")
	alert := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Render("This cannot be undone.")

	var rows []string
	rows = append(rows, title, "", warn)

	for idx := range m.multiDeleteSelected {
		if idx < len(m.folders) {
			name := m.folders[idx]
			rows = append(rows,
				"  "+lipgloss.NewStyle().Foreground(lipgloss.Color("#ff7b72")).Bold(true).Render("  "+name))
		}
	}

	rows = append(rows, "", alert, sep,
		helpStyle.Render("Enter / y: delete all    Esc / n: cancel"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// renderFolderSearchModal renders the folder search modal.
func (m model) renderFolderSearchModal() string {
	title := modalTitleStyle.Render(" Search Folder")
	sep := mutedStyle.Render(strings.Repeat("─", 48))

	// Search bar
	cur := lipgloss.NewStyle().Background(colorAccentBlue).Foreground(colorBg).Render(" ")
	searchBar := lipgloss.NewStyle().Foreground(colorAccentBlue).Render("/ ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(m.folderSearchModalQuery) +
		cur

	var rows []string
	rows = append(rows, title, searchBar, sep)

	const maxVisible = 12
	results := m.folderSearchModalResults
	start := 0
	if m.folderSearchModalCursor >= start+maxVisible {
		start = m.folderSearchModalCursor - maxVisible + 1
	}

	if len(results) == 0 {
		if m.folderSearchModalQuery != "" {
			rows = append(rows, mutedStyle.Render("No folders found for: ")+
				lipgloss.NewStyle().Foreground(colorAccentBlue).Render(m.folderSearchModalQuery))
		} else {
			rows = append(rows, mutedStyle.Render("Type to search folders..."))
		}
	} else {
		for i := start; i < len(results) && i < start+maxVisible; i++ {
			label := truncate(results[i].displayPath, 44)
			if i == m.folderSearchModalCursor {
				rows = append(rows,
					arrowStyle.Render("> ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render(label))
			} else {
				rows = append(rows,
					"   "+mutedStyle.Render("󰉋 ")+
						lipgloss.NewStyle().Foreground(colorFg).Render(label))
			}
		}
	}

	rows = append(rows, sep, helpStyle.Render("Type: filter  ↑↓: navigate  Enter: go to folder  Esc: close"))
	return modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}