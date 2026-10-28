package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct{ key, desc string }

type helpSection struct {
	title   string
	intro   string
	entries []helpEntry
}

type helpTab struct {
	name     string
	sections []helpSection
}

var helpTabs = []helpTab{
	{"Overview", []helpSection{
		{"CLIDOCS " + "V" + Version, "A terminal-native snippet manager. It organizes code snippets, notes and Markdown files in folders and lets you browse, preview, edit and sync them without leaving the terminal.", nil},
		{"Usage", "", []helpEntry{
			{"clidocs", "Start with the splash screen: open the default snippets directory or browse for another one."},
			{"clidocs .", "Open the current working directory as the snippets directory."},
			{"clidocs <path>", "Open the given directory (relative or absolute) directly, skipping the splash screen."},
			{"clidocs --help", "Show this interactive help (aliases: -h, help)."},
			{"clidocs --install", "Download the latest release from GitHub (F4NT0/clidocs), verify it and install it, adding the clidocs command to PATH and PowerShell. Add -y to skip the prompt."},
			{"clidocs --uninstall", "Remove clidocs from this computer (executable, PATH entry and PowerShell alias). Snippets are kept. Add -y to skip the prompt."},
			{"clidocs --version", "Print the version (aliases: -v, version)."},
		}},
		{"Layout", "The main screen has three panels. Use Tab to move between them; the active panel is highlighted with a blue border.", []helpEntry{
			{"Folders", "Left panel. Lists the folders (and subfolders) of the snippets directory."},
			{"Files", "Middle panel. Lists the files inside the selected folder, with language tag and age."},
			{"Preview", "Right panel. Shows the selected file with syntax highlighting or rendered Markdown."},
		}},
	}},
	{"Navigation", []helpSection{
		{"Moving around", "Navigation works in every panel and follows Vim-style conventions.", []helpEntry{
			{"Tab", "Cycle focus: Folders -> Files -> Preview."},
			{"s", "Jump straight back to the Folders panel."},
			{"Up / k", "Move the cursor up."},
			{"Down / j", "Move the cursor down."},
			{"Right / l", "Enter the selected folder (or move to the next panel)."},
			{"Left / h", "Go back to the parent folder (or the previous panel)."},
			{"Enter", "Open the selected folder or file."},
			{"H", "Return to the original snippets directory when you are elsewhere."},
			{"?", "Open the help panel with every available command."},
			{"q / Ctrl+C", "Quit the application."},
		}},
		{"Subfolders", "Folders marked with a '>' contain subfolders. Entering one pushes it on a navigation stack; Left or h pops back. The '~/' row represents files stored directly in the current directory.", nil},
	}},
	{"Folders", []helpSection{
		{"Folder management", "Available while the Folders panel is focused.", []helpEntry{
			{"n", "Create a new folder."},
			{"N", "Create a new subfolder inside the selected folder."},
			{"r", "Rename the selected folder."},
			{"d", "Delete the selected folder (asks for confirmation)."},
			{"D", "Multi-delete mode: Space selects folders, D confirms, Esc cancels."},
			{"/", "Search folders recursively and jump to a result."},
			{"o", "Show information about the current snippets directory."},
		}},
		{"Favorites", "Favorite folders are marked with a star and can be reached quickly from anywhere.", []helpEntry{
			{"f", "Toggle favorite on the selected folder."},
			{"F", "Open the favorites list."},
		}},
	}},
	{"Files", []helpSection{
		{"File management", "Available while the Files panel is focused.", []helpEntry{
			{"Enter / e", "Open the selected file in Neovim."},
			{"n", "Create a new file in the current folder."},
			{"r", "Rename the selected file."},
			{"d", "Delete the selected file (asks for confirmation)."},
			{"m", "Move the file to another folder."},
			{"c", "Copy a file from elsewhere into the current folder."},
			{"/", "Filter the file list by name as you type."},
		}},
		{"Supported content", "Any text file can be stored. Markdown files are rendered with styled headings, lists and tables; code files are syntax highlighted.", nil},
	}},
	{"Preview", []helpSection{
		{"Preview panel", "Available while the Preview panel is focused.", []helpEntry{
			{"c", "Copy the whole file content to the clipboard."},
			{"e", "Edit the file in Neovim."},
			{"o", "Open the file location in the system file explorer."},
			{"z / Z", "Toggle zoom: the preview fills the whole screen."},
			{"L", "Toggle line numbers."},
			{"/", "Search inside the file content."},
			{"Up / Down", "Scroll the content."},
		}},
	}},
	{"Git Sync", []helpSection{
		{"Syncing snippets", "Your snippets directory can be backed up to a Git repository.", []helpEntry{
			{"g", "Sync with Git. On first use a setup dialog asks for the remote; afterwards it commits and pushes/pulls automatically."},
			{"G", "Open the Git configuration (remote URL and settings)."},
		}},
		{"Notes", "The sync creates the repository and a .gitignore inside the snippets directory when they do not exist yet. Git must be installed and available in PATH.", nil},
	}},
	{"Console", []helpSection{
		{"Command console", "Press ':' on the main screen to open a small command console.", []helpEntry{
			{"time", "Work hours calculator: enter a start time (HH:MM)."},
			{"alive", "Show the alive screen."},
			{"whoami", "Show information about the author."},
			{"cmd", "Show the list of console commands."},
			{"nvim", "Neovim quick reference (navigation, editing, save & quit, search, comments)."},
			{"ls", "Open a tree view of the directories of the open directory."},
			{"cd <dir>", "Open a directory in clidocs (type it in the ls panel or in the console)."},
			{"exit / quit", "Close clidocs."},
		}},
	}},
}

type helpModel struct {
	tab, scroll int
	w, h        int
}

func (m helpModel) Init() tea.Cmd { return nil }

func (m helpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab, m.scroll = (m.tab+1)%len(helpTabs), 0
		case "shift+tab", "left", "h":
			m.tab, m.scroll = (m.tab+len(helpTabs)-1)%len(helpTabs), 0
		case "down", "j":
			m.scroll++
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "pgdown", " ":
			m.scroll += 10
		case "pgup":
			m.scroll = max(0, m.scroll-10)
		case "g", "home":
			m.scroll = 0
		default:
			if s := msg.String(); len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(helpTabs) {
				m.tab, m.scroll = int(s[0]-'1'), 0
			}
		}
	}
	return m, nil
}

func (m helpModel) View() string {
	if m.w == 0 {
		return ""
	}
	v, _ := renderHelp(helpTabs, m.tab, m.scroll, m.w, m.h, " clidocs V"+Version+" — Help", "q/Esc: quit")
	return v
}

// renderHelp draws the tabbed help for the given size and returns the view and
// the maximum scroll offset of the active tab.
func renderHelp(tabs []helpTab, tab, scroll, w, h int, header, quitHint string) (string, int) {
	innerW := w - 6
	if innerW > 100 {
		innerW = 100
	}
	if innerW < 30 {
		innerW = 30
	}

	activeTab := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorAccentBlue).Padding(0, 1)
	idleTab := lipgloss.NewStyle().Foreground(colorFgMuted).
		Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
	var tabViews []string
	for i, t := range tabs {
		label := fmt.Sprintf("%d %s", i+1, t.name)
		if i == tab {
			tabViews = append(tabViews, activeTab.Render(label))
		} else {
			tabViews = append(tabViews, idleTab.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)
	if lipgloss.Width(tabBar) > w {
		tabBar = lipgloss.NewStyle().MaxWidth(w).Render(tabBar)
	}

	title := lipgloss.NewStyle().Foreground(colorOrange).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(colorOrange)
	wrap := lipgloss.NewStyle().Foreground(colorFg).Width(innerW - 2)
	sep := mutedStyle.Render(strings.Repeat("─", innerW-2))

	var lines []string
	for _, s := range tabs[tab].sections {
		lines = append(lines, title.Render(" "+s.title), sep)
		if s.intro != "" {
			lines = append(lines, wrap.Render(s.intro))
		}
		if len(s.entries) > 0 {
			if s.intro != "" {
				lines = append(lines, "")
			}
			kw := 0
			for _, e := range s.entries {
				kw = max(kw, lipgloss.Width(e.key))
			}
			descW := max(10, innerW-2-kw-3)
			for _, e := range s.entries {
				k := keyStyle.Width(kw).Render(e.key)
				d := lipgloss.NewStyle().Foreground(colorFgMuted).Width(descW).Render(e.desc)
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, " ", k, "  ", d))
			}
		}
		lines = append(lines, "")
	}

	single := len(tabs) == 1
	tabBarH := lipgloss.Height(tabBar)
	if single {
		tabBarH = 0
	}
	bodyH := h - 2 - (4 + tabBarH)
	if bodyH < 3 {
		bodyH = 3
	}
	var all []string
	for _, l := range lines {
		all = append(all, strings.Split(l, "\n")...)
	}
	maxScroll := max(0, len(all)-bodyH)
	scroll = min(scroll, maxScroll)
	end := min(len(all), scroll+bodyH)
	body := strings.Join(all[scroll:end], "\n")

	panel := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder).
		Padding(0, 1).Width(innerW).Height(bodyH).Render(body)

	hint := "←/→ Tab: switch topic   1-9: jump   ↑/↓: scroll   " + quitHint
	if single {
		hint = quitHint
	}
	if maxScroll > 0 {
		hint += fmt.Sprintf("   (%d/%d)", scroll+1, maxScroll+1)
	}
	parts := []string{panelTitleStyle.Render(header)}
	if !single {
		parts = append(parts, tabBar)
	}
	parts = append(parts, panel, mutedStyle.Render(" "+hint))
	return lipgloss.JoinVertical(lipgloss.Left, parts...), maxScroll
}

var consoleCmdTab = helpTab{"Console commands", []helpSection{
	{"Available commands", "Type a command in the console and press Enter.", []helpEntry{
		{"cmd", "Show this list of console commands and the Neovim quick reference."},
		{"time", "Work hours calculator: enter your start time (HH:MM) to get the coffee break, lunch and exit times."},
		{"alive", "Keep the PC awake by pressing the B key every 2 seconds."},
		{"whoami", "Show information about the author."},
		{"nvim", "Open the Neovim quick reference tab directly."},
		{"ls", "Open a tree view of the directories of the open directory."},
		{"cd <dir>", "Open a directory in clidocs (type it in the ls panel or in the console)."},
		{"exit / quit", "Close clidocs."},
	}},
}}

var nvimTab = helpTab{"Neovim Quick Reference", []helpSection{
	{"Navigation", "", []helpEntry{
		{"h j k l", "← ↓ ↑ → (or arrow keys)"},
		{"gg", "go to top of file"},
		{"G", "go to bottom of file"},
		{"Ctrl+d", "scroll half page down"},
		{"Ctrl+u", "scroll half page up"},
		{"w / b", "next / previous word"},
		{"0 / $", "start / end of line"},
	}},
	{"Editing", "", []helpEntry{
		{"i", "insert before cursor"},
		{"a", "insert after cursor"},
		{"o", "new line below, insert"},
		{"O", "new line above, insert"},
		{"Esc", "return to Normal mode"},
		{"u", "undo"},
		{"Ctrl+r", "redo"},
		{"dd", "delete (cut) line"},
		{"yy", "yank (copy) line"},
		{"p", "paste after cursor"},
		{"x", "delete character"},
	}},
	{"Save & Quit", "", []helpEntry{
		{":w", "save file"},
		{":q", "quit (no unsaved changes)"},
		{":wq / :x", "save and quit"},
		{":q!", "quit without saving"},
	}},
	{"Search", "", []helpEntry{
		{"/word", "search forward"},
		{"n / N", "next / previous match"},
	}},
	{"Comment multiple lines", "", []helpEntry{
		{"Ctrl+V", "select lines (block mode)"},
		{":", "opens command line"},
		{"'<,'>s/^/#", "adds # at line start"},
	}},
	{"Uncomment multiple lines", "", []helpEntry{
		{"Ctrl+V", "select lines (block mode)"},
		{":", "opens command line"},
		{"'<,'>s/^#//", "removes # from start"},
	}},
	{"Visual mode", "", []helpEntry{
		{"v", "char select"},
		{"V", "line select"},
		{"Ctrl+V", "block select"},
	}},
}}

// RenderCmdHelp draws the help of the console commands, opened by the "cmd" command.
func RenderCmdHelp(tab, scroll, termW, termH int) string {
	v, _ := renderHelp([]helpTab{consoleCmdTab, nvimTab}, tab, scroll, min(termW-2, 90), min(termH-2, 22), " Console help — available commands", "Enter / Esc: back to console")
	return lipgloss.NewStyle().Background(colorBg).Render(v)
}

// consoleHelpMaxScroll returns the maximum scroll of the active tab of the console help.
func consoleHelpMaxScroll(tab, termW, termH int) int {
	_, ms := renderHelp([]helpTab{consoleCmdTab, nvimTab}, tab, 0, min(termW-2, 90), min(termH-2, 22), "", "")
	return ms
}

// helpPanelSize returns the width and height of the in-app help panel.
func helpPanelSize(termW, termH int) (int, int) {
	return min(termW-2, 106), min(termH-2, 34)
}

// RenderHelpPanel draws the floating help panel opened with "?" (it skips the
// command-line Overview tab).
func RenderHelpPanel(tab, scroll, termW, termH int) string {
	w, h := helpPanelSize(termW, termH)
	v, _ := renderHelp(helpTabs[1:], tab, scroll, w, h, " Help — available commands", "?/Esc: close")
	return lipgloss.NewStyle().Background(colorBg).Render(v)
}

// helpPanelMaxScroll returns the maximum scroll of the active tab of the help panel.
func helpPanelMaxScroll(tab, termW, termH int) int {
	w, h := helpPanelSize(termW, termH)
	_, ms := renderHelp(helpTabs[1:], tab, 0, w, h, "", "")
	return ms
}

func runHelp() {
	p := tea.NewProgram(helpModel{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
