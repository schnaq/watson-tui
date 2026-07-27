package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func TestAggregate(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, 2*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "alpha", day.Add(3*time.Hour), time.Hour, "review"),
		mkFrame("c3333333333333333333333333333333", "beta", day.Add(5*time.Hour), 30*time.Minute),
		// outside the period:
		mkFrame("d4444444444444444444444444444444", "alpha", day.AddDate(0, 0, 14), time.Hour),
	}
	lines, grand := aggregate(frames, period{unit: unitWeek, ref: day}, time.Monday)
	if grand != 3*time.Hour+30*time.Minute {
		t.Errorf("grand = %v", grand)
	}
	if len(lines) != 2 || lines[0].project != "alpha" || lines[1].project != "beta" {
		t.Fatalf("lines = %+v", lines)
	}
	if lines[0].total != 3*time.Hour {
		t.Errorf("alpha total = %v", lines[0].total)
	}
	if len(lines[0].tags) != 2 || lines[0].tags[0].tag != "code" {
		t.Errorf("alpha tags = %+v", lines[0].tags)
	}
}

func TestAggregateEmpty(t *testing.T) {
	lines, grand := aggregate(nil, period{unit: unitWeek, ref: time.Now()}, time.Monday)
	if len(lines) != 0 || grand != 0 {
		t.Errorf("lines=%v grand=%v", lines, grand)
	}
}

func TestReportKeyFlow(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("r"))
	if app.mode != modeReport {
		t.Fatal("r must open report")
	}
	app.Update(key("m"))
	if app.report.per.unit != unitMonth {
		t.Error("m must switch report to month")
	}
	before := app.report.per
	app.Update(key("["))
	if app.report.per == before {
		t.Error("[ must shift the period")
	}
	app.Update(key("esc"))
	if app.mode != modeList {
		t.Error("esc must return to list")
	}
}
