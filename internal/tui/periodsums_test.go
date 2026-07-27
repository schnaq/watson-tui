package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestSumInPeriodFollowsTheStartRule: a frame counts in full for the period it
// begins in — the rule every view in this project shares.
func TestSumInPeriodFollowsTheStartRule(t *testing.T) {
	// Mittwoch 2026-07-22; Woche (Mo) = 20.07.–26.07.
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	week := period{unit: unitWeek, ref: ref}

	inside := mkFrame("a1111111111111111111111111111111", "p",
		time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), 2*time.Hour)
	// Beginnt Sonntag vor der Woche, endet darin: zählt NICHT.
	before := mkFrame("b2222222222222222222222222222222", "p",
		time.Date(2026, 7, 19, 23, 0, 0, 0, time.Local), 3*time.Hour)
	// Beginnt Sonntag der Woche 23:00, endet Montag darauf: zählt VOLLSTÄNDIG.
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
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local) // Mittwoch, Juli

	week := comparisonPeriods(period{unit: unitWeek, ref: ref})
	if len(week) != 2 || week[0].label != "Vorwoche" || week[1].label != "Monat" {
		t.Fatalf("week comparisons = %+v", week)
	}
	// Vorwoche endet dort, wo die aktuelle Woche beginnt.
	curFrom, _, _ := (period{unit: unitWeek, ref: ref}).bounds(time.Monday)
	_, prevTo, _ := week[0].per.bounds(time.Monday)
	if !prevTo.Equal(curFrom) {
		t.Errorf("Vorwoche endet %v, aktuelle Woche beginnt %v", prevTo, curFrom)
	}
	// Der Monatsvergleich ist der Monat des Wochenstarts.
	monFrom, _, _ := week[1].per.bounds(time.Monday)
	if int(monFrom.Month()) != 7 || monFrom.Day() != 1 {
		t.Errorf("Monat beginnt %v, want 01.07.", monFrom)
	}

	day := comparisonPeriods(period{unit: unitDay, ref: ref})
	if len(day) != 2 || day[0].label != "Vortag" || day[1].label != "Woche" {
		t.Fatalf("day comparisons = %+v", day)
	}

	month := comparisonPeriods(period{unit: unitMonth, ref: ref})
	if len(month) != 2 || month[0].label != "Vormonat" || month[1].label != "Jahr" {
		t.Fatalf("month comparisons = %+v", month)
	}
	// Der Jahresvergleich umspannt das Kalenderjahr des Monats.
	yFrom, yTo, ok := month[1].per.bounds(time.Monday)
	if !ok || yFrom.Year() != 2026 || int(yFrom.Month()) != 1 || yTo.Year() != 2027 {
		t.Errorf("Jahr = %v..%v ok=%v", yFrom, yTo, ok)
	}

	if got := comparisonPeriods(period{unit: unitAll, ref: ref}); len(got) != 0 {
		t.Errorf("unitAll must have no comparisons, got %+v", got)
	}
}

// TestComparisonPeriodsRespectWeekStart: comparisonPeriods has no weekStart
// parameter, so the previous period must be found in a way that does not need
// one. Every assertion above uses Monday and would miss a Monday-only fix.
func TestComparisonPeriodsRespectWeekStart(t *testing.T) {
	// Sonntag 19.07.2026. Bei weekStart=Sonntag ist das der Wochenanfang,
	// die Vorwoche also 12.07.–18.07.
	ref := time.Date(2026, 7, 19, 12, 0, 0, 0, time.Local)
	cur := period{unit: unitWeek, ref: ref}
	curFrom, _, _ := cur.bounds(time.Sunday)

	prev := comparisonPeriods(cur)[0]
	prevFrom, prevTo, _ := prev.per.bounds(time.Sunday)
	if !prevTo.Equal(curFrom) {
		t.Errorf("Vorwoche %v..%v grenzt nicht an die aktuelle Woche ab %v",
			prevFrom, prevTo, curFrom)
	}
	if prevFrom.Day() != 12 || int(prevFrom.Month()) != 7 {
		t.Errorf("Vorwoche beginnt %v, want 12.07.", prevFrom)
	}

	// Tag und Monat hängen nicht vom Wochenstart ab, aber der Vergleich muss
	// auch dort anschließen.
	day := period{unit: unitDay, ref: ref}
	dayFrom, _, _ := day.bounds(time.Sunday)
	_, prevDayTo, _ := comparisonPeriods(day)[0].per.bounds(time.Sunday)
	if !prevDayTo.Equal(dayFrom) {
		t.Errorf("Vortag endet %v, Tag beginnt %v", prevDayTo, dayFrom)
	}
	month := period{unit: unitMonth, ref: ref}
	monFrom, _, _ := month.bounds(time.Sunday)
	_, prevMonTo, _ := comparisonPeriods(month)[0].per.bounds(time.Sunday)
	if !prevMonTo.Equal(monFrom) {
		t.Errorf("Vormonat endet %v, Monat beginnt %v", prevMonTo, monFrom)
	}
}

// TestYearPeriodShiftAndLabel pins the two unitYear cases the comparison tests
// above never reach. Without them unitYear falls into the month default and
// every other test still passes, so the unit would silently mean "month".
func TestYearPeriodShiftAndLabel(t *testing.T) {
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	y := period{unit: unitYear, ref: ref}
	if got := y.label(time.Monday); got != "Jahr 2026" {
		t.Errorf("label = %q, want Jahr 2026", got)
	}
	// +1 discriminates; -1 does not. Without the unitYear case, shift's default
	// gives Februar 2026 — whose bounds are still 2026, so -1 looks correct.
	from, _, ok := y.shift(time.Monday, 1).bounds(time.Monday)
	if !ok || from.Year() != 2027 || int(from.Month()) != 1 || from.Day() != 1 {
		t.Errorf("next year starts %v ok=%v, want 01.01.2027", from, ok)
	}
}
