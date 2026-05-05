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

type dirPickerResultMsg struct {
	dir string
	err error
}

type moveFileResultMsg struct {
	destFolder string
	err        error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// ~35% of width for preview panel inner content, minimum 40
		pw := (msg.Width * 35 / 100) - 4
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
			m.modal = modalError
			m.modalError = "Neovim (nvim) not found in PATH.\nPlease install and configure Neovim to edit files."
			return m, nil
		}
		m.modal = modalEditorReady
		m.editorPath = msg.path
		return m, nil

	case launchEditorMsg:
		// Try to open in a new Windows Terminal window
		wt, wtErr := exec.LookPath("wt")
		if wtErr == nil {
			// wt opens a new window; pwsh stays open after nvim exits
			pwsh := "pwsh"
			if _, err := exec.LookPath("pwsh"); err != nil {
				pwsh = "powershell"
			}
			safePath := strings.ReplaceAll(msg.path, `"`, `\"`)
			wtArgs := []string{
				"--window", "new",
				"new-tab",
				"--title", "clidocs editor",
				pwsh,
				"-NoLogo", "-NoExit",
				"-Command", fmt.Sprintf(`nvim "%s"`, safePath),
			}
			c := exec.Command(wt, wtArgs...)
			if err := c.Start(); err == nil {
				// WT launched detached — immediately return to TUI
				// Reload preview after a short moment when user comes back
				m.statusMsg = "Editing in new Windows Terminal window — reload with r"
				return m, nil
			}
		}
		// Fallback: take over the current terminal (old behaviour)
		pwsh := "pwsh"
		if _, err := exec.LookPath("pwsh"); err != nil {
			pwsh = "powershell"
		}
		safePath := strings.ReplaceAll(msg.path, `"`, `\"`)
		args := []string{"-NoLogo", "-NoExit", "-Command", fmt.Sprintf(`nvim "%s"`, safePath)}
		c := exec.Command(pwsh, args...)
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

	case dirPickerResultMsg:
		m.modal = modalNone
		if msg.err != nil {
			m.modal = modalError
			m.modalError = msg.err.Error()
			return m, nil
		}
		if msg.dir == "" {
			m.statusMsg = "No directory selected."
			return m, clearStatusAfter(3 * time.Second)
		}
		m.snippetsDir = msg.dir
		m.folderCursor = 0
		m.fileCursor = 0
		m.inFavSection = false
		m.favCursor = 0
		if err := os.MkdirAll(msg.dir, 0755); err != nil {
			m.modal = modalError
			m.modalError = fmt.Sprintf("Cannot use directory: %v", err)
			return m, nil
		}
		m.loadFolders()
		m.loadFavorites()
		m.loadFiles()
		m.loadPreview()
		m.statusMsg = "Snippets directory changed to: " + msg.dir
		return m, clearStatusAfter(5 * time.Second)

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
			// go back to parent dir if we navigated into a subfolder
			if len(m.folderDirStack) > 0 {
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
			if m.folderCursor > 0 {
				m.folderCursor--
				m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
				m.fileCursor = 0
				m.fileScroll = 0
				m.loadFiles()
				m.loadPreview()
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
			if m.folderCursor < len(m.folders)-1 {
				m.folderCursor++
				m.folderScroll = clampScroll(m.folderCursor, m.folderScroll, 10)
				m.fileCursor = 0
				m.fileScroll = 0
				m.loadFiles()
				m.loadPreview()
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
			folderName := m.currentFolderName()
			if folderName != "" {
				if m.hasSubfolders(folderName) {
					m.subNavStack = []string{folderName}
					m.subNavCursor = 0
					m.loadSubNavEntries()
					m.modal = modalSubfolderNav
				} else {
					m.activePanel = panelFiles
					m.fileCursor = 0
					m.loadFiles()
					m.loadPreview()
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
		if m.activePanel == panelFolders && len(m.folders) > 0 {
			m.openModal(modalNewSubfolder)
		}

	case "f":
		// toggle favorite on the currently selected folder
		if m.activePanel == panelFolders {
			name := m.currentFolderName()
			if name != "" {
				m.toggleFavorite(name)
				if m.inFavSection && !m.isFavorite(name) {
					// cursor was on a just-removed favorite — reset to main section
					m.inFavSection = false
					if m.favCursor >= len(m.favorites) {
						m.favCursor = max(0, len(m.favorites)-1)
					}
				}
				m.statusMsg = "★ " + name
				if m.isFavorite(name) {
					m.statusMsg += " added to favorites"
				} else {
					m.statusMsg += " removed from favorites"
				}
				m.statusIsSuccess = true
				return m, clearStatusAfter(2 * time.Second)
			}
		}

	case "F":
		// open favorites modal
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

	case "d":
		if m.activePanel == panelFiles && len(m.files) > 0 {
			m.modal = modalDeleteConfirm
		}

	case "m":
		// move current file to another folder
		if m.activePanel == panelFiles && len(m.files) > 0 && len(m.folders) > 1 {
			m.moveCursor = 0
			// skip current folder in selection — handled in modal render
			m.modal = modalMoveFile
		}

	case "c":
		switch m.activePanel {
		case panelFiles:
			if len(m.folders) > 0 {
				destDir := filepath.Join(m.snippetsDir, m.currentFolderName())
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

	case "v":
		// open current preview file in VS Code
		if m.previewFilePath != "" {
			_ = exec.Command("code", m.previewFilePath).Start()
			m.statusMsg = "Opened in VS Code"
			m.statusIsSuccess = true
			return m, clearStatusAfter(3 * time.Second)
		}

	case "o":
		if m.activePanel == panelPreview && m.previewFilePath != "" {
			// open file location in Explorer
			dir := filepath.Dir(m.previewFilePath)
			_ = exec.Command("explorer.exe", filepath.FromSlash(dir)).Start()
		} else {
			// show current snippets dir info (original behaviour on other panels)
			m.modal = modalDirInfo
		}

	case "L":
		m.previewLineNumbers = !m.previewLineNumbers

	case "/":
		switch m.activePanel {
		case panelFolders:
			m.folderSearchActive = true
			m.folderSearchQuery = ""
			m.folderCursor = 0
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

	case "R":
		if m.activePanel == panelFolders && len(m.folders) > 0 {
			m.modal = modalRenameFolder
			m.modalInput.SetValue(m.currentFolderName())
			m.modalInput.Focus()
			m.modalInput.Placeholder = "New folder name..."
		}

	case "D":
		if m.activePanel == panelFolders && len(m.folders) > 0 {
			m.modal = modalDeleteFolder
		}

	case "r":
		if m.activePanel == panelFiles && len(m.files) > 0 {
			m.modal = modalRenameFile
			m.modalInput.SetValue(m.currentFileName())
			m.modalInput.Focus()
			m.modalInput.Placeholder = "New file name..."
		} else {
			m.loadFiles()
			m.loadPreview()
			m.statusMsg = ""
		}

	case ":":
		// easter egg: open command console
		m.consoleInput = ""
		m.consoleOutput = ""
		m.modal = modalConsole

	case "?":
		m.statusMsg = "Tab/→←: panel | ↑↓: nav | Enter: edit | n: new | r: reload | g: sync | G: git config | q: quit"
	}

	return m, nil
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

func doPickDir() tea.Cmd {
	return func() tea.Msg {
		dir, err := openDirPicker()
		return dirPickerResultMsg{dir: dir, err: err}
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
		// build list of folders excluding current
		destinations := m.moveDestinations()
		switch msg.String() {
		case "up", "k":
			if m.moveCursor > 0 {
				m.moveCursor--
			}
		case "down", "j":
			if m.moveCursor < len(destinations)-1 {
				m.moveCursor++
			}
		case "enter":
			if len(destinations) == 0 {
				m.modal = modalNone
				return m, nil
			}
			destFolder := destinations[m.moveCursor]
			srcPath := m.currentFilePath()
			destDir := filepath.Join(m.snippetsDir, destFolder)
			return m, doMoveFile(srcPath, destDir, destFolder)
		case "esc", "q":
			m.modal = modalNone
		}
		return m, nil

	case modalDirInfo:
		switch msg.String() {
		case "enter":
			// open explorer at snippets dir
			_ = exec.Command("explorer.exe", filepath.FromSlash(m.snippetsDir)).Start()
		case "s":
			m.modal = modalChangeDirPicker
			return m, doPickDir()
		case "esc", "q", "o":
			m.modal = modalNone
		}
		return m, nil

	case modalChangeDirPicker:
		// waiting for async dir picker result, ignore keys
		return m, nil

	case modalEditorReady:
		switch msg.String() {
		case "enter":
			m.modal = modalNone
			path := m.editorPath
			return m, func() tea.Msg { return launchEditorMsg{path: path} }
		case "esc", "q":
			m.modal = modalNone
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
			dir := filepath.Join(m.snippetsDir, parentFolder, name)
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
				if len(m.folders) == 0 {
					m.modal = modalError
					m.modalError = "No folder selected. Create a folder first."
					return m, nil
				}
				path := filepath.Join(m.snippetsDir, m.folders[m.folderCursor], filename)
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
			oldPath := filepath.Join(m.snippetsDir, oldName)
			newPath := filepath.Join(m.snippetsDir, newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				m.modal = modalError
				m.modalError = fmt.Sprintf("Could not rename folder: %v", err)
				return m, nil
			}
			// update favorites if needed
			for i, f := range m.favorites {
				if f == oldName {
					m.favorites[i] = newName
					m.saveFavorites()
					break
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
			// remove from favorites
			for i, f := range m.favorites {
				if f == name {
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
			folderName := m.currentFolderName()
			if newName == "" || newName == oldName {
				m.modal = modalNone
				return m, nil
			}
			oldPath := filepath.Join(m.snippetsDir, folderName, oldName)
			newPath := filepath.Join(m.snippetsDir, folderName, newName)
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
				name := m.favorites[m.favCursor]
				// find folder index in main list
				for i, f := range m.folders {
					if f == name {
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
			// unfavorite from modal
			if m.favCursor < len(m.favorites) {
				name := m.favorites[m.favCursor]
				m.toggleFavorite(name)
				if m.favCursor >= len(m.favorites) {
					m.favCursor = max(0, len(m.favorites)-1)
				}
				if len(m.favorites) == 0 {
					m.modal = modalNone
				}
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
