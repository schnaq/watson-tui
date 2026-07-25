package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle     = lipgloss.NewStyle().Bold(true)
	styleDayHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleSelected  = lipgloss.NewStyle().Reverse(true)
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleRunning   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	styleError     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
)
