// The compact view: one block per day of the period, each listing its projects
// and their tags. It answers "what did I work on" — the frame list answers
// "which sessions were there", which is the question one asks when editing.

package tui

import (
	"slices"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// buildSummaryRows renders the period as day blocks. The numbers come from
// aggregate, the same function the report uses, so the two cannot drift apart:
// a frame counts towards the day it starts in, tags are counted individually,
// and a running timer counts up to now.
//
// The running timer is folded in before the filter, not after, so a filter for
// one project does not let another project's running timer through.
func buildSummaryRows(frames []watson.Frame, p period, weekStart time.Weekday,
	filter string, state *watson.State, now time.Time) []row {

	frames = filterFrames(withRunning(frames, state, now), filter)
	days := summaryDays(frames, p, weekStart)

	var rows []row
	for _, day := range days {
		if len(rows) > 0 {
			rows = append(rows, row{kind: rowBlank})
		}
		dayPeriod := period{unit: unitDay, ref: day}
		lines, total := aggregate(frames, dayPeriod, weekStart)
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

// summaryDays lists the days the view shows: every day of a bounded period,
// including the empty ones, so a gap is visible. unitAll has no bounds, so it
// lists only the days that carry frames — otherwise every day since the first
// frame would get a row.
func summaryDays(frames []watson.Frame, p period, weekStart time.Weekday) []time.Time {
	from, to, bounded := p.bounds(weekStart)
	if bounded {
		var days []time.Time
		for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
			days = append(days, d)
		}
		return days
	}
	seen := map[time.Time]bool{}
	var days []time.Time
	for _, f := range frames {
		y, m, d := f.Start.Local().Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		if !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })
	return days
}
