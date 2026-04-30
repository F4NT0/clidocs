package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type panel int

const (
	panelFolders panel = iota
	panelFiles
	panelPreview
)

type modalKind int

const (
	modalNone modalKind = iota
	modalNewFolder
	modalNewFile
	modalNewFileName
	modalError
	modalGitSetup
	modalGitConfig
	modalGitSuccess
	modalGitSyncing
)

type fileEntry struct {
	name    string
	modTime time.Time
}

type model struct {
	snippetsDir string
	width       int
	height      int

	activePanel panel

	folders      []string
	folderCursor int

	files      []fileEntry
	fileCursor int

	previewContent   string
	previewHighlight string
	previewScroll    int

	modal        modalKind
	modalInput   textinput.Model
	modalInput2  textinput.Model
	modalStep    int // for multi-step modals
	modalError   string

	statusMsg string

	gitCfg       GitConfig
	gitCfgLoaded bool
	modalInput3  textinput.Model
}

func newModel(dir string) model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 128
	ti.Width = 40

	ti2 := textinput.New()
	ti2.Placeholder = "go, py, js, md ..."
	ti2.CharLimit = 20
	ti2.Width = 40

	ti3 := textinput.New()
	ti3.CharLimit = 128
	ti3.Width = 40

	gitCfg, gitLoaded := loadGitConfig(dir)

	m := model{
		snippetsDir:  dir,
		modal:        modalNone,
		modalInput:   ti,
		modalInput2:  ti2,
		modalInput3:  ti3,
		activePanel:  panelFolders,
		gitCfg:       gitCfg,
		gitCfgLoaded: gitLoaded,
	}
	m.loadFolders()
	return m
}

func (m *model) loadFolders() {
	entries, err := os.ReadDir(m.snippetsDir)
	m.folders = []string{}
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			m.folders = append(m.folders, e.Name())
		}
	}
	sort.Strings(m.folders)
	if m.folderCursor >= len(m.folders) {
		m.folderCursor = max(0, len(m.folders)-1)
	}
}

func (m *model) loadFiles() {
	m.files = []fileEntry{}
	if len(m.folders) == 0 {
		return
	}
	dir := filepath.Join(m.snippetsDir, m.folders[m.folderCursor])
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			info, _ := e.Info()
			mod := time.Time{}
			if info != nil {
				mod = info.ModTime()
			}
			m.files = append(m.files, fileEntry{name: e.Name(), modTime: mod})
		}
	}
	if m.fileCursor >= len(m.files) {
		m.fileCursor = max(0, len(m.files)-1)
	}
}

func (m *model) loadPreview() {
	m.previewContent = ""
	m.previewHighlight = ""
	m.previewScroll = 0
	if len(m.files) == 0 || len(m.folders) == 0 {
		return
	}
	f := m.files[m.fileCursor]
	path := filepath.Join(m.snippetsDir, m.folders[m.folderCursor], f.name)
	data, err := os.ReadFile(path)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error reading file: %v", err)
		m.previewHighlight = m.previewContent
		return
	}
	m.previewContent = string(data)
	m.previewHighlight = highlightCode(m.previewContent, f.name)
}

func (m model) Init() tea.Cmd {
	return nil
}

type editorDoneMsg struct{}

func openNeovim(path string) tea.Cmd {
	return func() tea.Msg {
		// Check if nvim exists
		return openEditorMsg{path: path}
	}
}

type openEditorMsg struct{ path string }

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m model) currentFolderName() string {
	if len(m.folders) == 0 {
		return ""
	}
	return m.folders[m.folderCursor]
}

func (m model) currentFileName() string {
	if len(m.files) == 0 {
		return ""
	}
	return m.files[m.fileCursor].name
}

func (m model) currentFilePath() string {
	if len(m.folders) == 0 || len(m.files) == 0 {
		return ""
	}
	return filepath.Join(m.snippetsDir, m.folders[m.folderCursor], m.files[m.fileCursor].name)
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	}
}

func sanitizeExtension(ext string) string {
	ext = strings.TrimSpace(ext)
	ext = strings.TrimPrefix(ext, ".")
	return ext
}
