package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type splashModel struct {
	width      int
	height     int
	choice     int  // 0 = default dir, 1 = pick dir
	chosenDir  string
	quit       bool
	browsePick bool // tells main.go to open the dir picker after splash exits
}

func newSplashModel() splashModel {
	return splashModel{choice: 0}
}

func (s splashModel) Init() tea.Cmd { return nil }

func (s splashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			s.quit = true
			return s, tea.Quit
		case "up", "k":
			if s.choice > 0 {
				s.choice--
			}
		case "down", "j":
			if s.choice < 1 {
				s.choice++
			}
		case "enter":
			if s.choice == 0 {
				s.chosenDir = getSnippetsDir()
				return s, tea.Quit
			}
			// choice == 1: signal main.go to open the picker after TUI exits
			s.browsePick = true
			return s, tea.Quit
		}
	}
	return s, nil
}

func (s splashModel) View() string {
	if s.width == 0 {
		return ""
	}

	bg := colorBg
	green := lipgloss.Color("#00ff88")
	muted := lipgloss.NewStyle().Foreground(colorFgMuted)

	ascii := `
 ██████╗██╗     ██╗██████╗  ██████╗  ██████╗███████╗
██╔════╝██║     ██║██╔══██╗██╔═══██╗██╔════╝██╔════╝
██║     ██║     ██║██║  ██║██║   ██║██║     ███████╗
██║     ██║     ██║██║  ██║██║   ██║██║     ╚════██║
╚██████╗███████╗██║██████╔╝╚██████╔╝╚██████╗███████║
 ╚═════╝╚══════╝╚═╝╚═════╝  ╚═════╝  ╚═════╝╚══════╝`

	asciiStyle := lipgloss.NewStyle().
		Foreground(green).
		Bold(true)

	subtitle := lipgloss.NewStyle().
		Foreground(colorAccentBlue).
		Render("Terminal-native snippet manager")

	tagline := muted.Render("Organize, preview, and edit code snippets in a three-panel TUI")

	sep := muted.Render(strings.Repeat("─", 54))

	option0 := "  Open default snippets directory"
	option1 := "  Browse for a directory..."

	opt0Style := lipgloss.NewStyle().Foreground(colorFg)
	opt1Style := lipgloss.NewStyle().Foreground(colorFg)
	cursor0 := "  "
	cursor1 := "  "

	if s.choice == 0 {
		opt0Style = lipgloss.NewStyle().Foreground(green).Bold(true)
		cursor0 = lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
	} else {
		opt1Style = lipgloss.NewStyle().Foreground(green).Bold(true)
		cursor1 = lipgloss.NewStyle().Foreground(colorOrange).Render("> ")
	}

	boxContent := lipgloss.JoinVertical(lipgloss.Center,
		asciiStyle.Render(ascii),
		"",
		subtitle,
		tagline,
		"",
		sep,
		"",
		cursor0+opt0Style.Render(option0),
		cursor1+opt1Style.Render(option1),
		"",
		muted.Render("↑↓: select   enter: confirm   ctrl+c: quit"),
	)

	box := lipgloss.NewStyle().
		Background(bg).
		Width(s.width).
		Height(s.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(boxContent)

	return box
}
