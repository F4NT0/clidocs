package main

import "github.com/charmbracelet/lipgloss"

var (
	colorBg          = lipgloss.Color("#0d1117")
	colorBgPanel     = lipgloss.Color("#161b22")
	colorBgSelected  = lipgloss.Color("#1c2128")
	colorBgActive    = lipgloss.Color("#e6edf3")
	colorFg          = lipgloss.Color("#c9d1d9")
	colorFgMuted     = lipgloss.Color("#6e7681")
	colorFgSelected  = lipgloss.Color("#adbac7")
	colorAccent      = lipgloss.Color("#e6edf3")
	colorAccentBlue  = lipgloss.Color("#58a6ff")
	colorBorder      = lipgloss.Color("#30363d")
	colorArrow       = lipgloss.Color("#e6edf3")
	colorOrange      = lipgloss.Color("#e8912d")

	baseStyle = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorFg)

	panelStyle = lipgloss.NewStyle().
			Background(colorBgPanel).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	activePanelStyle = lipgloss.NewStyle().
				Background(colorBgPanel).
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorAccent)

	headerActiveStyle = lipgloss.NewStyle().
				Background(colorAccent).
				Foreground(colorBg).
				Bold(true).
				Padding(0, 1)

	headerInactiveStyle = lipgloss.NewStyle().
				Background(colorBgPanel).
				Foreground(colorFgMuted).
				Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Background(colorBgPanel).
				Foreground(colorFgSelected)

	normalItemStyle = lipgloss.NewStyle().
			Background(colorBgPanel).
			Foreground(colorFg)

	arrowStyle = lipgloss.NewStyle().
			Foreground(colorOrange)

	fileArrowStyle = lipgloss.NewStyle().
			Foreground(colorOrange)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorFgMuted).
			Bold(true).
			MarginBottom(0)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorFgMuted)

	accentStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	blueStyle = lipgloss.NewStyle().
			Foreground(colorAccentBlue)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	modalStyle = lipgloss.NewStyle().
			Background(colorBgPanel).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			MarginBottom(1)

	inputStyle = lipgloss.NewStyle().
			Background(colorBgSelected).
			Foreground(colorFgSelected).
			Padding(0, 1).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorFgMuted).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff7b72")).
			Bold(true)
)
