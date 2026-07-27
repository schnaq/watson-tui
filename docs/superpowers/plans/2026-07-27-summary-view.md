# Kompakte Tagesübersicht — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Beim Start zeigt watson-tui die Tage des Zeitraums mit ihren Projekt- und Tag-Summen; `f` schaltet auf die bisherige Frameliste um.

**Architecture:** Die kompakte Ansicht ist eine zweite Darstellung der Liste, keine eigene Betriebsart — Zeitraum-Navigation, Filter, Kopf und Höhenbudget gelten dadurch unverändert. `row` bekommt eine Art-Kennung statt des heutigen `isHeader bool`, die Aggregation liegt in einer eigenen Datei, und `refresh` wählt anhand eines Schalters, welcher Zeilenbauer läuft.

**Tech Stack:** Go — keine neuen Dependencies.

**Spec:** `docs/superpowers/specs/2026-07-27-summary-view-design.md`

**Ausgangspunkt:** `main` bei `c8bc35d`.

## Global Constraints

- Nur ANSI-Farben 0–15; keine neuen Dependencies; englische Oberfläche, englische Bezeichner
- Minuten, keine Sekunden — wie überall sonst
- Keine neue Berechnung: die Zahlen kommen aus dem vorhandenen `aggregate`
- Zahlen und IDs werden nie gekürzt; wenn eine Zeile nicht passt, weicht der Name mit `…`
- Bestehende Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in Liste, Report und Übersicht
- TDD: Test zuerst, RED verifizieren, dann implementieren; neue Tests durch Mutation der Implementierung auf Zähne prüfen
- Gate vor jedem Commit: `gofmt -l .` als eigener Schritt, dann `go build ./... && go vet ./... && go test ./... && golangci-lint run ./...`
- `commit.gpgsign=true` funktioniert — Commits signiert lassen
- Commit-Trailer: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` plus die `Claude-Session:`-Zeile

## Vorhandene Bausteine

- `row{isHeader bool, title string, total time.Duration, frame watson.Frame}` in `frameslist.go:121`
- `isHeader` wird gelesen in `app.go:486` (Frame-Zähler im Kopf) und `frameslist.go:189,200,211,266,275,431,466`
- `filterFrames(frames []watson.Frame, filter string) []watson.Frame` (`frameslist.go:145`)
- `aggregate(frames []watson.Frame, p period, weekStart time.Weekday) ([]reportLine, time.Duration)` in `report.go`; `reportLine{project string, total time.Duration, tags []tagLine}`, `tagLine{tag string, d time.Duration}`
- `withRunning(frames []watson.Frame, state *watson.State, now time.Time) []watson.Frame` in `overview.go`
- `period.bounds(weekStart) (from, to time.Time, ok bool)`, `formatDay(t) string`, `formatDuration(d) string`, `truncate(s string, max int) string`
- `styleDayHeader`, `styleDim`, `styleSelected`, `selectionMarker`

---

### Task 1: Zeilenarten statt `isHeader`

**Files:**
- Modify: `internal/tui/frameslist.go` (Typ `row`, alle sieben Lesestellen)
- Modify: `internal/tui/app.go:486`
- Test: `internal/tui/frameslist_test.go`, `frameslist_extra_test.go`

**Interfaces:**
- Produces: `type rowKind int` mit `rowFrame`, `rowDayHeader`, `rowProject`, `rowTag`, `rowBlank`; `row{kind rowKind, title string, total time.Duration, frame watson.Frame}`; `func (r row) selectable() bool`

Diese Task ändert **kein Verhalten** — sie ersetzt ein Bool durch eine Aufzählung, damit Task 2 vier Zeilenarten bauen kann. Die Suite muss danach unverändert grün sein.

- [ ] **Step 1: Typ und Helfer einführen**

In `frameslist.go`, `row` ersetzen:

```go
// rowKind distinguishes what a rendered line is. The frame list uses only
// rowFrame and rowDayHeader; the summary adds the other three.
type rowKind int

const (
	rowFrame rowKind = iota
	rowDayHeader
	rowProject
	rowTag
	rowBlank
)

type row struct {
	kind  rowKind
	title string
	total time.Duration
	frame watson.Frame
}

// selectable reports whether the cursor may rest on this row. Exactly one kind
// is selectable per view: frames in the list, projects in the summary — both
// are the row a user would act on.
func (r row) selectable() bool {
	return r.kind == rowFrame || r.kind == rowProject
}
```

- [ ] **Step 2: Die sieben Lesestellen umstellen**

- `frameslist.go:189` — `row{isHeader: true, title: …, total: …}` wird `row{kind: rowDayHeader, title: …, total: …}`
- `frameslist.go:200` (`firstFrameRow`) — `if !r.isHeader` wird `if r.selectable()`
- `frameslist.go:211` (`nextFrameRow`) — `if !rows[i].isHeader` wird `if rows[i].selectable()`
- `frameslist.go:266` (`refresh`, Cursor-Wiederherstellung) — `if !r.isHeader && r.frame.ID == prevID` wird `if r.kind == rowFrame && r.frame.ID == prevID`
- `frameslist.go:275` (`selected`) — `|| l.rows[l.cursor].isHeader` wird `|| l.rows[l.cursor].kind != rowFrame`; so liefert `selected()` in der kompakten Ansicht nie einen Frame, obwohl der Cursor dort auf Projektzeilen steht
- `frameslist.go:431` und `:466` (`view`, `renderRow`) — `if r.isHeader` wird `if r.kind == rowDayHeader`
- `app.go:486` (Frame-Zähler) — `if !r.isHeader` wird `if r.kind == rowFrame`; der Kopf zählt Frames, nicht Zeilen, und das bleibt in beiden Darstellungen richtig

Wo die Frame-Zeile gebaut wird (im Tagesgruppen-Block), `row{frame: fr}` explizit als `row{kind: rowFrame, frame: fr}` schreiben — `rowFrame` ist zwar der Nullwert, aber die Absicht soll dastehen.

- [ ] **Step 3: Test ergänzen, der die Auswahl-Regel festhält**

In `frameslist_extra_test.go`:

```go
// TestRowSelectability: the cursor rests on frames in the list and on projects
// in the summary — never on a header, a tag line or a blank.
func TestRowSelectability(t *testing.T) {
	cases := map[rowKind]bool{
		rowFrame: true, rowProject: true,
		rowDayHeader: false, rowTag: false, rowBlank: false,
	}
	for kind, want := range cases {
		if got := (row{kind: kind}).selectable(); got != want {
			t.Errorf("kind %d selectable = %v, want %v", kind, got, want)
		}
	}
}

// TestSelectedOnlyReturnsFrames: a project row is selectable but is not a
// frame, so selected() must refuse it — otherwise enter would edit a zero frame.
func TestSelectedOnlyReturnsFrames(t *testing.T) {
	l := listModel{rows: []row{{kind: rowProject, title: "alpha"}}, cursor: 0}
	if _, ok := l.selected(); ok {
		t.Error("selected() must not return a frame for a project row")
	}
}
```

- [ ] **Step 4: Suite grün**

Run: `go test ./... 2>&1 | tail -3`
Expected: PASS, unverändert — diese Task ändert kein Verhalten. Schlägt etwas fehl, ist eine Lesestelle übersehen worden.

- [ ] **Step 5: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "refactor: name what a rendered row is instead of flagging headers"
```

---

### Task 2: Die Zusammenfassung bauen

**Files:**
- Create: `internal/tui/summary.go`
- Test: `internal/tui/summary_test.go`

**Interfaces:**
- Consumes: `rowKind`/`row` (Task 1), `aggregate`, `withRunning`, `filterFrames`, `period.bounds`, `formatDay`, `formatDuration`
- Produces: `func buildSummaryRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string, state *watson.State, now time.Time) []row`

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/summary_test.go`)**

```go
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
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run TestSummary -v 2>&1 | tail -10`
Expected: FAIL — `undefined: buildSummaryRows`

- [ ] **Step 3: `internal/tui/summary.go` schreiben**

```go
// The compact view: one block per day of the period, each listing its projects
// and their tags. It answers "what did I work on" — the frame list answers
// "which sessions were there", which is the question one asks when editing.
package tui

import (
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// buildSummaryRows renders the period as day blocks. The numbers come from
// aggregate, the same function the report uses, so the two cannot drift apart:
// a frame counts towards the day it starts in, tags are counted individually,
// and a running timer counts up to now.
func buildSummaryRows(frames []watson.Frame, p period, weekStart time.Weekday,
	filter string, state *watson.State, now time.Time) []row {

	frames = filterFrames(withRunning(frames, state, now), filter)
	days := summaryDays(frames, p, weekStart)

	var rows []row
	for _, day := range days {
		if len(rows) > 0 {
			rows = append(rows, row{kind: rowBlank})
		}
		dayPeriod := period{unit: unitDay, ref: day}
		lines, total := aggregate(frames, dayPeriod, weekStart)
		rows = append(rows, row{kind: rowDayHeader, title: formatDay(day), total: total})
		for i, l := range lines {
			if i > 0 {
				rows = append(rows, row{kind: rowBlank})
			}
			rows = append(rows, row{kind: rowProject, title: l.project, total: l.total})
			for _, tl := range l.tags {
				rows = append(rows, row{kind: rowTag, title: tl.tag, total: tl.d})
			}
		}
	}
	return rows
}

// summaryDays lists the days the view shows: every day of a bounded period,
// including the empty ones, so a gap is visible. unitAll has no bounds, so it
// lists only the days that carry frames — otherwise every day since the first
// frame would get a row.
func summaryDays(frames []watson.Frame, p period, weekStart time.Weekday) []time.Time {
	from, to, bounded := p.bounds(weekStart)
	if bounded {
		var days []time.Time
		for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
			days = append(days, d)
		}
		return days
	}
	seen := map[time.Time]bool{}
	var days []time.Time
	for _, f := range frames {
		y, m, d := f.Start.Local().Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		if !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })
	return days
}
```

Der Import `slices` muss dazu. Falls `slices.SortFunc` in dieser Go-Version anders heißt, `sort.Slice` mit `days[i].Before(days[j])` nutzen und das im Bericht vermerken.

- [ ] **Step 4: Tests laufen grün**

Run: `go test ./internal/tui/ -run TestSummary -v 2>&1 | tail -10`
Expected: PASS (6 Tests)

- [ ] **Step 5: Zähne prüfen**

Drei Mutationen, einzeln, mit Ausgabe im Bericht:
1. In `summaryDays` den `bounded`-Zweig die leeren Tage überspringen lassen → `TestSummaryShowsEmptyDays` rot.
2. Die Tageszeile aus der Summe der Tag-Zeilen statt aus `aggregate`s `total` bauen → `TestSummaryDayTotalCountsFramesNotTags` rot.
3. `withRunning` weglassen → `TestSummaryCountsTheRunningTimer` rot.

- [ ] **Step 6: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: aggregate the period into day blocks"
```

---

### Task 3: Darstellung, Umschalten und Hinweise

**Files:**
- Modify: `internal/tui/frameslist.go` (`listModel.compact`, `refresh`, `renderRow`)
- Modify: `internal/tui/app.go` (`f` in `updateList`, `panelTitle`)
- Modify: `internal/tui/chrome.go` (`footerHints` für die Liste)
- Test: `internal/tui/summary_test.go`, `app_chrome_test.go`, `chrome_test.go`
- Regenerate: `internal/tui/testdata/*.golden`, plus eine neue Golden für die kompakte Ansicht

**Interfaces:**
- Consumes: `buildSummaryRows` (Task 2), `rowKind` (Task 1)
- Produces: `listModel.compact bool` (Startwert `true`); `footerHints` nimmt den Zustand entgegen — Signatur wird `footerHints(m mode, compact bool) [][]string`

- [ ] **Step 1: Failing Tests schreiben**

```go
// TestSummaryIsTheStartingView: the compact view is what one sees first.
func TestSummaryIsTheStartingView(t *testing.T) {
	app := newTestApp(t)
	if !app.list.compact {
		t.Error("watson-tui opens on the summary")
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if out := app.View(); !strings.Contains(out, "Summary") {
		t.Errorf("panel title must say Summary:\n%s", out)
	}
}

// TestFTogglesTheView: f switches, and the footer names the other side.
func TestFTogglesTheView(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour, "code"),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)

	if out := app.View(); !strings.Contains(out, "f frames") {
		t.Errorf("the summary offers the frame list:\n%s", out)
	}
	app.Update(key("f"))
	if app.list.compact {
		t.Fatal("f must switch to the frame list")
	}
	out := app.View()
	if !strings.Contains(out, "f summary") {
		t.Errorf("the frame list offers the summary:\n%s", out)
	}
	if !strings.Contains(out, "a111111") {
		t.Errorf("the frame list shows frame IDs:\n%s", out)
	}
	app.Update(key("f"))
	if !app.list.compact {
		t.Error("f must switch back")
	}
}

// TestCursorSkipsEverythingButProjects: j/k walk the project rows.
func TestCursorSkipsEverythingButProjects(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now, 2*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(3*time.Hour), time.Hour, "call"),
	}
	app.list.per = period{unit: unitDay, ref: now}
	app.list.refresh(app.frames, time.Monday)

	if app.list.cursor < 0 {
		t.Fatal("the cursor must find a project row")
	}
	if k := app.list.rows[app.list.cursor].kind; k != rowProject {
		t.Fatalf("cursor starts on kind %d, want a project row", k)
	}
	app.Update(key("j"))
	if k := app.list.rows[app.list.cursor].kind; k != rowProject {
		t.Errorf("j landed on kind %d, want a project row", k)
	}
}

// TestEditingKeysAreInertInTheSummary: enter and d need a frame.
func TestEditingKeysAreInertInTheSummary(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)

	app.Update(key("enter"))
	if app.mode != modeList {
		t.Error("enter must do nothing in the summary")
	}
	app.Update(key("d"))
	if app.mode != modeList {
		t.Error("d must do nothing in the summary")
	}
	if out := app.View(); strings.Contains(out, "enter edit") || strings.Contains(out, "d delete") {
		t.Errorf("the summary must not advertise keys that do nothing:\n%s", out)
	}
}

// TestSummaryNumbersSurviveEveryWidth: the rule the whole project follows —
// a name gives way, a duration never does.
func TestSummaryNumbersSurviveEveryWidth(t *testing.T) {
	now := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "a-rather-long-project-name", now, 124*time.Hour+30*time.Minute, "a-long-tag-name"),
	}
	app.list.per = period{unit: unitDay, ref: now}
	for width := 30; width <= 120; width++ {
		app.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		app.list.refresh(app.frames, time.Monday)
		out := app.View()
		if !strings.Contains(out, "124h 30m") {
			t.Errorf("width %d: the duration was cut:\n%s", width, out)
		}
		for i, line := range strings.Split(out, "\n") {
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d wide", width, i, w)
			}
		}
	}
}
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestSummaryIs|TestFToggles|TestCursorSkips|TestEditingKeys|TestSummaryNumbers' -v 2>&1 | tail -10`
Expected: FAIL — `app.list.compact` undefined

- [ ] **Step 3: `listModel` erweitern**

In `frameslist.go` das Feld ergänzen und in `newListModel` auf `true` setzen:

```go
	compact bool // the summary is the starting view; f switches to the frames
```

`refresh` wählt den Zeilenbauer. Die Signatur muss dafür `state` und `now` mitbekommen — sie wird zu `refresh(frames []watson.Frame, weekStart time.Weekday, state *watson.State, now time.Time)`. Alle Aufrufstellen in `app.go` reichen `a.state` und `a.now` durch:

```go
func (l *listModel) refresh(frames []watson.Frame, weekStart time.Weekday,
	state *watson.State, now time.Time) {
	prevID := ""
	if f, ok := l.selected(); ok {
		prevID = f.ID
	}
	if l.compact {
		l.rows = buildSummaryRows(frames, l.per, weekStart, l.filter, state, now)
	} else {
		l.rows = buildRows(frames, l.per, weekStart, l.filter)
	}
	l.cursor = firstFrameRow(l.rows)
	if prevID != "" {
		for i, r := range l.rows {
			if r.kind == rowFrame && r.frame.ID == prevID {
				l.cursor = i
				break
			}
		}
	}
}
```

- [ ] **Step 4: `renderRow` um die neuen Arten erweitern**

Die Breitenverhandlung folgt derselben Regel wie überall: die Zahl steht rechts und bleibt vollständig, der Name weicht mit `…`.

```go
	switch r.kind {
	case rowBlank:
		return ""
	case rowDayHeader:
		total := "–"
		if r.total > 0 {
			total = formatDuration(r.total)
		}
		return styleDayHeader.Render(summaryLine(r.title+" —", total, 0, width))
	case rowProject:
		line := summaryLine(r.title+" —", formatDuration(r.total), 2, width)
		if i == l.cursor {
			return styleSelected.Render(selectionMarker + line[1:])
		}
		return line
	case rowTag:
		return styleDim.Render(summaryLine("["+r.title, formatDuration(r.total)+"]", 6, width))
	}
```

`summaryLine` gehört zu `summary.go`:

```go
// summaryLine lays out "<indent><label> … <value>": the value right-aligned and
// always whole, the label truncated when the two do not fit. Same rule as the
// list, the report and the overview — a name that ends in … reads as cut, a
// number that lost its last digits does not.
func summaryLine(label, value string, indent, width int) string {
	avail := width - indent - lipgloss.Width(value) - 1
	if avail < 1 {
		avail = 1
	}
	label = truncate(label, avail)
	pad := width - indent - lipgloss.Width(label) - lipgloss.Width(value)
	if pad < 1 {
		pad = 1
	}
	return strings.Repeat(" ", indent) + label + strings.Repeat(" ", pad) + value
}
```

`renderRow` bekommt dafür die Breite. Sie steht in `listModel` noch nicht — ergänze ein Feld `width int`, das `view` vor dem Rendern aus seinem Parameter setzt, oder reiche die Breite als zweiten Parameter durch. Wähle das eine, und halte es konsistent; notiere die Wahl im Bericht.

- [ ] **Step 5: `f`, Panel-Titel und Hinweise**

In `app.go`, `updateList`:

```go
	case "f":
		a.list.compact = !a.list.compact
		a.list.refresh(a.frames, a.cfg.WeekStart, a.state, a.now)
```

`panelTitle` braucht den Zustand — Signatur `panelTitle(m mode, compact bool) string`, im `modeList`-Zweig `"Summary"` bei `compact`, sonst `"Frames"`.

In `chrome.go` wird `footerHints(m mode, compact bool) [][]string`. Der Listen-Zweig:

```go
	default:
		second := []string{"f frames", "n new", "? help", "q quit", "s timer",
			"/ filter", "r report", "o overview", "R reload"}
		if !compact {
			second = []string{"f summary", "enter edit", "n new", "? help", "q quit",
				"d delete", "s timer", "/ filter", "r report", "o overview", "R reload"}
		}
		return [][]string{
			{"j/k move", "← → period", "t/w/m/a day/week/month/all"},
			second,
		}
```

`enter` und `d` in `updateList` prüfen bereits über `a.list.selected()`, ob ein Frame da ist — in der kompakten Ansicht liefert das nichts, also sind sie von selbst wirkungslos. Prüfe das nach, statt es anzunehmen.

- [ ] **Step 6: Tests laufen grün**

Run: `go test ./... 2>&1 | tail -5`
Expected: PASS. Bestehende Tests, die von der Frameliste als Startbild ausgehen, müssen `app.list.compact = false` setzen — das ist eine Anpassung an geändertes Verhalten, keine Abschwächung. Notiere im Bericht, welche das waren.

- [ ] **Step 7: Golden-Dateien**

```bash
go test ./internal/tui/ -run TestGolden -update
```

Die bestehenden Golden zeigen jetzt die kompakte Ansicht; ergänze eine dritte für die Frameliste (`app.list.compact = false`), damit beide Darstellungen gepinnt sind. **Alle drei lesen** und beurteilen: Einrückung stimmig, Zahlen rechtsbündig, leere Tage mit `–`, Cursor auf einer Projektzeile, keine Zeile ragt heraus.

- [ ] **Step 8: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: open on the summary and toggle to the frames with f"
```

---

### Task 4: README und manuelle Abnahme

**Files:**
- Modify: `README.md`

- [ ] **Step 1: README ergänzen**

In der Tastentabelle nach `| \`/\` | filter (project, tag, ID) |` einfügen:

```markdown
| `f` | toggle summary / frame list |
```

Nach dem Absatz, der mit „Start `watson-tui`" beginnt, den Satz anpassen — die Anwendung öffnet jetzt die Zusammenfassung:

```markdown
Start `watson-tui` — it opens on the current week, summarised per day.
```

Und einen Absatz vor „The header shows which period" einfügen:

```markdown
The starting view sums each day of the period per project, and each project
by tag — the same numbers `watson aggregate` would give you. Days without
tracked time are listed too, with a dash, so a gap in the week is visible.
Press `f` for the individual frames, which is what you need to edit or delete
one, and `f` again to come back.
```

- [ ] **Step 2: Manuelle Abnahme**

```bash
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Durchgehen: die Anwendung öffnet die Zusammenfassung. `n` legt zwei Frames an verschiedenen Tagen mit je zwei Tags an — die Tage erscheinen mit Projekt- und Tag-Zeilen, leere Tage der Woche mit `–`. `j`/`k` springt über Projektzeilen. `f` zeigt die Frames mit IDs, `f` zurück. `←`/`→` blättert. `/` filtert, die Zusammenfassung folgt. `s` startet den Timer, der Tag zählt ihn mit. `?` nennt `f`. Notiere im Bericht, was du gesehen hast.

- [ ] **Step 3: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add README.md && git commit -m "docs: describe the summary view"
```

---

## Self-Review

**Spec-Abdeckung:** Zeilenarten → Task 1. Aggregation, leere Tage, `unitAll`, Filter, laufender Timer, Tageszeile aus Frames statt Tags → Task 2. Layout und Einrückung, Umschalten mit `f`, Cursor über Projektzeilen, wirkungslose Editiertasten, Panel-Titel, Hinweise, Zahlen bleiben vollständig → Task 3. README → Task 4. Golden für beide Darstellungen → Task 3, Step 7. Kopfzeile bleibt unverändert → keine Task fasst sie an, der Frame-Zähler wird in Task 1 auf `rowFrame` umgestellt und zählt damit weiter Frames. Non-Goals verletzt keine Task.

**Platzhalter:** keine. Task 3, Step 4 lässt die Wahl, wie die Breite zu `renderRow` kommt — mit der Auflage, eine zu wählen und sie zu notieren; das ist eine benannte Entscheidung, keine offene Stelle. Task 2, Step 3 nennt eine Rückfallvariante für `slices.SortFunc`.

**Typkonsistenz:** `rowKind` mit den fünf Werten in Task 1 definiert, in Tasks 2 und 3 genutzt. `row{kind, title, total, frame}` durchgehend. `buildSummaryRows(frames, p, weekStart, filter, state, now) []row` in Task 2 definiert, in Task 3 so aufgerufen. `refresh` wechselt in Task 3 auf vier Parameter — alle Aufrufstellen liegen in `app.go` und werden dort mitgezogen. `footerHints(m mode, compact bool)` und `panelTitle(m mode, compact bool)` in Task 3 geändert, beide nur aus `app.go` gerufen. `summaryLine(label, value string, indent, width int) string` in Task 3 definiert und nur dort genutzt.
