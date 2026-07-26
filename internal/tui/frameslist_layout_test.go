package tui

// This file holds the frame list's half of the rule the overview and the report
// already follow: when the width does not fit, a label gives way, never a number
// and never an identifier — and a shortened value must never look complete.

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// wideAndNarrowFrames are two frames of the same day whose IDs share the seven
// characters ShortID renders, one of them long enough to need a three-digit hour
// count. Both defects this file pins need exactly that pair: the wide duration
// shifts the columns behind it, and the shared prefix is what makes a shortened
// ID dangerous rather than merely ugly — watson resolves IDs by prefix
// (watson.FindByIDPrefix), so "a11111" read off the screen is ambiguous between
// these two frames and "watson edit a11111" either refuses or edits the wrong one.
func wideAndNarrowFrames(day time.Time) []watson.Frame {
	return []watson.Frame{
		// Three weeks on one frame: what the a (all) period shows for a project that
		// has been running for a while.
		mkFrame("a11111122222222222222222222222222", "schnaq", day, 130*time.Hour, "dev"),
		mkFrame("a11111199999999999999999999999999", "kunde-a", day.Add(2*time.Hour),
			30*time.Minute, "meeting"),
	}
}

// TestListKeepsColumnsAlignedPastNinetyNineHours: the duration field was a fixed
// %7s, but "130h 00m" is eight columns, so every row with a three-digit hour
// count pushed the columns behind it one to the right — and the ID sits at the
// end of the row, which is the column fitBody cuts first. The field is now as
// wide as the widest duration the list actually renders, the way report.go sizes
// its number column, and it stays right-aligned so the numbers line up.
func TestListKeepsColumnsAlignedPastNinetyNineHours(t *testing.T) {
	day := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := wideAndNarrowFrames(day)
	l := newListModel(day)
	l.per = period{unit: unitAll, ref: day}
	l.refresh(frames, time.Monday)
	// No row is the cursor row, so nothing is styled and the assertions can count
	// columns in the plain line instead of stepping over escape sequences.
	l.cursor = -1
	durW := listDurWidth(l.rows)

	var starts []int
	for i, r := range l.rows {
		if r.isHeader {
			continue
		}
		fr := r.frame
		line := l.renderRow(i, durW)
		times := fr.Start.Local().Format("15:04") + "–" + fr.Stop.Local().Format("15:04")
		_, rest, ok := strings.Cut(line, times)
		if !ok {
			t.Fatalf("row %d does not show its times %q: %q", i, times, line)
		}
		// Everything between the times and the project name is the duration cell,
		// padding included. Trimmed it has to be the duration verbatim: a cut number
		// would still be a substring of the line, so Contains would not catch it.
		cell, _, ok := strings.Cut(rest, fr.Project)
		if !ok {
			t.Fatalf("row %d does not show its project %q: %q", i, fr.Project, line)
		}
		if got, want := strings.TrimSpace(cell), formatDuration(fr.Duration()); got != want {
			t.Errorf("row %d duration = %q, want %q: %q", i, got, want, line)
		}
		starts = append(starts, lipgloss.Width(line[:strings.Index(line, fr.Project)]))
	}
	if len(starts) != 2 {
		t.Fatalf("got %d frame rows, want 2", len(starts))
	}
	if starts[0] != starts[1] {
		t.Errorf("project column starts at column %d in the 130h row and at %d in the 30m row",
			starts[0], starts[1])
	}
}
