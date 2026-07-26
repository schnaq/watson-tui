package tui

// This file holds the frame around every view: the context header, the bordered
// panel and the key hint footer. All of them take the width they may use and
// never draw wider than that, so a resize can never wrap a line and shift the
// whole layout. The header additionally takes the height, because on a short
// terminal the chrome has to give its rows back to the body.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// panel frames body and writes title into the top border line, the way k9s
// labels its views. width is the total width including the border, so callers
// pass the terminal width and the content gets width-4: two border columns and
// two spaces of gutter.
func panel(title, body string, width int, focused bool) string {
	// Four columns are the frame and its gutter. Anything narrower has no room
	// for a panel and could only be drawn by breaking the width promise.
	if width < 4 {
		return ""
	}
	inner := width - 2 // the run between the two corners
	border := styleBorder
	if focused {
		border = styleFocus
	}

	// The title sits in the top line: "╭─ title ─...─╮". Of that only the
	// dashes, the two spaces and the label count towards inner; the corners do
	// not. A label needs five columns of inner to leave a dash on either side.
	label := ""
	if title != "" && inner >= 5 {
		label = truncate(title, inner-4)
	}
	used := 1 // the dash right after the corner
	top := border.Render("╭─")
	if label != "" {
		top += " " + styleAccent.Render(label) + " "
		used += lipgloss.Width(label) + 2
	}
	top += border.Render(strings.Repeat("─", max(inner-used, 0)) + "╮")

	var b strings.Builder
	b.WriteString(top + "\n")
	for _, line := range strings.Split(body, "\n") {
		// Body lines arrive pre-styled from the views, so measure display width:
		// truncate counts runes, and the ANSI escapes around a styled row would
		// make a line that fits look too long. Cutting it there would drop the
		// reset escape and bleed the colour across the rest of the screen.
		if lipgloss.Width(line) > inner-2 {
			line = truncate(line, inner-2)
		}
		pad := max(inner-2-lipgloss.Width(line), 0)
		b.WriteString(border.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + border.Render("│") + "\n")
	}
	b.WriteString(border.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return b.String()
}

// renderFooter draws the key hints. An error replaces them: a failed write is
// more urgent than a reminder of which key moves the cursor.
func renderFooter(width int, hints, errMsg string) string {
	avail := width - 2 // one leading space, one column of margin on the right
	if avail < 1 {
		return ""
	}
	if errMsg != "" {
		return " " + styleError.Render(truncate(errMsg, avail))
	}
	return " " + styleDim.Render(truncate(hints, avail))
}

// footerHints lists the keys that work in the given mode.
func footerHints(m mode) string {
	switch m {
	case modeForm:
		return "tab Feld · → Vorschlag · enter speichern · esc abbrechen"
	case modeReport:
		return "t/w/m Zeitraum · [ ] verschieben · esc zurück"
	case modeOverview:
		return "esc zurück"
	case modeStartTimer:
		return "tab Feld · → Vorschlag · enter starten · esc abbrechen"
	case modeConfirmDelete:
		return "y löschen · andere Taste abbrechen"
	case modeConfirmCancel:
		return "y verwerfen · andere Taste abbrechen"
	case modeHelp:
		return "beliebige Taste schließt die Hilfe"
	case modeFatal:
		return "beliebige Taste beendet watson-tui"
	default:
		return "j/k bewegen · enter bearbeiten · n neu · d löschen · " +
			"s timer · / filtern · r report · o übersicht · ? hilfe · q ende"
	}
}

// headerField is one labelled value in the header panel.
type headerField struct {
	label string
	value string
}

// Collapse thresholds. The chrome must never grow so tall that the list it
// frames has no room left.
const (
	headerFullMinHeight = 20 // framed header (4 lines) + footer
	headerLineMinHeight = 12 // single-line header + footer
)

// chromeHeight reports how many lines the chrome occupies at this terminal
// height, so App.View can hand the rest to the body.
func chromeHeight(height int) int {
	switch {
	case height >= headerFullMinHeight:
		return 5
	case height >= headerLineMinHeight:
		return 2
	default:
		return 1
	}
}

// renderField formats "label  value"; a field without a label is just its value.
func renderField(f headerField) string {
	if f.label == "" {
		return f.value
	}
	return styleHeaderLabel.Render(f.label) + "  " + f.value
}

// clipWidth cuts s to at most max printable columns. Header fields arrive
// styled — a running timer is green, a filter comes from a bubbles input — so
// the plain truncate cannot be used here: it counts runes, so the escapes make
// it fire on a value that fits and the cut can land inside a sequence, dropping
// the reset and bleeding the colour across the rest of the screen. MaxWidth
// measures printable width, steps over the sequences and closes any style it
// cut open. The price is that it drops no ellipsis, so a clipped field just
// ends; do not "unify" the two, they cut different kinds of string.
func clipWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(max).Render(s)
}

// spread puts left at the start and right at the end of a line of the given
// width, with at least one space between them.
func spread(left, right string, width int) string {
	if right == "" {
		return clipWidth(left, width)
	}
	rightW := lipgloss.Width(right)
	if rightW >= width {
		return clipWidth(right, width)
	}
	left = clipWidth(left, max(width-rightW-1, 1))
	gap := width - lipgloss.Width(left) - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderHeader draws the context panel. Above headerFullMinHeight it is framed
// and shows every row; between the thresholds it collapses to a single
// unframed line carrying the outermost values; below, it disappears.
func renderHeader(width, height int, version string, rows [][]headerField) string {
	switch {
	case height >= headerFullMinHeight:
		var lines []string
		for _, r := range rows {
			var left, right string
			if len(r) > 0 {
				left = renderField(r[0])
			}
			if len(r) > 1 {
				right = renderField(r[1])
			}
			// panel keeps two border columns and a space of gutter on either
			// side, so its content area is width-4.
			lines = append(lines, spread(left, right, max(width-4, 1)))
		}
		return panel("watson-tui "+version, strings.Join(lines, "\n"), width, false)

	case height >= headerLineMinHeight:
		// One line has room for context plus the timer, not for four fields, so
		// keep the first value and the last one. Collect them all first and pick
		// afterwards: assigning inside the loop would make every field in
		// between briefly the right-hand one, which only works out because the
		// last write wins — too subtle to leave standing.
		var values []string
		for _, r := range rows {
			for _, f := range r {
				if f.value != "" {
					values = append(values, f.value)
				}
			}
		}
		var left, right string
		if n := len(values); n > 0 {
			left = values[0]
			if n > 1 {
				right = values[n-1]
			}
		}
		// One leading space to line the header up with the footer, one column of
		// margin on the right. Below that there is no line left to draw, and
		// drawing one anyway would break the width promise — the same guard
		// renderFooter needs, for the same reason.
		avail := width - 2
		if avail < 1 {
			return ""
		}
		return " " + spread(styleDim.Render(left), right, avail)

	default:
		return ""
	}
}
