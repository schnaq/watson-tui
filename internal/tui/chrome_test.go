package tui

import (
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
	hints := renderFooter(80, []string{"j/k bewegen"}, "")
	if !strings.Contains(hints, "j/k bewegen") {
		t.Errorf("footer lost its hints: %q", hints)
	}
	failed := renderFooter(80, []string{"j/k bewegen"}, "Speichern fehlgeschlagen")
	if !strings.Contains(failed, "Speichern fehlgeschlagen") {
		t.Errorf("footer lost the error: %q", failed)
	}
	if strings.Contains(failed, "j/k bewegen") {
		t.Errorf("error must replace the hints, got %q", failed)
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
		got := strings.Join(footerHints(m), hintSep)
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

// TestChromeHeightCollapses: the chrome must not eat the list on short
// terminals, so it sheds the header in two steps.
func TestChromeHeightCollapses(t *testing.T) {
	cases := map[int]int{30: 5, 20: 5, 19: 2, 12: 2, 11: 1, 5: 1}
	for height, want := range cases {
		if got := chromeHeight(height); got != want {
			t.Errorf("chromeHeight(%d) = %d, want %d", height, got, want)
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
// version and every field label and value.
func TestHeaderShowsFieldsFramed(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07.–26.07."}, {"Frames", "12"}},
		{{"Filter", "—"}, {"", "▶ schnaq 1:23:45"}},
	}
	out := renderHeader(100, 30, "0.1.0", rows)
	for _, want := range []string{"watson-tui", "0.1.0", "Zeitraum", "Woche 20.07.–26.07.", "Frames", "12", "Filter", "▶ schnaq 1:23:45"} {
		if !strings.Contains(out, want) {
			t.Errorf("framed header missing %q:\n%s", want, out)
		}
	}
	if lines := strings.Count(out, "\n") + 1; lines != 4 {
		t.Errorf("framed header has %d lines, want 4:\n%s", lines, out)
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

// TestFooterShedsWholeHints: the list hints are 116 columns, so on any normal
// terminal some have to go. They go from the tail and they go whole — a footer
// ending in "q en" or a dangling separator is worse than one hint fewer.
func TestFooterShedsWholeHints(t *testing.T) {
	hints := footerHints(modeList)
	whole := map[string]bool{}
	for _, h := range hints {
		whole[h] = true
	}
	for width := 0; width <= 140; width++ {
		out := renderFooter(width, hints, "")
		if w := lipgloss.Width(out); w > width {
			t.Errorf("width %d: footer is %d wide: %q", width, w, out)
		}
		body := strings.TrimSpace(out)
		if body == "" {
			continue
		}
		if strings.Contains(body, "…") {
			t.Errorf("width %d: a hint was cut instead of dropped: %q", width, out)
		}
		if strings.HasPrefix(body, "·") || strings.HasSuffix(body, "·") {
			t.Errorf("width %d: dangling separator: %q", width, out)
		}
		for _, part := range strings.Split(body, hintSep) {
			if !whole[part] {
				t.Errorf("width %d: %q is not a whole hint of %v", width, part, hints)
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

// TestHelpViewFitsTwentyLines: the help screen does not scroll, so fitBody cuts
// whatever does not fit — from the bottom of the list, where the quit key sits.
// The budget is taken from the code, not from a literal, so a change to the
// collapse thresholds cannot silently make the help screen too long again.
func TestHelpViewFitsTwentyLines(t *testing.T) {
	const height = 20
	// The panel spends two lines on its border, the chrome the rest.
	budget := height - chromeHeight(height) - 2
	if lines := strings.Count(helpView(), "\n") + 1; lines > budget {
		t.Errorf("help body has %d lines, a %d-line terminal fits %d — the tail "+
			"of the list would be clipped:\n%s", lines, height, budget, helpView())
	}
}
