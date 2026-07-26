package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

type tagLine struct {
	tag string
	d   time.Duration
}

type reportLine struct {
	project string
	total   time.Duration
	tags    []tagLine
}

// aggregate sums durations per project (with tag breakdown) for frames whose
// start lies in the period. Sorted by duration desc, ties alphabetically.
func aggregate(frames []watson.Frame, p period, weekStart time.Weekday) ([]reportLine, time.Duration) {
	from, to, bounded := p.bounds(weekStart)
	totals := map[string]time.Duration{}
	tagTotals := map[string]map[string]time.Duration{}
	var grand time.Duration
	for _, fr := range frames {
		if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
			continue
		}
		d := fr.Duration()
		totals[fr.Project] += d
		grand += d
		for _, tag := range fr.Tags {
			if tagTotals[fr.Project] == nil {
				tagTotals[fr.Project] = map[string]time.Duration{}
			}
			tagTotals[fr.Project][tag] += d
		}
	}
	projects := make([]string, 0, len(totals))
	for name := range totals {
		projects = append(projects, name)
	}
	sort.Slice(projects, func(i, j int) bool {
		if totals[projects[i]] != totals[projects[j]] {
			return totals[projects[i]] > totals[projects[j]]
		}
		return projects[i] < projects[j]
	})
	lines := make([]reportLine, 0, len(projects))
	for _, name := range projects {
		rl := reportLine{project: name, total: totals[name]}
		tags := make([]string, 0, len(tagTotals[name]))
		for tg := range tagTotals[name] {
			tags = append(tags, tg)
		}
		sort.Slice(tags, func(i, j int) bool {
			ti, tj := tagTotals[name][tags[i]], tagTotals[name][tags[j]]
			if ti != tj {
				return ti > tj
			}
			return tags[i] < tags[j]
		})
		for _, tg := range tags {
			rl.tags = append(rl.tags, tagLine{tag: tg, d: tagTotals[name][tg]})
		}
		lines = append(lines, rl)
	}
	return lines, grand
}

// reportLabelMax is the widest the label column gets: a project name, or a tag
// indented and bracketed below it. The label is the only column that gives way
// on a narrow terminal — see reportLayout.
const reportLabelMax = 32

// reportDurWidth is the width the number column needs: the widest duration this
// report actually renders. Data-driven instead of a fixed eight columns
// ("999h 59m") for two reasons. A five-digit total ("123456h 00m") does not fit a
// fixed column and would have to be cut. And padding a short number out to a
// fixed width is itself a clipping risk: right-aligned, "45m" becomes "     45m",
// so on a body too narrow for that the digits fall off the right edge where the
// bare "45m" would still have fitted.
func reportDurWidth(lines []reportLine, grand time.Duration) int {
	w := len(formatDuration(grand))
	for _, l := range lines {
		w = max(w, len(formatDuration(l.total)))
		for _, tl := range l.tags {
			w = max(w, len(formatDuration(tl.d)))
		}
	}
	return w
}

// reportLayout picks the label column width for a body of the given width, with
// the number column already sized by reportDurWidth. The number does not take
// part in the negotiation: it is what an invoice is written from, so the label
// gives way and shows an ellipsis, which reads as cut — a number that lost its
// last digits does not. Unlike the overview there is no second stage and no
// floor, because the report has no value column it could drop instead: the label
// shrinks all the way down, and at one column it is a bare ellipsis.
func reportLayout(width, durW int) int {
	labelW := reportLabelMax
	for labelW > 0 && labelW+1+durW > width {
		labelW--
	}
	return labelW
}

// reportRow lays out one "label  number" line: the label padded to labelW and
// truncated with an ellipsis when it must give way, the number right-aligned in
// durW. The two come back separately so a caller can colour the label without the
// number inheriting it, and because the padding has to be computed on the plain
// text — a %-*s over a styled string counts escape bytes as columns and pads the
// line past the width.
func reportRow(label, value string, labelW, durW int) (cell, number string) {
	return fmt.Sprintf("%-*s", labelW, truncate(label, labelW)), fmt.Sprintf("%*s", durW, value)
}

type reportModel struct {
	per period
}

func newReportModel(now time.Time) reportModel {
	return reportModel{per: period{unit: unitWeek, ref: now}}
}

// runningInPeriod reports whether the frame withRunning appends is counted for
// p, i.e. whether the running timer's start falls inside the period.
func runningInPeriod(state *watson.State, p period, weekStart time.Weekday) bool {
	if state == nil {
		return false
	}
	from, to, bounded := p.bounds(weekStart)
	if !bounded {
		return true
	}
	start := state.Start.Local()
	return !start.Before(from) && start.Before(to)
}

// view renders the report into a body of width columns. A running timer counts up
// to now and is flagged below the totals, matching the overview — both views
// inform billing.
//
// Every line is laid out against the width, because the lines used to be a fixed
// 43 columns wide: on a narrower body fitBody cut the right edge, and with the
// number right-aligned the cut took the least significant digits with no ellipsis
// to show for it ("124h 30m" as "124h 30" at 46 columns, as "124" at 42). The
// width is the body's, not the terminal's — the caller subtracts the panel.
func (m reportModel) view(frames []watson.Frame, state *watson.State, weekStart time.Weekday, now time.Time, width int) string {
	lines, grand := aggregate(withRunning(frames, state, now), m.per, weekStart)
	durW := reportDurWidth(lines, grand)
	labelW := reportLayout(width, durW)

	var b strings.Builder
	if len(lines) == 0 {
		b.WriteString(styleDim.Render(truncate("keine Frames im Zeitraum", width)) + "\n")
	}
	for _, l := range lines {
		cell, number := reportRow(l.project, formatDuration(l.total), labelW, durW)
		b.WriteString(cell + " " + number + "\n")
		for _, tl := range l.tags {
			// The tag lives in the label column, indented and bracketed, and is cut
			// there the same way a project name is. Only the label is dimmed: the
			// number beside it is read off the screen like any other.
			cell, number := reportRow("  ["+tl.tag+"]", formatDuration(tl.d), labelW, durW)
			b.WriteString(styleDim.Render(cell) + " " + number + "\n")
		}
	}
	cell, number := reportRow("Gesamt", formatDuration(grand), labelW, durW)
	b.WriteString("\n" + styleTitle.Render(cell+" "+number))
	if runningInPeriod(state, m.per, weekStart) {
		// Truncated twice: the project name to the label column, then the whole
		// sentence to the width. The second cut can reach the clock on a very narrow
		// terminal, but it leaves an ellipsis where fitBody left nothing — and the
		// running timer's contribution also stands in its project's row above.
		note := fmt.Sprintf("▶ %s läuft (%s) und ist eingerechnet",
			truncate(state.Project, labelW), formatClock(now.Sub(state.Start)))
		b.WriteString("\n\n" + styleRunning.Render(truncate(note, width)))
	}
	return b.String()
}
