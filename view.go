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

	header := m.renderHeader()
	statusbar := m.renderStatusBar()

	var body string
	if m.previewZoom {
		previewW := m.width - 2
		body = m.renderPreviewPanel(previewW, totalH)
	} else {
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

		body = lipgloss.JoinHorizontal(lipgloss.Top, folders, files, preview)
	}

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
	snippetsTab := headerInactiveStyle.Render("Files")
	previewTab := headerInactiveStyle.Render("Preview")

	switch m.activePanel {
	case panelFolders:
		folderTab = headerActiveStyle.Render("Folders")
	case panelFiles:
		snippetsTab = headerActiveStyle.Render("Files")
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
		sb.WriteString(panelTitleStyle.Render(" Files") + "\n")
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
		sb.WriteString(mutedStyle.Render("0 files") + "\n\n")
		sb.WriteString(mutedStyle.Render("Press n to create a file"))
	} else {
		countN := len(filtered)
		totalN := len(m.files)
		var countStr string
		if m.searchActive && countN < totalN {
			countStr = fmt.Sprintf("%d / %d files", countN, totalN)
		} else if totalN == 1 {
			countStr = "1 snippet"
		} else {
			countStr = fmt.Sprintf("%d files", totalN)
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

	// reserve: title(1) + sep(1) + optional search bar(1)
	headerLines := 2
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
	hl := func(s string) string {
		if m.previewSearchActive && len(m.previewSearchHits) > 0 {
			return highlightMatches(s, m.previewSearchQuery)
		}
		return s
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
				contentLines = append(contentLines, hl(truncateAnsiLine(line, lineMaxW)))
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
					rendered = prefix + hl(truncateAnsiLine(line, lineMaxW-prefixW))
				} else if absLine == currentHitLine {
					prefix := lipgloss.NewStyle().Foreground(colorOrange).Bold(true).Render("▶ ")
					rendered = prefix + hl(truncateAnsiLine(line, lineMaxW-2))
				} else if hitSet[absLine] {
					prefix := lipgloss.NewStyle().Foreground(colorGreen).Render("• ")
					rendered = prefix + hl(truncateAnsiLine(line, lineMaxW-2))
				} else {
					rendered = hl(truncateAnsiLine(line, lineMaxW))
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
		help = "MULTI-DELETE: Space=select/deselect  D=confirm  Esc=cancel  ↑↓: navigate"
	} else {
		switch m.activePanel {
		case panelFolders:
			if len(m.folderDirStack) > 0 {
				help = "Enter: open | ←: back | n: new | N: subfolder | r: rename | d: delete | f: fav | /: search | q: quit"
			} else {
				help = "Enter: open | n: new | N: subfolder | r: rename | d: delete | f: fav | /: search  | o: pwd | q: quit"
			}
		case panelFiles:
			help = "Enter: edit | /: search | n: new | r: rename | m: move | c: import | d: delete"
		case panelPreview:
			help = "↑↓: scroll | /: find | c: copy | e: edit |  o: open folder"
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
		modal = RenderNewFolderModal(m.modalInput.View())
	case modalNewFile:
		modal = RenderNewFileModal(m.modalInput.View(), m.modalInput2.View(), m.modalStep)
	case modalError:
		modal = RenderErrorModal(m.modalError, m.width)
	case modalGitSetup:
		modal = RenderGitSetupModal(m.modalInput.View(), m.modalInput2.View(), m.modalInput3.View(), m.modalStep)
	case modalGitConfig:
		modal = RenderGitConfigModal(m.modalInput.View(), m.modalInput2.View(), m.modalInput3.View(), m.modalStep)
	case modalGitSuccess:
		modal = RenderGitSuccessModal(m.modalError)
	case modalGitSyncing:
		modal = RenderGitSyncingModal()
	case modalEditorReady:
		modal = RenderEditorReadyModal()
	case modalCopyFile:
		modal = RenderCopyFileModal(m.currentFolderName())
	case modalDeleteConfirm:
		modal = RenderDeleteConfirmModal(m.currentFileName())
	case modalMoveFile:
		modal = RenderMoveFileModal(m.currentFileName(), m.moveDestinationsAll(), m.moveCursor)
	case modalMoveFileBrowse:
		modal = RenderDirBrowserModal(m.dirBrowser, m.width, m.height)
	case modalDirInfo:
		modal = RenderDirInfoModal(m.snippetsDir)
	case modalDirBrowser:
		modal = RenderDirBrowserModal(m.dirBrowser, m.width, m.height)
	case modalFavorites:
		modal = RenderFavoritesModal(m.favorites, m.favCursor, func(absPath string) string { return m.favDisplayLabel(absPath) })
	case modalRenameFolder:
		modal = RenderRenameFolderModal(m.folderOpTargetAbs, m.origSnippetsDir, m.snippetsDir, m.currentFolderName(), m.modalInput.View())
	case modalDeleteFolder:
		modal = RenderDeleteFolderModal(m.folderOpTargetAbs, m.origSnippetsDir, m.snippetsDir, m.currentFolderName())
	case modalRenameFile:
		modal = RenderRenameFileModal(m.currentFileName(), m.modalInput.View())
	case modalSubfolderNav:
		modal = RenderSubfolderNavModal(m.subNavStack, m.subNavEntries, m.subNavCursor)
	case modalNewSubfolder:
		modal = RenderNewSubfolderModal(m.newSubfolderParentAbs, m.origSnippetsDir, m.snippetsDir, m.currentFolderName(), m.modalInput.View())
	case modalConsole:
		modal = RenderConsoleModal(m.consoleInput, m.consoleOutput, m.width)
	case modalTimeCalc:
		modal = RenderTimeCalcModal(m.timeResult, m.modalInput.View())
	case modalWhoami:
		modal = RenderWhoamiModal()
	case modalLs:
		modal = RenderLsModal(m.snippetsDir, m.lsTree, m.lsScroll, m.lsInput, m.lsMsg, m.height)
	case modalHelp:
		modal = RenderHelpPanel(m.helpTab, m.helpScroll, m.width, m.height)
	case modalHelpConsole:
		modal = RenderCmdHelp(m.consoleHelpTab, m.consoleHelpScroll, m.width, m.height)
	case modalAlive:
		modal = RenderAliveModal(m.aliveInput)
	case modalSubfolderSelect:
		modal = RenderSubfolderSelectModal(m.subSelectEntries, m.subSelectCursor)
	case modalMultiDeleteConfirm:
		modal = RenderMultiDeleteConfirmModal(m.multiDeleteSelected)
	case modalFolderSearch:
		modal = RenderFolderSearchModal(m.folderSearchModalQuery, m.folderSearchModalResults, m.folderSearchModalCursor)
	}

	return overlayModal(base, modal, m.width, m.height)
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
