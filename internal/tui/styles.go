package tui

// This file holds the package-level theme. Everything stays inside ANSI 0-15 so
// the UI adopts the palette of whatever terminal theme the user runs — the same
// choice k9s and lazygit make by default.

import "github.com/charmbracelet/lipgloss"

// ANSI colour roles. Named so a change of taste happens in one place.
const (
	colDim    = lipgloss.Color("8") // bright black: borders, secondary text
	colAccent = lipgloss.Color("6") // cyan: panel titles, day headers
	colFocus  = lipgloss.Color("4") // blue: focused border, selection
	colOK     = lipgloss.Color("2") // green: running timer
	colWarn   = lipgloss.Color("3") // yellow: warnings
	colErr    = lipgloss.Color("1") // red: errors
)

// selectionMarker sits in the first column of the cursor row. A bar plus a
// coloured, bold line reads as a selection at a glance; reverse video does not.
const selectionMarker = "▌"

var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleDayHeader   = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleSelected    = lipgloss.NewStyle().Bold(true).Foreground(colFocus)
	styleDim         = lipgloss.NewStyle().Foreground(colDim)
	styleRunning     = lipgloss.NewStyle().Bold(true).Foreground(colOK)
	styleError       = lipgloss.NewStyle().Bold(true).Foreground(colErr)
	styleBorder      = lipgloss.NewStyle().Foreground(colDim)
	styleAccent      = lipgloss.NewStyle().Foreground(colAccent)
	styleFocus       = lipgloss.NewStyle().Foreground(colFocus)
	styleWarn        = lipgloss.NewStyle().Foreground(colWarn)
	styleHeaderLabel = lipgloss.NewStyle().Foreground(colDim)
	// styleKey sets a key off from its description in the footer, so "j/k" is
	// findable in a line of ten hints without reading the line.
	styleKey = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
)
