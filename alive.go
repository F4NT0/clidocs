package main

import (
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

const aliveMaxChars = 10

type aliveTickMsg struct{}

var procKeybdEvent = syscall.NewLazyDLL("user32.dll").NewProc("keybd_event")

func sendKeyB() {
	const vkB, keyUp = 0x42, 0x0002
	procKeybdEvent.Call(vkB, 0, 0, 0)
	procKeybdEvent.Call(vkB, 0, keyUp, 0)
}

func aliveTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return aliveTickMsg{} })
}

func aliveSendKey() tea.Cmd {
	return func() tea.Msg {
		sendKeyB()
		return nil
	}
}

func RenderAliveModal(input string) string {
	title := lipgloss.NewStyle().Foreground(colorAccentBlue).Bold(true).Render("Alive")
	sep := mutedStyle.Render(strings.Repeat("─", 50))
	prompt := lipgloss.NewStyle().Foreground(colorAccentBlue).Render("> ") +
		lipgloss.NewStyle().Foreground(colorFg).Render(input) +
		lipgloss.NewStyle().Foreground(colorFgMuted).Render("█")
	inputLine := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccentBlue).
		Padding(0, 1).
		Width(48).
		Render(prompt)
	rows := []string{title, sep, inputLine, "", sep,
		mutedStyle.Render("Pressing \"B\" every 2s  |  Esc: close")}
	return lipgloss.NewStyle().
		Background(colorBg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccentBlue).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
