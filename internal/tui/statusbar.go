package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
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

// renderStatus renders the global one-line status bar. left carries
// view-specific info (period, counts, filter); errMsg replaces left.
func renderStatus(width int, state *watson.State, now time.Time, left, errMsg string) string {
	if errMsg != "" {
		left = styleError.Render(errMsg)
	}
	var right string
	if state != nil {
		label := state.Project
		if len(state.Tags) > 0 {
			label += " [" + strings.Join(state.Tags, ", ") + "]"
		}
		right = styleRunning.Render(fmt.Sprintf("▶ %s %s", label, formatClock(now.Sub(state.Start))))
	} else {
		right = styleDim.Render("kein Timer")
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
