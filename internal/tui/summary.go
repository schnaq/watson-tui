// The compact view: one block per day of the period, each listing its projects
// and their tags. It answers "what did I work on" — the frame list answers
// "which sessions were there", which is the question one asks when editing.

package tui

import (
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// How far the summary indents its two nested levels. The day header sits at the
// left edge, its projects two columns in, their tags four columns further —
// enough that the nesting reads without a rule, and short enough to leave a
// narrow terminal something to print the name in.
const (
	summaryProjectIndent = 2
	summaryTagIndent     = 6
)

// summaryIndent is how far a row of this kind is indented. The day header sits
// at the left edge, its projects and their tags at the two levels above.
func summaryIndent(k rowKind) int {
	switch k {
	case rowProject:
		return summaryProjectIndent
	case rowTag:
		return summaryTagIndent
	}
	return 0
}

// summaryLabel is what a row writes into the label column, indent excluded: a
// day or project name followed by the em dash that sets its number off, a tag
// name preceded by the bracket that opens it.
func summaryLabel(r row) string {
	if r.kind == rowTag {
		return "[" + r.title
	}
	return r.title + " —"
}

// summaryValue is the number a row shows. An empty day reads as a dash: "0m"
// invites the question whether nothing was booked or nothing is known — the same
// answer the overview's cells and the header's comparison row give.
func summaryValue(r row) string {
	if r.kind == rowDayHeader && r.total == 0 {
		return "–"
	}
	return formatDuration(r.total)
}

// summaryTail is what follows a row's number: the bracket that closes a tag,
// nothing on the other kinds. It is deliberately outside the number column, so
// that every kind of row puts its digits in the same one.
func summaryTail(k rowKind) string {
	if k == rowTag {
		return "]"
	}
	return ""
}

// summaryColumns sizes the summary's two columns from the rows themselves: the
// number column to the widest duration the view actually renders, the label
// column to the longest name it actually renders, indent included.
//
// Sized from the data rather than from the width, because right-aligning the
// numbers against the body put them sixty columns from the names they belong to
// on a wide terminal. `watson aggregate` writes the number after the name, and
// so does this — but in one column across all three kinds of row, day headers
// included, because a number that stands in a different place on every line is
// harder to read down than one that stands far from its name. The day header is
// normally the longest label of the three, so it is normally what the column is
// sized to.
//
// Computed over every row of the view, not only the visible ones, so scrolling
// does not move the columns under the reader — the same reason listDurWidth
// gives.
//
// Only the label gives way to the width. Under about ten columns of body not
// even the number and a tag's closing bracket fit; the bracket is then clipped
// by the panel, the way report.go's last column is, and no ladder is spent on a
// terminal that narrow.
func summaryColumns(rows []row, width int) colWidths {
	var c colWidths
	tail := 0
	for _, r := range rows {
		if r.kind == rowBlank {
			continue
		}
		c.durW = max(c.durW, lipgloss.Width(summaryValue(r)))
		c.labelW = max(c.labelW, summaryIndent(r.kind)+lipgloss.Width(summaryLabel(r)))
		tail = max(tail, lipgloss.Width(summaryTail(r.kind)))
	}
	c.labelW = fitLabelWidth(c.labelW, width, c.durW+tail)
	return c
}

// summaryRow lays one row out: the indent, then the label and number columns
// reportRow builds, then whatever closes the line. The label is truncated with
// an ellipsis when it must give way and the number never is — the rule the frame
// list, the report and the overview all follow.
//
// The indent is what gives way first, before the name: below it the label column
// would otherwise be pushed past its width by rows that are only indented, and
// the numbers of the deepest rows would drift out of the column the others use.
func summaryRow(r row, c colWidths, cursor bool) string {
	indent := min(summaryIndent(r.kind), c.labelW)
	cell, number := reportRow(summaryLabel(r), summaryValue(r), c.labelW-indent, c.durW)
	lead := strings.Repeat(" ", indent)
	if cursor && indent > 0 {
		// The marker replaces the first column of the indent instead of being
		// prepended, so the line keeps its width and the numbers stay in their
		// column on the cursor row too.
		lead = selectionMarker + strings.Repeat(" ", indent-1)
	}
	return lead + cell + " " + number + summaryTail(r.kind)
}

// buildSummaryRows renders the period as day blocks. The numbers come from
// aggregate, the same function the report uses, so the two cannot drift apart:
// a frame counts towards the day it starts in, tags are counted individually,
// and a running timer counts up to now.
//
// The running timer is folded in before the filter, not after, so a filter for
// one project does not let another project's running timer through.
//
// The frames are bucketed by day once and each aggregate call is handed only its
// own day's, because aggregate walks everything it is given: one call per day
// over the whole slice is O(days × frames), and the a (all) period makes both
// factors grow together. Measured over three years and 5000 frames that came to
// 29 ms per refresh, at ten years 84 ms and at 15000 frames a quarter of a
// second — and refresh runs on the keystroke that changes the period, where a
// visible pause reads as a hung key. Handing aggregate a subset stays correct
// because it filters by period itself; the bucket is a superset of nothing and a
// subset of exactly the day it is passed for.
func buildSummaryRows(frames []watson.Frame, p period, weekStart time.Weekday,
	filter string, state *watson.State, now time.Time) []row {

	frames = filterFrames(withRunning(frames, state, now), filter)
	byDay := map[dayKey][]watson.Frame{}
	for _, f := range frames {
		k := localDay(f.Start)
		byDay[k] = append(byDay[k], f)
	}
	days := summaryDays(frames, p, weekStart)

	var rows []row
	for _, day := range days {
		if len(rows) > 0 {
			rows = append(rows, row{kind: rowBlank})
		}
		dayPeriod := period{unit: unitDay, ref: day}
		lines, total := aggregate(byDay[localDay(day)], dayPeriod, weekStart)
		// The day's total is aggregate's grand total, not the sum of the tag rows
		// below it: a frame carrying two tags is counted under both, so adding the
		// tag rows up would book it twice.
		rows = append(rows, row{kind: rowDayHeader, title: formatDay(day), total: total})
		for i, l := range lines {
			if i > 0 {
				rows = append(rows, row{kind: rowBlank})
			}
			rows = append(rows, row{kind: rowProject, title: l.project, total: l.total})
			for _, tl := range l.tags {
				rows = append(rows, row{kind: rowTag, title: tl.tag, total: tl.d})
			}
		}
	}
	return rows
}

// dayKey names one local calendar day. A time.Time cannot be that name: the
// day list and the frames are keyed from different constructions — a period's
// bounds carry the reference time's location, a frame's start is stored in UTC
// — and two Times that mean the same day but hold different locations are not
// equal, so a map keyed by Time would drop a day's frames on the floor instead
// of failing where one could see it.
type dayKey struct {
	year, day int
	month     time.Month
}

// localDay is the one place a moment becomes a day. Both sides of the bucketing
// go through it, so they cannot disagree about which day something falls on —
// and it is local time, because that is what every view in this package formats
// and groups by (buildRows does the same).
func localDay(t time.Time) dayKey {
	y, m, d := t.Local().Date()
	return dayKey{year: y, month: m, day: d}
}

// summaryDays lists the days the view shows: every day of a bounded period,
// including the empty ones, so a gap is visible. unitAll has no bounds, so it
// lists only the days that carry frames — otherwise every day since the first
// frame would get a row.
//
// The days come back in time.Local whatever location the period was built in,
// so that localDay agrees with the frames it buckets. Only the location is
// normalised, never the date: the loop still walks the period's own calendar.
func summaryDays(frames []watson.Frame, p period, weekStart time.Weekday) []time.Time {
	from, to, bounded := p.bounds(weekStart)
	if bounded {
		var days []time.Time
		for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
			y, m, dd := d.Date()
			days = append(days, time.Date(y, m, dd, 0, 0, 0, 0, time.Local))
		}
		return days
	}
	seen := map[dayKey]bool{}
	var days []time.Time
	for _, f := range frames {
		k := localDay(f.Start)
		if !seen[k] {
			seen[k] = true
			days = append(days, time.Date(k.year, k.month, k.day, 0, 0, 0, 0, time.Local))
		}
	}
	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })
	return days
}
