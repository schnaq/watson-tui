package tui

// This file holds the frame around every view: the bordered panel and the key
// hint footer. Both take the width they may use and never draw wider than that,
// so a resize can never wrap a line and shift the whole layout.

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
