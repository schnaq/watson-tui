package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// chromeModes lists every mode View() can be called in. The layout arithmetic
// has to hold in all of them, not just in the list.
var chromeModes = []mode{
	modeList, modeForm, modeReport, modeOverview, modeStartTimer,
	modeConfirmDelete, modeConfirmCancel, modeHelp, modeFatal,
}

// chromeHeights are the terminal heights the arithmetic has to hold at: the
// four collapse stages, both sides of each of their three boundaries, and the
// short terminals where the panel border no longer fits at all. The boundaries
// are the whole point of the list — the stages are exactly where header rows
// and hint lines are added and taken away, so a sweep that skips them checks
// the design nowhere near where it changes.
var chromeHeights = []int{30, 24, 23, 20, 19, 15, 14, 13, 11, 10, 5, 3, 2}

// chromeWidths bracket the narrow terminal where the billing table no longer
// fits its own minimum column widths (60) and the comfortable one (100).
var chromeWidths = []int{60, 80, 100}

// chromeApp builds an app with every sub-model filled, so View() has real
// content to render whichever mode it is put into. frames must not be empty.
func chromeApp(t *testing.T, now time.Time, width, height int, frames []watson.Frame) *App {
	t.Helper()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	app.frames = frames
	app.list.per = period{unit: unitAll, ref: now}
	// The frame list, not the summary the app opens on: the tests built on this
	// fixture ask about frame rows — the ID column, the day header's total, the
	// widest row there is. The sweeps that have to hold in both views flip the
	// flag themselves and refresh again; see TestViewNeverExceedsWidth.
	app.list.compact = false
	app.list.refresh(app.frames, time.Monday, app.state, app.now)
	app.form = newFormModel(&app.frames[0], app.frames, now)
	app.start = newStartModel(app.frames)
	app.report = newReportModel(now, time.Monday)
	app.pendingDelete = app.frames[0]
	// A running timer with tags: the widest header field there is, and it lands in
	// the second header row, which is where the width arithmetic is tightest. Left
	// unset, the sweep only ever measured the short "no timer".
	app.state = &watson.State{
		Project: "ein-ziemlich-langer-projektname", Start: now.Add(-90 * time.Minute),
		Tags: []string{"tag-eins", "tag-zwei"},
	}
	// A realistic fatal message: two lines, the second one a backup path far
	// wider than a narrow terminal, so the wrapping is exercised as well.
	app.fatalMsg = "cannot read the frames file: invalid character 'k'\n" +
		"backup: /var/folders/78/jyqksnb52nx_sm8zb5sj90dh0000gn/T/watson/001/frames.bak"
	return app
}

// TestViewHasHeaderBodyFooter: the full frame carries all three parts.
func TestViewHasHeaderBodyFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	out := app.View()
	for _, want := range []string{"watson-tui", "Period", "j/k move"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q:\n%s", want, out)
		}
	}
}

// TestViewNeverExceedsWidth: no line of any mode may be wider than the
// terminal — the whole point of the layout arithmetic.
//
// Both list views, because they lay their rows out by different arithmetic —
// the frame list negotiates columns, the summary indents and right-aligns — and
// the one a user meets first is the summary.
func TestViewNeverExceedsWidth(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "ein ziemlich langer projektname",
			now.Add(-2*time.Hour), time.Hour, "tag-eins", "tag-zwei"),
	}
	for _, width := range chromeWidths {
		for _, height := range chromeHeights {
			for _, m := range chromeModes {
				for _, compact := range []bool{false, true} {
					app := chromeApp(t, now, width, height, frames)
					app.mode = m
					app.list.compact = compact
					app.list.refresh(app.frames, time.Monday, app.state, app.now)
					for i, line := range strings.Split(app.View(), "\n") {
						if w := lipgloss.Width(line); w > width {
							t.Errorf("mode %d, compact %v at %dx%d: line %d is %d wide: %q",
								m, compact, width, height, i, w, line)
						}
					}
				}
			}
		}
	}
}

// TestViewBudgetsHeight: the rendered frame must fit the terminal height, in
// every mode — a frame one line too tall scrolls the alt screen and pushes the
// header out of sight.
func TestViewBudgetsHeight(t *testing.T) {
	now := time.Now()
	var frames []watson.Frame
	for i := 0; i < 40; i++ {
		frames = append(frames, mkFrame(
			strings.Repeat("a", 31)+string(rune('a'+i%26)), "projekt",
			now.Add(-time.Duration(i*30)*time.Hour), time.Hour))
	}
	for _, width := range chromeWidths {
		for _, height := range chromeHeights {
			for _, m := range chromeModes {
				for _, compact := range []bool{false, true} {
					app := chromeApp(t, now, width, height, frames)
					app.mode = m
					app.list.compact = compact
					app.list.refresh(app.frames, time.Monday, app.state, app.now)
					if lines := strings.Count(app.View(), "\n") + 1; lines > height {
						t.Errorf("mode %d, compact %v at %dx%d: view has %d lines",
							m, compact, width, height, lines)
					}
				}
			}
		}
	}
}

// TestViewChromeFillsItsBudget: header and footer together have to occupy
// exactly the lines chromeHeight reserves — in every mode and at every stage.
// View hands the body everything else, so a chrome that draws one line less
// than it claimed leaves a blank row at the bottom of the terminal, and one
// that draws more pushes the top line off the screen.
//
// The modes differ in how much they have to say: the list fills four header
// rows and two hint lines, the form one row and one line. Neither may decide
// the layout — filling the budget is the chrome's job, not the mode's.
func TestViewChromeFillsItsBudget(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	countLines := func(s string) int {
		if s == "" {
			return 0 // View leaves out an empty header rather than drawing a blank line
		}
		return strings.Count(s, "\n") + 1
	}
	for _, height := range chromeHeights {
		for _, m := range chromeModes {
			app := chromeApp(t, now, 100, height, frames)
			app.mode = m
			header, footer := countLines(app.headerView()), countLines(app.footerView())
			if header+footer != chromeHeight(height) {
				t.Errorf("mode %d at height %d: header %d + footer %d lines, "+
					"chromeHeight reserves %d", m, height, header, footer, chromeHeight(height))
			}
			if footer != footerLines(height) {
				t.Errorf("mode %d at height %d: footer has %d lines, want %d",
					m, height, footer, footerLines(height))
			}
			// The hints sit on the last row of the terminal, not one above it:
			// the modes with a single hint group are padded to their share of
			// the chrome, and padding below the hints would float them.
			lines := strings.Split(app.View(), "\n")
			if last := lines[len(lines)-1]; strings.TrimSpace(last) == "" {
				t.Errorf("mode %d at height %d: the bottom row is blank, the hints "+
					"float above it: %q", m, height, strings.Join(lines[len(lines)-2:], "\n"))
			}
		}
	}
}

// TestViewShedsHeaderOnShortTerminals: the collapse thresholds have to be
// visible in the assembled frame, not just in renderHeader. Between 14 and 19
// lines the header keeps the context but loses its frame and the version title;
// below 14 every line belongs to the body and the header is gone entirely. The
// footer stays in both cases — it is the only thing left that names the keys.
func TestViewShedsHeaderOnShortTerminals(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	for _, height := range []int{15, 10} {
		app := chromeApp(t, now, 100, height, frames)
		out := app.View()
		if strings.Contains(out, "watson-tui") {
			t.Errorf("height %d: header frame must be gone:\n%s", height, out)
		}
		if !strings.Contains(out, "j/k move") {
			t.Errorf("height %d: footer must survive:\n%s", height, out)
		}
	}
	// One line of header at 15, none at 10.
	if out := chromeApp(t, now, 100, 15, frames).View(); !strings.Contains(out, "all frames") {
		t.Errorf("height 15 lost the context line:\n%s", out)
	}
	if out := chromeApp(t, now, 100, 10, frames).View(); strings.Contains(out, "all frames") {
		t.Errorf("height 10 must give every line to the body:\n%s", out)
	}
}

// TestErrorGoesToFooter: a failed write shows up in the footer, replacing the hints.
func TestErrorGoesToFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.errMsg = "delete failed: broken"
	out := app.View()
	if !strings.Contains(out, "delete failed: broken") {
		t.Errorf("error missing from view:\n%s", out)
	}
	if strings.Contains(out, "j/k move") {
		t.Errorf("error must replace the hints:\n%s", out)
	}
}

// TestHeaderFieldsPerMode: each mode labels its own context.
func TestHeaderFieldsPerMode(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.state = &watson.State{Project: "laufend", Start: now.Add(-time.Minute), Tags: []string{}}

	app.mode = modeList
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Period") || !strings.Contains(got, "frames") {
		t.Errorf("list header = %q", got)
	}
	app.mode = modeOverview
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Overview") {
		t.Errorf("overview header = %q", got)
	}
	app.mode = modeReport
	app.report = newReportModel(now, time.Monday)
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Report") {
		t.Errorf("report header = %q", got)
	}
	// The timer belongs to every mode.
	for _, m := range chromeModes {
		app.mode = m
		if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "laufend") {
			t.Errorf("mode %d header lost the timer: %q", m, got)
		}
	}
	// Without a running timer every mode says so instead of leaving a gap.
	app.state = nil
	for _, m := range chromeModes {
		app.mode = m
		if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "no timer") {
			t.Errorf("mode %d header lost the idle timer: %q", m, got)
		}
	}
}

// headerFieldByLabel returns the value of the first field carrying label, so a
// test can compare one field exactly instead of searching the flattened header
// for a substring.
func headerFieldByLabel(rows [][]headerField, label string) (string, bool) {
	for _, r := range rows {
		for _, f := range r {
			if f.label == label {
				return f.value, true
			}
		}
	}
	return "", false
}

func renderFieldsFlat(rows [][]headerField) string {
	var parts []string
	for _, r := range rows {
		for _, f := range r {
			parts = append(parts, f.label+" "+f.value)
		}
	}
	return strings.Join(parts, " | ")
}

// TestViewKeepsTimerBottomRight: the spec puts the running timer in the
// header's bottom right in every mode ("field 4, bottom right, is the running
// timer in every mode"). Only the list mode fills the bottom left, so the
// eight modes that leave it empty are the ones that used to move the timer to
// the left edge. The assertion is its position, not its presence: a test that
// only looked for the timer stayed green while it sat bottom left.
func TestViewKeepsTimerBottomRight(t *testing.T) {
	const width, height = 100, 30
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	for _, m := range chromeModes {
		app := chromeApp(t, now, width, height, frames)
		app.now = now
		app.state = &watson.State{Project: "laufend", Start: now.Add(-90 * time.Second), Tags: []string{"dev"}}
		app.mode = m
		timer := "▶ laufend [dev] " + formatClock(90*time.Second)

		// Only the header rows: the report and the overview bodies disclose the
		// running timer as well, and those notes are left-aligned on purpose.
		lines := strings.Split(app.View(), "\n")[:chromeHeight(height)-footerLines(height)]
		var found bool
		for _, line := range lines {
			if !strings.Contains(line, timer) {
				continue
			}
			found = true
			// Inside the framed header the line closes with a space of gutter and
			// the border column, so the timer has to be the last thing before them.
			tail := strings.TrimRight(strings.TrimSuffix(strings.TrimRight(line, " "), "│"), " ")
			if !strings.HasSuffix(tail, timer) {
				t.Errorf("mode %d: timer is not flush right: %q", m, line)
			}
			if col := strings.Index(line, "▶"); col*2 < width {
				t.Errorf("mode %d: timer starts at column %d, want the right half of %d: %q",
					m, col, width, line)
			}
		}
		if !found {
			t.Errorf("mode %d: header lost the running timer:\n%s", m, strings.Join(lines, "\n"))
		}
	}
}

// TestQuitKeyIsReachableOnAShortTerminal: at 100x20 the quit key used to be
// discoverable nowhere — the footer cut off "q quit" and the help body was
// clipped before its own "q  quit" row. Neither view scrolls, so both have to
// fit.
//
// The heights are the tight ones, and they are tight for different reasons: 20
// is where the chrome costs six lines and leaves the help twelve, 14 is the
// shortest terminal that still draws a header and leaves it ten — the smallest
// budget there is. 15 and 16 sit between them. This renders the assembled frame;
// TestHelpFitsEveryTerminalWithAHeader is the arithmetic behind it.
func TestQuitKeyIsReachableOnAShortTerminal(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	for _, width := range []int{80, 100} {
		for _, height := range []int{14, 15, 16, 20} {
			app := chromeApp(t, now, width, height, frames)
			app.mode = modeList
			if out := app.View(); !strings.Contains(out, "q quit") {
				t.Errorf("%dx%d list: footer must name the quit key:\n%s", width, height, out)
			}
			app.mode = modeHelp
			out := app.View()
			var found bool
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "q ") && strings.Contains(line, "quit") {
					found = true
				}
			}
			if !found {
				t.Errorf("%dx%d help: the quit key must survive the panel:\n%s",
					width, height, out)
			}
		}
	}
}

// TestViewFillsTheTerminal: the frame has to reach the bottom of the screen.
// fitBody used to cut a body that was too long but never pad one that was too
// short, so three frames at 100x30 drew a 12-line box with the footer floating
// mid-screen and 18 blank rows below it — the single thing that made this look
// unfinished next to k9s or lazygit. The upper bound stays pinned by
// TestViewBudgetsHeight; this is the lower one.
func TestViewFillsTheTerminal(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-3*time.Hour), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(-2*time.Hour), time.Hour),
	}
	const width, height = 100, 30
	for _, m := range chromeModes {
		app := chromeApp(t, now, width, height, frames)
		app.mode = m
		out := app.View()
		if lines := strings.Count(out, "\n") + 1; lines != height {
			t.Errorf("mode %d at %dx%d: view has %d lines, want exactly %d:\n%s",
				m, width, height, lines, height, out)
		}
	}
}

// TestViewFillsEveryTerminalItFits: the same lower bound across both collapse
// thresholds and both extremes of the width sweep. Together with
// TestViewBudgetsHeight this pins the height to exactly the terminal — the frame
// may neither scroll the alt screen nor leave rows unclaimed at the bottom.
func TestViewFillsEveryTerminalItFits(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-3*time.Hour), time.Hour),
	}
	for _, width := range chromeWidths {
		for _, height := range chromeHeights {
			for _, m := range chromeModes {
				app := chromeApp(t, now, width, height, frames)
				app.mode = m
				if lines := strings.Count(app.View(), "\n") + 1; lines != height {
					t.Errorf("mode %d at %dx%d: view has %d lines, want exactly %d",
						m, width, height, lines, height)
				}
			}
		}
	}
}

// TestFatalPanelIsErrorColoured: the spec asks for the fatal screen in a panel
// of error colour, which is why panel takes a border style instead of the
// focused bool it used to take.
//
// The style was only half of it. Pinning panelBorder alone left View() free to
// hand panel a styleBorder unconditionally and keep the suite green, so the
// second half renders the fatal screen and looks for the colour on the frame it
// actually drew. That needs a colour profile: under go test the renderer detects
// Ascii and strips every escape, which would make such an assertion pass no
// matter which style the panel was given.
func TestFatalPanelIsErrorColoured(t *testing.T) {
	if got := panelBorder(modeFatal).GetForeground(); got != colErr {
		t.Errorf("fatal panel border = %v, want the error colour %v", got, colErr)
	}
	for _, m := range chromeModes {
		if m == modeFatal {
			continue
		}
		if got := panelBorder(m).GetForeground(); got != colDim {
			t.Errorf("mode %d panel border = %v, want the dim border %v", m, got, colDim)
		}
	}

	// 0 is termenv.TrueColor, written as a number so that one constant does not
	// turn termenv into a direct dependency of this module. A wrong value would
	// strip the colour again and fail the two assertions below rather than quietly
	// hollow them out. Restored afterwards: the profile is global, and every other
	// test in this package compares against colourless strings.
	saved := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(0)
	defer lipgloss.SetColorProfile(saved)

	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-3*time.Hour), time.Hour),
	}
	app := chromeApp(t, now, 60, 24, frames)
	// The panel's top-left corner, in the error colour: the exact substring panel
	// writes when it is handed styleError as its border.
	corner := styleError.Render("╭─")

	app.mode = modeFatal
	if out := app.View(); !strings.Contains(out, corner) {
		t.Errorf("the fatal screen must be framed in the error colour, got:\n%q", out)
	}
	app.mode = modeReport
	if out := app.View(); strings.Contains(out, corner) {
		t.Errorf("only the fatal screen may frame itself in the error colour, got:\n%q", out)
	}
}

// TestPeriodFieldShowsItIsShiftable: the brackets are the affordance for [ and ].
func TestPeriodFieldShowsItIsShiftable(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	for _, u := range []periodUnit{unitDay, unitWeek, unitMonth} {
		f := app.periodField("Period", period{unit: u, ref: now})
		if !strings.Contains(f.value, "‹") || !strings.Contains(f.value, "›") {
			t.Errorf("unit %d must show the shift affordance: %q", u, f.value)
		}
	}
	f := app.periodField("Period", period{unit: unitAll, ref: now})
	if strings.Contains(f.value, "‹") {
		t.Errorf("unitAll cannot be shifted, so no affordance: %q", f.value)
	}
}

// TestSumFieldMatchesTheListAndFlagsTheTimer: the header sum must add up to the
// day totals below it, and say so when a running timer is not in it.
func TestSumFieldMatchesTheListAndFlagsTheTimer(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p", now.Add(-3*time.Hour), 2*time.Hour),
		mkFrame("b2222222222222222222222222222222", "p", now.Add(-30*time.Hour), 30*time.Minute),
	}
	p := period{unit: unitAll, ref: now}

	f := app.sumFieldWithoutRunning(app.frames, p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("sum = %q, want 2h 30m", f.value)
	}
	if strings.Contains(f.value, "running") {
		t.Errorf("no timer runs, so no flag: %q", f.value)
	}

	app.state = &watson.State{Project: "p", Start: now.Add(-time.Hour), Tags: []string{}}
	f = app.sumFieldWithoutRunning(app.frames, p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("running timer must not change the sum: %q", f.value)
	}
	if !strings.Contains(f.value, "running") {
		t.Errorf("running timer must be flagged: %q", f.value)
	}
}

// TestHeaderSumAddsUpToTheDayTotals: the constraint the sum exists for. It sits
// one row above the list, so it has to count exactly the frames the list shows —
// and the list also filters. Summing a.frames unfiltered puts a header of
// "3h 00m" above day totals adding up to "1h 00m", which reads as a bug in the
// day totals. The filtered case is the one that used to disagree; the plain one
// is here so a sum field that always returns zero cannot pass.
//
// Both list views: each draws its own day headers, and after f the sum stands
// above a different set of them. The running timer is what makes the two views
// count differently, and it has a test of its own —
// TestSummaryHeaderTotalCountsWhatTheDayBlocksCount, below.
func TestHeaderSumAddsUpToTheDayTotals(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), time.Hour),
		mkFrame("b2222222222222222222222222222222", "kunde-a",
			time.Date(2026, 7, 21, 13, 0, 0, 0, time.Local), 2*time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}

	for _, compact := range []bool{false, true} {
		for _, filter := range []string{"", "schnaq"} {
			app.list.compact = compact
			app.list.filter = filter
			app.list.refresh(app.frames, app.cfg.WeekStart, app.state, app.now)
			var days time.Duration
			for _, r := range app.list.rows {
				if r.kind == rowDayHeader {
					days += r.total
				}
			}
			// The rendered header, not the sum field: the list narrows the frames at the
			// call site, and the constraint is about what the user reads.
			// Equality, not Contains: "1h 00m" is a substring of "11h 00m", so a
			// header that counted too much would pass a contains-check.
			got, ok := headerFieldByLabel(app.headerFields(), "Total")
			if !ok {
				t.Fatalf("compact %v, filter %q: the list header has no Total field", compact, filter)
			}
			if want := formatDuration(days); got != want {
				t.Errorf("compact %v, filter %q: header sum %q, day totals %q",
					compact, filter, got, want)
			}
		}
	}
}

// TestSummaryHeaderTotalCountsWhatTheDayBlocksCount: the same constraint one
// view further on, and the case the frame list never had. The summary folds the
// running timer into its day totals — that is what makes it agree with the
// report and the overview — so the Total above them has to count it too, and
// must not carry the "+ running" flag that says it did not.
//
// The combination this pins out is the one app.go's two sum fields exist to make
// unwritable: a flag reading "not counted" over a number that counts it, above
// day blocks that also count it. Everything else in the suite misses it, because
// newTestApp opens on an empty directory and chromeApp sets its state after the
// refresh — so a.state is nil wherever the header is measured. Here it is set
// before the rows are built, the way reload() does it in production.
func TestSummaryHeaderTotalCountsWhatTheDayBlocksCount(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 22, 9, 0, 0, 0, time.Local), 2*time.Hour),
	}
	app.state = &watson.State{Project: "kunde-a", Start: now.Add(-time.Hour), Tags: []string{}}
	app.list.per = period{unit: unitDay, ref: now}
	app.list.refresh(app.frames, app.cfg.WeekStart, app.state, app.now)

	var days time.Duration
	for _, r := range app.list.rows {
		if r.kind == rowDayHeader {
			days += r.total
		}
	}
	if days != 3*time.Hour {
		t.Fatalf("the fixture's day totals are %v, want 3h — two booked plus one running", days)
	}
	got, ok := headerFieldByLabel(app.headerFields(), "Total")
	if !ok {
		t.Fatal("the summary header has no Total field")
	}
	if want := formatDuration(days); got != want {
		t.Errorf("summary header Total %q, day blocks %q", got, want)
	}
	if strings.Contains(got, "running") {
		t.Errorf("Total %q flags the timer as uncounted while the day blocks count it", got)
	}

	// The neighbours are read against that Total, so they count the timer too —
	// otherwise the week containing today comes out smaller than today.
	week, ok := headerFieldByLabel(app.headerFields(), "Week")
	if !ok {
		t.Fatal("the summary header has no Week field")
	}
	if want := formatDuration(3 * time.Hour); week != want {
		t.Errorf("Week = %q, want %q — the neighbour must count the timer as the Total does",
			week, want)
	}
}

// durationPattern matches a duration the way formatDuration renders one. Used to
// read numbers back off a rendered body.
var durationPattern = regexp.MustCompile(`\d+h \d+m|\d+m`)

// bodyDurations returns the durations the body printed on its first line
// containing label, left to right. It lets a test compare a header field against
// the number the body actually put on screen instead of against a literal: a
// literal pins one arithmetic case, while the header's whole reason to exist is
// that it agrees with the body below it for every case.
//
// Contains rather than a prefix, and a regexp rather than field splitting,
// because the totals line is styled — under go test the profile is ASCII and the
// styling is a no-op, but a forced colour profile must not turn this into a
// puzzle about escape bytes.
func bodyDurations(t *testing.T, body, label string) []string {
	t.Helper()
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, label) {
			if got := durationPattern.FindAllString(line, -1); len(got) > 0 {
				return got
			}
		}
	}
	t.Fatalf("no %q line with a duration in the body:\n%s", label, body)
	return nil
}

// TestReportHeaderSumEqualsTheBodysTotal is the invariant the report header
// exists for, and the one it violated: the header said "2h 00m + running" over
// a body that said "Total 3h 00m" and "▶ … running (…), included". Two numbers
// for one period, and two annotations contradicting each other about whether
// the running hour is inside the number — in the view an invoice is written
// from.
//
// Asserted against the body's own Total, not against "3h 00m": a literal pins
// this fixture, whereas the header has to agree with the body whatever the
// frames are. The last check is what makes the fixture load-bearing — the
// running timer must actually move the number, or a header that ignored it
// entirely would pass by coincidence.
func TestReportHeaderSumEqualsTheBodysTotal(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.mode = modeReport
	app.report.per = period{unit: unitWeek, ref: now}.shift(app.cfg.WeekStart, 0)
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour),
	}
	app.state = &watson.State{Project: "kunde-a", Start: now.Add(-time.Hour), Tags: []string{}}

	head, ok := headerFieldByLabel(app.headerFields(), "Total")
	if !ok {
		t.Fatal("the report header has no Total field")
	}
	body := app.report.view(app.frames, app.state, app.cfg.WeekStart, app.now, app.width)
	total := bodyDurations(t, body, "Total")
	if want := total[len(total)-1]; head != want {
		t.Errorf("header sum %q, body Total %q", head, want)
	}
	// The body says the timer is counted; a flag that reads "not counted" on a
	// number that counts it is worse than no flag.
	if strings.Contains(head, "running") {
		t.Errorf("the report counts the timer, so its sum must not be flagged: %q", head)
	}
	if without := formatDuration(sumInPeriod(app.frames, app.report.per, app.cfg.WeekStart)); head == without {
		t.Fatalf("the fixture's timer contributes nothing (sum without it is also %q)", without)
	}
}

// TestOverviewHeaderSumEqualsTheTotalColumn: the overview's header number is
// correct today, and this keeps it that way. Same shape as the report's test,
// against the last cell of the table's Total row — the column billing reads.
func TestOverviewHeaderSumEqualsTheTotalColumn(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.mode = modeOverview
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour),
		mkFrame("b2222222222222222222222222222222", "kunde-b",
			time.Date(2026, 6, 30, 9, 0, 0, 0, time.Local), 45*time.Minute),
	}
	app.state = &watson.State{Project: "kunde-a", Start: now.Add(-time.Hour), Tags: []string{}}

	head, ok := headerFieldByLabel(app.headerFields(), "Total, all")
	if !ok {
		t.Fatal("the overview header has no \"Total, all\" field")
	}
	body := overviewView(app.frames, app.state, app.cfg.WeekStart, app.now, app.width)
	cells := bodyDurations(t, body, "Total")
	if want := cells[len(cells)-1]; head != want {
		t.Errorf("header sum %q, Total column %q", head, want)
	}
	if strings.Contains(head, "running") {
		t.Errorf("the overview counts the timer, so its sum must not be flagged: %q", head)
	}
	if without := formatDuration(sumInPeriod(app.frames, period{unit: unitAll, ref: now}, app.cfg.WeekStart)); head == without {
		t.Fatalf("the fixture's timer contributes nothing (sum without it is also %q)", without)
	}
}

// TestComparisonRowNamesNeighbours pins which value belongs to which label, in
// the order the row renders them.
//
// The three frames give the two neighbours three different sums, which is what
// makes the assertions bite. With a single frame in the previous week both
// neighbours came out at "5h 00m", so two independent Contains-checks over the
// flattened row passed no matter which label carried which number — swapping the
// entries of comparisonPeriods, or hanging the wrong period off a label, stayed
// green. The June frame is outside both neighbours, so a row that summed every
// frame it was given also fails.
//
// The order is asserted because it is part of the contract: the period before
// the current one first, the larger period containing it second (see
// comparisonPeriods), which is the order the spec renders as
// "Prev week … · Month …".
func TestComparisonRowNamesNeighbours(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour), // prev week and month
		mkFrame("b2222222222222222222222222222222", "p",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour), // month only
		mkFrame("c3333333333333333333333333333333", "p",
			time.Date(2026, 6, 30, 9, 0, 0, 0, time.Local), time.Hour), // neither
	}
	want := []headerField{
		{label: "Prev week", value: "5h 00m"},
		{label: "Month", value: "7h 00m"},
	}
	row := app.comparisonRow(app.frames, period{unit: unitWeek, ref: now})
	if len(row) != len(want) {
		t.Fatalf("comparison row = %+v, want %d fields", row, len(want))
	}
	for i, w := range want {
		if row[i] != w {
			t.Errorf("field %d = %+v, want %+v", i, row[i], w)
		}
	}
	if got := app.comparisonRow(app.frames, period{unit: unitAll, ref: now}); len(got) != 0 {
		t.Errorf("unitAll has no comparison row, got %+v", got)
	}
}

// TestComparisonRowFollowsTheListFilter: with a filter active the Total narrows
// to the filtered frames, so the neighbours beside it have to narrow too.
// Summing every frame made them describe a different set than the number they
// sit next to, with nothing on screen saying so — the reader has no way to tell
// "5h last week" (this project) from "8h last week" (all of them).
//
// Both cases are asserted, because a comparison row that ignores the filter is
// right in the unfiltered one, and only the pair shows the difference.
func TestComparisonRowFollowsTheListFilter(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour),
		mkFrame("b2222222222222222222222222222222", "kunde-a",
			time.Date(2026, 7, 14, 14, 0, 0, 0, time.Local), 3*time.Hour),
		mkFrame("c3333333333333333333333333333333", "schnaq",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour),
	}
	app.list.per = period{unit: unitWeek, ref: now}.shift(app.cfg.WeekStart, 0)

	cases := []struct{ filter, prevWeek, month string }{
		{"", "8h 00m", "10h 00m"},
		{"schnaq", "5h 00m", "7h 00m"},
	}
	for _, tc := range cases {
		app.list.filter = tc.filter
		app.list.refresh(app.frames, app.cfg.WeekStart, app.state, app.now)
		fields := app.headerFields()
		for _, want := range []struct{ label, value string }{
			{"Prev week", tc.prevWeek}, {"Month", tc.month},
		} {
			got, ok := headerFieldByLabel(fields, want.label)
			if !ok {
				t.Fatalf("filter %q: the header has no %s field", tc.filter, want.label)
			}
			if got != want.value {
				t.Errorf("filter %q: %s = %q, want %q", tc.filter, want.label, got, want.value)
			}
		}
	}
}

// TestReportComparisonCountsTheRunningTimer: the report's Total counts the
// running timer, so its neighbours have to as well. Otherwise the month renders
// smaller than the week it contains — "Total 3h 00m" beside "Month 2h 00m" —
// and nothing on screen explains the difference.
func TestReportComparisonCountsTheRunningTimer(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.mode = modeReport
	app.report.per = period{unit: unitWeek, ref: now}.shift(app.cfg.WeekStart, 0)
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq",
			time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour),
	}
	app.state = &watson.State{Project: "kunde-a", Start: now.Add(-time.Hour), Tags: []string{}}

	fields := app.headerFields()
	month := period{unit: unitMonth, ref: app.report.per.ref}
	_, want := aggregate(withRunning(app.frames, app.state, app.now), month, app.cfg.WeekStart)
	got, ok := headerFieldByLabel(fields, "Month")
	if !ok {
		t.Fatal("the report header has no Month field")
	}
	// Pinned to the month's own aggregate rather than to "the month is at least
	// the week": both sides of that comparison would be computed here from the
	// same frames, so it would hold whatever comparisonRow returned. Equality with
	// the timer counted gives the containment for free.
	if got != formatDuration(want) {
		t.Errorf("Month = %q, want %q", got, formatDuration(want))
	}
	if without := formatDuration(sumInPeriod(app.frames, month, app.cfg.WeekStart)); got == without {
		t.Fatalf("the fixture's timer contributes nothing to the month (also %q)", without)
	}
}

// TestComparisonRowShowsDashForZero.
func TestComparisonRowShowsDashForZero(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	row := app.comparisonRow(app.frames, period{unit: unitWeek, ref: now})
	for _, f := range row {
		if strings.Contains(f.value, "0m") {
			t.Errorf("zero must read as a dash, got %q", f.value)
		}
	}
	if len(row) == 0 || !strings.Contains(row[0].value, "–") {
		t.Errorf("comparison row = %+v", row)
	}
}

// TestViewDropsComparisonRowBeforeTheFrame: at 20-23 lines the comparison goes,
// the framed header stays.
func TestViewDropsComparisonRowBeforeTheFrame(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour),
	}
	app.list.per = period{unit: unitWeek, ref: now}
	app.list.refresh(app.frames, time.Monday, app.state, app.now)

	app.Update(tea.WindowSizeMsg{Width: 100, Height: 26})
	if !strings.Contains(app.View(), "Prev week") {
		t.Error("at 26 lines the comparison row belongs in the header")
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 21})
	out := app.View()
	if strings.Contains(out, "Prev week") {
		t.Errorf("at 21 lines the comparison row must give way:\n%s", out)
	}
	if !strings.Contains(out, "╭") {
		t.Errorf("but the frame stays at 21 lines:\n%s", out)
	}
}

// TestViewShowsPeriodKeysInTheFooter: the discoverability fix, end to end.
func TestViewShowsPeriodKeysInTheFooter(t *testing.T) {
	app := newTestApp(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 100, Height: 30}, {Width: 80, Height: 24}, {Width: 80, Height: 20}} {
		app.Update(size)
		out := app.View()
		if !strings.Contains(out, "← →") {
			t.Errorf("%dx%d: the period keys must be visible:\n%s", size.Width, size.Height, out)
		}
		if !strings.Contains(out, "q quit") {
			t.Errorf("%dx%d: the quit key must be visible:\n%s", size.Width, size.Height, out)
		}
	}
}

// TestArrowKeysShiftThePeriod: ← and → do what [ and ] do, in the list and in
// the report. The ‹ › affordance in the header reads as arrows, so the arrow
// keys are the form a user reaches for first.
func TestArrowKeysShiftThePeriod(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// List: left goes back one unit, right returns.
	start := app.list.per
	app.Update(key("left"))
	back := app.list.per
	if back == start {
		t.Fatal("← must shift the list period back")
	}
	app.Update(key("right"))
	if app.list.per != start {
		t.Errorf("→ must undo ←: got %v, want %v", app.list.per, start)
	}
	// The same step as [ produces.
	app.Update(key("["))
	if app.list.per != back {
		t.Errorf("← and [ must be the same step: %v vs %v", back, app.list.per)
	}

	// The unit in effect decides the step size.
	app.Update(key("m"))
	month := app.list.per
	app.Update(key("left"))
	from, _, _ := app.list.per.bounds(time.Monday)
	prevFrom, _, _ := month.bounds(time.Monday)
	if from.AddDate(0, 1, 0) != prevFrom {
		t.Errorf("← on a month must step one month: %v → %v", prevFrom, from)
	}

	// Report: same keys, same effect.
	app.Update(key("esc"))
	app.Update(key("r"))
	rStart := app.report.per
	app.Update(key("left"))
	if app.report.per == rStart {
		t.Fatal("← must shift the report period back")
	}
	app.Update(key("right"))
	if app.report.per != rStart {
		t.Errorf("→ must undo ← in the report too")
	}
}

// TestArrowKeysStayOutOfTheFilterInput: while filtering, ← and → move the text
// cursor — shifting the period out from under a half-typed filter would be a
// trap.
func TestArrowKeysStayOutOfTheFilterInput(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	before := app.list.per
	app.Update(key("/"))
	app.Update(key("a"))
	app.Update(key("left"))
	app.Update(key("right"))
	if app.list.per != before {
		t.Errorf("arrows must not shift the period while filtering: %v → %v", before, app.list.per)
	}
	if !app.list.filtering {
		t.Error("arrows must not leave filter mode")
	}
}

// TestFooterNamesTheArrowKeys: the hint has to name the keys a user will try.
func TestFooterNamesTheArrowKeys(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if out := app.View(); !strings.Contains(out, "← → period") {
		t.Errorf("footer must name the arrow keys:\n%s", out)
	}
}
