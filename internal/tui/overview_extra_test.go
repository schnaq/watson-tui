package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestCellDuration covers the zero-dash branch and the delegation to
// formatDuration for non-zero values.
func TestCellDuration(t *testing.T) {
	if got := cellDuration(0); got != "–" {
		t.Errorf("cellDuration(0) = %q, want dash", got)
	}
	if got, want := cellDuration(90*time.Minute), formatDuration(90*time.Minute); got != want {
		t.Errorf("cellDuration(90m) = %q, want %q", got, want)
	}
}

// TestBuildOverviewTiesAlphabetical pins the tie-break: equal totals sort by
// project name ascending, so the output is stable across map iterations.
func TestBuildOverviewTiesAlphabetical(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("c3333333333333333333333333333333", "charlie", day, time.Hour),
		mkFrame("a1111111111111111111111111111111", "alpha", day, time.Hour),
		mkFrame("b2222222222222222222222222222222", "bravo", day, time.Hour),
	}
	rows, _ := buildOverview(frames, cols, time.Monday)
	got := []string{rows[0].project, rows[1].project, rows[2].project}
	want := []string{"alpha", "bravo", "charlie"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row order = %v, want %v", got, want)
		}
	}
}

// TestBuildOverviewEmpty: no frames means no rows and zero totals, not a nil
// deref on the totals slice.
func TestBuildOverviewEmpty(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	rows, totals := buildOverview(nil, cols, time.Monday)
	if len(rows) != 0 {
		t.Errorf("got %d rows, want 0", len(rows))
	}
	if len(totals) != len(cols) {
		t.Fatalf("got %d totals, want %d", len(totals), len(cols))
	}
	for i, d := range totals {
		if d != 0 {
			t.Errorf("totals[%d] = %v, want 0", i, d)
		}
	}
}

// TestOverviewViewEmpty shows the placeholder instead of a bare header.
func TestOverviewViewEmpty(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	out := overviewView(nil, nil, time.Monday, now, 120)
	if !strings.Contains(out, "keine Frames vorhanden") {
		t.Errorf("empty overview missing placeholder:\n%s", out)
	}
	if !strings.Contains(out, "Gesamt") {
		t.Errorf("empty overview missing total line:\n%s", out)
	}
}

// TestOverviewViewTruncatesProject: long project names must not push the
// columns out of alignment.
func TestOverviewViewTruncatesProject(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	long := "ein-sehr-langer-projektname-der-nicht-passt"
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", long, time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), time.Hour),
	}
	out := overviewView(frames, nil, time.Monday, now, 120)
	if strings.Contains(out, long) {
		t.Errorf("long project name not truncated:\n%s", out)
	}
	if !strings.Contains(out, truncate(long, 24)) {
		t.Errorf("truncated project name missing:\n%s", out)
	}
}

// TestPeriodAttributionByStart pins the central billing rule for both money
// facing views: a frame counts fully into the period it *starts* in and is
// never split. The 31.07. 23:00 → 01.08. 02:00 session belongs to July with
// all three hours; August must see nothing of it. An invoice depends on this.
func TestPeriodAttributionByStart(t *testing.T) {
	start := time.Date(2026, 7, 31, 23, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", start, 3*time.Hour),
	}

	// buildOverview: mid-August "now", so dieser Monat = August (index 2) and
	// letzter Monat = Juli (index 3); neither week column contains the frame.
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	rows, totals := buildOverview(frames, cols, time.Monday)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	want := []time.Duration{0, 0, 0, 3 * time.Hour, 3 * time.Hour}
	for i := range want {
		if rows[0].cells[i] != want[i] {
			t.Errorf("column %q = %v, want %v", cols[i].title, rows[0].cells[i], want[i])
		}
		if totals[i] != want[i] {
			t.Errorf("total %q = %v, want %v", cols[i].title, totals[i], want[i])
		}
	}

	// aggregate: the same rule over month periods.
	july := period{unit: unitMonth, ref: time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)}
	lines, grand := aggregate(frames, july, time.Monday)
	if grand != 3*time.Hour || len(lines) != 1 || lines[0].total != 3*time.Hour {
		t.Errorf("Juli: grand=%v lines=%+v, want 3h on alpha", grand, lines)
	}
	august := period{unit: unitMonth, ref: now}
	lines, grand = aggregate(frames, august, time.Monday)
	if grand != 0 || len(lines) != 0 {
		t.Errorf("August: grand=%v lines=%+v, want nothing", grand, lines)
	}
}

// TestRunningNoteNamesColumns: the note must name the columns the running
// frame actually lands in. A timer left running over the weekend counts into
// letzte Woche — claiming it for the diese Woche column an invoice is written
// from would overstate that week. The assertions run against runningNote
// directly because the column titles also appear as table headers.
func TestRunningNoteNamesColumns(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local) // Mittwoch
	cols := overviewColumns(now, time.Monday)

	lastWeek := &watson.State{Project: "alpha", Start: time.Date(2026, 7, 18, 10, 0, 0, 0, time.Local), Tags: []string{}}
	note := runningNote(lastWeek, cols, time.Monday, now, overviewProjMax)
	if strings.Contains(note, "diese Woche") {
		t.Errorf("timer from last week must not be claimed for diese Woche:\n%s", note)
	}
	for _, want := range []string{"alpha", "läuft", "eingerechnet in:", "letzte Woche", "dieser Monat", "gesamt"} {
		if !strings.Contains(note, want) {
			t.Errorf("note missing %q:\n%s", want, note)
		}
	}

	thisWeek := &watson.State{Project: "alpha", Start: now.Add(-time.Hour), Tags: []string{}}
	note = runningNote(thisWeek, cols, time.Monday, now, overviewProjMax)
	if !strings.Contains(note, "diese Woche") {
		t.Errorf("timer from today must be claimed for diese Woche:\n%s", note)
	}
	if strings.Contains(note, "letzte Woche") {
		t.Errorf("timer from today must not be claimed for letzte Woche:\n%s", note)
	}

	out := overviewView(nil, lastWeek, time.Monday, now, 120)
	if !strings.Contains(out, "eingerechnet in:") {
		t.Errorf("view must disclose which columns count the timer:\n%s", out)
	}
}

// TestOverviewCloseKeys: o and q close the overview just like esc does.
func TestOverviewCloseKeys(t *testing.T) {
	for _, k := range []string{"o", "q"} {
		app := newTestApp(t)
		app.Update(key("o"))
		if app.mode != modeOverview {
			t.Fatalf("o must open overview")
		}
		app.Update(key(k))
		if app.mode != modeList {
			t.Errorf("%q must close overview, mode = %v", k, app.mode)
		}
	}
}
