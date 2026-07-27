package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestPanelFramesBodyAndTitle: the panel draws a border, carries its title in
// the top line and never exceeds the width it was given.
func TestPanelFramesBodyAndTitle(t *testing.T) {
	out := panel("Frames", "erste Zeile\nzweite Zeile", 40, styleBorder)
	if !strings.Contains(out, "Frames") {
		t.Errorf("panel lost its title:\n%s", out)
	}
	for _, want := range []string{"erste Zeile", "zweite Zeile"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel lost body line %q:\n%s", want, out)
		}
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 40 {
			t.Errorf("line %d is %d wide, want <= 40: %q", i, w, line)
		}
	}
}

// TestPanelTruncatesLongTitle: a title wider than the frame must not push the
// border past the given width. This is the branch where the label is truncated
// right up to its cap (len(label) == inner-4), which leaves exactly one dash
// between label and corner — the tightest case for the corner arithmetic. So
// assert the exact width, not just an upper bound, across both parities.
func TestPanelTruncatesLongTitle(t *testing.T) {
	for _, width := range []int{30, 37, 40} {
		out := panel(strings.Repeat("sehr langer titel ", 5), "body", width, styleBorder)
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w != width {
				t.Errorf("width %d: line %d is %d wide, want %d: %q",
					width, i, w, width, line)
			}
		}
	}
}

// TestPanelKeepsAnsiBodyLinesIntact: body lines arrive pre-styled from the
// views, so panel measures their display width instead of their rune count.
// Without that guard the escapes inflate the count, truncate fires on a line
// that fits, and the cut lands past the text — dropping the trailing reset and
// bleeding the colour across the rest of the terminal.
//
// lipgloss.Width parses the escapes out of the string itself, so it does not
// care that the no-TTY renderer strips styling under go test: a literal escape
// written here is measured at its display width all the same.
func TestPanelKeepsAnsiBodyLinesIntact(t *testing.T) {
	// Display width 30, but 39 runes. At width 40 the body may use inner-2 = 36
	// columns, so this line fits and must be passed through untouched.
	body := "\x1b[31m" + strings.Repeat("a", 30) + "\x1b[0m"
	out := panel("", body, 40, styleBorder)

	if !strings.Contains(out, body) {
		t.Errorf("panel altered a body line that fits:\nwant substring %q\ngot\n%q", body, out)
	}
	if !strings.Contains(out, "\x1b[0m") {
		t.Errorf("panel dropped the reset escape, the colour would bleed:\n%q", out)
	}
	if strings.Contains(out, "…") {
		t.Errorf("panel truncated a line that fits:\n%q", out)
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w != 40 {
			t.Errorf("line %d is %d wide, want 40: %q", i, w, line)
		}
	}
}

// TestPanelClosesEveryLineAtTheSameColumn: the top line has to end where the
// bottom line ends, otherwise the frame looks torn open at the top right.
func TestPanelClosesEveryLineAtTheSameColumn(t *testing.T) {
	for _, border := range []lipgloss.Style{styleBorder, styleFocus, styleError} {
		for _, title := range []string{"", "Frames"} {
			out := panel(title, "eine Zeile\n", 40, border)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w != 40 {
					t.Errorf("title %q: line %d is %d wide, want 40: %q",
						title, i, w, line)
				}
			}
		}
	}
}

// TestPanelStaysInsideNarrowWidths: a tiny terminal must not make the frame
// draw wider than it is allowed to.
func TestPanelStaysInsideNarrowWidths(t *testing.T) {
	for width := 0; width <= 10; width++ {
		out := panel("Frames", "erste Zeile\nzweite Zeile", width, styleFocus)
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d wide: %q", width, i, w, line)
			}
		}
	}
}

// TestFooterShowsHintsOrError: the error replaces the hints, it does not append.
func TestFooterShowsHintsOrError(t *testing.T) {
	hints := renderFooter(80, [][]string{{"j/k bewegen"}}, "")
	if !strings.Contains(hints, "j/k bewegen") {
		t.Errorf("footer lost its hints: %q", hints)
	}
	failed := renderFooter(80, [][]string{{"j/k bewegen"}}, "Speichern fehlgeschlagen")
	if !strings.Contains(failed, "Speichern fehlgeschlagen") {
		t.Errorf("footer lost the error: %q", failed)
	}
	if strings.Contains(failed, "j/k bewegen") {
		t.Errorf("error must replace the hints, got %q", failed)
	}
}

// TestFooterRendersOneLinePerGroup: two groups, two lines.
func TestFooterRendersOneLinePerGroup(t *testing.T) {
	groups := [][]string{{"j/k bewegen", "[ ] Zeitraum"}, {"? hilfe", "q ende"}}
	out := renderFooter(100, groups, "")
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "[ ] Zeitraum") || !strings.Contains(lines[1], "q ende") {
		t.Errorf("groups landed on the wrong lines: %q", out)
	}
}

// TestFooterMergesGroupsWhenOnlyOneLineFits: the period keys and the two exits
// survive; that is the whole point of the change.
func TestFooterMergesGroupsWhenOnlyOneLineFits(t *testing.T) {
	groups := [][]string{{"j/k bewegen", "[ ] Zeitraum"}, {"? hilfe", "q ende", "n neu", "d löschen"}}
	out := renderFooter(60, groups[:1], "") // caller passes what fits
	if strings.Contains(out, "\n") {
		t.Errorf("single group must be one line: %q", out)
	}
	merged := renderFooter(60, [][]string{mergeHints(groups)}, "")
	for _, want := range []string{"[ ] Zeitraum", "? hilfe", "q ende"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged footer at 60 lost %q: %q", want, merged)
		}
	}
}

// TestMergedHintsKeepThePeriodAndTheExits: the merge is not reading order. The
// navigation group ends in "t/w/m/a Tag/Woche/Monat/alles", 29 columns, and
// poured in as it stands it pushes "? hilfe" and "q ende" off an 80-column line
// — the two keys one cannot look up once they are gone. The real hints, at the
// widths a short terminal actually has.
func TestMergedHintsKeepThePeriodAndTheExits(t *testing.T) {
	merged := [][]string{mergeHints(footerHints(modeList))}
	for _, c := range []struct {
		width int
		wants []string
	}{
		{60, []string{"[ ]", "? hilfe"}}, // 58 columns: q ende no longer fits
		{80, []string{"[ ]", "? hilfe", "q ende"}},
		{100, []string{"[ ]", "? hilfe", "q ende"}},
	} {
		out := renderFooter(c.width, merged, "")
		if strings.Contains(out, "\n") {
			t.Errorf("width %d: the merged hints are one line: %q", c.width, out)
		}
		for _, want := range c.wants {
			if !strings.Contains(out, want) {
				t.Errorf("width %d: merged footer lost %q: %q", c.width, want, out)
			}
		}
	}
	// Nothing is dropped by the merge itself — shedHints decides what fits.
	var n int
	for _, g := range footerHints(modeList) {
		n += len(g)
	}
	if got := len(mergeHints(footerHints(modeList))); got != n {
		t.Errorf("merge kept %d of %d hints; it may reorder, not drop", got, n)
	}
}

// TestFooterErrorStillReplacesEverything.
func TestFooterErrorStillReplacesEverything(t *testing.T) {
	groups := [][]string{{"j/k bewegen"}, {"q ende"}}
	out := renderFooter(80, groups, "Speichern fehlgeschlagen")
	if !strings.Contains(out, "Speichern fehlgeschlagen") {
		t.Errorf("error missing: %q", out)
	}
	if strings.Contains(out, "j/k") || strings.Contains(out, "q ende") {
		t.Errorf("error must replace the hints: %q", out)
	}
	if strings.Contains(out, "\n") {
		t.Errorf("error is one line: %q", out)
	}
}

// TestFooterHintsListPeriodKeys: the gap that started this change.
func TestFooterHintsListPeriodKeys(t *testing.T) {
	groups := footerHints(modeList)
	if len(groups) != 2 {
		t.Fatalf("list hints must come in two groups, got %d", len(groups))
	}
	flat := strings.Join(append(append([]string{}, groups[0]...), groups[1]...), " ")
	for _, want := range []string{"[ ]", "t/w/m/a", "? hilfe", "q ende"} {
		if !strings.Contains(flat, want) {
			t.Errorf("list hints missing %q: %q", want, flat)
		}
	}
	// Group 1 is navigation and period, group 2 actions and views.
	if !strings.Contains(strings.Join(groups[0], " "), "[ ]") {
		t.Errorf("period keys belong in the first group: %q", groups[0])
	}
}

// TestFooterKeepsPeriodAndExitsAt80: shedding must not eat the keys a user
// cannot otherwise find.
func TestFooterKeepsPeriodAndExitsAt80(t *testing.T) {
	groups := footerHints(modeList)
	first := renderFooter(80, groups[:1], "")
	if !strings.Contains(first, "[ ]") {
		t.Errorf("period hint gone at 80 columns: %q", first)
	}
	second := renderFooter(80, groups[1:], "")
	for _, want := range []string{"? hilfe", "q ende"} {
		if !strings.Contains(second, want) {
			t.Errorf("exit hint %q gone at 80 columns: %q", want, second)
		}
	}
}

// TestFooterMeasuresBeforeStyling: colour must not change which hints survive.
// shedHints measures the plain hint and styles only what it keeps; measuring a
// styled one is harmless as long as the measure steps over escape sequences,
// which lipgloss.Width does — but swap it for one that counts runes or bytes
// and the escapes become columns, and the footer silently sheds hints that fit.
// That is the trap panel fell into with its body lines. Under go test the
// renderer strips colour, so the profile has to be forced on or there is
// nothing here to measure.
func TestFooterMeasuresBeforeStyling(t *testing.T) {
	count := func(out string) int {
		n := 0
		for _, line := range strings.Split(out, "\n") {
			if body := strings.TrimSpace(line); body != "" {
				n += len(strings.Split(body, hintSep))
			}
		}
		return n
	}
	groups := footerHints(modeList)
	widths := []int{40, 60, 80, 100, 140}
	plain := make(map[int]int, len(widths))
	for _, width := range widths {
		plain[width] = count(renderFooter(width, groups, ""))
	}

	// 0 is termenv.TrueColor; see TestFatalPanelIsErrorColoured for why it is
	// written as a number. Restored by defer, and the profile is set once
	// outside the loop: it is global, so a panic in between would leave every
	// later test comparing against coloured strings.
	saved := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(0)
	defer lipgloss.SetColorProfile(saved)

	for _, width := range widths {
		out := renderFooter(width, groups, "")
		if got := count(out); got != plain[width] {
			t.Errorf("width %d: %d hints in colour, %d without — the escapes were "+
				"measured as columns: %q", width, got, plain[width], out)
		}
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: coloured line %d is %d wide: %q", width, i, w, line)
			}
		}
	}
}

// TestFooterStaysInsideWidth: the footer is one line next to the status bar, so
// it must not wrap on a narrow terminal either.
func TestFooterStaysInsideWidth(t *testing.T) {
	for width := 0; width <= 20; width++ {
		out := renderFooter(width, footerHints(modeList), "")
		if w := lipgloss.Width(out); w > width {
			t.Errorf("width %d: footer is %d wide: %q", width, w, out)
		}
	}
}

// TestFooterHintsPerMode: every mode names the keys that actually work there.
func TestFooterHintsPerMode(t *testing.T) {
	cases := map[mode][]string{
		modeList:          {"j/k", "enter", "n", "d", "s", "o", "r", "?"},
		modeForm:          {"tab", "enter", "esc"},
		modeReport:        {"t/w/m", "esc"},
		modeOverview:      {"esc"},
		modeStartTimer:    {"enter", "esc"},
		modeConfirmDelete: {"y", "abbrechen"},
		modeConfirmCancel: {"y", "abbrechen"},
		modeHelp:          {"Taste"},
		modeFatal:         {"beendet"},
	}
	for m, wants := range cases {
		var flat []string
		for _, g := range footerHints(m) {
			flat = append(flat, g...)
		}
		got := strings.Join(flat, hintSep)
		if got == "" {
			t.Errorf("mode %d has no hints", m)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Errorf("mode %d hints %q missing %q", m, got, want)
			}
		}
	}
}

// TestChromeHeightStages: the chrome sheds in four steps so the body keeps room.
func TestChromeHeightStages(t *testing.T) {
	cases := map[int]int{40: 8, 24: 8, 23: 6, 20: 6, 19: 2, 14: 2, 13: 1, 5: 1}
	for height, want := range cases {
		if got := chromeHeight(height); got != want {
			t.Errorf("chromeHeight(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestHeaderRowBudgetStages: the comparison row is the first thing to go.
func TestHeaderRowBudgetStages(t *testing.T) {
	cases := map[int]int{40: 4, 24: 4, 23: 3, 20: 3, 19: 1, 14: 1, 13: 0, 5: 0}
	for height, want := range cases {
		if got := headerRowBudget(height); got != want {
			t.Errorf("headerRowBudget(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestChromeStagesAddUp: the three numbers are one budget seen from three
// sides. The framed header draws its rows plus two border lines, the footer
// draws its lines, and together they have to be exactly what chromeHeight
// reserves — App.View hands the body everything else, so a disagreement is a
// blank row at the bottom of the screen or a line pushed off the top.
func TestChromeStagesAddUp(t *testing.T) {
	for height := 2; height <= 40; height++ {
		header := headerRowBudget(height)
		if header > 0 && height >= chromeSlimMinHeight {
			header += 2 // the frame
		}
		if got := header + footerLines(height); got != chromeHeight(height) {
			t.Errorf("height %d: header %d + footer %d = %d, but chromeHeight says %d",
				height, header, footerLines(height), got, chromeHeight(height))
		}
	}
}

// TestFooterLinesStages: two hint lines only where the height pays for them.
func TestFooterLinesStages(t *testing.T) {
	cases := map[int]int{40: 2, 24: 2, 23: 1, 14: 1, 13: 1, 1: 1}
	for height, want := range cases {
		if got := footerLines(height); got != want {
			t.Errorf("footerLines(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestSpreadFillsTheLineAndKeepsBothFields: spread has to fill its line to the
// column, and it must not lose a field. Two adversarial cases hide here. A right
// string of exactly width-1 used to leave one column for the left field and then
// still force a space between them, so the line came out one column too wide. A
// right string that fills the budget on its own used to drop the left field
// altogether — on a narrow terminal a long running-timer value would silently
// hide "Filter  —", and naming the context is what the header is for.
func TestSpreadFillsTheLineAndKeepsBothFields(t *testing.T) {
	left := "Filter  —"
	for width := 3; width <= 40; width++ {
		for _, rightW := range []int{1, width - 2, width - 1, width, width + 5} {
			if rightW < 1 {
				continue
			}
			right := strings.Repeat("r", rightW)
			out := spread(left, right, width)
			if w := lipgloss.Width(out); w != width {
				t.Errorf("width %d right %d: line is %d wide: %q", width, rightW, w, out)
			}
			if !strings.Contains(out, "F") {
				t.Errorf("width %d right %d: left field dropped: %q", width, rightW, out)
			}
			if !strings.Contains(out, "r") {
				t.Errorf("width %d right %d: right field dropped: %q", width, rightW, out)
			}
		}
	}
	// Below three columns there is no room for two fields and the space between
	// them, and a single field must still stay inside the width.
	for width := 0; width < 3; width++ {
		for _, out := range []string{
			spread(left, "rechts", width), spread("", "rechts", width), spread(left, "", width),
		} {
			if w := lipgloss.Width(out); w > width {
				t.Errorf("width %d: line is %d wide: %q", width, w, out)
			}
		}
	}
}

// TestHeaderShowsFieldsFramed: at full height the header is framed, carries the
// version and every field label and value. Four field rows plus the two border
// lines are the six the chrome budgets for it from height 24 up.
func TestHeaderShowsFieldsFramed(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07.–26.07."}, {"Summe", "12h 30m"}},
		{{"Vorwoche", "8h 00m"}, {"Monat", "40h 15m"}},
		{{"Filter", "—"}, {"", "3 Frames · 2 Projekte"}},
		{{}, {"", "▶ schnaq 1:23:45"}},
	}
	out := renderHeader(100, 24, "0.1.0", rows)
	for _, want := range []string{"watson-tui", "0.1.0", "Zeitraum", "Woche 20.07.–26.07.",
		"Summe", "12h 30m", "Vorwoche", "Filter", "▶ schnaq 1:23:45"} {
		if !strings.Contains(out, want) {
			t.Errorf("framed header missing %q:\n%s", want, out)
		}
	}
	if lines := strings.Count(out, "\n") + 1; lines != 6 {
		t.Errorf("framed header has %d lines, want 6:\n%s", lines, out)
	}
}

// TestHeaderFillsItsRowBudget: the framed header occupies exactly the lines
// chromeHeight reserves, whatever the mode hands it. Modes differ in how much
// context they have — the form has one field row, the list four — and View
// gives the body everything the chrome does not take, so a header that
// shrink-wraps its rows leaves blank rows at the bottom of the screen.
//
// Rows are added and removed just before the last one, because the last row is
// the running timer in every mode and the spec puts it bottom right.
func TestHeaderFillsItsRowBudget(t *testing.T) {
	const timer = "▶ schnaq 1:23:45"
	for _, height := range []int{30, 24, 23, 20} {
		budget := headerRowBudget(height)
		for _, n := range []int{1, 2, 3, 4, 5, 6} {
			rows := make([][]headerField, n)
			for i := range rows {
				rows[i] = []headerField{{"Feld", fmt.Sprintf("wert %d", i)}}
			}
			rows[n-1] = []headerField{{}, {"", timer}}

			out := renderHeader(100, height, "0.1.0", rows)
			if lines := strings.Count(out, "\n") + 1; lines != budget+2 {
				t.Errorf("height %d with %d rows: header has %d lines, want %d:\n%s",
					height, n, lines, budget+2, out)
			}
			if !strings.Contains(out, timer) {
				t.Errorf("height %d with %d rows: the timer row must survive:\n%s",
					height, n, out)
			}
			// It survives as the last row, not merely somewhere.
			lines := strings.Split(out, "\n")
			if last := lines[len(lines)-2]; !strings.Contains(last, timer) {
				t.Errorf("height %d with %d rows: timer is not on the last row: %q",
					height, n, last)
			}
		}
	}
}

// TestHeaderCollapsesToOneLine: between 12 and 19 lines the header keeps the
// values but drops the frame.
func TestHeaderCollapsesToOneLine(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07."}, {"Frames", "12"}},
		{{"Filter", "—"}, {"", "kein Timer"}},
	}
	out := renderHeader(100, 15, "0.1.0", rows)
	if lines := strings.Count(out, "\n") + 1; lines != 1 {
		t.Errorf("collapsed header has %d lines, want 1: %q", lines, out)
	}
	if !strings.Contains(out, "Woche 20.07.") || !strings.Contains(out, "kein Timer") {
		t.Errorf("collapsed header lost context: %q", out)
	}
	if strings.Contains(out, "╭") {
		t.Errorf("collapsed header must not draw a frame: %q", out)
	}
}

// TestHeaderVanishesOnTinyTerminals: below 12 lines every row belongs to the body.
func TestHeaderVanishesOnTinyTerminals(t *testing.T) {
	rows := [][]headerField{{{"Zeitraum", "Woche"}, {"Frames", "1"}}}
	if out := renderHeader(100, 10, "0.1.0", rows); out != "" {
		t.Errorf("header must be empty at height 10, got %q", out)
	}
}

// TestHeaderRespectsWidth: no rendered line may exceed the terminal width.
func TestHeaderRespectsWidth(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", strings.Repeat("lang ", 30)}, {"Frames", "999"}},
		{{"Filter", strings.Repeat("filter ", 20)}, {"", "▶ projekt 1:23:45"}},
	}
	for _, width := range []int{40, 80, 100} {
		for _, height := range []int{30, 15} {
			out := renderHeader(width, height, "0.1.0", rows)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d height %d line %d is %d wide: %q", width, height, i, w, line)
				}
			}
		}
	}
}

// TestHeaderCutsBetweenAnsiSequences: header values arrive pre-styled — the
// running timer is green, the filter comes from a bubbles input — so a value
// that has to be shortened must be cut by display width, between the escape
// sequences. A rune-based cut counts the escapes, fires on a value that would
// have fit, lands inside a sequence and drops the trailing reset, which bleeds
// the colour across the rest of the screen. Both branches that shorten a value
// are covered: the framed one and the single line.
func TestHeaderCutsBetweenAnsiSequences(t *testing.T) {
	// Display width 60, but 69 runes, so it has to be cut at width 40 either way.
	long := "\x1b[31m" + strings.Repeat("a", 60) + "\x1b[0m"
	rows := [][]headerField{{{"Zeitraum", long}, {"Frames", "999"}}}
	for _, height := range []int{30, 15} {
		out := renderHeader(40, height, "0.1.0", rows)
		for i, line := range strings.Split(out, "\n") {
			start := strings.Index(line, "\x1b[31m")
			if start < 0 {
				continue
			}
			if !strings.Contains(line[start:], "\x1b[0m") {
				t.Errorf("height %d line %d opens a colour and never closes it: %q",
					height, i, line)
			}
		}
	}
}

// TestHeaderStaysInsideNarrowWidths: the width promise has to hold at the
// bottom end too. panel already refuses to draw below four columns, but the
// single-line branch composes its own line out of a leading space and the
// spread values, so it needs the same guard the footer has — otherwise a
// two-column line lands on a one-column terminal.
func TestHeaderStaysInsideNarrowWidths(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07."}, {"Frames", "12"}},
		{{"Filter", "—"}, {"", "▶ schnaq 1:23:45"}},
	}
	for width := 0; width <= 10; width++ {
		for _, height := range []int{30, 15} {
			out := renderHeader(width, height, "0.1.0", rows)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d height %d: line %d is %d wide: %q",
						width, height, i, w, line)
				}
			}
		}
	}
}

// TestSpreadKeepsTheRightFieldRight: a row without a left field must still put
// its right field at the last column. Every mode but the list passes an empty
// left field on the header's second row, and that row is the running timer — the
// spec puts it bottom right in every mode, so an unpadded return value moved it
// to the bottom left in eight of nine modes.
func TestSpreadKeepsTheRightFieldRight(t *testing.T) {
	const right = "▶ schnaq 1:23:45"
	for width := 1; width <= 40; width++ {
		out := spread("", right, width)
		if w := lipgloss.Width(out); w != width {
			t.Errorf("width %d: line is %d wide: %q", width, w, out)
		}
		clipped := clipWidth(right, width)
		if !strings.HasSuffix(out, clipped) {
			t.Errorf("width %d: %q does not end in the right field", width, out)
		}
		// The padding has to sit in front of the value, not behind it. (A clip can
		// land on a space inside the value, so a trailing space proves nothing.)
		if lipgloss.Width(clipped) < width && !strings.HasPrefix(out, " ") {
			t.Errorf("width %d: right field is not padded to the right edge: %q", width, out)
		}
	}
}

// TestFooterShedsWholeHints: the list hints are 58 and 116 columns, so on any
// normal terminal some have to go. They go from the tail and they go whole — a
// footer ending in "q en" or a dangling separator is worse than one hint fewer.
//
// Both numbers are asserted, not just recited. shedHints's doc quotes the 116
// as the reason it exists, and that figure had already gone stale once — it
// said 116 while the hints measured 109, and only came true again when the
// English two were said in German. A number a comment leans on is worth a line
// of test; when this fails, fix the two comments rather than the number.
func TestFooterShedsWholeHints(t *testing.T) {
	groups := footerHints(modeList)
	for i, want := range []int{58, 116} {
		if got := lipgloss.Width(strings.Join(groups[i], hintSep)); got != want {
			t.Errorf("list hint group %d is %d columns, the comments here and on "+
				"shedHints say %d", i, got, want)
		}
	}
	whole := map[string]bool{}
	for _, g := range groups {
		for _, h := range g {
			whole[h] = true
		}
	}
	for width := 0; width <= 140; width++ {
		out := renderFooter(width, groups, "")
		if w := lipgloss.Width(out); w > width {
			t.Errorf("width %d: footer is %d wide: %q", width, w, out)
		}
		for _, line := range strings.Split(out, "\n") {
			body := strings.TrimSpace(line)
			if body == "" {
				continue
			}
			if strings.Contains(body, "…") {
				t.Errorf("width %d: a hint was cut instead of dropped: %q", width, line)
			}
			if strings.HasPrefix(body, "·") || strings.HasSuffix(body, "·") {
				t.Errorf("width %d: dangling separator: %q", width, line)
			}
			for _, part := range strings.Split(body, hintSep) {
				if !whole[part] {
					t.Errorf("width %d: %q is not a whole hint of %v", width, part, groups)
				}
			}
		}
	}
}

// TestFooterKeepsHelpAndQuitAt80: below 100 columns the tail of the list hints
// used to be cut off, and the two keys that were cut first were the two that
// must never become undiscoverable. They now come before the keys the help
// screen can still teach, so both survive the narrowest terminal this UI
// targets.
func TestFooterKeepsHelpAndQuitAt80(t *testing.T) {
	for _, width := range []int{80, 100, 120} {
		out := renderFooter(width, footerHints(modeList), "")
		for _, want := range []string{"? hilfe", "q ende"} {
			if !strings.Contains(out, want) {
				t.Errorf("width %d: footer lost %q: %q", width, want, out)
			}
		}
	}
}

// TestStyleHintAccentsOnlyRealKeys: the accent on a hint's first word is a
// promise that the word is a key one can press. "andere Taste abbrechen" and
// "beliebige Taste …" name a class of keys, not one, so they are dimmed whole.
//
// The colour profile has to be forced: under go test the renderer strips
// colour, both styles render to the bare string, and every assertion here would
// pass on any implementation at all. Set once and restored by defer, because
// the profile is global — see TestFooterMeasuresBeforeStyling.
func TestStyleHintAccentsOnlyRealKeys(t *testing.T) {
	saved := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(0) // termenv.TrueColor
	defer lipgloss.SetColorProfile(saved)

	// Guard against the test proving nothing because the two styles agree.
	if styleKey.Render("x") == styleDim.Render("x") {
		t.Fatal("styleKey and styleDim render alike; this test cannot tell them apart")
	}
	accented := func(h, word string) bool {
		return strings.Contains(h, styleKey.Render(word))
	}
	for _, h := range []string{"j/k bewegen", "enter bearbeiten", "q ende"} {
		word := strings.SplitN(h, " ", 2)[0]
		if !accented(styleHint(h), word) {
			t.Errorf("%q: the key %q must carry the accent: %q", h, word, styleHint(h))
		}
	}
	for _, h := range []string{"andere Taste abbrechen", "beliebige Taste schließt die Hilfe",
		"beliebige Taste beendet watson-tui"} {
		word := strings.SplitN(h, " ", 2)[0]
		if accented(styleHint(h), word) {
			t.Errorf("%q: %q is no key and must not be offered as one: %q", h, word, styleHint(h))
		}
		if got, want := styleHint(h), styleDim.Render(h); got != want {
			t.Errorf("%q: a keyless hint is dimmed whole:\n got %q\nwant %q", h, got, want)
		}
	}
	// The deny-list is not dead code: the modes really do hand out hints that
	// begin with each of its words, and every one of them comes back dimmed. A
	// rewording that leaves an entry unused fails here rather than rotting.
	seen := map[string]bool{}
	for m := modeList; m <= modeOverview; m++ {
		for _, g := range footerHints(m) {
			for _, h := range g {
				word := strings.SplitN(h, " ", 2)[0]
				if !keylessLead[word] {
					continue
				}
				seen[word] = true
				if got, want := styleHint(h), styleDim.Render(h); got != want {
					t.Errorf("mode %d: %q must be dimmed whole:\n got %q\nwant %q", m, h, got, want)
				}
			}
		}
	}
	for word := range keylessLead {
		if !seen[word] {
			t.Errorf("keylessLead holds %q, but no mode hands out a hint starting with it", word)
		}
	}
}

// TestFooterHintsAreGerman: the UI speaks German, the identifiers English.
// "enter edit" and "/ filter" stood here once and contradicted the help screen,
// which teaches the same two keys as "Frame editieren" and "filtern".
//
// Whole hint strings, not substrings: "/ filtern" contains "filter", so a
// Contains check would pass on the very string it is supposed to reject. And
// presence alone does not pin anything — a refactor that adds the English
// spelling back beside the German one has to fail too, so the English ones are
// asserted absent.
func TestFooterHintsAreGerman(t *testing.T) {
	have := map[string]bool{}
	for _, g := range footerHints(modeList) {
		for _, h := range g {
			have[h] = true
		}
	}
	for _, want := range []string{"enter bearbeiten", "/ filtern"} {
		if !have[want] {
			t.Errorf("list hints lost the German hint %q: %v", want, footerHints(modeList))
		}
	}
	for _, unwanted := range []string{"enter edit", "/ filter"} {
		if have[unwanted] {
			t.Errorf("list hints went back to English with %q: %v", unwanted, footerHints(modeList))
		}
	}
}

// TestHelpFitsEveryTerminalWithAHeader: the help screen does not scroll, so
// fitBody cuts whatever does not fit — from the bottom of the list, where the
// quit key sits. The budget is taken from the code, not from a literal, so a
// change to the collapse thresholds cannot silently make the help too long again.
//
// A sweep rather than one height, because the budget does not grow with the
// terminal: at 19 lines the chrome costs two and the help gets fifteen, at 20 it
// costs six and the help gets twelve. The floor of the sweep is the shortest
// terminal that still draws a header, and it is the tightest of them all —
// exactly ten lines. Below it the help is clipped whatever it says, and the
// footer is the only thing left that names a key.
//
// This is the arithmetic explained; that the quit key really survives the
// assembled frame is TestQuitKeyIsReachableOnAShortTerminal, which renders it.
func TestHelpFitsEveryTerminalWithAHeader(t *testing.T) {
	for height := headerLineMinHeight; height <= 40; height++ {
		// What View leaves the help: the terminal minus the chrome, minus the two
		// border lines of the panel drawn around the body.
		budget := height - chromeHeight(height) - 2
		if lines := strings.Count(helpView(), "\n") + 1; lines > budget {
			t.Errorf("help body has %d lines, a %d-line terminal fits %d — the tail "+
				"of the list would be clipped:\n%s", lines, height, budget, helpView())
		}
	}
}

// TestHelpTeachesThePeriodKeysFirst: [ and ] are why this feature exists — they
// shifted the period and nothing on the screen said so. The help lists them
// right behind the navigation keys, ahead of the actions, for two reasons: a
// terminal too short even for ten lines loses the tail rather than them, and the
// eye that goes looking for "how do I get to last week" finds them at the top.
//
// The ‹ › is asserted too: periodField draws those angles around the period to
// stand for these keys, and the help is the one place that says so in words.
func TestHelpTeachesThePeriodKeysFirst(t *testing.T) {
	lines := strings.Split(helpView(), "\n")
	for _, want := range []string{"[ / ]", "t/w/m/a"} {
		at := -1
		for i, line := range lines {
			if strings.Contains(line, want) {
				at = i
			}
		}
		switch {
		case at < 0:
			t.Errorf("the help does not name %q:\n%s", want, helpView())
		case at > 2:
			t.Errorf("%q sits on line %d; the period keys belong in the first three, "+
				"where a clipped help still shows them:\n%s", want, at+1, helpView())
		}
	}
	if !strings.Contains(helpView(), "‹ ›") {
		t.Errorf("the help must explain the ‹ › periodField draws around the period:\n%s",
			helpView())
	}
}
