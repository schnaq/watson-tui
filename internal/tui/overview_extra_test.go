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
	out := overviewView(nil, time.Monday, now)
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
	out := overviewView(frames, time.Monday, now)
	if strings.Contains(out, long) {
		t.Errorf("long project name not truncated:\n%s", out)
	}
	if !strings.Contains(out, truncate(long, 24)) {
		t.Errorf("truncated project name missing:\n%s", out)
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
