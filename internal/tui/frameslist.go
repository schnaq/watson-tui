package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

type periodUnit int

const (
	unitWeek periodUnit = iota
	unitMonth
	// unitYear is not reachable by any key — it exists so a month can be
	// compared against the year containing it; see comparisonPeriods.
	unitYear
	unitDay
	unitAll
)

// period is a display time range: a unit plus a reference time inside it.
//
// Invariant, for the periods the user navigates — the list's and the report's:
// ref is the period's start from construction on. shift(ws, 0) establishes it
// and their constructors apply it. The short-lived ones built only to be
// measured do not bother (overviewColumns, comparisonPeriods): bounds() and
// label() normalise anyway, and nothing reads ref back off those.
//
// Everyone who does read ref back needs it: the header asks which month a week
// belongs to, and with ref left at the instant of construction a week across a
// month boundary answered whichever month that instant fell in, while the same
// week reached with ] and [ answered the other one.
type period struct {
	unit periodUnit
	ref  time.Time
}

// bounds returns the half-open range [from, to) in local time; ok=false for unitAll.
func (p period) bounds(weekStart time.Weekday) (from, to time.Time, ok bool) {
	if p.unit == unitAll {
		return time.Time{}, time.Time{}, false
	}
	y, m, d := p.ref.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, p.ref.Location())
	switch p.unit {
	case unitDay:
		return day, day.AddDate(0, 0, 1), true
	case unitWeek:
		diff := (int(day.Weekday()) - int(weekStart) + 7) % 7
		start := day.AddDate(0, 0, -diff)
		return start, start.AddDate(0, 0, 7), true
	case unitYear:
		start := time.Date(y, 1, 1, 0, 0, 0, 0, p.ref.Location())
		return start, start.AddDate(1, 0, 0), true
	default: // unitMonth
		start := time.Date(y, m, 1, 0, 0, 0, 0, p.ref.Location())
		return start, start.AddDate(0, 1, 0), true
	}
}

// shift moves the period by delta units (delta -1 = one week/month/day back).
// Normalizes ref to the period start first, so Jan 31 + 1 month = February.
func (p period) shift(weekStart time.Weekday, delta int) period {
	from, _, ok := p.bounds(weekStart)
	if !ok {
		return p
	}
	switch p.unit {
	case unitDay:
		p.ref = from.AddDate(0, 0, delta)
	case unitWeek:
		p.ref = from.AddDate(0, 0, 7*delta)
	case unitYear:
		p.ref = from.AddDate(delta, 0, 0)
	default:
		p.ref = from.AddDate(0, delta, 0)
	}
	return p
}

var germanDays = map[time.Weekday]string{
	time.Monday: "Montag", time.Tuesday: "Dienstag", time.Wednesday: "Mittwoch",
	time.Thursday: "Donnerstag", time.Friday: "Freitag", time.Saturday: "Samstag",
	time.Sunday: "Sonntag",
}

var germanMonths = [...]string{"", "Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember"}

func formatDay(t time.Time) string {
	return fmt.Sprintf("%s, %02d.%02d.%d", germanDays[t.Weekday()], t.Day(), int(t.Month()), t.Year())
}

// label describes the period for the status bar.
func (p period) label(weekStart time.Weekday) string {
	from, to, ok := p.bounds(weekStart)
	if !ok {
		return "alle Frames"
	}
	switch p.unit {
	case unitDay:
		return formatDay(from)
	case unitWeek:
		last := to.AddDate(0, 0, -1)
		return fmt.Sprintf("Woche %02d.%02d. – %02d.%02d.%d",
			from.Day(), int(from.Month()), last.Day(), int(last.Month()), last.Year())
	case unitYear:
		return fmt.Sprintf("Jahr %d", from.Year())
	default:
		return fmt.Sprintf("%s %d", germanMonths[int(from.Month())], from.Year())
	}
}

// row is one display line: a day header — the day and what was booked on it —
// or a frame. The header's two halves are kept apart instead of being formatted
// into one string here, because only renderRow knows the width they have to fit.
type row struct {
	isHeader bool
	title    string
	total    time.Duration
	frame    watson.Frame
}

func matchesFilter(f watson.Frame, needle string) bool {
	if strings.Contains(strings.ToLower(f.Project), needle) {
		return true
	}
	for _, tag := range f.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true
		}
	}
	return strings.HasPrefix(f.ID, needle)
}

// filterFrames narrows frames to the ones the filter matches, and hands them
// back untouched when there is none. It is not folded into buildRows because
// the header sums the same selection: its sum sits one row above the day totals
// and has to agree with them, and two places spelling out "lowercase, trim,
// match" is one place too many for that to keep holding.
func filterFrames(frames []watson.Frame, filter string) []watson.Frame {
	needle := strings.ToLower(strings.TrimSpace(filter))
	if needle == "" {
		return frames
	}
	sel := make([]watson.Frame, 0, len(frames))
	for _, fr := range frames {
		if matchesFilter(fr, needle) {
			sel = append(sel, fr)
		}
	}
	return sel
}

// buildRows filters frames to period+filter, sorts by start, groups by local day.
func buildRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string) []row {
	from, to, bounded := p.bounds(weekStart)
	var sel []watson.Frame
	for _, fr := range filterFrames(frames, filter) {
		if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
			continue
		}
		sel = append(sel, fr)
	}
	watson.SortFrames(sel)

	type group struct {
		day    time.Time
		frames []watson.Frame
		total  time.Duration
	}
	var groups []group
	for _, fr := range sel {
		y, m, d := fr.Start.Local().Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		if len(groups) == 0 || !groups[len(groups)-1].day.Equal(day) {
			groups = append(groups, group{day: day})
		}
		g := &groups[len(groups)-1]
		g.frames = append(g.frames, fr)
		g.total += fr.Duration()
	}
	var rows []row
	for _, g := range groups {
		rows = append(rows, row{isHeader: true, title: formatDay(g.day), total: g.total})
		for _, fr := range g.frames {
			rows = append(rows, row{frame: fr})
		}
	}
	return rows
}

// firstFrameRow returns the index of the first non-header row, -1 if none.
func firstFrameRow(rows []row) int {
	for i, r := range rows {
		if !r.isHeader {
			return i
		}
	}
	return -1
}

// nextFrameRow returns the next non-header row index in direction dir (+1/-1),
// or from when there is none.
func nextFrameRow(rows []row, from, dir int) int {
	for i := from + dir; i >= 0 && i < len(rows); i += dir {
		if !rows[i].isHeader {
			return i
		}
	}
	return from
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		if max <= 0 {
			return ""
		}
		return "…"
	}
	return string(r[:max-1]) + "…"
}

type listModel struct {
	rows        []row
	cursor      int // index into rows; always on a frame row; -1 when empty
	offset      int // first visible row
	per         period
	filter      string
	filtering   bool
	filterInput textinput.Model
}

// newListModel starts on the week around now. weekStart is not decoration: the
// period is normalised to its start right here, so ref means the same thing
// before the first shift as after it — see period.shift.
func newListModel(now time.Time, weekStart time.Weekday) listModel {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "Projekt, Tag oder ID"
	return listModel{
		per:         period{unit: unitWeek, ref: now}.shift(weekStart, 0),
		cursor:      -1,
		filterInput: ti,
	}
}

// refresh rebuilds rows and keeps the selection on the same frame if possible.
func (l *listModel) refresh(frames []watson.Frame, weekStart time.Weekday) {
	prevID := ""
	if f, ok := l.selected(); ok {
		prevID = f.ID
	}
	l.rows = buildRows(frames, l.per, weekStart, l.filter)
	l.cursor = firstFrameRow(l.rows)
	if prevID != "" {
		for i, r := range l.rows {
			if !r.isHeader && r.frame.ID == prevID {
				l.cursor = i
				break
			}
		}
	}
}

func (l *listModel) selected() (watson.Frame, bool) {
	if l.cursor < 0 || l.cursor >= len(l.rows) || l.rows[l.cursor].isHeader {
		return watson.Frame{}, false
	}
	return l.rows[l.cursor].frame, true
}

func (l *listModel) move(dir int) {
	if l.cursor >= 0 {
		l.cursor = nextFrameRow(l.rows, l.cursor, dir)
	}
}

// The frame row's columns, left to right: the selection marker, the start and
// stop time, the duration, the project name, the tags, the frame ID. The project
// and the tags are the elastic ones; these are their bounds. listProjMin is the
// point below which a project name stops telling two customers apart — the ID
// column drops rather than let the name go under it.
const (
	listMarkerWidth = 2  // "▌ ", or the two spaces that hold its column open
	listTimesWidth  = 11 // "09:00–17:00"
	listIDWidth     = 7  // what watson.Frame.ShortID renders
	listProjMax     = 24
	listProjMin     = 8
	listTagsMax     = 28
)

// rowLayout is the negotiated shape of a frame row: the width of each elastic
// column and whether the fixed ones are shown at all.
type rowLayout struct {
	durW, projW, tagW int
	times, id         bool
}

// columns is the number of columns a row of this layout occupies. Every column
// carries its own separator, so a dropped column takes its gap with it.
func (rl rowLayout) columns() int {
	w := listMarkerWidth + rl.durW
	if rl.times {
		w += listTimesWidth + 2
	}
	if rl.projW > 0 {
		w += 2 + rl.projW
	}
	if rl.tagW > 0 {
		w += 1 + rl.tagW
	}
	if rl.id {
		w += 1 + listIDWidth
	}
	return w
}

// grant hands the room left over after the fixed columns to the elastic ones:
// the project name up to its maximum first, the tags whatever is still there.
// That order *is* the priority — the tags are the last column to be granted room
// and therefore the first to give it back.
func (rl rowLayout) grant(width int) rowLayout {
	rl.projW, rl.tagW = 0, 0
	if room := width - rl.columns() - 2; room > 0 {
		rl.projW = min(listProjMax, room)
	}
	if room := width - rl.columns() - 1; room > 0 {
		rl.tagW = min(listTagsMax, room)
	}
	return rl
}

// listLayout negotiates a frame row against the width of the body it is rendered
// into. The row used to be a fixed format string that never looked at the width at
// all, and fitBody then cut the right edge — which is where the ID sits.
//
// What gives way, and in which order:
//
//  1. the tags shrink and then vanish. They are labels, they carry an ellipsis
//     when they are cut, and nothing outside this screen is looked up by them.
//  2. the project name shrinks, down to listProjMin. Also a label, also cut with
//     an ellipsis, but it is what tells one row from the next.
//  3. the ID column drops whole. Never narrowed: watson resolves frame IDs by
//     prefix (watson.FindByIDPrefix, and the watson CLI the same way), so a
//     shortened ID still looks like a usable one while it now names either
//     nothing, several frames, or the wrong one. No ellipsis can be spared in
//     seven columns to say otherwise, and the ID is one keystroke away in the
//     edit form anyway — so it is shown whole or not at all.
//  4. the project name shrinks on below its floor, down to a bare ellipsis.
//  5. the start and stop time drop whole, both of them, so what is left is not
//     mistaken for the other one. Only under 23 columns, and the day header still
//     dates the row.
//
// The duration never gives way: it is the number this row is billed from, and
// right-aligned as it is, a cut would take the minutes off the end and leave a
// plausible wrong number behind. Below the marker column plus the duration there
// is nothing left to give, and only there is the number itself cut — with an
// ellipsis, so it does not read as complete. See renderRow.
func listLayout(width, durW int) rowLayout {
	if full := (rowLayout{durW: durW, times: true, id: true}).grant(width); full.projW >= listProjMin {
		return full
	}
	if noID := (rowLayout{durW: durW, times: true}).grant(width); noID.columns() <= width {
		return noID
	}
	return (rowLayout{durW: min(durW, max(width-listMarkerWidth, 0))}).grant(width)
}

// view renders `height` rows into a body of `width` columns, scrolling so the
// cursor stays visible.
func (l *listModel) view(height, width int) string {
	if height < 1 {
		height = 1
	}
	if len(l.rows) == 0 {
		empty := "keine Frames im Zeitraum — n legt einen neuen an"
		if l.filter != "" {
			empty = "kein Treffer für Filter »" + l.filter + "«"
		}
		return styleDim.Render(empty)
	}
	if l.cursor >= 0 && l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+height {
		l.offset = l.cursor - height + 1
	}
	if l.offset > len(l.rows)-height {
		l.offset = len(l.rows) - height
	}
	if l.offset < 0 {
		l.offset = 0
	}
	var b strings.Builder
	end := l.offset + height
	if end > len(l.rows) {
		end = len(l.rows)
	}
	// One duration width for the whole list, computed once: see listDurWidth.
	durW := listDurWidth(l.rows)
	for i := l.offset; i < end; i++ {
		b.WriteString(l.renderRow(i, width, durW))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// listDurWidth is the width the duration column needs: the widest duration the
// list renders. Data-driven like report.go's number column, because the field was
// a fixed %7s and "130h 00m" is eight columns — so every row with a three-digit
// hour count pushed the columns behind it one to the right, and the frame ID sits
// in the last of them, where a line one column too long loses characters to
// fitBody. A few weeks on one frame or the a (all) period is enough to hit that.
//
// Computed over every frame row of the list, not only the visible ones, so
// scrolling does not move the columns under the reader.
func listDurWidth(rows []row) int {
	w := 0
	for _, r := range rows {
		if r.isHeader {
			continue
		}
		w = max(w, len(formatDuration(r.frame.Duration())))
	}
	return w
}

// dayHeaderSep separates the day from what was booked on it: three columns, a
// space and an em dash and a space.
const dayHeaderSep = " — "

// dayHeaderLine lays a day header out against the width: the day gives way, the
// day's total does not. The line used to be one pre-built string 31 columns wide
// ("Dienstag, 21.07.2026 — 130h 30m"), so on a narrower body fitBody cut it from
// the right — and the total sits at the right, which made "… — 130h" out of a day
// of 130h 30m. The same defect the frame row had, and the same fix: the label is
// cut with an ellipsis that reads as cut, the number is not cut at all.
func dayHeaderLine(day string, total time.Duration, width int) string {
	sum := formatDuration(total)
	if room := width - len([]rune(dayHeaderSep)) - len(sum); room > 0 {
		return truncate(day, room) + dayHeaderSep + sum
	}
	// Under about a dozen columns the total no longer fits beside even one
	// character of the day. Then the whole line is cut instead, which drops the
	// total rather than showing part of it — and leaves the ellipsis saying so.
	return truncate(day+dayHeaderSep+sum, width)
}

// renderRow renders row i into a body of `width` columns, with the duration
// column sized to durW for the whole list. The layout is negotiated per row
// rather than handed down, because it is arithmetic on two numbers that are the
// same for every row — which is also what keeps the columns of the rows aligned.
func (l *listModel) renderRow(i, width, durW int) string {
	r := l.rows[i]
	if r.isHeader {
		return styleDayHeader.Render(dayHeaderLine(r.title, r.total, width))
	}
	fr := r.frame
	rl := listLayout(width, durW)

	var b strings.Builder
	if i == l.cursor {
		b.WriteString(selectionMarker + " ")
	} else {
		b.WriteString("  ")
	}
	if rl.times {
		fmt.Fprintf(&b, "%s–%s  ", fr.Start.Local().Format("15:04"), fr.Stop.Local().Format("15:04"))
	}
	// The truncate is a no-op everywhere but at the bottom of the ladder, where
	// nothing but the duration is left and it no longer fits either: there it puts
	// an ellipsis where the minutes were, so the number does not read as complete.
	fmt.Fprintf(&b, "%*s", rl.durW, truncate(formatDuration(fr.Duration()), rl.durW))
	if rl.projW > 0 {
		fmt.Fprintf(&b, "  %-*s", rl.projW, truncate(fr.Project, rl.projW))
	}
	if rl.tagW > 0 {
		fmt.Fprintf(&b, " %-*s", rl.tagW, truncate(strings.Join(fr.Tags, ", "), rl.tagW))
	}
	if rl.id {
		fmt.Fprintf(&b, " %s", fr.ShortID())
	}
	line := b.String()
	if i == l.cursor {
		return styleSelected.Render(line)
	}
	return line
}
