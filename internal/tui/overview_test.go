package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func TestOverviewColumns(t *testing.T) {
	// Mittwoch 2026-07-22, Wochenstart Montag
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	if len(cols) != 5 {
		t.Fatalf("got %d columns, want 5", len(cols))
	}
	titles := []string{"diese Woche", "letzte Woche", "dieser Monat", "letzter Monat", "gesamt"}
	for i, want := range titles {
		if cols[i].title != want {
			t.Errorf("col %d = %q, want %q", i, cols[i].title, want)
		}
	}
	// letzte Woche endet, wo diese Woche beginnt
	thisFrom, _, _ := cols[0].per.bounds(time.Monday)
	_, lastTo, _ := cols[1].per.bounds(time.Monday)
	if !lastTo.Equal(thisFrom) {
		t.Errorf("last week to=%v, this week from=%v", lastTo, thisFrom)
	}
	// letzter Monat = Juni
	juneFrom, _, _ := cols[3].per.bounds(time.Monday)
	if int(juneFrom.Month()) != 6 || juneFrom.Day() != 1 {
		t.Errorf("letzter Monat from=%v, want 01.06.", juneFrom)
	}
	// gesamt unbegrenzt
	if _, _, ok := cols[4].per.bounds(time.Monday); ok {
		t.Error("gesamt must be unbounded")
	}
}

func TestBuildOverview(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	cols := overviewColumns(now, time.Monday)
	frames := []watson.Frame{
		// diese Woche (Mo 20.07.): zählt in Woche, Monat, gesamt
		mkFrame("a1111111111111111111111111111111", "alpha", time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local), time.Hour),
		// letzte Woche (Di 14.07.): letzte Woche, dieser Monat, gesamt
		mkFrame("b2222222222222222222222222222222", "beta", time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 2*time.Hour),
		// letzter Monat (10.06.): letzter Monat, gesamt
		mkFrame("c3333333333333333333333333333333", "gamma", time.Date(2026, 6, 10, 9, 0, 0, 0, time.Local), 3*time.Hour),
		// uralt (2020): nur gesamt
		mkFrame("d4444444444444444444444444444444", "alpha", time.Date(2020, 1, 6, 9, 0, 0, 0, time.Local), 4*time.Hour),
	}
	rows, totals := buildOverview(frames, cols, time.Monday)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	// Sortierung nach gesamt desc: alpha (5h), gamma (3h), beta (2h)
	if rows[0].project != "alpha" || rows[1].project != "gamma" || rows[2].project != "beta" {
		t.Fatalf("row order: %s, %s, %s", rows[0].project, rows[1].project, rows[2].project)
	}
	alpha := rows[0].cells
	if alpha[0] != time.Hour || alpha[1] != 0 || alpha[2] != time.Hour || alpha[3] != 0 || alpha[4] != 5*time.Hour {
		t.Errorf("alpha cells = %v", alpha)
	}
	beta := rows[2].cells
	if beta[0] != 0 || beta[1] != 2*time.Hour || beta[2] != 2*time.Hour || beta[4] != 2*time.Hour {
		t.Errorf("beta cells = %v", beta)
	}
	gamma := rows[1].cells
	if gamma[3] != 3*time.Hour || gamma[4] != 3*time.Hour || gamma[0] != 0 {
		t.Errorf("gamma cells = %v", gamma)
	}
	want := []time.Duration{time.Hour, 2 * time.Hour, 3 * time.Hour, 3 * time.Hour, 10 * time.Hour}
	for i, w := range want {
		if totals[i] != w {
			t.Errorf("totals[%d] = %v, want %v", i, totals[i], w)
		}
	}
}

func TestOverviewKeyFlow(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("o"))
	if app.mode != modeOverview {
		t.Fatal("o must open overview")
	}
	if !strings.Contains(app.View(), "Übersicht") {
		t.Error("overview view missing title")
	}
	app.Update(key("esc"))
	if app.mode != modeList {
		t.Error("esc must return to list")
	}
}
