package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// TestOverviewFitNeverExceedsWidth: whatever overviewFit keeps has to fit the
// width it was given, down to the narrowest terminal where a project column and
// one value column still go side by side (11 columns). The property is what
// protects the numbers: as long as the table fits, no line is ever clipped, and
// an unclipped right-aligned cell cannot lose its least significant digits.
func TestOverviewFitNeverExceedsWidth(t *testing.T) {
	const n = 5
	for width := 11; width <= 140; width++ {
		keep, dropped, projW, cellW := overviewFit(width, n)
		if len(keep) == 0 {
			t.Fatalf("width %d: every column dropped", width)
		}
		if keep[len(keep)-1] != n-1 {
			t.Errorf("width %d: kept %v, gesamt (%d) must be the last column standing",
				width, keep, n-1)
		}
		if total := projW + len(keep)*(cellW+1); total > width {
			t.Errorf("width %d: %d kept columns need %d columns (proj=%d cell=%d)",
				width, len(keep), total, projW, cellW)
		}
		if cellW < overviewCellMin {
			t.Errorf("width %d: cell width %d below the minimum %d — a value column "+
				"must be dropped, never squeezed", width, cellW, overviewCellMin)
		}
		if len(keep)+len(dropped) != n {
			t.Errorf("width %d: %d kept + %d dropped != %d columns", width, len(keep), len(dropped), n)
		}
	}
}

// TestOverviewRenderedLinesFitEveryWidth walks the rendered lines instead of the
// layout arithmetic. TestOverviewFitNeverExceedsWidth pins that the columns fit
// the width, and that property does not imply the rendered one: the totals row
// wrote its label with a bare %-*s, so from width 15 down it ran one to five
// columns past the table and fitBody ate the digits of the single number an
// invoice is copied from — "Gesamt    4h 02" at 15, "Gesamt    4" at 11 — while
// the arithmetic test stayed green.
//
// No running timer: its note is prose whose second line fitBody still cuts below
// ~50 columns (recorded in task-4-fixes-report.md), and this sweep is about the
// table. Frames are non-empty because the "keine Frames" hint is not width-bound
// either and would fail the sweep for a reason that is not the finding.
func TestOverviewRenderedLinesFitEveryWidth(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	monday := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "kunde-a", monday, 4*time.Hour+2*time.Minute),
		mkFrame("b2222222222222222222222222222222", "schnaq", monday.Add(6*time.Hour), 30*time.Minute),
	}
	cols := overviewColumns(now, time.Monday)
	_, totals := buildOverview(frames, cols, time.Monday)

	for width := 11; width <= 140; width++ {
		out := overviewView(frames, nil, time.Monday, now, width)
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: rendered line %d is %d wide: %q", width, i, w, line)
			}
		}
		// The totals row rebuilt from the layout: a substring check on "4h 32m"
		// alone would pass on a row whose label overflowed, because the same value
		// stands in several columns.
		keep, _, projW, cellW := overviewFit(width, len(cols))
		want := fmt.Sprintf("%-*s", projW, truncate("Gesamt", projW))
		for _, i := range keep {
			want += fmt.Sprintf(" %*s", cellW, truncate(cellDuration(totals[i]), cellW))
		}
		if !strings.Contains(out, want) {
			t.Errorf("width %d: totals row is not rendered as computed:\nwant %q\ngot\n%s",
				width, want, out)
		}
	}
}

// TestOverviewNarrowKeepsExactDurations is the regression for the worst defect
// this view can have: at 60 columns the table used to run two columns past the
// terminal, App.View clipped the overhang, and because the cells are
// right-aligned the clip ate the last digits of a duration without leaving an
// ellipsis — "4h 02m" rendered as "4h 0". Every value that is on screen must be
// exactly what buildOverview computed, so the assertions are whole lines
// rebuilt from its output, not substring checks: a project's row carries the
// same value in several columns, so "4h 02m appears somewhere" passes even when
// the wrong columns are shown.
func TestOverviewNarrowKeepsExactDurations(t *testing.T) {
	const width = 60
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	monday := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "kunde-a", monday, 4*time.Hour+2*time.Minute),
		mkFrame("b2222222222222222222222222222222", "schnaq", monday.Add(6*time.Hour), 30*time.Minute),
	}
	cols := overviewColumns(now, time.Monday)
	rows, totals := buildOverview(frames, cols, time.Monday)
	keep, _, projW, cellW := overviewFit(width, len(cols))

	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: width, Height: 30})
	app.frames = frames
	app.now = now
	app.mode = modeOverview
	out := app.View()

	row := func(label string, cells []time.Duration) string {
		s := fmt.Sprintf("%-*s", projW, truncate(label, projW))
		for _, i := range keep {
			s += fmt.Sprintf(" %*s", cellW, truncate(cellDuration(cells[i]), cellW))
		}
		return s
	}
	for _, r := range rows {
		if want := row(r.project, r.cells); !strings.Contains(out, want) {
			t.Errorf("row %q is not rendered as computed:\nwant %q\ngot\n%s", r.project, want, out)
		}
	}
	if want := row("Gesamt", totals); !strings.Contains(out, want) {
		t.Errorf("total row is not rendered as computed:\nwant %q\ngot\n%s", want, out)
	}
	// The digits themselves, spelled out: a clipped cell loses the tail, so
	// these are the exact strings an invoice is written from.
	for _, want := range []string{"4h 02m", "30m", "4h 32m"} {
		if !strings.Contains(out, want) {
			t.Errorf("value %q missing from the overview:\n%s", want, out)
		}
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line %d is %d wide, want <= %d: %q", i, w, width, line)
		}
	}
}

// TestOverviewDropOrderIsPriority pins the sequence in which value columns are
// given up. Only "gesamt survives last" was pinned, so the middle of the order
// was free: mutating overviewDropOrder from {1,3,2,0} to {1,2,3,0} — giving up
// dieser Monat before letzter Monat, the wrong way round for someone writing last
// month's invoice — left the whole suite green.
func TestOverviewDropOrderIsPriority(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	// The documented priority: what an invoice needs least goes first. gesamt is
	// not in it, so it is the column that always survives.
	want := []string{"letzte Woche", "letzter Monat", "dieser Monat", "diese Woche"}

	got := make([]string, 0, len(overviewDropOrder))
	for _, i := range overviewDropOrder {
		got = append(got, cols[i].title)
	}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Errorf("overviewDropOrder = %v, want %v", got, want)
	}

	// And behaviourally, which is what someone reading the table sees: whatever
	// overviewFit gave up at a width has to be a prefix of that priority. Between
	// 42 and 51 columns exactly two are dropped — that is the width where a swapped
	// middle pair shows.
	for width := 11; width <= 140; width++ {
		_, dropped, _, _ := overviewFit(width, len(cols))
		if len(dropped) > len(want) {
			t.Fatalf("width %d: %d of %d value columns dropped", width, len(dropped), len(want))
		}
		expect := map[string]bool{}
		for _, name := range want[:len(dropped)] {
			expect[name] = true
		}
		for _, i := range dropped {
			if !expect[cols[i].title] {
				t.Errorf("width %d: with %d columns dropped they must be %v, not %q",
					width, len(dropped), want[:len(dropped)], cols[i].title)
			}
		}
	}
}

// TestOverviewNamesDroppedColumns: a column that does not fit is dropped whole,
// so the view has to say so. Otherwise an empty screen reads as "no time booked
// last week" when it means "last week is not on screen".
func TestOverviewNamesDroppedColumns(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "kunde-a",
			time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), 4*time.Hour),
	}
	cols := overviewColumns(now, time.Monday)

	// Wide enough for all five: no note at all.
	if out := overviewView(frames, nil, time.Monday, now, 120); strings.Contains(out, "zu schmal") {
		t.Errorf("nothing is dropped at 120 columns, so there must be no note:\n%s", out)
	}
	// At 60 exactly one column goes, and the note fits a line, so it has to
	// stand in the rendered view word for word.
	if out := overviewView(frames, nil, time.Monday, now, 60); !strings.Contains(out, "zu schmal für: letzte Woche") {
		t.Errorf("60 columns must name the dropped column:\n%s", out)
	}
	// Narrower still: the note wraps, so the column names are asserted against
	// droppedNote — the same reason TestRunningNoteNamesColumns goes to the
	// note function instead of the rendered table.
	for _, width := range []int{60, 45, 30} {
		_, dropped, _, _ := overviewFit(width, len(cols))
		if len(dropped) == 0 {
			t.Fatalf("width %d: expected the layout to drop a column", width)
		}
		note := droppedNote(cols, dropped)
		for _, i := range dropped {
			if !strings.Contains(note, cols[i].title) {
				t.Errorf("width %d: note %q does not name the dropped column %q",
					width, note, cols[i].title)
			}
		}
		if out := overviewView(frames, nil, time.Monday, now, width); !strings.Contains(out, "zu schmal für:") {
			t.Errorf("width %d: dropped %v without saying so:\n%s", width, dropped, out)
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
	// 60 is the width below which all five value columns no longer fit their own
	// minimums, so it is where the table has to give a column up instead of
	// letting the line run past the terminal and be clipped.
	for _, width := range []int{120, 94, 80, 70, 60} {
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

// TestHeaderFieldsPerModeContext: the header must describe the view it belongs
// to, not always the list period.
func TestHeaderFieldsPerModeContext(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("o"))
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Übersicht") {
		t.Errorf("overview header = %q, want it to mention Übersicht", got)
	}
	app.Update(key("esc"))
	app.Update(key("r"))
	app.report.per = period{unit: unitMonth, ref: time.Date(2026, 7, 22, 0, 0, 0, 0, time.Local)}
	got := renderFieldsFlat(app.headerFields())
	if !strings.Contains(got, "Report") || !strings.Contains(got, "Juli 2026") {
		t.Errorf("report header = %q, want Report and the report period", got)
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

// TestOverviewDropNoteSurvivesShortTerminals: fitBody keeps the head of the
// body, so a note below the table drops off the screen as soon as there are more
// projects than rows — and then a missing column is silent again, this time
// because of the height rather than the width. That is why the note sits above
// the table: cut rows are obvious to whoever booked them, a cut note is not.
func TestOverviewDropNoteSurvivesShortTerminals(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	monday := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	var frames []watson.Frame
	for i := 0; i < 9; i++ {
		frames = append(frames, mkFrame(
			strings.Repeat("a", 31)+string(rune('a'+i)), "kunde-"+string(rune('a'+i)),
			monday.Add(time.Duration(i)*time.Hour), 30*time.Minute))
	}
	for _, height := range []int{30, 20, 15, 12} {
		app := newTestApp(t)
		app.Update(tea.WindowSizeMsg{Width: 60, Height: height})
		app.frames = frames
		app.now = now
		app.mode = modeOverview
		out := app.View()
		if !strings.Contains(out, "zu schmal für") {
			t.Errorf("60x%d: the note about the dropped column must survive the "+
				"height cut:\n%s", height, out)
		}
	}
}
