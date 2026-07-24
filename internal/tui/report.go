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

type reportModel struct {
	per period
}

func newReportModel(now time.Time) reportModel {
	return reportModel{per: period{unit: unitWeek, ref: now}}
}

func (m reportModel) view(frames []watson.Frame, weekStart time.Weekday) string {
	lines, grand := aggregate(frames, m.per, weekStart)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Report — "+m.per.label(weekStart)) + "\n\n")
	if len(lines) == 0 {
		b.WriteString(styleDim.Render("keine Frames im Zeitraum") + "\n")
	}
	for _, l := range lines {
		b.WriteString(fmt.Sprintf("%-32s %10s\n", truncate(l.project, 32), formatDuration(l.total)))
		for _, tl := range l.tags {
			b.WriteString(styleDim.Render(fmt.Sprintf("  [%s]", tl.tag)) +
				fmt.Sprintf("%*s\n", 42-len("  []")-len([]rune(tl.tag)), formatDuration(tl.d)))
		}
	}
	b.WriteString("\n" + styleTitle.Render(fmt.Sprintf("%-32s %10s", "Gesamt", formatDuration(grand))))
	b.WriteString("\n\n" + styleDim.Render("t/w/m: Zeitraum · [ / ]: verschieben · esc: zurück"))
	return b.String()
}
