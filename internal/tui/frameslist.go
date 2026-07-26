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
	unitDay
	unitAll
)

// period is a display time range: a unit plus a reference time inside it.
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
	default:
		return fmt.Sprintf("%s %d", germanMonths[int(from.Month())], from.Year())
	}
}

// row is one display line: a day header or a frame.
type row struct {
	isHeader bool
	title    string
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

// buildRows filters frames to period+filter, sorts by start, groups by local day.
func buildRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string) []row {
	from, to, bounded := p.bounds(weekStart)
	needle := strings.ToLower(strings.TrimSpace(filter))
	var sel []watson.Frame
	for _, fr := range frames {
		if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
			continue
		}
		if needle != "" && !matchesFilter(fr, needle) {
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
		rows = append(rows, row{isHeader: true, title: fmt.Sprintf("%s — %s", formatDay(g.day), formatDuration(g.total))})
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

func newListModel(now time.Time) listModel {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "Projekt, Tag oder ID"
	return listModel{per: period{unit: unitWeek, ref: now}, cursor: -1, filterInput: ti}
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

// view renders `height` rows, scrolling so the cursor stays visible.
func (l *listModel) view(height int) string {
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
		b.WriteString(l.renderRow(i, durW))
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

func (l *listModel) renderRow(i, durW int) string {
	r := l.rows[i]
	if r.isHeader {
		return styleDayHeader.Render(r.title)
	}
	fr := r.frame
	prefix := "  "
	if i == l.cursor {
		prefix = selectionMarker + " "
	}
	line := fmt.Sprintf("%s%s–%s  %*s  %-24s %-28s %s",
		prefix,
		fr.Start.Local().Format("15:04"),
		fr.Stop.Local().Format("15:04"),
		durW, formatDuration(fr.Duration()),
		truncate(fr.Project, 24),
		truncate(strings.Join(fr.Tags, ", "), 28),
		fr.ShortID())
	if i == l.cursor {
		return styleSelected.Render(line)
	}
	return line
}
