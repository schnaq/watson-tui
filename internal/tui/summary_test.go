package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// kinds lists the row kinds in order, for readable assertions.
func kinds(rows []row) []rowKind {
	out := make([]rowKind, len(rows))
	for i, r := range rows {
		out[i] = r.kind
	}
	return out
}

// titles lists the titles of the rows of one kind.
func titles(rows []row, k rowKind) []string {
	var out []string
	for _, r := range rows {
		if r.kind == k {
			out = append(out, r.title)
		}
	}
	return out
}

// TestSummaryGroupsDaysProjectsAndTags: the shape of the view.
func TestSummaryGroupsDaysProjectsAndTags(t *testing.T) {
	// Monday 2026-07-20, week runs 20.–26.
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), time.Hour, "call"),
		mkFrame("c3333333333333333333333333333333", "alpha", mon.AddDate(0, 0, 1), 30*time.Minute, "docs"),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)

	days := titles(rows, rowDayHeader)
	if len(days) != 7 {
		t.Fatalf("a week has seven day rows, got %d: %v", len(days), days)
	}
	if days[0] != "Monday, 2026-07-20" {
		t.Errorf("first day = %q", days[0])
	}

	// A blank line stands before every day but the first, so the blocks read
	// apart. Pinned here rather than in the one-day test below, which never has
	// a second day header to be separated from.
	if rows[0].kind == rowBlank {
		t.Errorf("no blank row stands before the first day")
	}
	for i, r := range rows {
		if r.kind == rowDayHeader && i > 0 && rows[i-1].kind != rowBlank {
			t.Errorf("day header %q at row %d is not preceded by a blank", r.title, i)
		}
	}

	// Monday: alpha (3h) before beta (1h), each followed by its tag.
	var mondayBlock []row
	for i, r := range rows {
		if r.kind == rowDayHeader && r.title == "Monday, 2026-07-20" {
			for _, next := range rows[i+1:] {
				if next.kind == rowDayHeader {
					break
				}
				mondayBlock = append(mondayBlock, next)
			}
			break
		}
	}
	projects := titles(mondayBlock, rowProject)
	if len(projects) != 2 || projects[0] != "alpha" || projects[1] != "beta" {
		t.Errorf("monday projects = %v, want [alpha beta] sorted by duration", projects)
	}
	if got := titles(mondayBlock, rowTag); len(got) != 2 {
		t.Errorf("monday tags = %v, want one per project", got)
	}
}

// TestSummaryNestsRowsAndSeparatesProjects pins the exact sequence of kinds for
// one day with two projects: the tags sit under their project, a blank line
// separates the project blocks, and none stands before the first day.
func TestSummaryNestsRowsAndSeparatesProjects(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 3*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(4*time.Hour), time.Hour, "call"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)

	want := []rowKind{rowDayHeader, rowProject, rowTag, rowBlank, rowProject, rowTag}
	got := kinds(rows)
	if len(got) != len(want) {
		t.Fatalf("row kinds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row kinds = %v, want %v", got, want)
		}
	}
}

// TestSummaryShowsEmptyDays: a period is legible only if its gaps are visible.
func TestSummaryShowsEmptyDays(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitWeek, ref: mon}, time.Monday, "", nil, mon)
	var empty int
	for _, r := range rows {
		if r.kind == rowDayHeader && r.total == 0 {
			empty++
		}
	}
	if empty != 6 {
		t.Errorf("six of seven days are empty, got %d", empty)
	}
}

// TestSummaryDayTotalCountsFramesNotTags: a frame with two tags counts once
// towards the day, though it appears under both tags.
func TestSummaryDayTotalCountsFramesNotTags(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, 2*time.Hour, "code", "review"),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "", nil, mon)
	for _, r := range rows {
		if r.kind == rowDayHeader {
			if r.total != 2*time.Hour {
				t.Errorf("day total = %v, want 2h — the tags must not be added up", r.total)
			}
		}
	}
	if got := len(titles(rows, rowTag)); got != 2 {
		t.Errorf("both tags must be listed, got %d", got)
	}
}

// TestSummaryRespectsTheFilter: the same filter the frame list applies.
func TestSummaryRespectsTheFilter(t *testing.T) {
	mon := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", mon, time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", mon.Add(2*time.Hour), time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitDay, ref: mon}, time.Monday, "alpha", nil, mon)
	if got := titles(rows, rowProject); len(got) != 1 || got[0] != "alpha" {
		t.Errorf("filtered projects = %v, want [alpha]", got)
	}
}

// TestSummaryCountsTheRunningTimer: like the report and the overview.
func TestSummaryCountsTheRunningTimer(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	state := &watson.State{Project: "running", Start: now.Add(-time.Hour), Tags: []string{}}
	rows := buildSummaryRows(nil, period{unit: unitDay, ref: now}, time.Monday, "", state, now)
	if got := titles(rows, rowProject); len(got) != 1 || got[0] != "running" {
		t.Errorf("projects = %v, want the running timer", got)
	}
	for _, r := range rows {
		if r.kind == rowDayHeader && r.total != time.Hour {
			t.Errorf("day total = %v, want 1h from the running timer", r.total)
		}
	}
}

// TestSummaryUnboundedSkipsEmptyDays: with no period bounds, a row per day
// since the first frame would be thousands of lines.
func TestSummaryUnboundedSkipsEmptyDays(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.AddDate(-1, 0, 0), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now, time.Hour),
	}
	rows := buildSummaryRows(frames, period{unit: unitAll, ref: now}, time.Monday, "", nil, now)
	if got := len(titles(rows, rowDayHeader)); got != 2 {
		t.Errorf("unitAll must list only days with frames, got %d day rows", got)
	}
}
