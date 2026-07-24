package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// overviewColumn is one time-range column of the billing overview.
type overviewColumn struct {
	title string
	per   period
}

// overviewColumns defines the fixed billing columns:
// diese Woche, letzte Woche, dieser Monat, letzter Monat, gesamt.
func overviewColumns(now time.Time, weekStart time.Weekday) []overviewColumn {
	week := period{unit: unitWeek, ref: now}
	month := period{unit: unitMonth, ref: now}
	return []overviewColumn{
		{"diese Woche", week},
		{"letzte Woche", week.shift(weekStart, -1)},
		{"dieser Monat", month},
		{"letzter Monat", month.shift(weekStart, -1)},
		{"gesamt", period{unit: unitAll, ref: now}},
	}
}

// overviewRow is one project's sums, one cell per column.
type overviewRow struct {
	project string
	cells   []time.Duration
}

// buildOverview sums frame durations per project per column. Rows sort by
// the last column (gesamt) descending, ties alphabetically. The second
// return value holds the per-column totals.
func buildOverview(frames []watson.Frame, cols []overviewColumn, weekStart time.Weekday) ([]overviewRow, []time.Duration) {
	sums := map[string][]time.Duration{}
	totals := make([]time.Duration, len(cols))
	for _, fr := range frames {
		row, ok := sums[fr.Project]
		if !ok {
			row = make([]time.Duration, len(cols))
			sums[fr.Project] = row
		}
		d := fr.Duration()
		for i, col := range cols {
			from, to, bounded := col.per.bounds(weekStart)
			if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
				continue
			}
			row[i] += d
			totals[i] += d
		}
	}
	rows := make([]overviewRow, 0, len(sums))
	for project, cells := range sums {
		rows = append(rows, overviewRow{project: project, cells: cells})
	}
	last := len(cols) - 1
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].cells[last] != rows[j].cells[last] {
			return rows[i].cells[last] > rows[j].cells[last]
		}
		return rows[i].project < rows[j].project
	})
	return rows, totals
}

// cellDuration renders one overview cell; zero shows as a dash.
func cellDuration(d time.Duration) string {
	if d == 0 {
		return "–"
	}
	return formatDuration(d)
}

// overviewView renders the billing overview (stateless).
func overviewView(frames []watson.Frame, weekStart time.Weekday, now time.Time) string {
	cols := overviewColumns(now, weekStart)
	rows, totals := buildOverview(frames, cols, weekStart)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Übersicht — Summen pro Projekt") + "\n\n")
	fmt.Fprintf(&b, "%-24s", "Projekt")
	for _, c := range cols {
		fmt.Fprintf(&b, " %13s", c.title)
	}
	b.WriteString("\n\n")
	if len(rows) == 0 {
		b.WriteString(styleDim.Render("keine Frames vorhanden") + "\n")
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "%-24s", truncate(r.project, 24))
		for _, d := range r.cells {
			fmt.Fprintf(&b, " %13s", cellDuration(d))
		}
		b.WriteByte('\n')
	}
	totalLine := fmt.Sprintf("%-24s", "Gesamt")
	for _, d := range totals {
		totalLine += fmt.Sprintf(" %13s", cellDuration(d))
	}
	b.WriteString("\n" + styleTitle.Render(totalLine))
	b.WriteString("\n\n" + styleDim.Render("esc: zurück"))
	return b.String()
}
