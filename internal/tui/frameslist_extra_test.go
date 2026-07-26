package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestPeriodLabel pins the exact German status-bar strings for every unit.
// label() is an exported-contract item (Produces, Task 9) that the eight
// brief tests never exercise; Tasks 10-16 read these strings.
func TestPeriodLabel(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local) // Mittwoch
	cases := []struct {
		unit periodUnit
		want string
	}{
		{unitDay, "Mittwoch, 22.07.2026"},
		{unitWeek, "Woche 20.07. – 26.07.2026"},
		{unitMonth, "Juli 2026"},
		{unitAll, "alle Frames"},
	}
	for _, c := range cases {
		if got := (period{unit: c.unit, ref: ref}).label(time.Monday); got != c.want {
			t.Errorf("unit %d: label = %q, want %q", c.unit, got, c.want)
		}
	}
}

// TestPeriodShiftAllNoop covers the !ok early return in shift(): unitAll is
// unbounded, so shifting must leave the period untouched.
func TestPeriodShiftAllNoop(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	p := period{unit: unitAll, ref: ref}
	got := p.shift(time.Monday, 3)
	if got.unit != unitAll || !got.ref.Equal(ref) {
		t.Errorf("shift on unitAll changed period: %+v", got)
	}
}

// TestTruncate covers both branches: short strings pass through unchanged,
// long strings are cut rune-safely with an ellipsis.
func TestTruncate(t *testing.T) {
	if got := truncate("abcd", 4); got != "abcd" {
		t.Errorf("equal-length: got %q", got)
	}
	if got := truncate("abc", 10); got != "abc" {
		t.Errorf("short: got %q", got)
	}
	// Unicode: each umlaut is one rune, so the cut must not split bytes.
	if got := truncate("äöüxyz", 4); got != "äöü…" {
		t.Errorf("long unicode: got %q, want %q", got, "äöü…")
	}
}

// TestFirstAndNextFrameRowEmpty covers the "no frame" paths of the row
// helpers used for navigation.
func TestFirstAndNextFrameRowEmpty(t *testing.T) {
	if firstFrameRow(nil) != -1 {
		t.Error("firstFrameRow(nil) must be -1")
	}
	headers := []row{{isHeader: true}, {isHeader: true}}
	if firstFrameRow(headers) != -1 {
		t.Error("firstFrameRow(all headers) must be -1")
	}
	// No frame in the requested direction: return the starting index.
	if got := nextFrameRow(headers, 0, +1); got != 0 {
		t.Errorf("nextFrameRow with no target = %d, want 0", got)
	}
}

// TestListMoveEmpty ensures move() and selected() are safe on an empty list
// (cursor == -1).
func TestListMoveEmpty(t *testing.T) {
	l := newListModel(time.Now())
	if _, ok := l.selected(); ok {
		t.Error("empty list must not report a selection")
	}
	l.move(+1) // must not panic and must not leave -1
	if l.cursor != -1 {
		t.Errorf("move on empty list changed cursor to %d", l.cursor)
	}
}

// TestListViewEmptyMessages covers view()'s two empty-state messages.
func TestListViewEmptyMessages(t *testing.T) {
	l := newListModel(time.Now())
	if got := l.view(10, 80); !strings.Contains(got, "keine Frames im Zeitraum") {
		t.Errorf("empty view = %q", got)
	}
	l.filter = "zzz"
	if got := l.view(10, 80); !strings.Contains(got, "kein Treffer für Filter »zzz«") {
		t.Errorf("no-match view = %q", got)
	}
}

// TestHeaderFieldsListBranches covers the three filter branches of the list
// header: no filter (the field says so instead of vanishing), a set filter, and
// the live input while typing, which has to replace the stored filter rather
// than sit next to it.
func TestHeaderFieldsListBranches(t *testing.T) {
	app := newTestApp(t)
	app.mode = modeList
	app.list.per = period{unit: unitAll, ref: time.Now()}
	app.list.refresh(app.frames, time.Monday)

	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Frames 0") ||
		!strings.Contains(got, "Filter —") {
		t.Errorf("plain header = %q", got)
	}

	app.list.filter = "foo"
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Filter foo") {
		t.Errorf("filter header = %q", got)
	}

	app.list.filtering = true
	app.list.filterInput.Focus()
	app.list.filterInput.SetValue("bar")
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "bar") ||
		strings.Contains(got, "Filter foo") {
		t.Errorf("filtering header = %q", got)
	}
}

// TestListViewRendersRows covers the non-empty view/renderRow path (header +
// frame line), which the App.View default branch reaches at runtime but the
// brief tests do not.
func TestListViewRendersRows(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local) // Montag
	l := newListModel(day)
	l.per = period{unit: unitWeek, ref: day}
	l.refresh([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, time.Hour, "tag1"),
	}, time.Monday)
	out := l.view(10, 80)
	if !strings.Contains(out, "alpha") {
		t.Errorf("view missing project: %q", out)
	}
	if !strings.Contains(out, "Montag") { // day header (2026-07-20 is a Monday)
		t.Errorf("view missing day header: %q", out)
	}
}
