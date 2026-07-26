package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestReportKeepsEveryDurationItComputed is the report's half of the rule the
// overview already follows: when the width does not fit, the label gives way and
// the number survives. The view formatted its lines as "%-32s %10s" — 43 columns
// regardless of the terminal — so from a body width of 46 down fitBody cut the
// right edge, and because the number is right-aligned the digits went first:
// "kunde-a  124h 30" at 46, "Gesamt  124" and "kunde-a  124" at 42. Unlike the
// overview's clip this one is not inherited from v0.1.0; before the chrome the
// line simply wrapped.
//
// The sweep therefore asserts both halves: every duration aggregate computed has
// to stand in the output verbatim, at the end of its line where the right
// alignment puts it, and no line may be wider than the body it was given.
func TestReportKeepsEveryDurationItComputed(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	monday := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		// 124h 30m is a month of work on one customer: the widest number this view
		// renders in practice, and the one an invoice is written from.
		mkFrame("a1111111111111111111111111111111", "kunde-a", monday, 124*time.Hour+30*time.Minute, "code"),
		mkFrame("b2222222222222222222222222222222", "schnaq-intern-lange-projektbezeichnung",
			monday.Add(2*time.Hour), 45*time.Minute, "ops", "review"),
		// Years of tracked time in one frame: "12345h 00m" is ten columns, wider than
		// the eight "999h 59m" needs, so the number column cannot be a fixed width
		// without cutting this total.
		mkFrame("c3333333333333333333333333333333", "altprojekt", monday.Add(time.Hour),
			12345*time.Hour, "wartung"),
	}
	state := &watson.State{Project: "laufend", Start: now.Add(-90 * time.Minute), Tags: []string{"live"}}
	per := period{unit: unitWeek, ref: now}
	m := reportModel{per: per}

	lines, grand := aggregate(withRunning(frames, state, now), per, time.Monday)
	want := []string{formatDuration(grand)}
	for _, l := range lines {
		want = append(want, formatDuration(l.total))
		for _, tl := range l.tags {
			want = append(want, formatDuration(tl.d))
		}
	}

	for width := 20; width <= 100; width++ {
		out := m.view(frames, state, time.Monday, now, width)
		rendered := strings.Split(out, "\n")
		for i, line := range rendered {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: rendered line %d is %d wide: %q", width, i, w, line)
			}
		}
		for _, d := range want {
			// At the end of a line, not merely somewhere: "30m" is a substring of
			// "124h 30m", so a plain Contains would pass on a clipped total.
			found := false
			for _, line := range rendered {
				if strings.HasSuffix(strings.TrimRight(line, " "), d) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("width %d: duration %q is not rendered as computed:\n%s", width, d, out)
			}
		}
	}

	// The same numbers through the whole pipeline: App.View hands the report a
	// body width and fitBody cuts whatever comes back wider. That is where the
	// digits actually went, so the check has to stand on this side of it too.
	for width := 20; width <= 100; width++ {
		app := chromeApp(t, now, width, 30, frames)
		app.now, app.state, app.report = now, state, m
		app.mode = modeReport
		out := app.View()
		for _, d := range want {
			found := false
			for _, line := range strings.Split(out, "\n") {
				// The panel closes every body line with padding and a border column;
				// trim those before asking what the line ends with.
				if strings.HasSuffix(strings.TrimRight(line, " │"), d) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%dx30: duration %q did not survive the panel:\n%s", width, d, out)
			}
		}
	}
}

// TestAggregateTieBreak: equal durations sort alphabetically, for both
// projects and tags within a project.
func TestAggregateTieBreak(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("f1111111111111111111111111111111", "gamma", day, time.Hour, "zeta", "alef"),
		mkFrame("f2222222222222222222222222222222", "beta", day.Add(2*time.Hour), time.Hour),
	}
	lines, grand := aggregate(frames, period{unit: unitWeek, ref: day}, time.Monday)
	if grand != 2*time.Hour {
		t.Errorf("grand = %v", grand)
	}
	// same duration (1h each) → alphabetical: beta before gamma
	if len(lines) != 2 || lines[0].project != "beta" || lines[1].project != "gamma" {
		t.Fatalf("project tie order = %+v", lines)
	}
	// gamma's tags have equal duration → alphabetical: alef before zeta
	g := lines[1]
	if len(g.tags) != 2 || g.tags[0].tag != "alef" || g.tags[1].tag != "zeta" {
		t.Errorf("tag tie order = %+v", g.tags)
	}
}

// TestAggregateAll: unitAll is unbounded, so frames far apart both count.
func TestAggregateAll(t *testing.T) {
	day := time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, time.Hour),
		mkFrame("a2222222222222222222222222222222", "alpha", day.AddDate(0, 6, 0), time.Hour),
	}
	lines, grand := aggregate(frames, period{unit: unitAll, ref: day}, time.Monday)
	if grand != 2*time.Hour || len(lines) != 1 || lines[0].total != 2*time.Hour {
		t.Errorf("unitAll must count all frames: lines=%+v grand=%v", lines, grand)
	}
}

// TestReportViewRenders: the view shows project totals, a tag line and the grand total.
func TestReportViewRenders(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, 2*time.Hour, "code"),
	}
	out := reportModel{per: period{unit: unitWeek, ref: day}}.view(frames, nil, time.Monday, day, 80)
	for _, want := range []string{"alpha", "[code]", "Gesamt", "2h 00m"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q in:\n%s", want, out)
		}
	}
	// The title moved into the header, which names the view and its period, so
	// the body must not repeat it.
	if strings.Contains(out, "Report") {
		t.Errorf("body must not repeat the header's title:\n%s", out)
	}
	app := newTestApp(t)
	app.mode = modeReport
	app.report = reportModel{per: period{unit: unitWeek, ref: day}}
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Report") {
		t.Errorf("header must name the view: %q", got)
	}
}

// TestReportViewEmpty: empty period shows the hint line.
func TestReportViewEmpty(t *testing.T) {
	out := newReportModel(time.Now()).view(nil, nil, time.Monday, time.Now(), 80)
	if !strings.Contains(out, "keine Frames im Zeitraum") {
		t.Errorf("empty report must show hint, got:\n%s", out)
	}
}

// TestReportKeysDayWeekForwardClose covers t/w/] and q (close), which the
// primary flow test does not exercise.
func TestReportKeysDayWeekForwardClose(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("r"))
	app.Update(key("t"))
	if app.report.per.unit != unitDay {
		t.Error("t must switch report to day")
	}
	app.Update(key("w"))
	if app.report.per.unit != unitWeek {
		t.Error("w must switch report to week")
	}
	before := app.report.per
	app.Update(key("]"))
	if app.report.per == before {
		t.Error("] must shift the period forward")
	}
	app.Update(key("q"))
	if app.mode != modeList {
		t.Error("q must return to list")
	}
}

// TestReportCountsRunningTimer: the report is a money-facing view, so a running
// timer counts up to now and is disclosed — same rule as the overview.
func TestReportCountsRunningTimer(t *testing.T) {
	now := time.Now()
	state := &watson.State{Project: "laufend", Start: now.Add(-90 * time.Minute), Tags: []string{"live"}}
	out := reportModel{per: period{unit: unitWeek, ref: now}}.view(nil, state, time.Monday, now, 80)
	for _, want := range []string{"laufend", "1h 30m", "eingerechnet"} {
		if !strings.Contains(out, want) {
			t.Errorf("running timer missing %q in:\n%s", want, out)
		}
	}
}

// TestReportRunningTimerOutsidePeriod: a shifted period must not pick up the
// running timer, because its start lies outside those bounds.
func TestReportRunningTimerOutsidePeriod(t *testing.T) {
	now := time.Now()
	state := &watson.State{Project: "laufend", Start: now.Add(-30 * time.Minute), Tags: []string{}}
	lastWeek := period{unit: unitWeek, ref: now}.shift(time.Monday, -1)
	out := reportModel{per: lastWeek}.view(nil, state, time.Monday, now, 80)
	if strings.Contains(out, "laufend") {
		t.Errorf("running timer must not appear in a past period:\n%s", out)
	}
}
