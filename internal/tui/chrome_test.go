package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestPanelFramesBodyAndTitle: the panel draws a border, carries its title in
// the top line and never exceeds the width it was given.
func TestPanelFramesBodyAndTitle(t *testing.T) {
	out := panel("Frames", "erste Zeile\nzweite Zeile", 40, false)
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
		out := panel(strings.Repeat("sehr langer titel ", 5), "body", width, false)
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
	out := panel("", body, 40, false)

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
	for _, focused := range []bool{false, true} {
		for _, title := range []string{"", "Frames"} {
			out := panel(title, "eine Zeile\n", 40, focused)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w != 40 {
					t.Errorf("title %q focused %v: line %d is %d wide, want 40: %q",
						title, focused, i, w, line)
				}
			}
		}
	}
}

// TestPanelStaysInsideNarrowWidths: a tiny terminal must not make the frame
// draw wider than it is allowed to.
func TestPanelStaysInsideNarrowWidths(t *testing.T) {
	for width := 0; width <= 10; width++ {
		out := panel("Frames", "erste Zeile\nzweite Zeile", width, true)
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d wide: %q", width, i, w, line)
			}
		}
	}
}

// TestFooterShowsHintsOrError: the error replaces the hints, it does not append.
func TestFooterShowsHintsOrError(t *testing.T) {
	hints := renderFooter(80, "j/k bewegen", "")
	if !strings.Contains(hints, "j/k bewegen") {
		t.Errorf("footer lost its hints: %q", hints)
	}
	failed := renderFooter(80, "j/k bewegen", "Speichern fehlgeschlagen")
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
		got := footerHints(m)
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
