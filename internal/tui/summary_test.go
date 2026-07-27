package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// kinds lists the row kinds in order, for readable assertions.
func kinds(rows []row) []rowKind {
	out := make([]rowKind, len(rows))
	for i, r := range rows {
		out[i] = r.kind
	}
	return out
}

// titles lists the titles of the rows of one kind.
func titles(rows []row, k rowKind) []string {
	var out []string
	for _, r := range rows {
		if r.kind == k {
			out = append(out, r.title)
		}
	}
	return out
}

// TestSummaryGroupsDaysProjectsAndTags: the shape of the view.
func TestSummaryGroupsDaysProjectsAndTags(t *testing.T) {
	// Monday 2026-07-20, week runs 20.–26.
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), time.Hour, "call"),
		mkFrame("c3333333333333333333333333333333", "alpha", mon.AddDate(0, 0, 1), 30*time.Minute, "docs"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)

	days := titles(rows, rowDayHeader)
	if len(days) != 7 {
		t.Fatalf("a week has seven day rows, got %d: %v", len(days), days)
	}
	if days[0] != "Monday, 2026-07-20" {
		t.Errorf("first day = %q", days[0])
	}

	// A blank line stands before every day but the first, so the blocks read
	// apart. Pinned here rather than in the one-day test below, which never has
	// a second day header to be separated from.
	if rows[0].kind == rowBlank {
		t.Errorf("no blank row stands before the first day")
	}
	for i, r := range rows {
		if r.kind == rowDayHeader && i > 0 && rows[i-1].kind != rowBlank {
			t.Errorf("day header %q at row %d is not preceded by a blank", r.title, i)
		}
	}

	// Monday: alpha (3h) before beta (1h), each followed by its tag.
	var mondayBlock []row
	for i, r := range rows {
		if r.kind == rowDayHeader && r.title == "Monday, 2026-07-20" {
			for _, next := range rows[i+1:] {
				if next.kind == rowDayHeader {
					break
				}
				mondayBlock = append(mondayBlock, next)
			}
			break
		}
	}
	projects := titles(mondayBlock, rowProject)
	if len(projects) != 2 || projects[0] != "alpha" || projects[1] != "beta" {
		t.Errorf("monday projects = %v, want [alpha beta] sorted by duration", projects)
	}
	if got := titles(mondayBlock, rowTag); len(got) != 2 {
		t.Errorf("monday tags = %v, want one per project", got)
	}
}

// TestSummaryNestsRowsAndSeparatesProjects pins the exact sequence of kinds for
// one day with two projects: the tags sit under their project, a blank line
// separates the project blocks, and none stands before the first day.
func TestSummaryNestsRowsAndSeparatesProjects(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), time.Hour, "call"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)

	want := []rowKind{rowDayHeader, rowProject, rowTag, rowBlank, rowProject, rowTag}
	got := kinds(rows)
	if len(got) != len(want) {
		t.Fatalf("row kinds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row kinds = %v, want %v", got, want)
		}
	}
}

// TestSummaryShowsEmptyDays: a period is legible only if its gaps are visible.
func TestSummaryShowsEmptyDays(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)
	var empty int
	for _, r := range rows {
		if r.kind == rowDayHeader && r.total == 0 {
			empty++
		}
	}
	if empty != 6 {
		t.Errorf("six of seven days are empty, got %d", empty)
	}
}

// TestSummaryDayTotalCountsFramesNotTags: a frame with two tags counts once
// towards the day, though it appears under both tags.
func TestSummaryDayTotalCountsFramesNotTags(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 2*time.Hour, "code", "review"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)
	for _, r := range rows {
		if r.kind == rowDayHeader {
			if r.total != 2*time.Hour {
				t.Errorf("day total = %v, want 2h — the tags must not be added up", r.total)
			}
		}
	}
	if got := len(titles(rows, rowTag)); got != 2 {
		t.Errorf("both tags must be listed, got %d", got)
	}
}

// TestSummaryRespectsTheFilter: the same filter the frame list applies.
func TestSummaryRespectsTheFilter(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(2*time.Hour), time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "alpha", nil, mon)
	if got := titles(rows, rowProject); len(got) != 1 || got[0] != "alpha" {
		t.Errorf("filtered projects = %v, want [alpha]", got)
	}
}

// TestSummaryCountsTheRunningTimer: like the report and the overview.
func TestSummaryCountsTheRunningTimer(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	state := &watson.State{Project: "running", Start: now.Add(-time.Hour), Tags: []string{}}
	rows := buildSummaryRows(nil, period{unit: unitDay, ref: now}, time.Monday, "", state, now)
	if got := titles(rows, rowProject); len(got) != 1 || got[0] != "running" {
		t.Errorf("projects = %v, want the running timer", got)
	}
	for _, r := range rows {
		if r.kind == rowDayHeader && r.total != time.Hour {
			t.Errorf("day total = %v, want 1h from the running timer", r.total)
		}
	}
}

// TestSummaryUnboundedSkipsEmptyDays: with no period bounds, a row per day
// since the first frame would be thousands of lines.
func TestSummaryUnboundedSkipsEmptyDays(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.AddDate(-1, 0, 0), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now, time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitAll, ref: now}, time.Monday, "", nil, now)
	if got := len(titles(rows, rowDayHeader)); got != 2 {
		t.Errorf("unitAll must list only days with frames, got %d day rows", got)
	}
}

// TestSummaryIsTheStartingView: the compact view is what one sees first. The
// question watson-tui is opened with is "what did I work on", and the frame
// list only answers "which sessions were there" — which one asks when editing.
func TestSummaryIsTheStartingView(t *testing.T) {
	app := newTestApp(t)
	if !app.list.compact {
		t.Error("watson-tui opens on the summary")
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if out := app.View(); !strings.Contains(out, "Summary") {
		t.Errorf("panel title must say Summary:\n%s", out)
	}
}

// TestFTogglesTheView: f switches, and the footer names the other side. A view
// one cannot get back out of is worse than no second view.
func TestFTogglesTheView(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour, "code"),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday, app.state, app.now)

	if out := app.View(); !strings.Contains(out, "f frames") {
		t.Errorf("the summary offers the frame list:\n%s", out)
	}
	app.Update(key("f"))
	if app.list.compact {
		t.Fatal("f must switch to the frame list")
	}
	out := app.View()
	if !strings.Contains(out, "f summary") {
		t.Errorf("the frame list offers the summary:\n%s", out)
	}
	if !strings.Contains(out, "a111111") {
		t.Errorf("the frame list shows frame IDs:\n%s", out)
	}
	if !strings.Contains(out, "Frames") {
		t.Errorf("the panel title follows the view:\n%s", out)
	}
	app.Update(key("f"))
	if !app.list.compact {
		t.Error("f must switch back")
	}
}

// TestCursorSkipsEverythingButProjects: j/k walk the project rows. Day headers,
// tags and the blank lines between blocks are not something one acts on, so the
// cursor must never come to rest on one.
func TestCursorSkipsEverythingButProjects(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 2*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(3*time.Hour), time.Hour, "call"),
	}
	app.list.per = period{unit: unitDay, ref: now}
	app.list.refresh(app.frames, time.Monday, app.state, app.now)

	if app.list.cursor < 0 {
		t.Fatal("the cursor must find a project row")
	}
	if k := app.list.rows[app.list.cursor].kind; k != rowProject {
		t.Fatalf("cursor starts on kind %d, want a project row", k)
	}
	app.Update(key("j"))
	if k := app.list.rows[app.list.cursor].kind; k != rowProject {
		t.Errorf("j landed on kind %d, want a project row", k)
	}
	// Teeth: a j that never moved would satisfy the check above.
	if app.list.rows[app.list.cursor].title != "beta" {
		t.Errorf("j stayed on %q, want the second project", app.list.rows[app.list.cursor].title)
	}
}

// TestEditingKeysAreInertInTheSummary: enter and d need a frame, and the
// summary selects a project. They are inert of their own accord — selected()
// hands back nothing — and the footer must not advertise them anyway.
func TestEditingKeysAreInertInTheSummary(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday, app.state, app.now)

	app.Update(key("enter"))
	if app.mode != modeList {
		t.Error("enter must do nothing in the summary")
	}
	app.Update(key("d"))
	if app.mode != modeList {
		t.Error("d must do nothing in the summary")
	}
	if out := app.View(); strings.Contains(out, "enter edit") || strings.Contains(out, "d delete") {
		t.Errorf("the summary must not advertise keys that do nothing:\n%s", out)
	}
}

// summaryPanelBody returns the lines inside the body panel, chrome stripped.
//
// The sweep below has to look at the body alone: the header's Total carries the
// same duration, and it is capped by clipWidth rather than laid out by
// summaryLine, so a Contains over the whole rendered frame comes back true for a
// body that lost every number it had. It did — this helper exists because the
// sweep passed against a summaryLine that had stopped truncating labels
// altogether, on the strength of the header alone.
func summaryPanelBody(t *testing.T, out string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "─ Summary ") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatalf("no summary panel in:\n%s", out)
	}
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "╰") {
			return strings.Join(lines[start:i], "\n")
		}
	}
	t.Fatalf("the summary panel is never closed:\n%s", out)
	return ""
}

// TestSummaryNumbersSurviveEveryWidth: the rule the whole project follows — a
// name gives way with an ellipsis, a number never does. The same sweep the
// frame list, the report and the overview each carry.
func TestSummaryNumbersSurviveEveryWidth(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "a-rather-long-project-name", now,
			124*time.Hour+30*time.Minute, "a-long-tag-name"),
	}
	app.list.per = period{unit: unitDay, ref: now}
	for width := 30; width <= 120; width++ {
		app.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		app.list.refresh(app.frames, time.Monday, app.state, app.now)
		out := app.View()
		if !strings.Contains(summaryPanelBody(t, out), "124h 30m") {
			t.Errorf("width %d: the duration was cut:\n%s", width, out)
		}
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d wide", width, i, w)
			}
		}
	}
}

// TestSummaryRowsEndInTheirNumber: every row is laid out to the full width with
// its own total at the right edge, so the numbers stand in a column instead of
// trailing their labels.
//
// Two edges, not one, and that is the design: a day header and a project row end
// in the number itself, a tag row ends in the bracket that closes it, one column
// further out. The tag line is a breakdown of the row above it and reads as one
// because of that offset — see renderRow. Both are asserted, so neither the
// bracket nor the alignment can quietly go.
func TestSummaryRowsEndInTheirNumber(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(4*time.Hour), 45*time.Minute, "call"),
	}
	l := newListModel(now, time.Monday)
	l.per = period{unit: unitDay, ref: now}
	l.refresh(frames, time.Monday, nil, now)
	l.cursor = -1 // nothing styled, so the columns can be counted in plain text

	const width = 60
	seen := map[rowKind]bool{}
	for i, r := range l.rows {
		if r.kind == rowBlank {
			continue
		}
		line := l.renderRow(i, width, 0)
		seen[r.kind] = true
		if w := lipgloss.Width(line); w != width {
			t.Errorf("row %d (kind %d) is %d columns, want %d: %q", i, r.kind, w, width, line)
		}
		// Nothing but the closing bracket may follow the number. A pad after it
		// would take that row's digits out of the column its neighbours use.
		want := formatDuration(r.total)
		if r.kind == rowTag {
			want += "]"
		}
		if !strings.HasSuffix(line, want) {
			t.Errorf("row %d (kind %d) does not end in %q: %q", i, r.kind, want, line)
		}
	}
	// Teeth for the teeth: all three kinds have to have been rendered, or a
	// renderRow that dropped one of them would pass the loop above by never
	// entering it.
	for _, k := range []rowKind{rowDayHeader, rowProject, rowTag} {
		if !seen[k] {
			t.Errorf("the fixture rendered no row of kind %d", k)
		}
	}
}

// TestListDurWidthIgnoresSummaryRows: the duration column is sized off frame
// rows and nothing else.
//
// listDurWidth used to skip day headers and read r.frame.Duration() on
// everything else, which held while frames and headers were the only two kinds.
// A project, tag or blank row carries no frame, and watson.Frame{}.Duration() is
// guarded — so each of them would have measured the zero frame's "0m" and sized
// a column none of them uses. Two columns are never the widest, so nothing on
// screen would have moved; this is the pin that keeps the measure named after
// what it measures rather than after what it happens to exclude.
func TestListDurWidthIgnoresSummaryRows(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	summary := buildSummaryRows(
		[]watson.Frame{mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code")},
		period{unit: unitDay, ref: now}, time.Monday, "", nil, now)
	if got := listDurWidth(summary); got != 0 {
		t.Errorf("summary rows sized the duration column to %d; none of them is a frame", got)
	}
	// Teeth: the same function still measures a frame row, so returning zero
	// unconditionally is not a way to pass.
	frames := buildRows(
		[]watson.Frame{mkFrame("a1111111111111111111111111111111", "alpha", now, 130*time.Hour)},
		period{unit: unitDay, ref: now}, time.Monday, "")
	if got, want := listDurWidth(frames), len("130h 00m"); got != want {
		t.Errorf("frame rows sized the duration column to %d, want %d", got, want)
	}
}

// TestSummaryBucketsFramesByTheirLocalDay: no frame may fall out of the view
// between the day list and the frames bucketed under it.
//
// The two are keyed from different constructions — a period's bounds carry the
// reference time's location, a frame's start is stored in UTC — and the day
// blocks are built by handing each aggregate call one bucket. A key reading the
// stored UTC date instead of the local one files a frame booked just after local
// midnight under the day before, which no day of the period then asks for: the
// frame does not move, it disappears, and the day total is short by its duration
// with nothing on screen saying so. aggregate re-filters what it is handed, so
// it cannot put back what the bucket left out.
//
// time.Local is forced to a zone east of UTC for the duration, because where the
// two agree there is nothing here to test. Restored by defer, the way the colour
// profile is in chrome_test.go — both are global.
func TestSummaryBucketsFramesByTheirLocalDay(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("no zoneinfo for Europe/Berlin: %v", err)
	}
	saved := time.Local
	time.Local = berlin
	defer func() { time.Local = saved }()

	// 00:15 local on Monday is 22:15 UTC on the Sunday before it.
	mon := time.Date(2026, 7, 20, 0, 15, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(23*time.Hour), 30*time.Minute, "call"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)

	var booked time.Duration
	for _, r := range rows {
		if r.kind == rowDayHeader {
			booked += r.total
		}
	}
	if want := 90 * time.Minute; booked != want {
		t.Errorf("the day blocks book %v of %v — a frame fell between the day list "+
			"and the bucket it was filed in", booked, want)
	}
	// Under the right day, not merely somewhere in the week.
	for _, r := range rows {
		if r.kind == rowDayHeader && r.title == "Monday, 2026-07-20" && r.total != 90*time.Minute {
			t.Errorf("Monday books %v, want 1h 30m", r.total)
		}
	}
}
