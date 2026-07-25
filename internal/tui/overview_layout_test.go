package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestOverviewLayoutFitsWidth: the layout must never exceed the terminal
// width as long as the minimum widths still fit.
func TestOverviewLayoutFitsWidth(t *testing.T) {
	for _, width := range []int{200, 120, 94, 80, 70} {
		projW, cellW := overviewLayout(width, 5)
		total := projW + 5*(cellW+1)
		if total > width {
			t.Errorf("width %d: layout needs %d columns (proj=%d cell=%d)", width, total, projW, cellW)
		}
		if cellW < overviewCellMin {
			t.Errorf("width %d: cell width %d below minimum %d", width, cellW, overviewCellMin)
		}
		if projW < overviewProjMin {
			t.Errorf("width %d: project width %d below minimum %d", width, projW, overviewProjMin)
		}
	}
}

// TestOverviewLayoutKeepsProjectWidth: value columns shrink before the
// project column does, so project names stay readable at 80 columns.
func TestOverviewLayoutKeepsProjectWidth(t *testing.T) {
	projW, cellW := overviewLayout(80, 5)
	if projW != overviewProjMax {
		t.Errorf("project width = %d, want %d at 80 columns", projW, overviewProjMax)
	}
	if cellW >= overviewCellMax {
		t.Errorf("cell width = %d, want it shrunk below %d", cellW, overviewCellMax)
	}
}

// TestOverviewViewFitsWidth: no rendered line may be wider than the terminal.
func TestOverviewViewFitsWidth(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "ein-langer-projektname-ueberlang", time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), 123*time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", time.Date(2026, 6, 10, 9, 0, 0, 0, time.Local), 2*time.Hour),
		// Several years of tracked time: "123456h 00m" is wider than a narrow
		// value column, so the cell content has to be cut, not just padded.
		mkFrame("c3333333333333333333333333333333", "langlaeufer", time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 123456*time.Hour),
	}
	// A running timer adds the note below the table, which must fit too.
	state := &watson.State{Project: "ein-langer-projektname-ueberlang", Start: now.Add(-3 * time.Hour), Tags: []string{}}
	for _, width := range []int{120, 94, 80, 70} {
		for _, st := range []*watson.State{nil, state} {
			out := overviewView(frames, st, time.Monday, now, width)
			for _, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d: line is %d wide: %q", width, w, line)
				}
			}
		}
	}
}

// TestOverviewShortTitles: narrow value columns fall back to short headers
// instead of cutting "letzter Monat" mid-word.
func TestOverviewShortTitles(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	wide := overviewView(nil, nil, time.Monday, now, 200)
	if !strings.Contains(wide, "letzter Monat") {
		t.Errorf("wide view must use full titles:\n%s", wide)
	}
	narrow := overviewView(nil, nil, time.Monday, now, 80)
	if !strings.Contains(narrow, "Vormonat") {
		t.Errorf("narrow view must use short titles:\n%s", narrow)
	}
	if strings.Contains(narrow, "letzter Monat") {
		t.Errorf("narrow view must not use full titles:\n%s", narrow)
	}
}

// TestOverviewCountsRunningTimer: a running timer counts towards its project
// up to now, otherwise the billing sums lag behind reality.
func TestOverviewCountsRunningTimer(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), time.Hour),
	}
	state := &watson.State{Project: "beta", Start: now.Add(-2 * time.Hour)}

	rows, totals := buildOverview(withRunning(frames, state, now), cols, time.Monday)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (alpha + running beta)", len(rows))
	}
	if rows[0].project != "beta" || rows[0].cells[0] != 2*time.Hour {
		t.Errorf("running project row = %s %v, want beta 2h", rows[0].project, rows[0].cells)
	}
	if totals[4] != 3*time.Hour {
		t.Errorf("grand total = %v, want 3h", totals[4])
	}

	out := overviewView(frames, state, time.Monday, now, 120)
	if !strings.Contains(out, "beta") {
		t.Errorf("view must list the running project:\n%s", out)
	}
	if !strings.Contains(out, "läuft") {
		t.Errorf("view must flag that a running timer is included:\n%s", out)
	}
}

// TestWithRunningNoState leaves the frames untouched when nothing runs.
func TestWithRunningNoState(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-time.Hour), time.Hour),
	}
	got := withRunning(frames, nil, now)
	if len(got) != 1 || got[0].Project != "alpha" {
		t.Errorf("withRunning(nil state) = %+v", got)
	}
}

// TestStatusLeftPerMode: the status bar must describe the view it belongs to,
// not always the list period.
func TestStatusLeftPerMode(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("o"))
	if left := app.statusLeft(); !strings.Contains(left, "Übersicht") {
		t.Errorf("overview status = %q, want it to mention Übersicht", left)
	}
	app.Update(key("esc"))
	app.Update(key("r"))
	app.report.per = period{unit: unitMonth, ref: time.Date(2026, 7, 22, 0, 0, 0, 0, time.Local)}
	left := app.statusLeft()
	if !strings.Contains(left, "Report") || !strings.Contains(left, "Juli 2026") {
		t.Errorf("report status = %q, want Report and the report period", left)
	}
}

// TestTruncateGuard: truncate must not panic on degenerate widths.
func TestTruncateGuard(t *testing.T) {
	for _, max := range []int{0, -1, 1} {
		got := truncate("projekt", max)
		if lipgloss.Width(got) > max && max > 0 {
			t.Errorf("truncate(%q, %d) = %q", "projekt", max, got)
		}
	}
}
