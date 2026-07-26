// This file holds the timer plumbing: the one-second tick that drives the
// running clock and the two formatters that render a duration for the views.
// It was statusbar.go until the chrome took the status bar's job — the header
// carries the context and the timer, the footer the key hints.

package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// formatDuration renders "6h 30m" / "45m", rounded to minutes.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}

// formatClock renders a live timer as H:MM:SS.
func formatClock(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}
