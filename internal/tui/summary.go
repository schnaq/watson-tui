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

// summaryLine lays out "<indent><label>   <value>": the value right-aligned at
// the width and always whole, the label truncated with an ellipsis when the two
// do not both fit. The same rule the frame list, the report and the overview
// follow — a name that ends in "…" reads as cut, a number that lost its last
// digits does not, it reads as a smaller number.
//
// Every kind of summary row goes through it, which is what puts the day total,
// the project totals and the tag totals in one right-hand column.
func summaryLine(label, value string, indent, width int) string {
	avail := width - indent - lipgloss.Width(value) - 1
	if avail < 1 {
		avail = 1
	}
	label = truncate(label, avail)
	pad := width - indent - lipgloss.Width(label) - lipgloss.Width(value)
	if pad < 1 {
		pad = 1
	}
	return strings.Repeat(" ", indent) + label + strings.Repeat(" ", pad) + value
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
