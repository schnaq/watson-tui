package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestSumInPeriodFollowsTheStartRule: a frame counts in full for the period it
// begins in — the rule every view in this project shares.
func TestSumInPeriodFollowsTheStartRule(t *testing.T) {
	// Wednesday 2026-07-22; the week (Mon) runs 20.–26.
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	week := period{unit: unitWeek, ref: ref}

	inside := mkFrame("a1111111111111111111111111111111", "p",
		time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), 2*time.Hour)
	// Starts the Sunday before the week and ends inside it: does NOT count.
	before := mkFrame("b2222222222222222222222222222222", "p",
		time.Date(2026, 7, 19, 23, 0, 0, 0, time.Local), 3*time.Hour)
	// Starts 23:00 on the week's Sunday and ends the Monday after: counts IN FULL.
	spanning := mkFrame("c3333333333333333333333333333333", "p",
		time.Date(2026, 7, 26, 23, 0, 0, 0, time.Local), 3*time.Hour)

	if got := sumInPeriod([]watson.Frame{inside}, week, time.Monday); got != 2*time.Hour {
		t.Errorf("frame inside: got %v, want 2h", got)
	}
	if got := sumInPeriod([]watson.Frame{before}, week, time.Monday); got != 0 {
		t.Errorf("frame starting before the period must not count, got %v", got)
	}
	if got := sumInPeriod([]watson.Frame{spanning}, week, time.Monday); got != 3*time.Hour {
		t.Errorf("frame starting inside counts in full: got %v, want 3h", got)
	}
	all := []watson.Frame{inside, before, spanning}
	if got := sumInPeriod(all, week, time.Monday); got != 5*time.Hour {
		t.Errorf("sum: got %v, want 5h", got)
	}
}

// TestSumInPeriodUnbounded: unitAll has no bounds, so everything counts.
func TestSumInPeriodUnbounded(t *testing.T) {
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p", ref.AddDate(-3, 0, 0), time.Hour),
		mkFrame("b2222222222222222222222222222222", "p", ref, 30*time.Minute),
	}
	if got := sumInPeriod(frames, period{unit: unitAll, ref: ref}, time.Monday); got != 90*time.Minute {
		t.Errorf("got %v, want 1h30m", got)
	}
}

// TestComparisonPeriodsPerUnit: the labels and bounds the header shows.
func TestComparisonPeriodsPerUnit(t *testing.T) {
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local) // Wednesday, July

	week := comparisonPeriods(period{unit: unitWeek, ref: ref})
	if len(week) != 2 || week[0].label != "Prev week" || week[1].label != "Month" {
		t.Fatalf("week comparisons = %+v", week)
	}
	// The previous week ends where the current one begins.
	curFrom, _, _ := (period{unit: unitWeek, ref: ref}).bounds(time.Monday)
	_, prevTo, _ := week[0].per.bounds(time.Monday)
	if !prevTo.Equal(curFrom) {
		t.Errorf("prev week ends %v, current week begins %v", prevTo, curFrom)
	}
	// The month comparison is the month the week starts in.
	monFrom, _, _ := week[1].per.bounds(time.Monday)
	if int(monFrom.Month()) != 7 || monFrom.Day() != 1 {
		t.Errorf("month begins %v, want 2026-07-01", monFrom)
	}

	day := comparisonPeriods(period{unit: unitDay, ref: ref})
	if len(day) != 2 || day[0].label != "Prev day" || day[1].label != "Week" {
		t.Fatalf("day comparisons = %+v", day)
	}

	month := comparisonPeriods(period{unit: unitMonth, ref: ref})
	if len(month) != 2 || month[0].label != "Prev month" || month[1].label != "Year" {
		t.Fatalf("month comparisons = %+v", month)
	}
	// The year comparison spans the calendar year the month is in.
	yFrom, yTo, ok := month[1].per.bounds(time.Monday)
	if !ok || yFrom.Year() != 2026 || int(yFrom.Month()) != 1 || yTo.Year() != 2027 {
		t.Errorf("year = %v..%v ok=%v", yFrom, yTo, ok)
	}

	if got := comparisonPeriods(period{unit: unitAll, ref: ref}); len(got) != 0 {
		t.Errorf("unitAll must have no comparisons, got %+v", got)
	}
}

// TestComparisonPeriodsRespectWeekStart: comparisonPeriods has no weekStart
// parameter, so the previous period must be found in a way that does not need
// one. Every assertion above uses Monday and would miss a Monday-only fix.
func TestComparisonPeriodsRespectWeekStart(t *testing.T) {
	// Sunday 2026-07-19. With weekStart=Sunday that is the start of the week,
	// so the previous week runs 2026-07-12 to 2026-07-18.
	ref := time.Date(2026, 7, 19, 12, 0, 0, 0, time.Local)
	cur := period{unit: unitWeek, ref: ref}
	curFrom, _, _ := cur.bounds(time.Sunday)

	prev := comparisonPeriods(cur)[0]
	prevFrom, prevTo, _ := prev.per.bounds(time.Sunday)
	if !prevTo.Equal(curFrom) {
		t.Errorf("prev week %v..%v does not abut the current week from %v",
			prevFrom, prevTo, curFrom)
	}
	if prevFrom.Day() != 12 || int(prevFrom.Month()) != 7 {
		t.Errorf("prev week begins %v, want 2026-07-12", prevFrom)
	}

	// Day and month do not depend on the week start, but the comparison has to
	// abut there too.
	day := period{unit: unitDay, ref: ref}
	dayFrom, _, _ := day.bounds(time.Sunday)
	_, prevDayTo, _ := comparisonPeriods(day)[0].per.bounds(time.Sunday)
	if !prevDayTo.Equal(dayFrom) {
		t.Errorf("prev day ends %v, day begins %v", prevDayTo, dayFrom)
	}
	month := period{unit: unitMonth, ref: ref}
	monFrom, _, _ := month.bounds(time.Sunday)
	_, prevMonTo, _ := comparisonPeriods(month)[0].per.bounds(time.Sunday)
	if !prevMonTo.Equal(monFrom) {
		t.Errorf("prev month ends %v, month begins %v", prevMonTo, monFrom)
	}
}

// TestYearPeriodShiftAndLabel pins the two unitYear cases the comparison tests
// above never reach. Without them unitYear falls into the month default and
// every other test still passes, so the unit would silently mean "month".
func TestYearPeriodShiftAndLabel(t *testing.T) {
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	y := period{unit: unitYear, ref: ref}
	if got := y.label(time.Monday); got != "Year 2026" {
		t.Errorf("label = %q, want Year 2026", got)
	}
	// +1 discriminates; -1 does not. Without the unitYear case, shift's default
	// gives February 2026 — whose bounds are still 2026, so -1 looks correct.
	from, _, ok := y.shift(time.Monday, 1).bounds(time.Monday)
	if !ok || from.Year() != 2027 || int(from.Month()) != 1 || from.Day() != 1 {
		t.Errorf("next year starts %v ok=%v, want 01.01.2027", from, ok)
	}
}
