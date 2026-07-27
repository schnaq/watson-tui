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
//
// border is the frame's colour. It replaced a focused bool that production code
// never set to true, because the spec wants the fatal screen framed in the error
// colour and a bool cannot say that; styleFocus is what a focused panel passes
// once there is more than one panel to focus.
func panel(title, body string, width int, border lipgloss.Style) string {
	// Four columns are the frame and its gutter. Anything narrower has no room
	// for a panel and could only be drawn by breaking the width promise.
	if width < 4 {
		return ""
	}
	inner := width - 2 // the run between the two corners

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

// renderFooter draws the key hints, one line per group. An error replaces them
// all: a failed write outranks a reminder of which key moves the cursor.
func renderFooter(width int, groups [][]string, errMsg string) string {
	avail := width - 2 // one leading space, one column of margin on the right
	if avail < 1 {
		return ""
	}
	if errMsg != "" {
		// Cut, not shed: an error is one sentence, and its beginning says what
		// failed. The ellipsis truncate leaves marks that there is more.
		return " " + styleError.Render(truncate(errMsg, avail))
	}
	lines := make([]string, 0, len(groups))
	for _, g := range groups {
		lines = append(lines, " "+shedHints(g, avail))
	}
	return strings.Join(lines, "\n")
}

// hintSep separates two key hints in the footer.
const hintSep = " · "

// shedHints joins the hints that fit avail columns and drops the rest from the
// tail. Whole hints only: the list mode's second group alone needs 116 columns,
// so on any normal terminal something has to go, and a footer ending in "q en"
// reads as a key that does not exist. Callers order their hints by how badly the
// user needs them, so what goes is what the help screen can still teach.
// TestFooterShedsWholeHints asserts the 116 instead of leaving it to rot here.
//
// Every hint is measured plain and styled only once it is kept. lipgloss.Width
// does step over escape sequences, so that order is not what makes the
// arithmetic come out today — it is what keeps it coming out: a measure that
// counts runes or bytes (truncate, len) is one edit away, and on a styled hint
// it would read the escapes as columns and shed hints that fit. panel was
// bitten by exactly that. TestFooterMeasuresBeforeStyling forces a colour
// profile and pins it, because under go test the renderer strips colour and
// there would be nothing to measure.
func shedHints(hints []string, avail int) string {
	var kept []string
	used := 0
	for _, h := range hints {
		w := lipgloss.Width(h)
		if len(kept) > 0 {
			w += lipgloss.Width(hintSep)
		}
		if used+w > avail {
			break
		}
		used += w
		kept = append(kept, styleHint(h))
	}
	return strings.Join(kept, hintSep)
}

// keylessLead are the words a hint that names no single key begins with.
// "andere Taste abbrechen" and "beliebige Taste schließt die Hilfe" describe a
// class of keys, and the accent styleHint puts on a first word offers "andere"
// and "beliebige" as keys one could press.
//
// A deny-list of lead words rather than an allow-list of keys, because the two
// fail in opposite directions: a new hint whose key is missing from an
// allow-list loses its accent and the key stops standing out, while a new
// keyless phrase missing from this list merely gets one accent too many — and
// no key in this UI is spelled like a German adjective, so the list stays short.
var keylessLead = map[string]bool{"andere": true, "beliebige": true}

// styleHint sets the key off from what it does: "j/k" in the accent colour,
// "bewegen" dimmed. Two shades in a line of ten hints are what let the eye find
// the key it is looking for instead of reading the line. A hint that names no
// key is dimmed whole — see keylessLead.
func styleHint(h string) string {
	if i := strings.IndexByte(h, ' '); i > 0 {
		if keylessLead[h[:i]] {
			return styleDim.Render(h)
		}
		return styleKey.Render(h[:i]) + styleDim.Render(h[i:])
	}
	return styleKey.Render(h)
}

// mergeHints pours the groups into a single line, for a terminal too short to
// give each of them one. Not in reading order: the first group ends in
// "t/w/m/a Tag/Woche/Monat/alles", twenty-nine columns, and poured in as it
// stands it pushes "? hilfe" and "q ende" off an 80-column line — the two keys
// nobody can look up once they are gone. So the first group keeps its two
// leading hints ahead of the rest and its tail goes last. Nothing is dropped
// here; shedHints decides what fits.
func mergeHints(groups [][]string) []string {
	if len(groups) == 0 {
		return nil
	}
	const lead = 2 // "j/k bewegen" and "← → Zeitraum"
	head, tail := groups[0], []string(nil)
	if len(head) > lead {
		head, tail = head[:lead], head[lead:]
	}
	merged := append([]string{}, head...)
	for _, g := range groups[1:] {
		merged = append(merged, g...)
	}
	return append(merged, tail...)
}

// footerHints lists the keys that work in the given mode, in groups of one line
// each, most important first — shedHints drops from the tail of every group.
// Only the list has enough keys to need two lines; the other modes keep one.
func footerHints(m mode) [][]string {
	switch m {
	case modeForm:
		return [][]string{{"tab Feld", "→ Vorschlag", "enter speichern", "esc abbrechen"}}
	case modeReport:
		return [][]string{{"t/w/m Zeitraum", "[ ] verschieben", "esc zurück"}}
	case modeOverview:
		return [][]string{{"esc zurück"}}
	case modeStartTimer:
		return [][]string{{"tab Feld", "→ Vorschlag", "enter starten", "esc abbrechen"}}
	case modeConfirmDelete:
		return [][]string{{"y löschen", "andere Taste abbrechen"}}
	case modeConfirmCancel:
		return [][]string{{"y verwerfen", "andere Taste abbrechen"}}
	case modeHelp:
		return [][]string{{"beliebige Taste schließt die Hilfe"}}
	case modeFatal:
		return [][]string{{"beliebige Taste beendet watson-tui"}}
	default:
		// Navigation and period on the first line, actions and views on the
		// second. "← → Zeitraum" and "t/w/m/a" are why this change exists: the
		// period read as a state because no hint ever said it could be moved.
		//
		// "? hilfe" and "q ende" sit second and third in the second group on
		// purpose. About six hints of it survive at 80 columns, and these two
		// are the ones there is no way left to look up once they are gone. They
		// used to sit third and fourth, behind "n neu"; "enter bearbeiten" is six
		// columns wider than the "enter edit" that stood here in English, and at
		// the 60 columns the merged single line has to work at, that width came
		// out of "? hilfe". Moving it up one slot is what buys it back.
		return [][]string{
			{"j/k bewegen", "← → Zeitraum", "t/w/m/a Tag/Woche/Monat/alles"},
			{"enter bearbeiten", "? hilfe", "q ende", "n neu", "d löschen", "s timer",
				"/ filtern", "r report", "o übersicht", "R neu laden"},
		}
	}
}

// headerField is one labelled value in the header panel.
type headerField struct {
	label string
	value string
}

// Collapse thresholds. The chrome must never grow so tall that the list it
// frames has no room left. With the context header and the second hint line it
// stands at eight lines, so it sheds in four steps rather than two: first the
// comparison row and the second hint line, then the frame around the header,
// then the header itself.
const (
	chromeFullMinHeight = 24 // framed header, 4 field rows, 2 hint lines
	chromeSlimMinHeight = 20 // framed header, 3 field rows, 1 hint line
	headerLineMinHeight = 14 // single unframed header line, 1 hint line
)

// chromeHeight reports how many lines the chrome occupies at this terminal
// height, so App.View can hand the rest to the body. It is a promise, not an
// estimate: headerRowBudget and footerLines add up to exactly this, and the
// header and the footer fill their share whatever the mode has to say.
func chromeHeight(height int) int {
	switch {
	case height >= chromeFullMinHeight:
		return 8
	case height >= chromeSlimMinHeight:
		return 6
	case height >= headerLineMinHeight:
		return 2
	default:
		return 1
	}
}

// headerRowBudget reports how many field rows the header shows. The comparison
// row is the first to go: it is the one field a user can work without.
func headerRowBudget(height int) int {
	switch {
	case height >= chromeFullMinHeight:
		return 4
	case height >= chromeSlimMinHeight:
		return 3
	case height >= headerLineMinHeight:
		return 1
	default:
		return 0
	}
}

// footerLines reports how many hint lines fit. Two is what it takes to show the
// period keys next to the actions; below that the caller pours both groups into
// the one line it has (see mergeHints).
func footerLines(height int) int {
	if height >= chromeFullMinHeight {
		return 2
	}
	return 1
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
		// Padded, not just clipped: the header's second row is the running timer
		// in every mode, and every mode but the list leaves the field beside it
		// empty. Returning the value bare would put the timer at the left edge —
		// the spec has it bottom right, and that is where the eye looks for it
		// when the mode changes but the timer keeps running.
		right = clipWidth(right, width)
		return strings.Repeat(" ", max(width-lipgloss.Width(right), 0)) + right
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

// fitBody fits body to exactly maxLines lines of at most maxWidth columns, so
// the panel around it neither grows past the height App.View budgeted for it nor
// shrink-wraps around its content. Padding matters as much as cutting: a
// bordered box drawn around three rows with the footer floating in the middle of
// the screen is what makes a TUI look unfinished — k9s and lazygit fill the
// viewport, and so the panel bottom sits just above the footer here.
//
// The views hand out pre-styled lines, so the cut counts printable columns
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
	// Empty lines, not spaces: panel pads every row to its inner width anyway,
	// and the unframed overview has nothing to gain from trailing blanks.
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// fitHeaderRows makes rows exactly budget long, so the framed header fills the
// lines chromeHeight reserved for it. Modes differ in how much context they
// have — the list four rows, the form one — and View gives the body every line
// the chrome does not take, so a header that shrink-wraps its rows would leave
// blank rows at the bottom of the screen and the count would depend on the mode.
//
// Rows are added and taken away just before the last one, because the last row
// is the running timer in every mode and the spec puts it bottom right. The
// caller decides which row above it gives way (the comparison row does); this
// is only the hard guarantee that the count holds either way.
func fitHeaderRows(rows [][]headerField, budget int) [][]headerField {
	if budget < 1 {
		return nil
	}
	if len(rows) == 0 {
		return make([][]headerField, budget)
	}
	last := rows[len(rows)-1]
	fitted := make([][]headerField, budget)
	for i := 0; i < budget-1 && i < len(rows)-1; i++ {
		fitted[i] = rows[i]
	}
	fitted[budget-1] = last
	return fitted
}

// renderHeader draws the context panel. From chromeSlimMinHeight up it is
// framed and shows as many rows as headerRowBudget allows; between the
// thresholds it collapses to a single unframed line carrying the outermost
// values; below, it disappears.
func renderHeader(width, height int, version string, rows [][]headerField) string {
	switch {
	case height >= chromeSlimMinHeight:
		var lines []string
		for _, r := range fitHeaderRows(rows, headerRowBudget(height)) {
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
		return panel("watson-tui "+version, strings.Join(lines, "\n"), width, styleBorder)

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
