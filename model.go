package main

import (
	"encoding/json"
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
	modalEditorReady
	modalCopyFile
	modalDeleteConfirm
	modalMoveFile
	modalDirInfo
	modalChangeDirPicker
	modalFavorites
	modalRenameFolder
	modalDeleteFolder
	modalRenameFile
	modalSubfolderNav
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
	previewIsImage   bool
	previewFilePath  string // absolute path of previewed file (may differ from currentFilePath when in subNav)

	modal        modalKind
	modalInput   textinput.Model
	modalInput2  textinput.Model
	modalStep    int // for multi-step modals
	modalError   string

	statusMsg       string
	statusIsSuccess bool

	gitCfg       GitConfig
	gitCfgLoaded bool
	modalInput3  textinput.Model

	editorPath string

	// move-file modal
	moveCursor int

	// custom snippets dir (persisted across session)
	newSnippetsDir string

	// inline search in files panel
	searchActive bool
	searchQuery  string

	// preview panel features
	previewLineNumbers  bool
	previewSearchActive bool
	previewSearchQuery  string
	previewSearchHits   []int // line indices (0-based) of matches
	previewSearchCursor int   // which hit is currently focused

	// folder favorites
	favorites       []string // folder names marked as favorite
	origSnippetsDir string   // original snippetsDir before any change-dir
	inFavSection    bool     // cursor is in the favorites section
	favCursor       int      // cursor within favorites section

	// scroll offsets for virtual lists
	folderScroll int // top visible index in folders list
	fileScroll   int // top visible index in files list

	// subfolder navigation modal
	subNavStack   []string // stack of folder path segments relative to snippetsDir
	subNavEntries []subNavEntry
	subNavCursor  int
}

type subNavEntry struct {
	name     string
	isDir    bool
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
		gitCfg:          gitCfg,
		gitCfgLoaded:    gitLoaded,
		origSnippetsDir: dir,
	}
	m.loadFolders()
	m.loadFavorites()
	return m
}

// clampScroll adjusts scroll so that cursor stays within [scroll, scroll+visible).
func clampScroll(cursor, scroll, visible int) int {
	if cursor < scroll {
		return cursor
	}
	if cursor >= scroll+visible {
		return cursor - visible + 1
	}
	return scroll
}

const favoritesFile = ".clidocs_favorites.json"

func (m *model) loadFavorites() {
	path := filepath.Join(m.snippetsDir, favoritesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		m.favorites = nil
		return
	}
	var favs []string
	if err := json.Unmarshal(data, &favs); err != nil {
		m.favorites = nil
		return
	}
	m.favorites = favs
}

func (m *model) saveFavorites() {
	path := filepath.Join(m.snippetsDir, favoritesFile)
	data, _ := json.MarshalIndent(m.favorites, "", "  ")
	_ = os.WriteFile(path, data, 0644)
}

func (m *model) isFavorite(name string) bool {
	for _, f := range m.favorites {
		if f == name {
			return true
		}
	}
	return false
}

func (m *model) toggleFavorite(name string) {
	for i, f := range m.favorites {
		if f == name {
			m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
			m.saveFavorites()
			return
		}
	}
	m.favorites = append(m.favorites, name)
	m.saveFavorites()
}

// currentFolderInContext returns the folder name the cursor is currently on.
func (m model) currentFolderInContext() string {
	if false { // inFavSection removed; favorites navigate via modal then set folderCursor
		if m.favCursor >= 0 && m.favCursor < len(m.favorites) {
			return m.favorites[m.favCursor]
		}
		return ""
	}
	if m.folderCursor < len(m.folders) {
		return m.folders[m.folderCursor]
	}
	return ""
}

func (m *model) loadFolders() {
	entries, err := os.ReadDir(m.snippetsDir)
	m.folders = []string{}
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
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
	folderName := m.currentFolderName()
	if folderName == "" {
		return
	}
	dir := filepath.Join(m.snippetsDir, folderName)
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

// resolvedFile returns the currently selected file entry, respecting the
// active search filter (fileCursor indexes into filteredFiles when search is on).
func (m model) resolvedFile() (fileEntry, bool) {
	list := m.filteredFiles()
	if len(list) == 0 {
		return fileEntry{}, false
	}
	idx := m.fileCursor
	if idx < 0 || idx >= len(list) {
		idx = 0
	}
	return list[idx], true
}

func (m *model) loadPreview() {
	m.previewContent = ""
	m.previewHighlight = ""
	m.previewScroll = 0
	m.previewIsImage = false
	m.previewFilePath = ""
	folderName := m.currentFolderName()
	if folderName == "" {
		return
	}
	f, ok := m.resolvedFile()
	if !ok {
		return
	}
	path := filepath.Join(m.snippetsDir, folderName, f.name)
	m.previewFilePath = path

	if isImageFile(f.name) {
		m.previewIsImage = true
		m.previewHighlight = renderImagePreview(path, m.width/3)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error reading file: %v", err)
		m.previewHighlight = m.previewContent
		return
	}
	if isBinary(data) {
		m.previewContent = ""
		m.previewHighlight = ""
		return
	}
	m.previewContent = string(data)
	m.previewHighlight = highlightCode(m.previewContent, f.name)
}

// loadPreviewFromPath loads a preview for any arbitrary absolute file path.
func (m *model) loadPreviewFromPath(path string) {
	m.previewContent = ""
	m.previewHighlight = ""
	m.previewScroll = 0
	m.previewIsImage = false
	m.previewFilePath = path
	name := filepath.Base(path)
	if isImageFile(name) {
		m.previewIsImage = true
		m.previewHighlight = renderImagePreview(path, m.width/3)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		m.previewContent = fmt.Sprintf("Error reading file: %v", err)
		m.previewHighlight = m.previewContent
		return
	}
	if isBinary(data) {
		m.previewContent = ""
		m.previewHighlight = ""
		return
	}
	m.previewContent = string(data)
	m.previewHighlight = highlightCode(m.previewContent, name)
}

// isBinary returns true when data contains null bytes or too many non-printable
// characters, which indicates a binary file that should not be rendered.
func isBinary(data []byte) bool {
	check := data
	if len(check) > 8000 {
		check = check[:8000]
	}
	nonPrintable := 0
	for _, b := range check {
		if b == 0 {
			return true
		}
		if b < 0x08 || (b >= 0x0e && b < 0x20 && b != 0x1b) {
			nonPrintable++
		}
	}
	return len(check) > 0 && nonPrintable*100/len(check) > 10
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
type launchEditorMsg struct{ path string }

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

// filteredFiles returns the files list filtered by m.searchQuery.
// When searchActive is false or query is empty, returns all files.
func (m model) filteredFiles() []fileEntry {
	if !m.searchActive || strings.TrimSpace(m.searchQuery) == "" {
		return m.files
	}
	q := strings.ToLower(strings.TrimSpace(m.searchQuery))
	var out []fileEntry
	for _, f := range m.files {
		if matchName(f.name, q) {
			out = append(out, f)
		}
	}
	return out
}

// computePreviewSearchHits returns the 0-based line indices that contain query.
func computePreviewSearchHits(content, query string) []int {
	if query == "" || content == "" {
		return nil
	}
	q := strings.ToLower(query)
	var hits []int
	for i, line := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(line), q) {
			hits = append(hits, i)
		}
	}
	return hits
}

func (m model) currentFolderName() string {
	return m.currentFolderInContext()
}

func (m model) currentFileName() string {
	f, ok := m.resolvedFile()
	if !ok {
		return ""
	}
	return f.name
}

func (m model) currentFilePath() string {
	if len(m.folders) == 0 {
		return ""
	}
	f, ok := m.resolvedFile()
	if !ok {
		return ""
	}
	return filepath.Join(m.snippetsDir, m.folders[m.folderCursor], f.name)
}

// hasSubfolders returns true if the folder at relPath (relative to snippetsDir) contains subdirectories.
func (m *model) hasSubfolders(relPath string) bool {
	dir := filepath.Join(m.snippetsDir, relPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			return true
		}
	}
	return false
}

// loadSubNavEntries populates subNavEntries for the current subNavStack path.
func (m *model) loadSubNavEntries() {
	rel := strings.Join(m.subNavStack, string(filepath.Separator))
	dir := filepath.Join(m.snippetsDir, rel)
	entries, err := os.ReadDir(dir)
	m.subNavEntries = nil
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		m.subNavEntries = append(m.subNavEntries, subNavEntry{
			name:  e.Name(),
			isDir: e.IsDir(),
		})
	}
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
