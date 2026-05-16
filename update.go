package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type clearStatusMsg struct{}


type moveFileResultMsg struct {
	destFolder string
	err        error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Match the actual previewW computed in renderMain: width - foldersW(22) - filesW(32) - 6 borders, minus 4 inner padding
		pw := msg.Width - 22 - 32 - 6 - 4
		if pw < 40 {
			pw = 40
		}
		m.previewWidth = pw
		// re-render markdown with new width if a .md file is open
		if m.previewIsMarkdown && m.previewContent != "" {
			m.previewHighlight = renderMarkdown(m.previewContent, pw)
		}
		return m, nil

	case openEditorMsg:
		_, err := exec.LookPath("nvim")
		if err != nil {
			m.modal = modalEditorReady
			return m, nil
		}
		// Launch nvim directly in the current terminal via tea.ExecProcess
		c := exec.Command("nvim", msg.path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return editorDoneMsg{}
		})

	case editorDoneMsg:
		m.loadPreview()
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		m.statusIsSuccess = false
		return m, nil

	case fileCopyResultMsg:
		if msg.err != nil {
			m.modal = modalError
			m.modalError = msg.err.Error()
		} else if msg.copied == 0 {
			m.modal = modalNone
			m.statusMsg = "No file selected."
			return m, clearStatusAfter(3 * time.Second)
		} else {
			m.modal = modalNone
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = fmt.Sprintf("%d file(s) copied successfully.", msg.copied)
			return m, clearStatusAfter(3 * time.Second)
		}
		return m, nil

	case moveFileResultMsg:
		m.modal = modalNone
		if msg.err != nil {
			m.modal = modalError
			m.modalError = msg.err.Error()
			return m, nil
		}
		m.loadFiles()
		m.loadPreview()
		m.statusMsg = "File moved to " + msg.destFolder + "."
		return m, clearStatusAfter(3 * time.Second)

	case gitSyncResultMsg:
		if msg.err != nil {
			m.modal = modalError
			m.modalError = msg.err.Error()
		} else {
			m.modal = modalGitSuccess
			m.modalError = msg.output
		}
		return m, nil

	case tea.KeyMsg:
		if m.modal != modalNone {
			return m.handleModalKey(msg)
		}
		if m.folderSearchActive {
			return m.handleFolderSearchKey(msg)
		}
		if m.searchActive {
			return m.handleSearchKey(msg)
		}
		if m.previewSearchActive {
			return m.handlePreviewSearchKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Exit multi-delete mode on Escape
	if m.multiDeleteMode && msg.String() == "esc" {
		m.multiDeleteMode = false
		m.multiDeleteSelected = nil
		m.statusMsg = "Multi-delete cancelled"
		return m, clearStatusAfter(2 * time.Second)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "tab":
		switch m.activePanel {
		case panelFolders:
			m.activePanel = panelFiles
		case panelFiles:
			m.activePanel = panelPreview
		case panelPreview:
			m.activePanel = panelFolders
		}

	case "left":
		if m.activePanel == panelFolders {
			if m.inParentView {
				// exit parent-view mode back to normal folder list
				m.inParentView = false
				m.parentViewDir = ""
				// restore folderCursor to the folder we came from
				if len(m.folderDirStack) > 0 {
					m.snippetsDir = m.folderDirStack[len(m.folderDirStack)-1]
					m.folderDirStack = m.folderDirStack[:len(m.folderDirStack)-1]
				}
				m.folderCursor = 0
				m.fileCursor = 0
				m.loadFolders()
				m.loadFiles()
				m.loadPreview()
			} else if len(m.folderDirStack) > 0 {
				// go back to parent dir if we navigated into a subfolder
				m.snippetsDir = m.folderDirStack[len(m.folderDirStack)-1]
				m.folderDirStack = m.folderDirStack[:len(m.folderDirStack)-1]
				m.folderCursor = 0
				m.fileCursor = 0
				m.loadFolders()
				m.loadFiles()
				m.loadPreview()
			} else if m.activePanel > panelFolders {
				m.activePanel--
			}
		} else {
			m.activePanel--
		}

	case "h":
		if m.activePanel > panelFolders {
			m.activePanel--
		}

	case "right", "l":
		if m.activePanel < panelPreview {
			m.activePanel++
		}

	case "up", "k":
		switch m.activePanel {
		case panelFolders:
			if m.inParentView {
				if m.folderCursor > 0 {
					m.folderCursor--
					m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
					m.fileCursor = 0
					m.fileScroll = 0
					m.loadFiles()
					m.loadPreview()
				}
			} else if m.folderCursor > 0 {
				m.folderCursor--
				m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
				m.fileCursor = 0
				m.fileScroll = 0
				m.loadFiles()
				m.loadPreview()
			} else if m.hasRootFiles && m.folderCursor == 0 {
				// already at ~/ root — nothing to do
			}
		case panelFiles:
			if m.fileCursor > 0 {
				m.fileCursor--
				m.fileScroll = clampScroll(m.fileCursor, m.fileScroll, 10)
				m.loadPreview()
			}
		case panelPreview:
			if m.previewScroll > 0 {
				m.previewScroll--
			}
		}

	case "down", "j":
		switch m.activePanel {
		case panelFolders:
			if m.inParentView {
				subs := m.subfolderNames(m.parentViewDir)
				maxIdx := len(subs) // 0=~/, 1..len(subs)=subfolders
				if m.folderCursor < maxIdx {
					m.folderCursor++
					m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
					m.fileCursor = 0
					m.fileScroll = 0
					m.loadFiles()
					m.loadPreview()
				}
			} else {
				totalFolderItems := len(m.folders)
				if m.hasRootFiles {
					totalFolderItems++
				}
				if m.folderCursor < totalFolderItems-1 {
					m.folderCursor++
					m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
					m.fileCursor = 0
					m.fileScroll = 0
					m.loadFiles()
					m.loadPreview()
				}
			}
		case panelFiles:
			if m.fileCursor < len(m.files)-1 {
				m.fileCursor++
				m.fileScroll = clampScroll(m.fileCursor, m.fileScroll, 10)
				m.loadPreview()
			}
		case panelPreview:
			m.previewScroll++
		}

	case "enter":
		switch m.activePanel {
		case panelFolders:
			if m.inParentView {
				// In parent-view: Enter on ~/ does nothing (already showing its files)
				// Enter on a subfolder: check if it itself has subfolders → open subSelect modal
				if m.folderCursor == 0 {
					// ~/ row — no-op, files already showing
					break
				}
				subs := m.subfolderNames(m.parentViewDir)
				idx := m.folderCursor - 1
				if idx >= len(subs) {
					break
				}
				subDir := filepath.Join(m.parentViewDir, subs[idx])
				subSubs := m.subfolderNames(subDir)
				if len(subSubs) > 0 {
					// subfolder also has children — open subfolder select modal
					// subSelectStack is relative to m.snippetsDir
					rel, _ := filepath.Rel(m.snippetsDir, subDir)
					parts := strings.Split(filepath.ToSlash(rel), "/")
					m.subSelectStack = parts
					m.subSelectCursor = 0
					m.loadSubSelectEntries()
					m.modal = modalSubfolderSelect
					return m, nil
				}
				// plain subfolder — navigate into it directly (no subfolders)
				m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
				m.inParentView = false
				m.parentViewDir = ""
				m.snippetsDir = subDir
				m.folderCursor = 0
				m.fileCursor = 0
				m.loadFolders()
				m.loadFiles()
				m.loadPreview()
				m.statusMsg = "Inside: " + filepath.Base(subDir) + "  (← to go back)"
				m.statusIsSuccess = true
				return m, clearStatusAfter(4 * time.Second)
			} else {
				folderName := m.currentFolderName()
				if folderName != "" {
					folderAbs := filepath.Join(m.snippetsDir, folderName)
					subs := m.subfolderNames(folderAbs)
					if len(subs) > 0 {
						// folder has subfolders — open subfolder select modal
						m.subSelectStack = []string{folderName}
						m.subSelectCursor = 0
						m.loadSubSelectEntries()
						m.modal = modalSubfolderSelect
						return m, nil
					}
					// plain folder — navigate directly
					m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
					m.snippetsDir = folderAbs
					m.folderCursor = 0
					m.fileCursor = 0
					m.loadFolders()
					m.loadFiles()
					m.loadPreview()
					m.statusMsg = "Navigated into: " + folderName + "  (← to go back)"
					m.statusIsSuccess = true
					return m, clearStatusAfter(4 * time.Second)
				}
			}
		case panelFiles:
			if len(m.files) > 0 {
				path := m.currentFilePath()
				return m, openNeovim(path)
			}
		}


	case "n":
		switch m.activePanel {
		case panelFolders:
			m.openModal(modalNewFolder)
		case panelFiles:
			m.openModal(modalNewFile)
		}

	case "N":
		// create a new subfolder inside the currently selected folder
		if m.activePanel == panelFolders {
			m.openModal(modalNewSubfolder)
		}

	case "d":
		switch m.activePanel {
		case panelFolders:
			// favorite/unfavorite selected folder
			name := m.currentFolderName()
			if name != "" {
				m.toggleFavoriteFolder()
			}
		case panelFiles:
			// delete selected file
			if len(m.files) > 0 {
				m.modal = modalDeleteConfirm
			}
		}

	case "D":
		switch m.activePanel {
		case panelFolders:
			// open favorites modal
			if len(m.favorites) > 0 {
				m.favCursor = 0
				m.modal = modalFavorites
			}
		}

	case "f":
		// legacy: toggle favorite (kept for compatibility)
		if m.activePanel == panelFolders {
			m.toggleFavoriteFolder()
		}

	case "F":
		// open favorites modal (legacy binding)
		if m.activePanel == panelFolders && len(m.favorites) > 0 {
			m.favCursor = 0
			m.modal = modalFavorites
		}

	case "H":
		// return to original snippets directory
		if m.activePanel == panelFolders && m.snippetsDir != m.origSnippetsDir {
			m.snippetsDir = m.origSnippetsDir
			m.folderCursor = 0
			m.fileCursor = 0
			m.inFavSection = false
			m.favCursor = 0
			m.folderDirStack = nil
			m.inParentView = false
			m.parentViewDir = ""
			m.loadFolders()
			m.loadFavorites()
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "Returned to snippets directory"
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		}

	case "s":
		m.activePanel = panelFolders
		m.inFavSection = false

	case "g":
		if !m.gitCfgLoaded {
			m.openGitSetupModal()
		} else {
			m.modal = modalGitSyncing
			return m, doGitSync(m.snippetsDir, m.gitCfg)
		}

	case "G":
		m.openGitConfigModal()

	case "m":
		// move current file to another folder
		if m.activePanel == panelFiles && len(m.files) > 0 {
			m.moveCursor = 0
			m.modal = modalMoveFile
		}

	case "c":
		switch m.activePanel {
		case panelFiles:
			destDir := m.currentFilesDir()
			if destDir != "" {
				m.modal = modalCopyFile
				return m, doCopyFile(destDir)
			}
		case panelPreview:
			if m.previewContent != "" {
				if err := clipboard.WriteAll(m.previewContent); err != nil {
					m.statusMsg = "Failed to copy: " + err.Error()
					m.statusIsSuccess = false
				} else {
					m.statusMsg = "✓ Copied to clipboard!"
					m.statusIsSuccess = true
				}
				return m, clearStatusAfter(3 * time.Second)
			}
		}

	case "e":
		// open current preview file in Neovim
		if m.previewFilePath != "" {
			return m, openNeovim(m.previewFilePath)
		}

	case "o":
		switch m.activePanel {
		case panelPreview:
			if m.previewFilePath != "" {
				// open file location in Explorer
				dir := filepath.Dir(m.previewFilePath)
				_ = exec.Command("explorer.exe", filepath.FromSlash(dir)).Start()
			}
		case panelFolders:
			// show current snippets dir info / location switch modal
			m.modal = modalDirInfo
		default:
			m.modal = modalDirInfo
		}

	case "L":
		m.previewLineNumbers = !m.previewLineNumbers

	case "/":
		switch m.activePanel {
		case panelFolders:
			// Open folder search modal
			m.folderSearchModalQuery = ""
			m.folderSearchModalResults = m.searchFoldersRecursive("")
			m.folderSearchModalCursor = 0
			m.modal = modalFolderSearch
		case panelFiles:
			m.searchActive = true
			m.searchQuery = ""
			m.fileCursor = 0
			m.loadPreview()
		case panelPreview:
			if m.previewContent != "" {
				m.previewSearchActive = true
				m.previewSearchQuery = ""
				m.previewSearchHits = nil
				m.previewSearchCursor = 0
			}
		}

	case "r":
		switch m.activePanel {
		case panelFolders:
			// rename selected folder
			if len(m.folders) > 0 {
				name := m.currentFolderName()
				if name != "" {
					m.modal = modalRenameFolder
					m.modalInput.SetValue(name)
					m.modalInput.Focus()
					m.modalInput.Placeholder = "New folder name..."
				}
			}
		case panelFiles:
			if len(m.files) > 0 {
				m.modal = modalRenameFile
				m.modalInput.SetValue(m.currentFileName())
				m.modalInput.Focus()
				m.modalInput.Placeholder = "New file name..."
			} else {
				m.loadFiles()
				m.loadPreview()
				m.statusMsg = ""
			}
		}

	case "R":
		// legacy rename folder binding (kept for compatibility)
		if m.activePanel == panelFolders && len(m.folders) > 0 {
			name := m.currentFolderName()
			if name != "" {
				m.modal = modalRenameFolder
				m.modalInput.SetValue(name)
				m.modalInput.Focus()
				m.modalInput.Placeholder = "New folder name..."
			}
		}

	case "x":
		// delete selected folder with confirmation
		if m.activePanel == panelFolders {
			name := m.currentFolderName()
			if name != "" {
				m.modal = modalDeleteFolder
			}
		}

	case "X":
		// multi-folder delete mode
		if m.activePanel == panelFolders && !m.inParentView && len(m.folders) > 0 {
			if m.multiDeleteMode {
				// already in mode — open confirmation if any selected
				if len(m.multiDeleteSelected) > 0 {
					m.modal = modalMultiDeleteConfirm
				}
			} else {
				m.multiDeleteMode = true
				m.multiDeleteSelected = make(map[int]bool)
				m.statusMsg = "Multi-delete: Space=select  Enter=confirm  Esc=cancel"
				return m, clearStatusAfter(4 * time.Second)
			}
		}

	case " ":
		// select/deselect folder in multi-delete mode
		if m.activePanel == panelFolders && m.multiDeleteMode && !m.inParentView {
			idx := m.folderCursor
			if m.hasRootFiles {
				if idx == 0 {
					break // can't delete ~/ root
				}
				idx-- // map to folders slice
			}
			if idx >= 0 && idx < len(m.folders) {
				if m.multiDeleteSelected[idx] {
					delete(m.multiDeleteSelected, idx)
				} else {
					m.multiDeleteSelected[idx] = true
				}
			}
		}

	case ":":
		// easter egg: open command console
		m.consoleInput = ""
		m.consoleOutput = ""
		m.modal = modalConsole

	case "?":
		m.statusMsg = "Tab/→←: panel | ↑↓: nav | Enter: open | n: new folder | N: new subfolder | r: rename | x: delete | d: fav | D: favs | /: search | q: quit"
	}

	return m, nil
}

// toggleFavoriteFolder toggles favorite for the currently selected folder and shows status.
func (m *model) toggleFavoriteFolder() {
	name := m.currentFolderName()
	if name == "" {
		return
	}
	m.toggleFavorite(name)
	m.statusMsg = "★ " + name
	if m.isFavorite(name) {
		m.statusMsg += " added to favorites"
	} else {
		m.statusMsg += " removed from favorites"
	}
	m.statusIsSuccess = true
}

func doCopyFile(destDir string) tea.Cmd {
	return func() tea.Msg {
		paths, err := openFilePicker()
		if err != nil {
			return fileCopyResultMsg{err: err}
		}
		if len(paths) == 0 {
			return fileCopyResultMsg{copied: 0}
		}
		for _, p := range paths {
			if err := copyFileToDir(p, destDir); err != nil {
				return fileCopyResultMsg{err: err}
			}
		}
		return fileCopyResultMsg{copied: len(paths)}
	}
}

// moveDestinations returns the list of folders excluding the current one.
func (m model) moveDestinations() []string {
	current := m.currentFolderName()
	var dest []string
	for _, f := range m.folders {
		if f != current {
			dest = append(dest, f)
		}
	}
	return dest
}

// moveDestEntry is a single destination option in the hierarchical move-file list.
type moveDestEntry struct {
	label   string // display label (relative path from snippetsDir)
	absPath string // absolute destination directory
	depth   int    // indentation depth
}

// moveDestinationsAll returns a flat list of all folders and subfolders under
// snippetsDir (recursively), excluding the directory that contains the current file.
func (m model) moveDestinationsAll() []moveDestEntry {
	currentFilePath := m.currentFilePath()
	currentDir := ""
	if currentFilePath != "" {
		currentDir = filepath.Dir(currentFilePath)
	}
	var entries []moveDestEntry
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		children, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range children {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			childAbs := filepath.Join(dir, e.Name())
			rel, err := filepath.Rel(m.snippetsDir, childAbs)
			if err != nil {
				rel = e.Name()
			}
			if childAbs != currentDir {
				entries = append(entries, moveDestEntry{
					label:   rel,
					absPath: childAbs,
					depth:   depth,
				})
			}
			walk(childAbs, depth+1)
		}
	}
	walk(m.snippetsDir, 0)
	return entries
}

func doMoveFile(srcPath, destDir, destFolder string) tea.Cmd {
	return func() tea.Msg {
		name := filepath.Base(srcPath)
		dest := filepath.Join(destDir, name)
		if err := os.Rename(srcPath, dest); err != nil {
			// Rename may fail across drives — fallback to copy+delete
			if err2 := copyFileToDir(srcPath, destDir); err2 != nil {
				return moveFileResultMsg{err: fmt.Errorf("move failed: %v", err2)}
			}
			if err2 := os.Remove(srcPath); err2 != nil {
				return moveFileResultMsg{err: fmt.Errorf("cleanup failed: %v", err2)}
			}
		}
		return moveFileResultMsg{destFolder: destFolder}
	}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func doGitSync(snippetsDir string, cfg GitConfig) tea.Cmd {
	return func() tea.Msg {
		out, err := gitSync(snippetsDir, cfg)
		return gitSyncResultMsg{err: err, output: out}
	}
}

func (m *model) openModal(kind modalKind) {
	m.modal = kind
	m.modalStep = 0
	m.modalInput.SetValue("")
	m.modalInput2.SetValue("")
	m.modalInput.Focus()
	m.modalInput.Placeholder = ""
	switch kind {
	case modalNewFolder:
		m.modalInput.Placeholder = "Folder name..."
	case modalNewFile:
		m.modalInput.Placeholder = "File name (without extension)..."
		m.modalInput2.Placeholder = "Extension (e.g. go, py, md)..."
	case modalNewSubfolder:
		m.modalInput.Placeholder = "Subfolder name..."
	}
}

func (m *model) openGitSetupModal() {
	m.modal = modalGitSetup
	m.modalStep = 0
	m.modalInput.SetValue("")
	m.modalInput2.SetValue("")
	m.modalInput3.SetValue("")
	m.modalInput.Placeholder = "https://github.com/user/snippets.git"
	m.modalInput2.Placeholder = "GitHub username"
	m.modalInput3.Placeholder = "email@example.com"
	m.modalInput.Focus()
	m.modalInput2.Blur()
	m.modalInput3.Blur()
}

func (m *model) openGitConfigModal() {
	m.modal = modalGitConfig
	m.modalStep = 0
	m.modalInput.SetValue(m.gitCfg.RepoURL)
	m.modalInput2.SetValue(m.gitCfg.Username)
	m.modalInput3.SetValue(m.gitCfg.Email)
	m.modalInput.Placeholder = "https://github.com/user/snippets.git"
	m.modalInput2.Placeholder = "GitHub username"
	m.modalInput3.Placeholder = "email@example.com"
	m.modalInput.Focus()
	m.modalInput2.Blur()
	m.modalInput3.Blur()
}

func (m model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// ctrl+c always quits regardless of which modal is open
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.modal {
	case modalError:
		switch msg.String() {
		case "enter", "esc", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalGitSuccess:
		switch msg.String() {
		case "enter", "esc", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalEditorReady:
		switch msg.String() {
		case "enter", "esc", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalGitSyncing:
		// waiting for async result, ignore keys
		return m, nil

	case modalCopyFile:
		// waiting for file picker result, ignore keys
		return m, nil

	case modalDeleteConfirm:
		switch msg.String() {
		case "enter", "y":
			path := m.currentFilePath()
			if path == "" {
				m.modal = modalNone
				return m, nil
			}
			if err := os.Remove(path); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not delete file: %v", err)
				return m, nil
			}
			m.modal = modalNone
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "File deleted."
			return m, clearStatusAfter(3 * time.Second)
		case "esc", "n", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalMoveFile:
		// build hierarchical list of all folders/subfolders
		destinations := m.moveDestinationsAll()
		// +1 for the "Browse external..." entry at the end
		totalItems := len(destinations) + 1
		switch msg.String() {
		case "up", "k":
			if m.moveCursor > 0 {
				m.moveCursor--
			}
		case "down", "j":
			if m.moveCursor < totalItems-1 {
				m.moveCursor++
			}
		case "enter":
			if m.moveCursor == len(destinations) {
				// Browse external folder
				m.dirBrowser = newDirBrowser(m.snippetsDir)
				m.modal = modalMoveFileBrowse
				return m, nil
			}
			if len(destinations) == 0 {
				m.modal = modalNone
				return m, nil
			}
			dest := destinations[m.moveCursor]
			srcPath := m.currentFilePath()
			return m, doMoveFile(srcPath, dest.absPath, dest.label)
		case "esc", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalMoveFileBrowse:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalMoveFile
		case "up", "k":
			m.dirBrowser.moveUp()
		case "down", "j":
			m.dirBrowser.moveDown()
		case "left", "backspace":
			m.dirBrowser.goUp()
		case "right":
			m.dirBrowser.enter()
		case "enter":
			picked := m.dirBrowser.cwd
			if len(m.dirBrowser.entries) > 0 {
				picked = m.dirBrowser.selectedPath()
			}
			if err := os.MkdirAll(picked, 0755); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Cannot use directory: %v", err)
				return m, nil
			}
			srcPath := m.currentFilePath()
			folderLabel := filepath.Base(picked)
			return m, doMoveFile(srcPath, picked, folderLabel)
		}
		return m, nil

	case modalDirInfo:
		switch msg.String() {
		case "enter":
			// open explorer at snippets dir
			_ = exec.Command("explorer.exe", filepath.FromSlash(m.snippetsDir)).Start()
		case "s":
			m.dirBrowser = newDirBrowser(m.snippetsDir)
			m.modal = modalDirBrowser
		case "esc", "q", "o":
			m.modal = modalNone
		}
		return m, nil

	case modalDirBrowser:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalNone
		case "up", "k":
			m.dirBrowser.moveUp()
		case "down", "j":
			m.dirBrowser.moveDown()
		case "left", "backspace":
			m.dirBrowser.goUp()
		case "right":
			m.dirBrowser.enter()
		case "enter":
			picked := m.dirBrowser.cwd
			if len(m.dirBrowser.entries) > 0 {
				picked = m.dirBrowser.selectedPath()
			}
			m.modal = modalNone
			if err := os.MkdirAll(picked, 0755); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Cannot use directory: %v", err)
				return m, nil
			}
			if m.origSnippetsDir == "" {
				m.origSnippetsDir = m.snippetsDir
			}
			m.snippetsDir = picked
			m.folderCursor = 0
			m.fileCursor = 0
			m.inFavSection = false
			m.favCursor = 0
			m.loadFolders()
			m.loadFavorites()
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "Snippets directory changed to: " + picked
			return m, clearStatusAfter(5 * time.Second)
		}
		return m, nil

	case modalNewSubfolder:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "enter":
			name := strings.TrimSpace(m.modalInput.Value())
			if name == "" {
				return m, nil
			}
			parentFolder := m.currentFolderName()
			parentDir := filepath.Join(m.snippetsDir, parentFolder)
			dir := filepath.Join(parentDir, name)
			if err := os.MkdirAll(dir, 0755); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not create subfolder: %v", err)
				return m, nil
			}
			m.modal = modalNone
			m.statusMsg = "Subfolder '" + name + "' created inside " + parentFolder
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		default:
			m.modalInput, cmd = m.modalInput.Update(msg)
		}
		return m, cmd

	case modalConsole:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalNone
			m.consoleInput = ""
			m.consoleOutput = ""
		case "backspace", "ctrl+h":
			if len(m.consoleInput) > 0 {
				m.consoleInput = m.consoleInput[:len(m.consoleInput)-1]
			}
		case "enter":
			consoleCmd := strings.TrimSpace(m.consoleInput)
			m.consoleInput = ""
			switch strings.ToLower(consoleCmd) {
			case "time":
				m.modal = modalTimeCalc
				m.timeInput = ""
				m.timeResult = ""
				m.modalInput.SetValue("")
				m.modalInput.Focus()
				m.modalInput.Placeholder = "HH:MM (e.g. 08:00)"
			case "whoami":
				m.modal = modalWhoami
			case "help":
				m.modal = modalHelpConsole
			case "nvim":
				m.modal = modalNvimGuide
			case "clear":
				m.consoleOutput = ""
				m.modal = modalConsole
			case "exit", "quit":
				m.modal = modalNone
			default:
				if consoleCmd != "" {
					m.consoleOutput = "Unknown command: '" + consoleCmd + "'\nType 'help' for available commands."
				}
			}
		default:
			r := msg.String()
			if len(r) == 1 && r[0] >= 0x20 {
				m.consoleInput += r
			}
		}
		return m, nil

	case modalTimeCalc:
		switch msg.String() {
		case "esc":
			m.modal = modalConsole
			m.timeInput = ""
			m.timeResult = ""
		case "enter":
			if m.timeResult != "" {
				// already computed — go back to console
				m.modal = modalConsole
				m.timeInput = ""
				m.timeResult = ""
			} else {
				val := strings.TrimSpace(m.modalInput.Value())
				if val == "" {
					return m, nil
				}
				m.timeInput = val
				m.timeResult = calcWorkHours(val)
			}
		default:
			if m.timeResult == "" {
				m.modalInput, cmd = m.modalInput.Update(msg)
			}
		}
		return m, cmd

	case modalWhoami:
		switch msg.String() {
		case "esc", "q", "enter":
			m.modal = modalConsole
		}
		return m, nil

	case modalHelpConsole:
		switch msg.String() {
		case "esc", "q", "enter":
			m.modal = modalConsole
		}
		return m, nil

	case modalNvimGuide:
		switch msg.String() {
		case "esc", "q", "enter":
			m.modal = modalConsole
		}
		return m, nil

	case modalGitSetup, modalGitConfig:
		return m.handleGitModalKey(msg)

	case modalNewFolder:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "enter":
			name := m.modalInput.Value()
			if name == "" {
				return m, nil
			}
			dir := filepath.Join(m.snippetsDir, name)
			if err := os.MkdirAll(dir, 0755); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not create folder: %v", err)
				return m, nil
			}
			m.modal = modalNone
			m.loadFolders()
			// Select newly created folder
			for i, f := range m.folders {
				if f == name {
					m.folderCursor = i
					break
				}
			}
			m.loadFiles()
			m.loadPreview()
		default:
			m.modalInput, cmd = m.modalInput.Update(msg)
		}

	case modalNewFile:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "tab":
			if m.modalStep == 0 {
				m.modalStep = 1
				m.modalInput.Blur()
				m.modalInput2.Focus()
			} else {
				m.modalStep = 0
				m.modalInput2.Blur()
				m.modalInput.Focus()
			}
		case "enter":
			if m.modalStep == 0 {
				m.modalStep = 1
				m.modalInput.Blur()
				m.modalInput2.Focus()
			} else {
				name := m.modalInput.Value()
				ext := sanitizeExtension(m.modalInput2.Value())
				if name == "" {
					return m, nil
				}
				filename := name
				if ext != "" {
					filename = name + "." + ext
				}
				destDir := m.currentFilesDir()
				if destDir == "" {
					m.modal = modalError
					m.modalError = "No folder selected. Create a folder first."
					return m, nil
				}
				path := filepath.Join(destDir, filename)
				f, err := os.Create(path)
				if err != nil {
					m.modal = modalError
					m.modalError = fmt.Sprintf("Could not create file: %v", err)
					return m, nil
				}
				f.Close()
				m.modal = modalNone
				m.loadFiles()
				// Select the new file
				for i, fi := range m.files {
					if fi.name == filename {
						m.fileCursor = i
						break
					}
				}
				m.loadPreview()
				// Open nvim immediately
				return m, openNeovim(path)
			}
		default:
			if m.modalStep == 0 {
				m.modalInput, cmd = m.modalInput.Update(msg)
			} else {
				m.modalInput2, cmd = m.modalInput2.Update(msg)
			}
		}

	case modalNewFileName:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "enter":
			m.modal = modalNone
		default:
			m.modalInput, cmd = m.modalInput.Update(msg)
		}

	case modalRenameFolder:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "enter":
			newName := strings.TrimSpace(m.modalInput.Value())
			oldName := m.currentFolderName()
			if newName == "" || newName == oldName {
				m.modal = modalNone
				return m, nil
			}
			// Validation: name cannot contain spaces (use _ or - instead)
			if strings.Contains(newName, " ") {
				m.modal = modalError
				m.modalError = "Folder name cannot contain spaces.\nUse underscore (user_name) or hyphen (user-name) instead."
				return m, nil
			}
			oldPath := filepath.Join(m.snippetsDir, oldName)
			newPath := filepath.Join(m.snippetsDir, newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not rename folder: %v", err)
				return m, nil
			}
			// update favorites if needed (stored as abs paths) — also update subpaths
			for i, f := range m.favorites {
				if f == oldPath {
					m.favorites[i] = newPath
					m.saveFavorites()
				} else if strings.HasPrefix(f, oldPath+string(filepath.Separator)) {
					m.favorites[i] = newPath + f[len(oldPath):]
					m.saveFavorites()
				}
			}
			m.modal = modalNone
			m.loadFolders()
			for i, f := range m.folders {
				if f == newName {
					m.folderCursor = i
					break
				}
			}
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "Folder renamed to " + newName
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		default:
			var cmd tea.Cmd
			m.modalInput, cmd = m.modalInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case modalDeleteFolder:
		switch msg.String() {
		case "enter", "y":
			name := m.currentFolderName()
			if name == "" {
				m.modal = modalNone
				return m, nil
			}
			path := filepath.Join(m.snippetsDir, name)
			if err := os.RemoveAll(path); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not delete folder: %v", err)
				return m, nil
			}
			// remove from favorites (stored as abs paths)
			for i, f := range m.favorites {
				if f == path {
					m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
					m.saveFavorites()
					break
				}
			}
			m.modal = modalNone
			m.loadFolders()
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "Folder \"" + name + "\" deleted."
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		case "esc", "n", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalRenameFile:
		switch msg.String() {
		case "esc":
			m.modal = modalNone
		case "enter":
			newName := strings.TrimSpace(m.modalInput.Value())
			oldName := m.currentFileName()
			if newName == "" || newName == oldName {
				m.modal = modalNone
				return m, nil
			}
			fileDir := m.currentFilesDir()
			if fileDir == "" {
				m.modal = modalNone
				return m, nil
			}
			oldPath := filepath.Join(fileDir, oldName)
			newPath := filepath.Join(fileDir, newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not rename file: %v", err)
				return m, nil
			}
			m.modal = modalNone
			m.loadFiles()
			for i, f := range m.files {
				if f.name == newName {
					m.fileCursor = i
					break
				}
			}
			m.loadPreview()
			m.statusMsg = "Snippet renamed to " + newName
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		default:
			var cmd tea.Cmd
			m.modalInput, cmd = m.modalInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case modalSubfolderNav:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalNone
		case "up", "k":
			if m.subNavCursor > 0 {
				m.subNavCursor--
			}
		case "down", "j":
			if m.subNavCursor < len(m.subNavEntries)-1 {
				m.subNavCursor++
			}
		case "backspace":
			if len(m.subNavStack) > 1 {
				// pop one level
				m.subNavStack = m.subNavStack[:len(m.subNavStack)-1]
				m.subNavCursor = 0
				m.loadSubNavEntries()
			} else {
				// already at root folder — close modal
				m.modal = modalNone
			}
		case "enter":
			if m.subNavCursor >= len(m.subNavEntries) {
				break
			}
			e := m.subNavEntries[m.subNavCursor]
			if e.isDir {
				// descend into this subfolder as the new Folders panel root
				rel := strings.Join(m.subNavStack, string(filepath.Separator))
				newRoot := filepath.Join(m.snippetsDir, rel, e.name)
				m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
				m.snippetsDir = newRoot
				m.modal = modalNone
				m.folderCursor = 0
				m.fileCursor = 0
				m.loadFolders()
				m.loadFiles()
				m.loadPreview()
				m.statusMsg = "Navigated into: " + e.name + "  (← to go back)"
				m.statusIsSuccess = true
				return m, clearStatusAfter(4 * time.Second)
			} else {
				// load file into preview and close modal
				rel := strings.Join(m.subNavStack, string(filepath.Separator))
				path := filepath.Join(m.snippetsDir, rel, e.name)
				m.modal = modalNone
				m.activePanel = panelPreview
				m.loadPreviewFromPath(path)
			}
		}
		return m, nil

	case modalFavorites:
		switch msg.String() {
		case "esc", "q", "F":
			m.modal = modalNone
		case "up", "k":
			if m.favCursor > 0 {
				m.favCursor--
			}
		case "down", "j":
			if m.favCursor < len(m.favorites)-1 {
				m.favCursor++
			}
		case "enter":
			if m.favCursor < len(m.favorites) {
				absPath := m.favorites[m.favCursor]
				// The favorite is an abs path to a folder.
				// Navigate to its parent dir, then select the folder by name.
				parentDir := filepath.Dir(absPath)
				folderName := filepath.Base(absPath)
				// Push dir stack and switch to parent
				if parentDir != m.snippetsDir {
					m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
					m.snippetsDir = parentDir
					m.loadFolders()
				}
				// find folder index
				for i, f := range m.folders {
					if f == folderName {
						m.folderCursor = i
						m.folderScroll = clampScroll(i, m.folderScroll, 10)
						break
					}
				}
				m.inFavSection = false
				m.fileCursor = 0
				m.fileScroll = 0
				m.loadFiles()
				m.loadPreview()
			}
			m.modal = modalNone
		case "f":
			// unfavorite from modal — favorites store abs paths, remove directly
			if m.favCursor < len(m.favorites) {
				absPath := m.favorites[m.favCursor]
				for i, f := range m.favorites {
					if f == absPath {
						m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
						m.saveFavorites()
						break
					}
				}
				if m.favCursor >= len(m.favorites) {
					m.favCursor = max(0, len(m.favorites)-1)
				}
				if len(m.favorites) == 0 {
					m.modal = modalNone
				}
			}
		case "o":
			// open selected favorite in Windows Explorer
			if m.favCursor < len(m.favorites) {
				absPath := m.favorites[m.favCursor]
				_ = exec.Command("explorer.exe", filepath.FromSlash(absPath)).Start()
			}
		}

	case modalSubfolderSelect:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalNone
		case "up", "k":
			if m.subSelectCursor > 0 {
				m.subSelectCursor--
			}
		case "down", "j":
			if m.subSelectCursor < len(m.subSelectEntries)-1 {
				m.subSelectCursor++
			}
		case "right":
			// descend into the currently selected subfolder
			if m.subSelectCursor < len(m.subSelectEntries) {
				child := m.subSelectEntries[m.subSelectCursor]
				newStack := append(append([]string{}, m.subSelectStack...), child)
				childAbsParts := append([]string{m.snippetsDir}, newStack...)
				childAbs := filepath.Join(childAbsParts...)
				childSubs := m.subfolderNames(childAbs)
				if len(childSubs) > 0 {
					m.subSelectStack = newStack
					m.subSelectCursor = 0
					m.loadSubSelectEntries()
				}
			}
		case "left":
			// go back one level (up to root folder level)
			if len(m.subSelectStack) > 1 {
				m.subSelectStack = m.subSelectStack[:len(m.subSelectStack)-1]
				m.subSelectCursor = 0
				m.loadSubSelectEntries()
			} else {
				m.modal = modalNone
			}
		case "enter":
			if m.subSelectCursor >= len(m.subSelectEntries) {
				break
			}
			child := m.subSelectEntries[m.subSelectCursor]
			selectedStack := append(append([]string{}, m.subSelectStack...), child)

			// Build absolute paths
			buildAbs := func(parts []string) string {
				allParts := append([]string{m.snippetsDir}, parts...)
				return filepath.Join(allParts...)
			}
			selectedAbs := buildAbs(selectedStack)
			parentParts := selectedStack[:len(selectedStack)-1]
			parentAbs := buildAbs(parentParts)
			_ = selectedAbs

			// Navigate the folder panel: show parentAbs in parent-view mode,
			// with cursor pointing at the selected child.
			// Per docs: folder panel shows ~/ (parent files) + all subfolders of parent;
			// the selected child is highlighted.
			m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
			m.snippetsDir = parentAbs
			m.inParentView = true
			m.parentViewDir = parentAbs

			// Find the selected child in subfolders of parentAbs
			childSubs := m.subfolderNames(parentAbs)
			m.folderCursor = 0
			for idx, s := range childSubs {
				if s == child {
					m.folderCursor = idx + 1 // +1 for the ~/ entry
					break
				}
			}
			m.fileCursor = 0
			m.modal = modalNone
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = "Navigated into: " + strings.Join(selectedStack, "/") + "  (← to go back)"
			m.statusIsSuccess = true
			return m, clearStatusAfter(4 * time.Second)
		}

	case modalMultiDeleteConfirm:
		switch msg.String() {
		case "esc", "n", "q":
			m.modal = modalNone
		case "enter", "y":
			// delete all selected folders
			deleted := 0
			var errs []string
			for idx := range m.multiDeleteSelected {
				if idx < len(m.folders) {
					name := m.folders[idx]
					path := filepath.Join(m.snippetsDir, name)
					// remove from favorites
					absPath := path
					for i, f := range m.favorites {
						if f == absPath {
							m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
							m.saveFavorites()
							break
						}
					}
					if err := os.RemoveAll(path); err != nil {
						errs = append(errs, name+": "+err.Error())
					} else {
						deleted++
					}
				}
			}
			m.multiDeleteMode = false
			m.multiDeleteSelected = nil
			m.modal = modalNone
			m.folderCursor = 0
			m.fileCursor = 0
			m.loadFolders()
			m.loadFiles()
			m.loadPreview()
			if len(errs) > 0 {
				m.modal = modalError
				m.modalError = "Some folders could not be deleted:\n" + strings.Join(errs, "\n")
			} else {
				m.statusMsg = fmt.Sprintf("%d folder(s) deleted.", deleted)
				m.statusIsSuccess = true
				return m, clearStatusAfter(3 * time.Second)
			}
		}

	case modalFolderSearch:
		switch msg.String() {
		case "esc", "q":
			m.modal = modalNone
			m.folderSearchModalQuery = ""
			m.folderSearchModalResults = nil
			m.folderSearchModalCursor = 0
		case "ctrl+c":
			return m, tea.Quit
		case "backspace", "ctrl+h":
			if len(m.folderSearchModalQuery) > 0 {
				m.folderSearchModalQuery = m.folderSearchModalQuery[:len(m.folderSearchModalQuery)-1]
				m.folderSearchModalResults = m.searchFoldersRecursive(m.folderSearchModalQuery)
				m.folderSearchModalCursor = 0
			}
		case "up", "k":
			if m.folderSearchModalCursor > 0 {
				m.folderSearchModalCursor--
			}
		case "down", "j":
			if m.folderSearchModalCursor < len(m.folderSearchModalResults)-1 {
				m.folderSearchModalCursor++
			}
		case "enter":
			if m.folderSearchModalCursor < len(m.folderSearchModalResults) {
				result := m.folderSearchModalResults[m.folderSearchModalCursor]
				// Navigate to selected folder
				parentDir := filepath.Dir(result.absPath)
				folderName := filepath.Base(result.absPath)
				m.modal = modalNone
				m.folderSearchModalQuery = ""
				m.folderSearchModalResults = nil
				m.folderSearchModalCursor = 0
				// Push stack and switch
				if parentDir != m.snippetsDir {
					m.folderDirStack = append(m.folderDirStack, m.snippetsDir)
					m.snippetsDir = parentDir
				}
				m.inParentView = false
				m.parentViewDir = ""
				m.loadFolders()
				for i, f := range m.folders {
					if f == folderName {
						m.folderCursor = i
						m.folderScroll = clampScroll(i, m.folderScroll, 10)
						break
					}
				}
				m.fileCursor = 0
				m.fileScroll = 0
				m.loadFiles()
				m.loadPreview()
			}
		default:
			r := msg.String()
			if len(r) == 1 && r[0] >= 0x20 {
				m.folderSearchModalQuery += r
				m.folderSearchModalResults = m.searchFoldersRecursive(m.folderSearchModalQuery)
				m.folderSearchModalCursor = 0
			}
		}
	}

	return m, cmd
}

func (m model) handleGitModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	focusField := func(step int) {
		m.modalInput.Blur()
		m.modalInput2.Blur()
		m.modalInput3.Blur()
		switch step {
		case 0:
			m.modalInput.Focus()
		case 1:
			m.modalInput2.Focus()
		case 2:
			m.modalInput3.Focus()
		}
	}

	switch msg.String() {
	case "esc":
		m.modal = modalNone
		return m, nil
	case "tab":
		m.modalStep = (m.modalStep + 1) % 3
		focusField(m.modalStep)
		return m, nil
	case "shift+tab":
		m.modalStep = (m.modalStep + 2) % 3
		focusField(m.modalStep)
		return m, nil
	case "enter":
		if m.modalStep < 2 {
			m.modalStep++
			focusField(m.modalStep)
			return m, nil
		}
		// Final confirm
		repoURL := m.modalInput.Value()
		username := m.modalInput2.Value()
		email := m.modalInput3.Value()
		if repoURL == "" || username == "" || email == "" {
			m.modal = modalError
			m.modalError = "All fields are required.\nPlease fill in repo URL, username and email."
			return m, nil
		}
		cfg := GitConfig{RepoURL: repoURL, Username: username, Email: email}
		if err := saveGitConfig(m.snippetsDir, cfg); err != nil {
			m.modal = modalError
			m.modalError = fmt.Sprintf("Could not save git config: %v", err)
			return m, nil
		}
		m.gitCfg = cfg
		m.gitCfgLoaded = true
		m.modal = modalGitSyncing
		return m, doGitSync(m.snippetsDir, cfg)
	}

	switch m.modalStep {
	case 0:
		m.modalInput, cmd = m.modalInput.Update(msg)
	case 1:
		m.modalInput2, cmd = m.modalInput2.Update(msg)
	case 2:
		m.modalInput3, cmd = m.modalInput3.Update(msg)
	}

	return m, cmd
}

// handleFolderSearchKey handles keypresses while folder search is active.
func (m model) handleFolderSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.folderSearchActive = false
		m.folderSearchQuery = ""
	case "ctrl+c":
		return m, tea.Quit
	case "backspace", "ctrl+h":
		if len(m.folderSearchQuery) > 0 {
			m.folderSearchQuery = m.folderSearchQuery[:len(m.folderSearchQuery)-1]
		}
		m.folderCursor = 0
	case "up", "k":
		if m.folderCursor > 0 {
			m.folderCursor--
			m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
		}
	case "down", "j":
		list := m.filteredFolders()
		if m.folderCursor < len(list)-1 {
			m.folderCursor++
			m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
		}
	case "enter":
		// confirm selection — map filtered index back to real folders index
		list := m.filteredFolders()
		if m.folderCursor < len(list) {
			selected := list[m.folderCursor]
			for i, f := range m.folders {
				if f == selected {
					m.folderCursor = i
					m.folderScroll = clampScroll(i, m.folderScroll, 10)
					break
				}
			}
		}
		m.folderSearchActive = false
		m.folderSearchQuery = ""
		m.loadFiles()
		m.loadPreview()
	default:
		r := msg.String()
		if len(r) == 1 && r[0] >= 0x20 {
			m.folderSearchQuery += r
			m.folderCursor = 0
		}
	}
	return m, nil
}

// handleSearchKey handles keypresses while inline search is active in the files panel.
func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// restore real fileCursor to the currently previewed file before exiting
		if f, ok := m.resolvedFile(); ok {
			for i, rf := range m.files {
				if rf.name == f.name {
					m.fileCursor = i
					break
				}
			}
		}
		m.searchActive = false
		m.searchQuery = ""
		m.loadPreview()

	case "ctrl+c":
		return m, tea.Quit

	case "backspace", "ctrl+h":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		m.fileCursor = 0
		m.fileScroll = 0
		m.loadPreview()

	case "up", "k":
		if m.fileCursor > 0 {
			m.fileCursor--
			m.fileScroll = clampScroll(m.fileCursor, m.fileScroll, 10)
			m.loadPreview()
		}

	case "down", "j":
		filtered := m.filteredFiles()
		if m.fileCursor < len(filtered)-1 {
			m.fileCursor++
			m.fileScroll = clampScroll(m.fileCursor, m.fileScroll, 10)
			m.loadPreview()
		}

	case "enter":
		// map fileCursor back to real index in m.files, exit search
		filtered := m.filteredFiles()
		if len(filtered) > 0 && m.fileCursor < len(filtered) {
			selected := filtered[m.fileCursor]
			for i, f := range m.files {
				if f.name == selected.name {
					m.fileCursor = i
					break
				}
			}
		}
		m.searchActive = false
		m.searchQuery = ""
		m.loadPreview()

	default:
		// append printable characters to query; reset cursor to top of results
		r := msg.String()
		if len(r) == 1 && r[0] >= 0x20 {
			m.searchQuery += r
			m.fileCursor = 0
			m.fileScroll = 0
			m.loadPreview()
		}
	}
	return m, nil
}

// handlePreviewSearchKey handles keypresses while preview word-search is active.
func (m model) handlePreviewSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.previewSearchActive = false
		m.previewSearchQuery = ""
		m.previewSearchHits = nil
		m.previewSearchCursor = 0

	case "backspace", "ctrl+h":
		if len(m.previewSearchQuery) > 0 {
			m.previewSearchQuery = m.previewSearchQuery[:len(m.previewSearchQuery)-1]
		}
		m.previewSearchHits = nil
		m.previewSearchCursor = 0

	case "enter":
		// compute hits and jump to first one
		hits := computePreviewSearchHits(m.previewContent, m.previewSearchQuery)
		m.previewSearchHits = hits
		m.previewSearchCursor = 0
		if len(hits) > 0 {
			m.previewScroll = max(0, hits[0]-3)
		}

	default:
		r := msg.String()
		if len(r) == 1 && r[0] >= 0x20 {
			if r == "n" && len(m.previewSearchHits) > 0 {
				// next hit (only when hits already computed)
				m.previewSearchCursor = (m.previewSearchCursor + 1) % len(m.previewSearchHits)
				m.previewScroll = max(0, m.previewSearchHits[m.previewSearchCursor]-3)
			} else if r == "N" && len(m.previewSearchHits) > 0 {
				// previous hit (only when hits already computed)
				m.previewSearchCursor = (m.previewSearchCursor - 1 + len(m.previewSearchHits)) % len(m.previewSearchHits)
				m.previewScroll = max(0, m.previewSearchHits[m.previewSearchCursor]-3)
			} else {
				m.previewSearchQuery += r
				m.previewSearchHits = nil
				m.previewSearchCursor = 0
			}
		}
	}
	return m, nil
}

// Needed for textinput
var _ = textinput.New
