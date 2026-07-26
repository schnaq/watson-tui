package tui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestThemeUsesANSIOnly: the theme must stay inside ANSI 0-15 so it adapts to
// whatever palette the user's terminal defines.
func TestThemeUsesANSIOnly(t *testing.T) {
	styles := map[string]lipgloss.Style{
		"title": styleTitle, "dayHeader": styleDayHeader, "selected": styleSelected,
		"dim": styleDim, "running": styleRunning, "error": styleError,
		"border": styleBorder, "accent": styleAccent, "focus": styleFocus,
		"warn": styleWarn, "headerLabel": styleHeaderLabel,
	}
	for name, s := range styles {
		assertANSIColor(t, name+" fg", s.GetForeground())
		assertANSIColor(t, name+" bg", s.GetBackground())
	}
}

// assertANSIColor accepts an unset colour or a plain lipgloss.Color holding an
// ANSI index 0-15. Everything else fails, including the types that would let a
// hex value in through the back door: a Color("#ff0000"), an AdaptiveColor or a
// CompleteColor must not pass just because they are not a bare index.
func assertANSIColor(t *testing.T, what string, c lipgloss.TerminalColor) {
	t.Helper()
	switch v := c.(type) {
	case nil, lipgloss.NoColor:
		return // not set on this style
	case lipgloss.Color:
		if !isANSIIndex(string(v)) {
			t.Errorf("%s = %q, want a plain ANSI index 0-15", what, string(v))
		}
	default:
		t.Errorf("%s is a %T, want a plain lipgloss.Color with an ANSI index 0-15", what, c)
	}
}

// isANSIIndex reports whether s is one or two decimal digits denoting 0-15.
func isANSIIndex(s string) bool {
	if s == "" || len(s) > 2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(s)
	return err == nil && n <= 15
}

// TestSelectedRowCarriesMarker: the cursor row is marked by a bar in the first
// column, not by reverse video.
func TestSelectedRowCarriesMarker(t *testing.T) {
	now := time.Now()
	l := listModel{
		per:    period{unit: unitAll, ref: now},
		cursor: 1,
		rows: []row{
			{isHeader: true, title: "Montag"},
			{frame: watson.Frame{
				ID: "a1111111111111111111111111111111", Project: "alpha",
				Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Tags: []string{},
			}},
		},
	}
	selected := l.renderRow(1, 80, 7)
	if !strings.Contains(selected, selectionMarker) {
		t.Errorf("selected row lacks %q marker:\n%s", selectionMarker, selected)
	}
	l.cursor = -1
	if plain := l.renderRow(1, 80, 7); strings.Contains(plain, selectionMarker) {
		t.Errorf("unselected row must not carry the marker:\n%s", plain)
	}
	if styleSelected.GetReverse() {
		t.Error("styleSelected still uses reverse video; the marker replaces it")
	}
}
