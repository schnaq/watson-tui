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
// three header branches (framed, one line, gone), both of their boundaries, and
// the short terminals where the panel border no longer fits at all.
var chromeHeights = []int{30, 20, 19, 15, 12, 11, 10, 5, 3, 2}

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
	app.report = newReportModel(now)
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

// TestViewHeaderRowsMatchChromeBudget: chromeHeight reserves four lines for the
// framed header, so every mode has to fill exactly those. A mode with a single
// field row draws a three-line header, which leaves the body a line it never
// uses and the footer floating one row above the bottom.
func TestViewHeaderRowsMatchChromeBudget(t *testing.T) {
	now := time.Now()
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	const height = 30
	want := chromeHeight(height) - 1 // the footer takes the remaining line
	for _, m := range chromeModes {
		app := chromeApp(t, now, 100, height, frames)
		app.mode = m
		rows := app.headerFields()
		if len(rows) != 2 {
			t.Errorf("mode %d has %d header rows, want 2", m, len(rows))
		}
		out := renderHeader(100, height, app.version, rows)
		if lines := strings.Count(out, "\n") + 1; lines != want {
			t.Errorf("mode %d: framed header has %d lines, chromeHeight reserves %d:\n%s",
				m, lines, want, out)
		}
	}
}

// TestViewShedsHeaderOnShortTerminals: the collapse thresholds have to be
// visible in the assembled frame, not just in renderHeader. Between 12 and 19
// lines the header keeps the context but loses its frame and the version title;
// below 12 every line belongs to the body and the header is gone entirely. The
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
	app.report = newReportModel(now)
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
		lines := strings.Split(app.View(), "\n")[:chromeHeight(height)-1]
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
// focused bool it used to take. Under go test the renderer strips colour out of
// the rendered string, so the assertion goes to the style the panel is handed.
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
}
