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
// width, with at least one space between them. When the two do not both fit, the
// right field is capped first so that the left one survives: a long
// running-timer value must not push the field naming the view off the line, that
// naming is what the header is for. The floor the left field keeps — a third of
// the line — is a judgement call, not a derived number; it only has to leave
// enough to recognise the label by.
func spread(left, right string, width int) string {
	if width < 1 {
		return ""
	}
	if left == "" {
		return clipWidth(right, width)
	}
	if right == "" {
		return clipWidth(left, width)
	}
	if width < 3 {
		// No room for two fields and the space between them: keep the context.
		return clipWidth(left, width)
	}
	if lipgloss.Width(left)+1+lipgloss.Width(right) > width {
		// Cap the right field, then fit the left one into what is left. Both caps
		// stay above zero for width >= 3, so the line comes out exactly width
		// wide: clipping the left field to width-rightW-1 is what keeps the gap
		// below from having to squeeze a space in that does not fit.
		floor := min(lipgloss.Width(left), max(width/3, 1))
		right = clipWidth(right, width-floor-1)
		left = clipWidth(left, width-lipgloss.Width(right)-1)
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// fitBody cuts body down to maxLines lines of at most maxWidth columns, so the
// panel around it cannot grow past the height App.View budgeted for it. The
// views hand out pre-styled lines, so the cut counts printable columns
// (clipWidth) instead of runes — see there for why. Lines go from the bottom:
// letting the terminal scroll instead would push the header off the top, and the
// head of a view is the part that says what one is looking at.
func fitBody(body string, maxLines, maxWidth int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	// A trailing newline would otherwise become a blank padded row inside the
	// panel and cost the body a line it has content for.
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, line := range lines {
		lines[i] = clipWidth(line, maxWidth)
	}
	return strings.Join(lines, "\n")
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
