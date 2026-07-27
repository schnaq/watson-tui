package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// overviewColumn is one time-range column of the billing overview. short is
// the header used when the column is too narrow for title.
type overviewColumn struct {
	title string
	short string
	per   period
}

// overviewColumns defines the fixed billing columns:
// diese Woche, letzte Woche, dieser Monat, letzter Monat, gesamt.
func overviewColumns(now time.Time, weekStart time.Weekday) []overviewColumn {
	week := period{unit: unitWeek, ref: now}
	month := period{unit: unitMonth, ref: now}
	return []overviewColumn{
		{"diese Woche", "Woche", week},
		{"letzte Woche", "Vorwoche", week.shift(weekStart, -1)},
		{"dieser Monat", "Monat", month},
		{"letzter Monat", "Vormonat", month.shift(weekStart, -1)},
		{"gesamt", "gesamt", period{unit: unitAll, ref: now}},
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

// withRunning appends the running timer as a frame ending at now, so billing
// sums include time that has not been stopped yet.
func withRunning(frames []watson.Frame, state *watson.State, now time.Time) []watson.Frame {
	if state == nil {
		return frames
	}
	running := watson.Frame{Project: state.Project, Start: state.Start.Local(), Stop: now, Tags: state.Tags}
	return append(append(make([]watson.Frame, 0, len(frames)+1), frames...), running)
}

// cellDuration renders one overview cell; zero shows as a dash.
func cellDuration(d time.Duration) string {
	if d == 0 {
		return "–"
	}
	return formatDuration(d)
}

// Column widths of the overview table. The value columns shrink before the
// project column does, because project names carry more information than the
// two extra digits a wide cell allows for.
const (
	overviewProjMax = 24
	overviewProjMin = 12
	overviewCellMax = 13
	overviewCellMin = 9 // fits "999h 59m"
)

// overviewLayout picks the project and value column widths for a terminal of
// the given width and n value columns.
func overviewLayout(width, n int) (projW, cellW int) {
	projW, cellW = overviewProjMax, overviewCellMax
	fits := func() bool { return projW+n*(cellW+1) <= width }
	for !fits() && cellW > overviewCellMin {
		cellW--
	}
	for !fits() && projW > overviewProjMin {
		projW--
	}
	// Last resort, below the project minimum: the caller has already dropped
	// every value column it may drop, so the only thing left to give is name
	// width. A cut project name keeps its ellipsis and reads as cut; a cut value
	// column would not — see overviewFit.
	for !fits() && projW > 1 {
		projW--
	}
	return projW, cellW
}

// overviewDropOrder lists the value columns in the order they are given up on a
// narrow terminal: least useful for writing an invoice first. gesamt is not in
// the list, so it is the column that always survives.
var overviewDropOrder = []int{1, 3, 2, 0} // letzte Woche, letzter Monat, dieser Monat, diese Woche

// overviewFit decides which value columns the table shows and how wide its
// columns are. A column that no longer fits its minimum is dropped whole
// instead of being narrowed: the cells are right-aligned, so a too-narrow table
// gets clipped on the right and the clip eats the least significant digits —
// "4h 02m" renders as "4h 0", a plausible wrong number in the one view an
// invoice is written from. Dropped columns are named below the table
// (droppedNote), so a missing column is never silent.
//
// keep and dropped are column indices into the caller's column slice, both in
// display order; the sums themselves are always computed for every column, so
// what is dropped is only the rendering.
func overviewFit(width, n int) (keep, dropped []int, projW, cellW int) {
	keep = make([]int, 0, n)
	for i := 0; i < n; i++ {
		keep = append(keep, i)
	}
	// The minimums decide: as long as the narrowest legible table fits, nothing
	// is dropped and overviewLayout hands out whatever extra room there is.
	fits := func() bool { return overviewProjMin+len(keep)*(overviewCellMin+1) <= width }
	for _, drop := range overviewDropOrder {
		if fits() || len(keep) <= 1 {
			break
		}
		for i, idx := range keep {
			if idx == drop {
				keep = append(keep[:i], keep[i+1:]...)
				dropped = append(dropped, idx)
				break
			}
		}
	}
	sort.Ints(dropped)
	projW, cellW = overviewLayout(width, len(keep))
	return keep, dropped, projW, cellW
}

// droppedNote names the columns overviewFit gave up. In column order, the order
// they would have been read in — the drop order is an internal priority and
// would put the names in a sequence the table never had.
func droppedNote(cols []overviewColumn, dropped []int) string {
	names := make([]string, 0, len(dropped))
	for _, i := range dropped {
		names = append(names, cols[i].title)
	}
	return "zu schmal für: " + strings.Join(names, ", ")
}

// columnHeader picks the longest header variant that fits cellW.
func columnHeader(c overviewColumn, cellW int) string {
	title := c.title
	if len([]rune(title)) > cellW {
		title = c.short
	}
	return truncate(title, cellW)
}

// runningNote describes the running timer below the table: how long it runs
// and which columns count it. Naming the columns matters for billing — a timer
// started before this week's boundary counts into letzte Woche, while the
// diese Woche column an invoice is written from does not see it at all.
func runningNote(state *watson.State, cols []overviewColumn, weekStart time.Weekday, now time.Time, projW int) string {
	note := fmt.Sprintf("▶ %s läuft (%s)", truncate(state.Project, projW), formatClock(now.Sub(state.Start)))
	var in []string
	for _, c := range cols {
		if runningInPeriod(state, c.per, weekStart) {
			in = append(in, c.title)
		}
	}
	if len(in) == 0 {
		return note
	}
	// Second line: the column list is the billing-relevant part and must not
	// compete with the project name for the terminal width.
	return note + "\n  eingerechnet in: " + strings.Join(in, ", ")
}

// overviewView renders the billing overview (stateless). A running timer is
// counted up to now and flagged below the table.
func overviewView(frames []watson.Frame, state *watson.State, weekStart time.Weekday, now time.Time, width int) string {
	cols := overviewColumns(now, weekStart)
	rows, totals := buildOverview(withRunning(frames, state, now), cols, weekStart)
	keep, dropped, projW, cellW := overviewFit(width, len(cols))

	var b strings.Builder
	if len(dropped) > 0 {
		// Above the table, not below it: fitBody keeps the head of the body, so a
		// note under the totals falls off the screen as soon as there are more
		// projects than rows — and a column missing without a word is exactly what
		// this note exists to prevent. Cut rows are obvious to whoever booked
		// them; a cut note is not. Wrapped, because on the narrow terminal that
		// dropped a column the list of names is longer than the line.
		b.WriteString(styleDim.Width(width).Render(droppedNote(cols, dropped)) + "\n\n")
	}
	fmt.Fprintf(&b, "%-*s", projW, truncate("Projekt", projW))
	for _, i := range keep {
		fmt.Fprintf(&b, " %*s", cellW, columnHeader(cols[i], cellW))
	}
	b.WriteString("\n\n")
	if len(rows) == 0 {
		b.WriteString(styleDim.Render("keine Frames vorhanden") + "\n")
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "%-*s", projW, truncate(r.project, projW))
		for _, i := range keep {
			fmt.Fprintf(&b, " %*s", cellW, truncate(cellDuration(r.cells[i]), cellW))
		}
		b.WriteByte('\n')
	}
	// Truncated like the project rows above: without it the label keeps all six
	// of its columns while the rows give theirs up, the line runs past the table
	// from width 15 down, and fitBody cuts the least significant digits off the
	// grand total — the one number on this screen that goes on an invoice, and the
	// only line that was still illegible. A cut label reads as cut; a cut number
	// does not.
	totalLine := fmt.Sprintf("%-*s", projW, truncate("Gesamt", projW))
	for _, i := range keep {
		totalLine += fmt.Sprintf(" %*s", cellW, truncate(cellDuration(totals[i]), cellW))
	}
	b.WriteString("\n" + styleTitle.Render(totalLine))
	if state != nil {
		b.WriteString("\n\n" + styleRunning.Render(runningNote(state, cols, weekStart, now, projW)))
	}
	return b.String()
}
