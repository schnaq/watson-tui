package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func mkFrame(id, project string, start time.Time, dur time.Duration, tags ...string) watson.Frame {
	return watson.Frame{
		ID: id, Project: project, Start: start.UTC(), Stop: start.Add(dur).UTC(),
		Tags: tags, UpdatedAt: start.UTC(),
	}
}

func TestPeriodBoundsWeek(t *testing.T) {
	// Wednesday 2026-07-22, week start Monday -> 2026-07-20 to (excl.) 2026-07-27
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	from, to, ok := period{unit: unitWeek, ref: ref}.bounds(time.Monday)
	if !ok || from.Day() != 20 || to.Day() != 27 {
		t.Errorf("from=%v to=%v ok=%v", from, to, ok)
	}
}

func TestPeriodBoundsWeekSundayStart(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	from, _, _ := period{unit: unitWeek, ref: ref}.bounds(time.Sunday)
	if from.Day() != 19 {
		t.Errorf("from=%v, want 2026-07-19", from)
	}
}

func TestPeriodShiftMonthFromJan31(t *testing.T) {
	ref := time.Date(2026, 1, 31, 12, 0, 0, 0, time.Local)
	p := period{unit: unitMonth, ref: ref}.shift(time.Monday, 1)
	from, _, _ := p.bounds(time.Monday)
	if int(from.Month()) != 2 {
		t.Errorf("shift from Jan 31 must land in February, got %v", from)
	}
}

func TestPeriodAllUnbounded(t *testing.T) {
	if _, _, ok := (period{unit: unitAll, ref: time.Now()}).bounds(time.Monday); ok {
		t.Error("unitAll must be unbounded")
	}
}

func TestBuildRowsGroupsAndFilters(t *testing.T) {
	day1 := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	day2 := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day1, time.Hour, "x"),
		mkFrame("b2222222222222222222222222222222", "beta", day2, 30*time.Minute),
		mkFrame("c3333333333333333333333333333333", "alpha", day2, time.Hour),
	}
	p := period{unit: unitWeek, ref: day1}
	rows := buildRows(frames, p, time.Monday, "")
	if len(rows) != 5 { // 2 day headers + 3 frames
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	if rows[0].kind != rowDayHeader || rows[2].kind != rowDayHeader {
		t.Errorf("headers at wrong positions")
	}

	rows = buildRows(frames, p, time.Monday, "beta")
	if len(rows) != 2 || rows[1].frame.Project != "beta" {
		t.Errorf("filter beta: %+v", rows)
	}

	rows = buildRows(frames, p, time.Monday, "x") // tag filter
	if len(rows) != 2 || rows[1].frame.Project != "alpha" {
		t.Errorf("filter tag x: %+v", rows)
	}

	rows = buildRows(frames, period{unit: unitDay, ref: day1}, time.Monday, "")
	if len(rows) != 2 {
		t.Errorf("day period: %+v", rows)
	}
}

func TestListNavigationSkipsHeaders(t *testing.T) {
	app := newTestApp(t)
	now := time.Now()
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-50*time.Hour), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(-2*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.compact = false // selected() names a frame, and only the frame list has one
	app.list.refresh(app.frames, time.Monday, app.state, app.now)
	if f, ok := app.list.selected(); !ok || f.Project != "alpha" {
		t.Fatalf("initial selection: %+v", f)
	}
	app.Update(key("j"))
	if f, _ := app.list.selected(); f.Project != "beta" {
		t.Errorf("after j: %v", f.Project)
	}
	app.Update(key("j")) // at the end: stays put
	if f, _ := app.list.selected(); f.Project != "beta" {
		t.Errorf("j at end moved cursor")
	}
	app.Update(key("k"))
	if f, _ := app.list.selected(); f.Project != "alpha" {
		t.Errorf("after k: %v", f.Project)
	}
}

func TestFilterKeyFlow(t *testing.T) {
	app := newTestApp(t)
	now := time.Now()
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(-4*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.compact = false // selected() names a frame, and only the frame list has one
	app.list.refresh(app.frames, time.Monday, app.state, app.now)
	app.Update(key("/"))
	if !app.list.filtering {
		t.Fatal("/ must enter filter mode")
	}
	app.Update(key("b"))
	app.Update(key("enter"))
	if app.list.filtering || app.list.filter != "b" {
		t.Fatalf("filter state: %q filtering=%v", app.list.filter, app.list.filtering)
	}
	if f, ok := app.list.selected(); !ok || f.Project != "beta" {
		t.Errorf("selected = %+v", f)
	}
	app.Update(key("/"))
	app.Update(key("esc"))
	if app.list.filter != "" {
		t.Error("esc must clear filter")
	}
}
