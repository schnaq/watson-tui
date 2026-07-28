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
	summaryTagIndent     = 4
)

// The columns the summary spends on things that are not the label or the number:
// one for the rail glyph plus one of air, and two between the number and the bar.
const (
	summaryGutter = 2
	summaryBarGap = 2
)

// What the bar may and must have.
//
// The cap is there because 150 columns of bar on a wide terminal is decoration,
// not a reading aid. The floor is the more interesting number: under it a bar can
// no longer tell a tenth from a half, and it then drops whole rather than shrink
// — a bar showing the wrong proportion is worse than no bar, the same reason the
// frame list shows an ID in full or not at all. A terminal below the floor
// therefore renders exactly what it did before the bar existed, so no new rung is
// added to the width ladder.
const (
	summaryBarMin = 10
	summaryBarMax = 24
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

// summaryLabel is what row i writes into the label column, indent excluded: a day
// or project name as it stands, a tag name hanging from the branch that says
// whether it is the last of its project.
//
// No em dash behind the names any more. It used to set the number off; the fixed
// number column does that, and the rail does it one level up, so the dash only
// repeated them — for two columns the bar had a use for.
//
// Which branch a tag hangs from is a statement about its neighbour, so it is read
// off the rows here rather than stored on them, for the reason summaryRail gives.
func summaryLabel(rows []row, i int) string {
	r := rows[i]
	if r.kind != rowTag {
		return r.title
	}
	if i+1 < len(rows) && rows[i+1].kind == rowTag {
		return tagBranch + r.title
	}
	return tagLast + r.title
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

// summaryRail is the glyph row i puts in the rail column: the corner that opens
// a day block, the line that runs down its side, the corner that closes it.
//
// Derived from the rows rather than stored on them, because it is a statement
// about a row's neighbours and not about the row — a field would have to be kept
// in step with every change to how the blocks are built, and this cannot fall out
// of step with anything.
//
// The closing corner follows the last row of the block whatever kind that row is,
// so a last project carrying no tags closes the bracket on itself. rows[i+1]
// being a day header cannot happen while a day block is always followed by a
// blank line, and is spelled out anyway: it is the case that would otherwise
// leave a block hanging open, and it costs one term to rule out.
//
// A day with nothing under it has nothing to bracket, so it gets no rail at all —
// that is what keeps a run of empty days from drawing a rail down a column of
// dashes. "Nothing under it" is read off the rows, not off the day's total: a
// frame that starts and stops at the same moment books nothing, so its day reads
// "–" while still carrying a project row, and a corner keyed on the total would
// go missing above a block that draws one.
func summaryRail(rows []row, i int) string {
	r := rows[i]
	switch {
	case r.kind == rowBlank:
		return ""
	case r.kind == rowDayHeader:
		if i+1 < len(rows) && rows[i+1].kind == rowProject {
			return railOpen
		}
		return ""
	case i+1 >= len(rows), rows[i+1].kind == rowBlank, rows[i+1].kind == rowDayHeader:
		return railClose
	}
	return railMid
}

// summaryDayStyle is the weight a day header carries. Three of them, because the
// three questions one asks of a day list are different: which day is today, which
// days were worked, and where the gaps are. Bold marks today even when it has
// booked nothing yet — a day one is standing in is worth seeing before it fills up.
func summaryDayStyle(r row) lipgloss.Style {
	switch {
	case r.today:
		return styleDayHeader
	case r.total == 0:
		return styleDim
	}
	return styleDayPlain
}

// summaryRailStyle colours the rail: the accent for the block of the current day,
// dim for the others. Together with the bold day header that is the second of
// today's two signals, and it needs no glyph of its own — one in the gutter would
// have had to displace the corner that opens the block.
func summaryRailStyle(r row, cursor bool) lipgloss.Style {
	switch {
	case cursor:
		return styleSelected
	case r.today:
		return styleAccent
	}
	return styleDim
}

// summaryPeak is the scale every bar of the view is a share of: the largest day
// total in it. One scale for the whole view rather than one per day, so that a
// quiet day looks quiet — and because the projects of a day then add up to exactly
// their day's bar, which makes it read as their sum instead of a second scale.
//
// Day rows only. A project or tag total can never exceed the day it sits in, so
// measuring them would change nothing today and would let a later kind of row move
// the scale. Computed over every row of the view, not only the visible ones, so
// scrolling does not rescale the bars under the reader — the same reason
// summaryColumns and listDurWidth give.
//
// Zero when the period holds no frames, or a filter matched none. That is the
// guard on the division in summaryBar as much as it is an answer.
func summaryPeak(rows []row) time.Duration {
	var peak time.Duration
	for _, r := range rows {
		if r.kind == rowDayHeader {
			peak = max(peak, r.total)
		}
	}
	return peak
}

// summaryBarWidth is how many columns the bar gets: what the label and number
// columns left, capped, or nothing at all below the floor.
//
// The order matters and is deliberate. The label column is sized from the data
// first and the bar takes the remainder, so on a wide terminal with very long
// project names the bar can be gone while space looks free. The other way round —
// reserving the bar and truncating names into it — would put the decoration ahead
// of the information, and sizing the label column from its own content is exactly
// what the commit before this one was for.
func summaryBarWidth(width, labelW, durW int) int {
	left := width - summaryGutter - labelW - 1 - durW - summaryBarGap
	if left < summaryBarMin {
		return 0
	}
	return min(left, summaryBarMax)
}

// summaryBar renders d as a share of peak across w columns, the filled part and
// the track behind it apart.
//
// Apart, because the two are coloured differently and a line assembled from
// styled pieces cannot be measured: %-*s and truncate count the escape bytes as
// columns, so the line comes out too long, panel cuts it by display width, the
// reset sequence goes with the cut and the colour bleeds across the rest of the
// screen. Everything here is plain text, and every width is arithmetic on it.
//
// Any duration above zero occupies at least the thinnest eighth. A row that
// rendered as nothing would say nothing was booked, which is a different claim
// from "not much was" — and in the "all" period, where one strong day sets the
// scale, that floor is what keeps the quiet days on the screen.
func summaryBar(d, peak time.Duration, w int) (filled, track string) {
	if peak <= 0 || d <= 0 || w <= 0 {
		return "", ""
	}
	eighths := int(float64(d)/float64(peak)*float64(w*8) + 0.5)
	eighths = min(max(eighths, 1), w*8)
	filled = strings.Repeat(barFull, eighths/8) + barEighths[eighths%8]
	return filled, strings.Repeat(barTrack, w-lipgloss.Width(filled))
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
// Only the label gives way to the width, and it gives way against the body less
// the gutter, which is a fixed cost the rail column takes off the top. Under about
// ten columns of body not even the number fits; it is then clipped by the panel,
// the way report.go's last column is, and no ladder is spent on a terminal that
// narrow.
//
// The bar is sized last, from what the two columns left. See summaryBarWidth for
// why that order and not the other one.
func summaryColumns(rows []row, width int) viewLayout {
	var c viewLayout
	for i, r := range rows {
		if r.kind == rowBlank {
			continue
		}
		c.durW = max(c.durW, lipgloss.Width(summaryValue(r)))
		c.labelW = max(c.labelW, summaryIndent(r.kind)+lipgloss.Width(summaryLabel(rows, i)))
	}
	c.labelW = fitLabelWidth(c.labelW, width-summaryGutter, c.durW)
	c.barW = summaryBarWidth(width, c.labelW, c.durW)
	c.peak = summaryPeak(rows)
	return c
}

// summaryParts is one summary row as plain text, cut where its colour changes.
//
// Cut rather than returned as one string, because the pieces are coloured
// differently and a line assembled from styled pieces can no longer be measured:
// reportRow's %-*s and truncate count escape bytes as columns. The width of every
// piece here is arithmetic on plain text; renderRow puts the colour on afterwards.
type summaryParts struct {
	rail   string // the gutter: the rail glyph or the cursor marker, plus its air
	body   string // indent, label, number — and the gap in front of the bar
	filled string // the bar's filled head
	track  string // the bar's remainder
}

// render puts the three colours on: the rail, the text of the row, and the bar's
// filled head. The track is always dim — it is the absence the bar is measured
// against, and on the cursor row too it must not compete with the head.
func (p summaryParts) render(rail, text, bar lipgloss.Style) string {
	return rail.Render(p.rail) + text.Render(p.body) + bar.Render(p.filled) + styleDim.Render(p.track)
}

// summaryRow lays row i out: the rail column, the indent, the label and number
// columns reportRow builds, and the bar. The label is truncated with an ellipsis
// when it must give way and the number never is — the rule the frame list, the
// report and the overview all follow.
//
// The marker replaces the rail glyph rather than being prepended, so the line keeps
// its width and the numbers stay in their column on the cursor row too. It wins
// over a closing corner: where the cursor stands is the more urgent of the two, and
// the corner comes back with the next keystroke.
//
// The indent gives way before the name: below it the label column would otherwise
// be pushed past its width by rows that are only indented, and the numbers of the
// deepest rows would drift out of the column the others use.
//
// Tag rows carry no bar. Tags are counted individually, so a frame with two of them
// counts under both, and their bars would together outrun the project's — claiming
// a division of it that is not there.
func summaryRow(rows []row, i int, c viewLayout, cursor bool) summaryParts {
	r := rows[i]
	glyph := summaryRail(rows, i)
	if cursor {
		glyph = selectionMarker
	}
	if glyph == "" {
		glyph = " "
	}
	p := summaryParts{rail: glyph + strings.Repeat(" ", summaryGutter-1)}

	indent := min(summaryIndent(r.kind), c.labelW)
	cell, number := reportRow(summaryLabel(rows, i), summaryValue(r), c.labelW-indent, c.durW)
	p.body = strings.Repeat(" ", indent) + cell + " " + number

	if r.kind != rowTag {
		p.filled, p.track = summaryBar(r.total, c.peak, c.barW)
		if p.filled != "" {
			p.body += strings.Repeat(" ", summaryBarGap)
		}
	}
	return p
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

	today := localDay(now)
	var rows []row
	prevEmpty := false
	for _, day := range days {
		dayPeriod := period{unit: unitDay, ref: day}
		lines, total := aggregate(byDay[localDay(day)], dayPeriod, weekStart)
		empty := len(lines) == 0
		isToday := localDay(day) == today
		// A blank line stands between two day blocks — but not between two empty
		// ones: a quiet second half of the week would otherwise spend ten lines on
		// five dashes. Nothing separates the projects within a day any more; the
		// rail does that, and while a project boundary carried the same blank line
		// as a day boundary the view had no blocks, only a list.
		runTogether := empty && prevEmpty
		if len(rows) > 0 && !runTogether {
			rows = append(rows, row{kind: rowBlank})
		}
		prevEmpty = empty
		// The day's total is aggregate's grand total, not the sum of the tag rows
		// below it: a frame carrying two tags is counted under both, so adding the
		// tag rows up would book it twice.
		rows = append(rows, row{kind: rowDayHeader, title: formatDay(day), total: total, today: isToday})
		for _, l := range lines {
			rows = append(rows, row{kind: rowProject, title: l.project, total: l.total, today: isToday})
			for _, tl := range l.tags {
				rows = append(rows, row{kind: rowTag, title: tl.tag, total: tl.d, today: isToday})
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
