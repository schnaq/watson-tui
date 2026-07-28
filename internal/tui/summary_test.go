package tui

import (
	"slices"
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

	// A blank line stands between two day blocks, but not between two empty ones.
	// Monday and Tuesday carry frames here, so Tuesday and the Wednesday behind it
	// open with one; Thursday to Sunday each follow an empty day and do not.
	// Pinned here rather than in the one-day test below, which never has a second
	// day header to be separated from.
	if rows[0].kind == rowBlank {
		t.Errorf("no blank row stands before the first day")
	}
	var opened []string
	for i, r := range rows {
		if r.kind == rowDayHeader && i > 0 && rows[i-1].kind == rowBlank {
			opened = append(opened, r.title)
		}
	}
	wantOpened := []string{"Tuesday, 2026-07-21", "Wednesday, 2026-07-22"}
	if !slices.Equal(opened, wantOpened) {
		t.Errorf("blank lines open %v, want %v", opened, wantOpened)
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

// TestSummaryNestsTagsUnderTheirProject pins the exact sequence of kinds for one
// day with two projects: each tag sits under its project, nothing separates the
// two project blocks — the rail does that now — and no blank stands before the
// first day.
func TestSummaryNestsTagsUnderTheirProject(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), time.Hour, "call"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)

	want := []rowKind{rowDayHeader, rowProject, rowTag, rowProject, rowTag}
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

// TestSummaryRowsEndInTheirNumber: every row ends in its own total, so nothing
// stands between the number and the eye running down the column.
//
// Two edges, not one, and that is the design: a day header and a project row end
// in the number itself, a tag row ends in the bracket that closes it, one column
// further out. The numbers themselves still line up — see
// TestSummaryNumbersShareOneColumn — the bracket is the one column a tag row has
// that the others do not, which is what makes it read as a breakdown of the row
// above. Both are asserted, so neither the bracket nor the alignment can quietly
// go.
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
	c := summaryColumns(l.rows, width)
	seen := map[rowKind]bool{}
	for i, r := range l.rows {
		if r.kind == rowBlank {
			continue
		}
		line := l.renderRow(i, width, c)
		seen[r.kind] = true
		if w := lipgloss.Width(line); w > width {
			t.Errorf("row %d (kind %d) is %d columns, want at most %d: %q", i, r.kind, w, width, line)
		}
		// Nothing but the bar may follow the number. A pad between the two would take
		// that row's digits out of the column its neighbours use.
		want := formatDuration(r.total)
		if !strings.HasSuffix(stripBar(line), want) {
			t.Errorf("row %d (kind %d) does not end in %q once its bar is taken off: %q",
				i, r.kind, want, line)
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

// summaryColumn is where one rendered row puts its number and how far that is
// from the end of its name. The number is right-aligned in its column, so it is
// the last digit that stands in the same place on every row, not the first.
type summaryColumn struct {
	kind   rowKind
	numEnd int // the column after the number's last digit
	gap    int // blank columns between the end of the name and the first digit
}

// stripBar takes the bar and the gap in front of it off the end of a rendered row,
// leaving the line to end in its number. The alignment tests all read the number
// off the end, and a day or project row now ends in its bar instead.
func stripBar(line string) string {
	runes := []rune(line)
	end := len(runes)
	for end > 0 && strings.ContainsRune(barGlyphs, runes[end-1]) {
		end--
	}
	for end > 0 && runes[end-1] == ' ' {
		end--
	}
	return string(runes[:end])
}

// summaryColumnsOf renders every non-blank row of l into a body of width columns
// and reads the two numbers off the rendered line — not off the widths
// summaryColumns computed, which would only restate the arithmetic under test
// back at itself.
func summaryColumnsOf(t *testing.T, l *listModel, width int) []summaryColumn {
	t.Helper()
	l.cursor = -1 // nothing styled, so the columns can be counted in plain text
	c := summaryColumns(l.rows, width)
	var out []summaryColumn
	for i, r := range l.rows {
		if r.kind == rowBlank {
			continue
		}
		line := l.renderRow(i, width, c)
		if strings.Contains(line, "\x1b") {
			t.Fatalf("row %d is styled (CLICOLOR_FORCE?); its columns cannot be counted: %q", i, line)
		}
		runes := []rune(stripBar(line))
		value := summaryValue(r)
		end := len(runes)
		start := end - len([]rune(value))
		if start < 0 || string(runes[start:end]) != value {
			t.Fatalf("row %d does not end in its number %q: %q", i, value, line)
		}
		name := start
		for name > 0 && runes[name-1] == ' ' {
			name--
		}
		out = append(out, summaryColumn{kind: r.kind, numEnd: end, gap: start - name})
	}
	if len(out) == 0 {
		t.Fatal("the fixture rendered no rows at all")
	}
	return out
}

// summaryFixture is a week with two worked days and five empty ones, which is
// what the alignment tests need: three kinds of row, more than one day block,
// and a day whose total is the dash.
func summaryFixture(t *testing.T) *listModel {
	t.Helper()
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), 45*time.Minute, "call"),
		mkFrame("c3333333333333333333333333333333", "alpha", mon.AddDate(0, 0, 2), 90*time.Minute, "docs"),
	}
	l := newListModel(mon, time.Monday)
	l.per = period{unit: unitWeek, ref: mon}
	l.refresh(frames, time.Monday, nil, mon)
	return &l
}

// TestSummaryNumbersFollowTheirNames: the duration column is sized to the
// longest name of the view, not to the terminal.
//
// The numbers used to be right-aligned against the body, which on a hundred
// columns put "4h 00m" sixty columns away from the "kunde-a" it belonged to.
// `watson aggregate` writes the number after the name, and so does this. The
// longest name of the view is the one that ends closest to the column — it is
// what the column was sized from — so the smallest gap is what is asserted here,
// and under the old right-alignment it was around sixty.
func TestSummaryNumbersFollowTheirNames(t *testing.T) {
	const width = 100
	cols := summaryColumnsOf(t, summaryFixture(t), width)
	closest := width
	for _, c := range cols {
		closest = min(closest, c.gap)
	}
	if closest >= 6 {
		t.Errorf("the longest name of the view ends %d columns before its number at width %d — "+
			"the column is sized to the terminal, not to the names", closest, width)
	}
}

// TestSummaryCursorDoesNotMoveTheNumbers: the marker replaces the first column
// of the project row's indent instead of being prepended, so the numbers stand
// still while the cursor walks the view. Prepended, every cursor row would be one
// column longer than its neighbours and the column would wobble as j is held
// down — the one thing an aligned column cannot survive.
//
// Asserted rather than left to the golden file, because summaryRow builds that
// row's lead two different ways and only this says the two are the same width.
func TestSummaryCursorDoesNotMoveTheNumbers(t *testing.T) {
	l := summaryFixture(t)
	const width = 100
	c := summaryColumns(l.rows, width)
	var checked int
	for i, r := range l.rows {
		if r.kind != rowProject {
			continue
		}
		l.cursor = -1
		plain := l.renderRow(i, width, c)
		l.cursor = i
		marked := l.renderRow(i, width, c)
		if strings.Contains(marked, "\x1b") {
			t.Fatalf("row %d is styled (CLICOLOR_FORCE?); its columns cannot be counted: %q", i, marked)
		}
		if !strings.Contains(marked, selectionMarker) {
			t.Fatalf("the cursor row %q carries no %q marker", marked, selectionMarker)
		}
		// The marker takes the rail's column, so the two rows differ in that one
		// column and are identical behind it — which is what says it replaced the
		// rail glyph rather than pushing the line right.
		if lipgloss.Width(marked) != lipgloss.Width(plain) {
			t.Errorf("row %d is %d columns under the cursor and %d without it:\n  plain  %q\n  cursor %q",
				i, lipgloss.Width(marked), lipgloss.Width(plain), plain, marked)
		}
		if rest, plainRest := string([]rune(marked)[1:]), string([]rune(plain)[1:]); rest != plainRest {
			t.Errorf("row %d shifts under the cursor:\n  plain  %q\n  cursor %q", i, plain, marked)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("the fixture rendered no project row for the cursor to sit on")
	}
}

// TestSummaryNumbersShareOneColumn: all of them, across the day blocks and
// across the three kinds of row. A column sized per day or per kind would be
// ragged, which is harder to read down than a column that stands too far right.
func TestSummaryNumbersShareOneColumn(t *testing.T) {
	for _, width := range []int{40, 60, 100} {
		cols := summaryColumnsOf(t, summaryFixture(t), width)
		seen := map[rowKind]bool{}
		want := cols[0].numEnd
		for _, c := range cols {
			seen[c.kind] = true
			if c.numEnd != want {
				t.Errorf("width %d: a row of kind %d ends its number in column %d, "+
					"the first row in column %d", width, c.kind, c.numEnd, want)
			}
		}
		// Teeth: all three kinds have to have been rendered, or a renderRow that
		// dropped one would agree with itself about the one column it has left.
		for _, k := range []rowKind{rowDayHeader, rowProject, rowTag} {
			if !seen[k] {
				t.Errorf("width %d: the fixture rendered no row of kind %d", width, k)
			}
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

// TestSummaryRailBracketsTheDayBlock: the rail glyph follows where a row sits in
// its day block, which is what makes a block read as a block. An empty day has
// nothing to bracket and gets no rail.
func TestSummaryRailBracketsTheDayBlock(t *testing.T) {
	rows := []row{
		{kind: rowDayHeader, title: "Monday, 2026-07-20", total: 4 * time.Hour},
		{kind: rowProject, title: "alpha", total: 3 * time.Hour},
		{kind: rowTag, title: "code", total: 3 * time.Hour},
		{kind: rowProject, title: "beta", total: time.Hour},
		{kind: rowTag, title: "call", total: time.Hour},
		{kind: rowBlank},
		{kind: rowDayHeader, title: "Tuesday, 2026-07-21"},
		{kind: rowDayHeader, title: "Wednesday, 2026-07-22"},
	}
	want := []string{"╭", "│", "│", "│", "╰", "", "", ""}
	for i, w := range want {
		if got := summaryRail(rows, i); got != w {
			t.Errorf("row %d (%q): rail = %q, want %q", i, rows[i].title, got, w)
		}
	}
}

// TestSummaryRailClosesOnAProjectWithoutTags: the closing glyph follows the last
// row of the block whatever kind it is. A project carrying no tags is the last
// row itself, and the bracket still has to close on it.
func TestSummaryRailClosesOnAProjectWithoutTags(t *testing.T) {
	rows := []row{
		{kind: rowDayHeader, title: "Monday, 2026-07-20", total: time.Hour},
		{kind: rowProject, title: "alpha", total: time.Hour},
	}
	if got := summaryRail(rows, 1); got != "╰" {
		t.Errorf("rail on the last row of the view = %q, want %q", got, "╰")
	}
}

// TestSummaryRunsEmptyDaysTogether: a blank line stands between two day blocks
// unless both are empty. A quiet second half of the week otherwise spends ten
// lines on five dashes.
func TestSummaryRunsEmptyDaysTogether(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)

	blanks := 0
	for _, r := range rows {
		if r.kind == rowBlank {
			blanks++
		}
	}
	// One: between Monday's block and Tuesday. The five days after Tuesday are
	// empty and follow an empty day, so none of them opens with one.
	if blanks != 1 {
		t.Errorf("a week with one booked day holds %d blank rows, want 1", blanks)
	}
}

// TestSummaryRailWithAFilteredEmptyDayInTheMiddle: a filter can empty a day that
// the period still lists, which is the only way a day header without a rail comes
// to stand between two bracketed blocks. The bounded-period cases put their empty
// days at the end, so they never exercise it.
func TestSummaryRailWithAFilteredEmptyDayInTheMiddle(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.AddDate(0, 0, 1), time.Hour, "call"),
		mkFrame("c3333333333333333333333333333333", "alpha", mon.AddDate(0, 0, 2), time.Hour, "docs"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "alpha", nil, mon)

	// Monday and Wednesday are bracketed, Tuesday is empty between them, and the
	// four days after Wednesday carry neither a rail nor a blank line.
	var got []string
	for i, r := range rows {
		if r.kind == rowDayHeader {
			got = append(got, summaryRail(rows, i))
		}
	}
	want := []string{railOpen, "", railOpen, "", "", "", ""}
	if len(got) != len(want) {
		t.Fatalf("got %d day headers, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("day header %d: rail = %q, want %q", i, got[i], want[i])
		}
	}
	// Three blanks: before Tuesday, before Wednesday and before Thursday — each of
	// those three neighbours a day that carries frames. Friday, Saturday and
	// Sunday follow an empty day and open with nothing.
	blanks := 0
	for _, r := range rows {
		if r.kind == rowBlank {
			blanks++
		}
	}
	if blanks != 3 {
		t.Errorf("holds %d blank rows, want 3 (before Tuesday, Wednesday and Thursday)", blanks)
	}
}

// TestSummaryRailFollowsTheBlockNotTheTotal: a frame that starts and stops at the
// same moment books nothing, so its day header reads "–" — but the day still has
// a project row under it. The rail has to bracket what is there, or the corner
// goes missing above a block that draws one.
//
// A degenerate frame, and exactly the kind of input that makes two conditions
// derived from different things disagree: "shows a dash" is about the total,
// "opens a block" is about the rows.
func TestSummaryRailFollowsTheBlockNotTheTotal(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 0, "code"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)

	if len(rows) < 2 || rows[0].kind != rowDayHeader || rows[1].kind != rowProject {
		t.Fatalf("expected a day header with a project under it, got %v", kinds(rows))
	}
	if rows[0].total != 0 {
		t.Fatalf("the day books %v, want nothing — the fixture is wrong", rows[0].total)
	}
	if got := summaryRail(rows, 0); got != railOpen {
		t.Errorf("rail on a day header with rows under it = %q, want %q", got, railOpen)
	}
}

// TestSummaryPeakIsTheLargestDayTotal: the scale every bar in the view is a share
// of. Day totals only — a project or tag row can never exceed the day it sits in,
// so measuring them would only invite a later kind of row to move the scale.
func TestSummaryPeakIsTheLargestDayTotal(t *testing.T) {
	rows := []row{
		{kind: rowDayHeader, total: 2 * time.Hour},
		{kind: rowProject, total: 2 * time.Hour},
		{kind: rowBlank},
		{kind: rowDayHeader, total: 5 * time.Hour},
		{kind: rowProject, total: 5 * time.Hour},
	}
	if got := summaryPeak(rows); got != 5*time.Hour {
		t.Errorf("peak = %v, want 5h", got)
	}
}

// TestSummaryPeakOfAnEmptyPeriodIsZero: a period without a single frame — or a
// filter without a match — has no scale, and the division that would size a bar
// has to be guarded somewhere.
func TestSummaryPeakOfAnEmptyPeriodIsZero(t *testing.T) {
	rows := []row{{kind: rowDayHeader}, {kind: rowBlank}, {kind: rowDayHeader}}
	if got := summaryPeak(rows); got != 0 {
		t.Errorf("peak = %v, want 0", got)
	}
}

// TestSummaryBarWidthTakesWhatIsLeftUpToTheCap: the bar is computed after the
// label and number columns stand, so it can only have what they left. Capped,
// because 150 columns of bar on a wide terminal is not a reading aid.
func TestSummaryBarWidthTakesWhatIsLeftUpToTheCap(t *testing.T) {
	if got := summaryBarWidth(100, 23, 6); got != summaryBarMax {
		t.Errorf("bar width at 100 columns = %d, want the cap %d", got, summaryBarMax)
	}
	// Between the floor and the cap the bar takes exactly what is left: 52 columns
	// less the two of gutter, the label's 23, the separator, the number's 6 and the
	// two of gap.
	if got := summaryBarWidth(52, 23, 6); got != 18 {
		t.Errorf("bar width at 52 columns = %d, want 18", got)
	}
}

// TestSummaryBarWidthDropsWholeRatherThanLie: under summaryBarMin a bar can no
// longer tell a tenth from a half, and a bar that shows the wrong proportion is
// worse than none — the rule the frame list's ID column follows for the same kind
// of reason.
func TestSummaryBarWidthDropsWholeRatherThanLie(t *testing.T) {
	// 44 columns is the narrowest that still carries one, at exactly the minimum.
	if got := summaryBarWidth(44, 23, 6); got != summaryBarMin {
		t.Errorf("bar width at 44 columns = %d, want %d", got, summaryBarMin)
	}
	if got := summaryBarWidth(43, 23, 6); got != 0 {
		t.Errorf("bar width at 43 columns = %d, want 0 — no bar at all", got)
	}
}

// TestSummaryBarFillsTheWidthBetweenItsTwoParts: filled part and track come back
// apart so that renderRow can colour them differently, and together they are
// exactly the width they were given — the line's width is arithmetic on plain
// text, before any style is applied.
func TestSummaryBarFillsTheWidthBetweenItsTwoParts(t *testing.T) {
	for _, tc := range []struct{ d, peak time.Duration }{
		{8 * time.Hour, 8 * time.Hour},
		{4 * time.Hour, 8 * time.Hour},
		{30 * time.Minute, 8 * time.Hour},
		{time.Minute, 100 * time.Hour},
	} {
		filled, track := summaryBar(tc.d, tc.peak, 16)
		if got := lipgloss.Width(filled) + lipgloss.Width(track); got != 16 {
			t.Errorf("%v of %v: filled %q plus track %q is %d columns, want 16",
				tc.d, tc.peak, filled, track, got)
		}
	}
}

// TestSummaryBarUsesEighthsToPlaceAPartialColumn: full blocks alone would round a
// day and a half of work down to the same length as a day of it.
func TestSummaryBarUsesEighthsToPlaceAPartialColumn(t *testing.T) {
	// Half of one column: 30 minutes of an eight-hour peak over eight columns.
	filled, _ := summaryBar(30*time.Minute, 8*time.Hour, 8)
	if filled != "▌" {
		t.Errorf("filled = %q, want a half block", filled)
	}
	// Four columns and a half.
	filled, _ = summaryBar(4*time.Hour+30*time.Minute, 8*time.Hour, 8)
	if filled != "████▌" {
		t.Errorf("filled = %q, want four blocks and a half", filled)
	}
}

// TestSummaryBarShowsAnythingAboveZero: a minute against a hundred hours rounds to
// no columns at all, and a row rendering as nothing claims nothing was booked. The
// floor is what keeps the small days of the "all" period on the screen.
func TestSummaryBarShowsAnythingAboveZero(t *testing.T) {
	filled, _ := summaryBar(time.Minute, 100*time.Hour, 16)
	if filled == "" {
		t.Error("a minute of a hundred hours renders as nothing at all")
	}
}

// TestSummaryBarWithoutAScaleIsNothing: no frames means no peak, and a bar is a
// share of something. Also the guard on the division.
func TestSummaryBarWithoutAScaleIsNothing(t *testing.T) {
	filled, track := summaryBar(time.Hour, 0, 16)
	if filled != "" || track != "" {
		t.Errorf("filled %q track %q, want both empty", filled, track)
	}
	filled, track = summaryBar(0, 8*time.Hour, 16)
	if filled != "" || track != "" {
		t.Errorf("a day that booked nothing draws filled %q track %q, want both empty", filled, track)
	}
}

// TestSummaryMarksEveryRowOfTodaysBlock: the mark sits on every row of the current
// day, not only its header, because the rail of the whole block is coloured from
// it. Deriving it per row beats making the renderer search back to the nearest day
// header to pick a colour.
func TestSummaryMarksEveryRowOfTodaysBlock(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	tue := mon.AddDate(0, 0, 1)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", tue, time.Hour, "call"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil,
		tue.Add(3*time.Hour))

	marked := 0
	block := ""
	for i, r := range rows {
		if r.kind == rowDayHeader {
			block = r.title
		}
		if r.kind == rowBlank {
			continue
		}
		want := block == "Tuesday, 2026-07-21"
		if r.today != want {
			t.Errorf("row %d (%q under %q): today = %v, want %v", i, r.title, block, r.today, want)
		}
		if r.today {
			marked++
		}
	}
	// Tuesday's header, its one project and its one tag.
	if marked != 3 {
		t.Errorf("%d rows carry the mark, want 3", marked)
	}
}

// TestSummaryMarksNothingInAPeriodWithoutToday: looking at last week, no day is
// the current one, so no row may be emphasised as such.
func TestSummaryMarksNothingInAPeriodWithoutToday(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour, "code"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil,
		mon.AddDate(0, 0, 14))

	for i, r := range rows {
		if r.today {
			t.Errorf("row %d (%q) is marked as today in a week that does not hold it", i, r.title)
		}
	}
}

// barGlyphs is every rune a bar may be built from, for asserting that a line ends
// in one without restating how it is drawn.
const barGlyphs = barFull + barTrack + "▏▎▍▌▋▊▉"

// summaryTestModel is a summary of the given frames, rendered plain: the cursor is
// parked off the list so no row is styled and columns can be counted in the text.
func summaryTestModel(frames []watson.Frame, now time.Time) listModel {
	l := newListModel(now, time.Monday)
	l.per = period{unit: unitDay, ref: now}
	l.refresh(frames, time.Monday, nil, now)
	l.cursor = -1
	return l
}

// TestSummaryRowsOpenWithTheRail: the rail is the first thing on the line, in a
// column of its own — that is what makes a day block read as a block rather than
// as three indent levels that happen to follow each other.
func TestSummaryRowsOpenWithTheRail(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code"),
	}, now)

	c := summaryColumns(l.rows, 60)
	for i, r := range l.rows {
		if r.kind == rowBlank {
			continue
		}
		line := l.renderRow(i, 60, c)
		want := summaryRail(l.rows, i)
		if want == "" {
			want = " "
		}
		if got := string([]rune(line)[0]); got != want {
			t.Errorf("row %d (%q) opens with %q, want the rail %q", i, r.title, got, want)
		}
	}
}

// TestSummaryTagRowsHangFromABranch: a tag row says which project it breaks down
// and whether it is the last of them. The square bracket it used to carry did the
// first and not the second.
func TestSummaryTagRowsHangFromABranch(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code", "review"),
	}, now)

	c := summaryColumns(l.rows, 60)
	var branches []string
	for i, r := range l.rows {
		line := l.renderRow(i, 60, c)
		if strings.ContainsAny(line, "[]") {
			t.Errorf("row %d still carries a bracket: %q", i, line)
		}
		if r.kind == rowTag {
			switch {
			case strings.Contains(line, tagLast):
				branches = append(branches, "last")
			case strings.Contains(line, tagBranch):
				branches = append(branches, "branch")
			default:
				t.Errorf("tag row %d hangs from nothing: %q", i, line)
			}
		}
	}
	// Two tags on one project: the first branches, the second closes.
	if want := []string{"branch", "last"}; !slices.Equal(branches, want) {
		t.Errorf("tag glyphs = %v, want %v", branches, want)
	}
}

// TestSummaryDropsTheEmDash: the dash used to set a number off from its name. The
// fixed number column does that, and the rail does it one level up, so the dash
// only repeated them — at the price of two columns the bar wanted.
func TestSummaryDropsTheEmDash(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code"),
	}, now)

	c := summaryColumns(l.rows, 60)
	for i := range l.rows {
		if line := l.renderRow(i, 60, c); strings.Contains(line, "—") {
			t.Errorf("row %d still carries an em dash: %q", i, line)
		}
	}
}

// TestSummaryDayAndProjectRowsEndInTheirBar: the bar is the last thing on the line
// and fills its column exactly, so the rows line up down the right edge as well as
// the left.
func TestSummaryDayAndProjectRowsEndInTheirBar(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(4*time.Hour), time.Hour, "call"),
	}, now)

	const width = 60
	c := summaryColumns(l.rows, width)
	if c.barW == 0 {
		t.Fatalf("60 columns must leave room for a bar, got barW 0 (labelW %d, durW %d)", c.labelW, c.durW)
	}
	for i, r := range l.rows {
		if r.kind != rowDayHeader && r.kind != rowProject {
			continue
		}
		line := l.renderRow(i, width, c)
		runes := []rune(line)
		if len(runes) < c.barW {
			t.Errorf("row %d is shorter than its bar: %q", i, line)
			continue
		}
		for _, ch := range runes[len(runes)-c.barW:] {
			if !strings.ContainsRune(barGlyphs, ch) {
				t.Errorf("row %d does not end in a full bar: %q", i, line)
				break
			}
		}
	}
}

// TestSummaryTagRowsCarryNoBar: tags are counted individually, so a frame with two
// of them counts under both. Their bars would together outrun the project's and
// claim a division that is not there.
func TestSummaryTagRowsCarryNoBar(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code", "review"),
	}, now)

	c := summaryColumns(l.rows, 60)
	for i, r := range l.rows {
		if r.kind != rowTag {
			continue
		}
		line := l.renderRow(i, 60, c)
		if strings.ContainsAny(line, barGlyphs) {
			t.Errorf("tag row %d carries a bar: %q", i, line)
		}
		if !strings.HasSuffix(line, formatDuration(r.total)) {
			t.Errorf("tag row %d does not end in its number: %q", i, line)
		}
	}
}

// TestSummaryCursorTakesTheRailColumn: the marker sits in the rail's own column,
// not in front of it. That the line keeps its width as a result is asserted by
// TestSummaryCursorDoesNotMoveTheNumbers; this pins where the marker goes.
func TestSummaryCursorTakesTheRailColumn(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	l := summaryTestModel([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 3*time.Hour, "code"),
	}, now)
	l.cursor = 1 // the project row

	c := summaryColumns(l.rows, 60)
	line := l.renderRow(1, 60, c)
	if got := string([]rune(line)[0]); got != selectionMarker {
		t.Errorf("the cursor row opens with %q, want %q", got, selectionMarker)
	}
}
