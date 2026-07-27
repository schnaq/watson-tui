package tui

import (
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
	app.list.refresh(app.frames, time.Monday)
	app.form = newFormModel(&app.frames[0], app.frames, now)
	app.start = newStartModel(app.frames)
	app.report = newReportModel(now, time.Monday)
	app.pendingDelete = app.frames[0]
	// A running timer with tags: the widest header field there is, and it lands in
	// the second header row, which is where the width arithmetic is tightest. Left
	// unset, the sweep only ever measured the short "kein Timer".
	app.state = &watson.State{
		Project: "ein-ziemlich-langer-projektname", Start: now.Add(-90 * time.Minute),
		Tags: []string{"tag-eins", "tag-zwei"},
	}
	// A realistic fatal message: two lines, the second one a backup path far
	// wider than a narrow terminal, so the wrapping is exercised as well.
	app.fatalMsg = "frames-Datei nicht lesbar: invalid character 'k'\n" +
		"Backup: /var/folders/78/jyqksnb52nx_sm8zb5sj90dh0000gn/T/watson/001/frames.bak"
	return app
}

// TestViewHasHeaderBodyFooter: the full frame carries all three parts.
func TestViewHasHeaderBodyFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	out := app.View()
	for _, want := range []string{"watson-tui", "Zeitraum", "j/k bewegen"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q:\n%s", want, out)
		}
	}
}

// TestViewNeverExceedsWidth: no line of any mode may be wider than the
// terminal — the whole point of the layout arithmetic.
func TestViewNeverExceedsWidth(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "ein ziemlich langer projektname",
			now.Add(-2*time.Hour), time.Hour, "tag-eins", "tag-zwei"),
	}
	for _, width := range chromeWidths {
		for _, height := range chromeHeights {
			for _, m := range chromeModes {
				app := chromeApp(t, now, width, height, frames)
				app.mode = m
				for i, line := range strings.Split(app.View(), "\n") {
					if w := lipgloss.Width(line); w > width {
						t.Errorf("mode %d at %dx%d: line %d is %d wide: %q", m, width, height, i, w, line)
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
				app := chromeApp(t, now, width, height, frames)
				app.mode = m
				if lines := strings.Count(app.View(), "\n") + 1; lines > height {
					t.Errorf("mode %d at %dx%d: view has %d lines", m, width, height, lines)
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
		if !strings.Contains(out, "j/k bewegen") {
			t.Errorf("height %d: footer must survive:\n%s", height, out)
		}
	}
	// One line of header at 15, none at 10.
	if out := chromeApp(t, now, 100, 15, frames).View(); !strings.Contains(out, "alle Frames") {
		t.Errorf("height 15 lost the context line:\n%s", out)
	}
	if out := chromeApp(t, now, 100, 10, frames).View(); strings.Contains(out, "alle Frames") {
		t.Errorf("height 10 must give every line to the body:\n%s", out)
	}
}

// TestErrorGoesToFooter: a failed write shows up in the footer, replacing the hints.
func TestErrorGoesToFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.errMsg = "Löschen fehlgeschlagen: kaputt"
	out := app.View()
	if !strings.Contains(out, "Löschen fehlgeschlagen: kaputt") {
		t.Errorf("error missing from view:\n%s", out)
	}
	if strings.Contains(out, "j/k bewegen") {
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
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Zeitraum") || !strings.Contains(got, "Frames") {
		t.Errorf("list header = %q", got)
	}
	app.mode = modeOverview
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Übersicht") {
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
		if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "kein Timer") {
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
// header's bottom right in every mode ("Feld 4 (rechts unten) ist in jedem
// Modus der laufende Timer"). Only the list mode fills the bottom left, so the
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
// discoverable nowhere — the footer cut off "q ende" and the help body was
// clipped before "q beenden". Neither view scrolls, so both have to fit.
func TestQuitKeyIsReachableOnAShortTerminal(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	for _, width := range []int{80, 100} {
		app := chromeApp(t, now, width, 20, frames)
		app.mode = modeList
		if out := app.View(); !strings.Contains(out, "q ende") {
			t.Errorf("%dx20 list: footer must name the quit key:\n%s", width, out)
		}
		app.mode = modeHelp
		out := app.View()
		var found bool
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "beenden") && strings.Contains(line, "q") {
				found = true
			}
		}
		if !found {
			t.Errorf("%dx20 help: the quit key must survive the panel:\n%s", width, out)
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
		f := app.periodField("Zeitraum", period{unit: u, ref: now})
		if !strings.Contains(f.value, "‹") || !strings.Contains(f.value, "›") {
			t.Errorf("unit %d must show the shift affordance: %q", u, f.value)
		}
	}
	f := app.periodField("Zeitraum", period{unit: unitAll, ref: now})
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

	f := app.sumField(p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("sum = %q, want 2h 30m", f.value)
	}
	if strings.Contains(f.value, "läuft") {
		t.Errorf("no timer runs, so no flag: %q", f.value)
	}

	app.state = &watson.State{Project: "läuft", Start: now.Add(-time.Hour), Tags: []string{}}
	f = app.sumField(p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("running timer must not change the sum: %q", f.value)
	}
	if !strings.Contains(f.value, "läuft") {
		t.Errorf("running timer must be flagged: %q", f.value)
	}
}

// TestHeaderSumAddsUpToTheDayTotals: the constraint the sum exists for. It sits
// one row above the list, so it has to count exactly the frames the list shows —
// and the list also filters. Summing a.frames unfiltered puts a header of
// "3h 00m" above day totals adding up to "1h 00m", which reads as a bug in the
// day totals. The filtered case is the one that used to disagree; the plain one
// is here so a sumField that always returns zero cannot pass.
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

	for _, filter := range []string{"", "schnaq"} {
		app.list.filter = filter
		app.list.refresh(app.frames, app.cfg.WeekStart)
		var days time.Duration
		for _, r := range app.list.rows {
			if r.isHeader {
				days += r.total
			}
		}
		// The rendered header, not sumField: the list narrows the frames at the
		// call site, and the constraint is about what the user reads.
		// Equality, not Contains: "1h 00m" is a substring of "11h 00m", so a
		// header that counted too much would pass a contains-check.
		got, ok := headerFieldByLabel(app.headerFields(), "Summe")
		if !ok {
			t.Fatalf("filter %q: the list header has no Summe field", filter)
		}
		if want := formatDuration(days); got != want {
			t.Errorf("filter %q: header sum %q, day totals %q", filter, got, want)
		}
	}
}

// TestComparisonRowNamesNeighbours.
func TestComparisonRowNamesNeighbours(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour), // Vorwoche
	}
	row := app.comparisonRow(period{unit: unitWeek, ref: now})
	flat := ""
	for _, f := range row {
		flat += f.label + " " + f.value + " "
	}
	if !strings.Contains(flat, "Vorwoche") || !strings.Contains(flat, "5h 00m") {
		t.Errorf("comparison row = %q", flat)
	}
	if got := app.comparisonRow(period{unit: unitAll, ref: now}); len(got) != 0 {
		t.Errorf("unitAll has no comparison row, got %+v", got)
	}
}

// TestComparisonRowShowsDashForZero.
func TestComparisonRowShowsDashForZero(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	row := app.comparisonRow(period{unit: unitWeek, ref: now})
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
	app.list.refresh(app.frames, time.Monday)

	app.Update(tea.WindowSizeMsg{Width: 100, Height: 26})
	if !strings.Contains(app.View(), "Vorwoche") {
		t.Error("at 26 lines the comparison row belongs in the header")
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 21})
	out := app.View()
	if strings.Contains(out, "Vorwoche") {
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
		if !strings.Contains(out, "[ ]") {
			t.Errorf("%dx%d: the period keys must be visible:\n%s", size.Width, size.Height, out)
		}
		if !strings.Contains(out, "q ende") {
			t.Errorf("%dx%d: the quit key must be visible:\n%s", size.Width, size.Height, out)
		}
	}
}
