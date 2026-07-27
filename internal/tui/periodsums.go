// Sums over periods, and the neighbouring periods the header compares against.

package tui

import (
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// sumInPeriod adds the duration of every frame that begins inside p. A frame
// belongs entirely to the period it starts in — the same rule buildRows,
// aggregate and buildOverview follow, so all four agree.
func sumInPeriod(frames []watson.Frame, p period, weekStart time.Weekday) time.Duration {
	from, to, bounded := p.bounds(weekStart)
	var total time.Duration
	for _, f := range frames {
		if bounded && (f.Start.Before(from) || !f.Start.Before(to)) {
			continue
		}
		total += f.Duration()
	}
	return total
}

// comparison is a neighbouring period the header shows alongside the current
// one, so a week reads against the week before it and its month.
type comparison struct {
	label string
	per   period
}

// comparisonPeriods returns the two neighbours the header shows for p: the
// period before it, and the larger period containing it. unitYear exists only
// for this comparison — no key selects it.
func comparisonPeriods(p period) []comparison {
	switch p.unit {
	case unitDay:
		return []comparison{
			// The weekStart given to shift is ignored for day and month
			// bounds, so Monday here is arbitrary and harmless.
			{"Prev day", p.shift(time.Monday, -1)},
			{"Week", period{unit: unitWeek, ref: p.ref}},
		}
	case unitWeek:
		return []comparison{
			// Not p.shift(): shift needs the weekStart to find the week's
			// start, and this function has none. Moving ref back seven days
			// lands in the previous week whatever the week starts on, and
			// bounds normalizes from there.
			{"Prev week", period{unit: unitWeek, ref: p.ref.AddDate(0, 0, -7)}},
			{"Month", period{unit: unitMonth, ref: p.ref}},
		}
	case unitMonth:
		return []comparison{
			{"Prev month", p.shift(time.Monday, -1)},
			{"Year", period{unit: unitYear, ref: p.ref}},
		}
	// unitAll has no neighbour to shift to; unitYear is only ever a comparison
	// itself, never the period the header is showing.
	default:
		return nil
	}
}
