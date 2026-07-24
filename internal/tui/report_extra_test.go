package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

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
	out := reportModel{per: period{unit: unitWeek, ref: day}}.view(frames, time.Monday)
	for _, want := range []string{"Report", "alpha", "[code]", "Gesamt", "2h 00m"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q in:\n%s", want, out)
		}
	}
}

// TestReportViewEmpty: empty period shows the hint line.
func TestReportViewEmpty(t *testing.T) {
	out := newReportModel(time.Now()).view(nil, time.Monday)
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
