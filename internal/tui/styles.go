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

// The summary's rail: one column left of the labels that brackets a day block.
// A blank line alone did not separate the days, because a project boundary
// carried the same blank line as a day boundary — so there were no blocks, only
// a list.
const (
	railOpen  = "╭"
	railMid   = "│"
	railClose = "╰"
)

// The glyphs a tag row hangs from, so that it reads as a breakdown of the
// project above it and says which of them is the last. They replace the square
// bracket the first draft of the summary used, which did the first job and not
// the second — and read like a data format rather than an interface.
const (
	tagBranch = "├─ "
	tagLast   = "└─ "
)

// The share bar. barEighths is indexed by how many eighths of a column are left
// over after the full blocks, so index 0 is the empty string and the array is
// read straight from the remainder. Eighths rather than full blocks alone,
// because the smallest entries of a period otherwise round to no columns and a
// row that renders as nothing says nothing was booked.
const (
	barFull  = "█"
	barTrack = "░"
)

var barEighths = [8]string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

var (
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleDayHeader = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	// styleDayPlain is a day of the summary that is not today. The bold weight is
	// what marks the current day there, so the others give it up — and it is a
	// style of its own rather than styleDayHeader losing its Bold, because the
	// frame list draws its day headers with that one and has no such distinction to
	// make.
	styleDayPlain = lipgloss.NewStyle().Foreground(colAccent)
	// stylePlain leaves text in the terminal's own foreground colour. Named so that
	// a row rendered from styled parts can say "this part carries no colour"
	// instead of holding an empty style nobody recognises.
	stylePlain       = lipgloss.NewStyle()
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
