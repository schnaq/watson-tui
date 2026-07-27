# Kontext-Header und entdeckbare Hinweise — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Der Zeitraum ist als verschiebbar erkennbar, der Kopf zeigt Summe und Vergleich zum Vorzeitraum, und die Tastenhinweise nennen `[ ]` und `t/w/m/a`, die bisher nirgends im laufenden Bild standen.

**Architecture:** Die Vergleichs- und Summenlogik kommt in eine neue Datei `internal/tui/periodsums.go`, damit `app.go` nicht weiter wächst. `chrome.go` bekommt gestufte Kollaps-Schwellen und einen Footer, der Hinweisgruppen zeilenweise rendert. `headerFields` in `app.go` setzt die neuen Zeilen aus diesen Bausteinen zusammen.

**Tech Stack:** Go, charmbracelet/lipgloss — keine neuen Dependencies.

**Spec:** `docs/superpowers/specs/2026-07-26-context-and-discoverability-design.md`

**Ausgangspunkt:** Branch `feature/tui-chrome` bei `2f99628`.

## Global Constraints

- Nur ANSI-Farben 0–15; keine neuen Dependencies
- UI-Texte deutsch, Code/Bezeichner englisch
- Kein neues Tastenkürzel, kein Verhaltenswechsel — nur Darstellung
- Zuordnungsregel unverändert: ein Frame zählt vollständig in den Zeitraum, in dem er **beginnt**
- Die Kopfsumme der Liste zählt den laufenden Timer **nicht** mit und weist ihn mit ` + läuft` aus; Report und Übersicht zählen ihn weiter mit
- Bestehende Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in Liste, Report und Übersicht
- TDD: Test zuerst, RED verifizieren, dann implementieren; neue Tests durch Mutation der Implementierung auf Zähne prüfen
- Gate vor jedem Commit: `gofmt -l .` als eigener Schritt (der Befehl endet mit 0, auch wenn er Dateien listet), dann `go build ./... && go vet ./... && go test ./... && golangci-lint run ./...`
- `commit.gpgsign=true` ist gesetzt, gpg kann in dieser Umgebung nicht nach der Passphrase fragen: hängt das Signieren, mit `--no-gpg-sign` committen und im Bericht vermerken
- Commit-Trailer: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` plus die `Claude-Session:`-Zeile der Session

## Vorhandene Bausteine

Aus dem Chrome-Redesign, unverändert nutzbar:

- `headerField{label, value string}`, `renderField(f headerField) string`, `spread(left, right string, width int) string`, `clipWidth(s string, max int) string`, `panel(title, body string, width int, border lipgloss.Style) string`
- `renderHeader(width, height int, version string, rows [][]headerField) string`
- `renderFooter(width int, hints []string, errMsg string) string`, `footerHints(m mode) []string`, `const hintSep = " · "`
- `chromeHeight(height int) int` mit `headerFullMinHeight = 20`, `headerLineMinHeight = 12`
- `period` mit `bounds(weekStart) (from, to time.Time, ok bool)`, `shift(weekStart, delta) period`, `label(weekStart) string`; Einheiten `unitDay`, `unitWeek`, `unitMonth`, `unitAll`
- `formatDuration(d time.Duration) string`, `styleAccent`, `styleDim`, `styleRunning`, `styleHeaderLabel`
- Test-Helfer `mkFrame(id, project string, start time.Time, dur time.Duration, tags ...string) watson.Frame`, `newTestApp(t)`, `key(s string) tea.KeyMsg`

---

### Task 1: Summen und Vergleichszeiträume

**Files:**
- Create: `internal/tui/periodsums.go`
- Test: `internal/tui/periodsums_test.go`

**Interfaces:**
- Consumes: `period`, `unitDay`/`unitWeek`/`unitMonth`/`unitAll`, `watson.Frame`
- Produces:
  - `func sumInPeriod(frames []watson.Frame, p period, weekStart time.Weekday) time.Duration`
  - `type comparison struct { label string; per period }`
  - `func comparisonPeriods(p period) []comparison` — zwei Einträge für Tag/Woche/Monat, leer für `unitAll`

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/periodsums_test.go`)**

```go
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
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestSumInPeriod|TestComparisonPeriods' -v`
Expected: FAIL — `undefined: sumInPeriod`

- [ ] **Step 3: `internal/tui/periodsums.go` schreiben**

Für den Jahresvergleich fehlt eine Einheit — `period` kennt nur Tag, Woche, Monat, alles. Statt eine `unitYear` einzuführen (die dann `label`, `shift` und `bounds` mitziehen müsste), trägt `comparison` den fertigen Zeitraum als zwölf Monate: der Vergleich braucht nur `bounds`, nicht Navigation. Deshalb ein eigener kleiner Typ statt einer neuen Einheit.

```go
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

// yearPeriod spans the calendar year around ref. period has no year unit —
// a comparison only needs bounds, never navigation, so a month period shifted
// to January with a twelve-month span would need a new unit for no gain. The
// month unit's bounds cover one month, so the year is expressed as its own
// period with unitMonth on January plus an explicit span; see yearBounds.
func comparisonPeriods(p period) []comparison {
	switch p.unit {
	case unitDay:
		return []comparison{
			{"Vortag", p.shift(time.Monday, -1)},
			{"Woche", period{unit: unitWeek, ref: p.ref}},
		}
	case unitWeek:
		return []comparison{
			{"Vorwoche", p.shift(time.Monday, -1)},
			{"Monat", period{unit: unitMonth, ref: p.ref}},
		}
	case unitMonth:
		return []comparison{
			{"Vormonat", p.shift(time.Monday, -1)},
			{"Jahr", period{unit: unitYear, ref: p.ref}},
		}
	default: // unitAll — shifting does nothing, so a comparison says nothing
		return nil
	}
}
```

Der Code oben verweist auf `unitYear`, das es noch nicht gibt. Zwei Wege stehen offen; nimm den ersten, er ist der kleinere Eingriff:

**Weg A (empfohlen): `unitYear` als vierte Einheit in `frameslist.go` ergänzen.** `period.bounds` bekommt einen Fall, `shift` einen, `label` einen. Die Einheit ist über keine Taste erreichbar — `setPeriodUnit` wird nicht erweitert —, sie existiert nur für den Vergleich. Konkret in `internal/tui/frameslist.go`:

```go
// im const-Block der periodUnit-Werte, nach unitMonth:
	unitYear
```

```go
// in bounds(), vor dem default-Zweig:
	case unitYear:
		start := time.Date(y, 1, 1, 0, 0, 0, 0, p.ref.Location())
		return start, start.AddDate(1, 0, 0), true
```

```go
// in shift(), im switch nach case unitWeek:
	case unitYear:
		p.ref = from.AddDate(delta, 0, 0)
```

```go
// in label(), vor dem default-Zweig:
	case unitYear:
		return fmt.Sprintf("Jahr %d", from.Year())
```

**Weg B:** `comparison` trägt statt `period` ein Paar `from, to time.Time`. Dann braucht `sumInPeriod` eine zweite Variante für rohe Grenzen, und der Kopf kann den Vergleich nicht mehr über `period.label` benennen. Mehr Code an mehr Stellen — nur nehmen, wenn Weg A an etwas scheitert, und dann im Bericht begründen.

Nach Weg A ist der `yearPeriod`-Kommentar im Code oben falsch (es gibt jetzt eine Einheit). Ersetze ihn beim Implementieren durch:

```go
// comparisonPeriods returns the two neighbours the header shows for p: the
// period before it, and the larger period containing it. unitYear exists only
// for this comparison — no key selects it.
```

- [ ] **Step 4: Tests laufen grün**

Run: `go test ./internal/tui/ -run 'TestSumInPeriod|TestComparisonPeriods' -v`
Expected: PASS (4 Tests)

- [ ] **Step 5: Zähne prüfen**

Mutiere `sumInPeriod`, sodass es die Grenzen mit `f.Stop` statt `f.Start` vergleicht, und bestätige, dass `TestSumInPeriodFollowsTheStartRule` rot wird. Stelle zurück. Notiere die Ausgabe im Bericht.

- [ ] **Step 6: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: sum periods and name their neighbours for the header"
```

---

### Task 2: Gestufter Kollaps und zweizeiliger Fuß

**Files:**
- Modify: `internal/tui/chrome.go` (`chromeHeight`, `renderHeader`, `renderFooter`, `footerHints`)
- Test: `internal/tui/chrome_test.go` (Tests ergänzen und die bestehenden Schwellen-Tests anpassen)

**Interfaces:**
- Consumes: `panel`, `spread`, `renderField`, `clipWidth`, `headerField`, `mode`
- Produces:
  - `func chromeHeight(height int) int` — 8 / 6 / 2 / 1 nach den neuen Stufen
  - `func headerRowBudget(height int) int` — 4 / 3 / 1 / 0: wie viele Feldzeilen der Kopf zeigt
  - `func renderFooter(width int, groups [][]string, errMsg string) string` — eine Zeile pro Gruppe; bei einer erlaubten Zeile werden die Gruppen zu einer Liste verkettet und von hinten gekürzt
  - `func footerHints(m mode) [][]string`
  - `func footerLines(height int) int` — 2 ab Höhe 24, sonst 1
  - `styleKey` (in `styles.go`) für die abgesetzte Taste

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/chrome_test.go` ergänzen)**

```go
// TestChromeHeightStages: the chrome sheds in four steps so the body keeps room.
func TestChromeHeightStages(t *testing.T) {
	cases := map[int]int{40: 8, 24: 8, 23: 6, 20: 6, 19: 2, 14: 2, 13: 1, 5: 1}
	for height, want := range cases {
		if got := chromeHeight(height); got != want {
			t.Errorf("chromeHeight(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestHeaderRowBudgetStages: the comparison row is the first thing to go.
func TestHeaderRowBudgetStages(t *testing.T) {
	cases := map[int]int{40: 4, 24: 4, 23: 3, 20: 3, 19: 1, 14: 1, 13: 0, 5: 0}
	for height, want := range cases {
		if got := headerRowBudget(height); got != want {
			t.Errorf("headerRowBudget(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestFooterRendersOneLinePerGroup: two groups, two lines.
func TestFooterRendersOneLinePerGroup(t *testing.T) {
	groups := [][]string{{"j/k bewegen", "[ ] Zeitraum"}, {"? hilfe", "q ende"}}
	out := renderFooter(100, groups, "")
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "[ ] Zeitraum") || !strings.Contains(lines[1], "q ende") {
		t.Errorf("groups landed on the wrong lines: %q", out)
	}
}

// TestFooterMergesGroupsWhenOnlyOneLineFits: the period keys and the two exits
// survive; that is the whole point of the change.
func TestFooterMergesGroupsWhenOnlyOneLineFits(t *testing.T) {
	groups := [][]string{{"j/k bewegen", "[ ] Zeitraum"}, {"? hilfe", "q ende", "n neu", "d löschen"}}
	out := renderFooter(60, groups[:1], "") // caller passes what fits
	if strings.Contains(out, "\n") {
		t.Errorf("single group must be one line: %q", out)
	}
	merged := renderFooter(60, [][]string{append(append([]string{}, groups[0]...), groups[1]...)}, "")
	for _, want := range []string{"[ ] Zeitraum", "? hilfe", "q ende"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged footer at 60 lost %q: %q", want, merged)
		}
	}
}

// TestFooterErrorStillReplacesEverything.
func TestFooterErrorStillReplacesEverything(t *testing.T) {
	groups := [][]string{{"j/k bewegen"}, {"q ende"}}
	out := renderFooter(80, groups, "Speichern fehlgeschlagen")
	if !strings.Contains(out, "Speichern fehlgeschlagen") {
		t.Errorf("error missing: %q", out)
	}
	if strings.Contains(out, "j/k") || strings.Contains(out, "q ende") {
		t.Errorf("error must replace the hints: %q", out)
	}
	if strings.Contains(out, "\n") {
		t.Errorf("error is one line: %q", out)
	}
}

// TestFooterHintsListPeriodKeys: the gap that started this change.
func TestFooterHintsListPeriodKeys(t *testing.T) {
	groups := footerHints(modeList)
	if len(groups) != 2 {
		t.Fatalf("list hints must come in two groups, got %d", len(groups))
	}
	flat := strings.Join(append(append([]string{}, groups[0]...), groups[1]...), " ")
	for _, want := range []string{"[ ]", "t/w/m/a", "? hilfe", "q ende"} {
		if !strings.Contains(flat, want) {
			t.Errorf("list hints missing %q: %q", want, flat)
		}
	}
	// Group 1 is navigation and period, group 2 actions and views.
	if !strings.Contains(strings.Join(groups[0], " "), "[ ]") {
		t.Errorf("period keys belong in the first group: %q", groups[0])
	}
}

// TestFooterKeepsPeriodAndExitsAt80: shedding must not eat the keys a user
// cannot otherwise find.
func TestFooterKeepsPeriodAndExitsAt80(t *testing.T) {
	groups := footerHints(modeList)
	first := renderFooter(80, groups[:1], "")
	if !strings.Contains(first, "[ ]") {
		t.Errorf("period hint gone at 80 columns: %q", first)
	}
	second := renderFooter(80, groups[1:], "")
	for _, want := range []string{"? hilfe", "q ende"} {
		if !strings.Contains(second, want) {
			t.Errorf("exit hint %q gone at 80 columns: %q", want, second)
		}
	}
}
```

Die bestehenden `TestChromeHeightCollapses` und `TestHeaderShowsFieldsFramed` prüfen die alten Schwellen (20/19/12/11) und vier Kopfzeilen. Passe sie auf die neuen Werte an — `TestChromeHeightCollapses` wird durch `TestChromeHeightStages` ersetzt, `TestHeaderShowsFieldsFramed` bekommt Höhe 24 statt 30 und erwartet 6 statt 4 Zeilen bei vier Feldzeilen. Alle anderen `renderFooter`-Aufrufe in den Tests bekommen `[][]string{{...}}` statt `[]string{...}`.

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestChromeHeightStages|TestHeaderRowBudget|TestFooter' -v`
Expected: FAIL — `undefined: headerRowBudget`, und die `renderFooter`-Aufrufe passen nicht auf `[]string`

- [ ] **Step 3: `chrome.go` anpassen**

Schwellen ersetzen:

```go
// Collapse thresholds. The chrome grew to eight lines with the context header
// and the two hint rows, so it sheds in four steps rather than two: first the
// comparison row and the second hint line, then the frame around the header,
// then the header itself.
const (
	chromeFullMinHeight = 24 // framed header, 4 field rows, 2 hint lines
	chromeSlimMinHeight = 20 // framed header, 3 field rows, 1 hint line
	headerLineMinHeight = 14 // single unframed header line, 1 hint line
)

// chromeHeight reports how many lines the chrome occupies.
func chromeHeight(height int) int {
	switch {
	case height >= chromeFullMinHeight:
		return 8
	case height >= chromeSlimMinHeight:
		return 6
	case height >= headerLineMinHeight:
		return 2
	default:
		return 1
	}
}

// headerRowBudget reports how many field rows the header shows.
func headerRowBudget(height int) int {
	switch {
	case height >= chromeFullMinHeight:
		return 4
	case height >= chromeSlimMinHeight:
		return 3
	case height >= headerLineMinHeight:
		return 1
	default:
		return 0
	}
}

// footerLines reports how many hint lines fit.
func footerLines(height int) int {
	if height >= chromeFullMinHeight {
		return 2
	}
	return 1
}
```

`renderHeader` schneidet die Zeilen auf das Budget zu. Ersetze im gerahmten Zweig die Zeilenschleife so, dass sie nach `headerRowBudget(height)` Zeilen aufhört — und weil die Vergleichszeile die erste ist, die weichen soll, entfernt der Aufrufer sie (siehe Task 3), nicht `renderHeader`. `renderHeader` kürzt nur hart, damit das Budget in jedem Fall hält:

```go
	case height >= chromeSlimMinHeight:
		budget := headerRowBudget(height)
		if len(rows) > budget {
			rows = rows[:budget]
		}
		var lines []string
		for _, r := range rows {
			var left, right string
			if len(r) > 0 {
				left = renderField(r[0])
			}
			if len(r) > 1 {
				right = renderField(r[1])
			}
			lines = append(lines, spread(left, right, max(width-4, 1)))
		}
		return panel("watson-tui "+version, strings.Join(lines, "\n"), width, styleBorder)
```

Den bisherigen `case height >= headerFullMinHeight` durch den obigen ersetzen und `headerFullMinHeight` löschen; der einzeilige Zweig prüft weiter `height >= headerLineMinHeight`.

`renderFooter` auf Gruppen umstellen:

```go
// renderFooter draws the key hints, one line per group. An error replaces them
// all: a failed write outranks a reminder of which key moves the cursor.
func renderFooter(width int, groups [][]string, errMsg string) string {
	if errMsg != "" {
		return " " + styleError.Render(clipWidth(errMsg, max(width-2, 1)))
	}
	var lines []string
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		lines = append(lines, " "+shedHints(g, max(width-2, 1)))
	}
	return strings.Join(lines, "\n")
}
```

`shedHints` ist die bisherige Kürzungslogik aus `renderFooter` — ziehe sie in eine eigene Funktion, damit `renderFooter` nur noch über Gruppen schleift, und rendere darin jede Taste abgesetzt:

```go
// shedHints joins as many hints as fit, dropping whole ones from the tail so a
// hint is never cut mid-word. The key itself is set off from its description.
func shedHints(hints []string, avail int) string {
	var kept []string
	used := 0
	for _, h := range hints {
		w := lipgloss.Width(h)
		if len(kept) > 0 {
			w += lipgloss.Width(hintSep)
		}
		if used+w > avail {
			break
		}
		used += w
		kept = append(kept, styleHint(h))
	}
	return strings.Join(kept, hintSep)
}

// styleHint colours the key part of "j/k bewegen" and dims the rest.
func styleHint(h string) string {
	if i := strings.IndexByte(h, ' '); i > 0 {
		return styleKey.Render(h[:i]) + styleDim.Render(h[i:])
	}
	return styleKey.Render(h)
}
```

Wichtig: `shedHints` misst `lipgloss.Width(h)` auf dem **unstilierten** Hinweis und stiliert erst danach — sonst zählen unter einem Farbprofil die Escape-Sequenzen mit. Genau diese Falle hat `panel` schon einmal getroffen.

`footerHints` liefert Gruppen:

```go
func footerHints(m mode) [][]string {
	switch m {
	case modeForm:
		return [][]string{{"tab Feld", "→ Vorschlag", "enter speichern", "esc abbrechen"}}
	case modeReport:
		return [][]string{{"t/w/m Zeitraum", "[ ] verschieben", "esc zurück"}}
	case modeOverview:
		return [][]string{{"esc zurück"}}
	case modeStartTimer:
		return [][]string{{"tab Feld", "→ Vorschlag", "enter starten", "esc abbrechen"}}
	case modeConfirmDelete:
		return [][]string{{"y löschen", "andere Taste abbrechen"}}
	case modeConfirmCancel:
		return [][]string{{"y verwerfen", "andere Taste abbrechen"}}
	case modeHelp:
		return [][]string{{"beliebige Taste schließt die Hilfe"}}
	case modeFatal:
		return [][]string{{"beliebige Taste beendet watson-tui"}}
	default:
		return [][]string{
			{"j/k bewegen", "[ ] Zeitraum", "t/w/m/a Tag/Woche/Monat/alles"},
			{"enter bearbeiten", "n neu", "? hilfe", "q ende", "d löschen", "s timer",
				"/ filtern", "r report", "o übersicht", "R neu laden"},
		}
	}
}
```

`? hilfe` und `q ende` stehen bewusst an dritter und vierter Stelle der zweiten Gruppe: bei 80 Spalten überleben etwa sechs Hinweise, und diese beiden kann man ohne sie nicht mehr nachschlagen.

- [ ] **Step 4: `styleKey` in `styles.go` ergänzen**

```go
	styleKey = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
```

- [ ] **Step 5: Tests laufen grün**

Run: `go test ./internal/tui/ -v 2>&1 | tail -25`
Expected: PASS. Der Aufruf in `app.go` passt noch nicht auf die neue `renderFooter`-Signatur — das behebt Task 3. Bis dahin schlägt `go build` fehl; halte diesen Task also erst nach Task 3 für abgeschlossen, oder passe den Aufruf hier minimal an (`renderFooter(a.width, footerHints(a.mode), a.errMsg)` funktioniert unverändert, weil `footerHints` jetzt Gruppen liefert).

- [ ] **Step 6: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: shed the chrome in four steps and group the key hints"
```

---

### Task 3: Kopf mit Zeitraum, Summe und Vergleich

**Files:**
- Modify: `internal/tui/app.go` (`headerFields`, `View`)
- Test: `internal/tui/app_chrome_test.go` (Tests ergänzen)

**Interfaces:**
- Consumes: `sumInPeriod`, `comparisonPeriods`, `comparison` (Task 1); `headerRowBudget`, `footerLines`, `renderFooter`, `footerHints` (Task 2)
- Produces:
  - `func (a *App) periodField(label string, p period) headerField` — Wert als `‹ label ›`, ohne Winkel bei `unitAll`
  - `func (a *App) sumField(p period) headerField` — `Summe` plus ` + läuft`, wenn ein laufender Timer im Zeitraum beginnt
  - `func (a *App) comparisonRow(p period) []headerField` — eine Zeile mit den Vergleichswerten, leer bei `unitAll`
  - `headerFields()` liefert bis zu vier Zeilen; `View()` schneidet die Vergleichszeile nach `headerRowBudget` weg und rendert den Fuß mit `footerLines` Gruppen

- [ ] **Step 1: Failing Tests schreiben**

```go
// TestPeriodFieldShowsItIsShiftable: the brackets are the affordance for [ and ].
func TestPeriodFieldShowsItIsShiftable(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	for _, u := range []periodUnit{unitDay, unitWeek, unitMonth} {
		f := app.periodField("Zeitraum", period{unit: u, ref: now})
		if !strings.Contains(f.value, "‹") || !strings.Contains(f.value, "›") {
			t.Errorf("unit %d must show the shift affordance: %q", u, f.value)
		}
	}
	f := app.periodField("Zeitraum", period{unit: unitAll, ref: now})
	if strings.Contains(f.value, "‹") {
		t.Errorf("unitAll cannot be shifted, so no affordance: %q", f.value)
	}
}

// TestSumFieldMatchesTheListAndFlagsTheTimer: the header sum must add up to the
// day totals below it, and say so when a running timer is not in it.
func TestSumFieldMatchesTheListAndFlagsTheTimer(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p", now.Add(-3*time.Hour), 2*time.Hour),
		mkFrame("b2222222222222222222222222222222", "p", now.Add(-30*time.Hour), 30*time.Minute),
	}
	p := period{unit: unitAll, ref: now}

	f := app.sumField(p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("sum = %q, want 2h 30m", f.value)
	}
	if strings.Contains(f.value, "läuft") {
		t.Errorf("no timer runs, so no flag: %q", f.value)
	}

	app.state = &watson.State{Project: "läuft", Start: now.Add(-time.Hour), Tags: []string{}}
	f = app.sumField(p)
	if !strings.Contains(f.value, "2h 30m") {
		t.Errorf("running timer must not change the sum: %q", f.value)
	}
	if !strings.Contains(f.value, "läuft") {
		t.Errorf("running timer must be flagged: %q", f.value)
	}
}

// TestComparisonRowNamesNeighbours.
func TestComparisonRowNamesNeighbours(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour), // Vorwoche
	}
	row := app.comparisonRow(period{unit: unitWeek, ref: now})
	flat := ""
	for _, f := range row {
		flat += f.label + " " + f.value + " "
	}
	if !strings.Contains(flat, "Vorwoche") || !strings.Contains(flat, "5h 00m") {
		t.Errorf("comparison row = %q", flat)
	}
	if got := app.comparisonRow(period{unit: unitAll, ref: now}); len(got) != 0 {
		t.Errorf("unitAll has no comparison row, got %+v", got)
	}
}

// TestComparisonRowShowsDashForZero.
func TestComparisonRowShowsDashForZero(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	row := app.comparisonRow(period{unit: unitWeek, ref: now})
	for _, f := range row {
		if strings.Contains(f.value, "0m") {
			t.Errorf("zero must read as a dash, got %q", f.value)
		}
	}
	if len(row) == 0 || !strings.Contains(row[0].value, "–") {
		t.Errorf("comparison row = %+v", row)
	}
}

// TestViewDropsComparisonRowBeforeTheFrame: at 20-23 lines the comparison goes,
// the framed header stays.
func TestViewDropsComparisonRowBeforeTheFrame(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	app := newTestApp(t)
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "p",
			time.Date(2026, 7, 14, 9, 0, 0, 0, time.Local), 5*time.Hour),
	}
	app.list.per = period{unit: unitWeek, ref: now}
	app.list.refresh(app.frames, time.Monday)

	app.Update(tea.WindowSizeMsg{Width: 100, Height: 26})
	if !strings.Contains(app.View(), "Vorwoche") {
		t.Error("at 26 lines the comparison row belongs in the header")
	}
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 21})
	out := app.View()
	if strings.Contains(out, "Vorwoche") {
		t.Errorf("at 21 lines the comparison row must give way:\n%s", out)
	}
	if !strings.Contains(out, "╭") {
		t.Errorf("but the frame stays at 21 lines:\n%s", out)
	}
}

// TestViewShowsPeriodKeysInTheFooter: the discoverability fix, end to end.
func TestViewShowsPeriodKeysInTheFooter(t *testing.T) {
	app := newTestApp(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 100, Height: 30}, {Width: 80, Height: 24}, {Width: 80, Height: 20}} {
		app.Update(size)
		out := app.View()
		if !strings.Contains(out, "[ ]") {
			t.Errorf("%dx%d: the period keys must be visible:\n%s", size.Width, size.Height, out)
		}
		if !strings.Contains(out, "q ende") {
			t.Errorf("%dx%d: the quit key must be visible:\n%s", size.Width, size.Height, out)
		}
	}
}
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestPeriodField|TestSumField|TestComparisonRow|TestViewDrops|TestViewShowsPeriod' -v`
Expected: FAIL — `undefined: (*App).periodField`

- [ ] **Step 3: Helfer in `app.go` ergänzen**

```go
// periodField renders the period as "‹ label ›" so the brackets show that [ and
// ] move it. unitAll cannot be shifted, so it gets no brackets.
func (a *App) periodField(label string, p period) headerField {
	text := p.label(a.cfg.WeekStart)
	if p.unit == unitAll {
		return headerField{label: label, value: text}
	}
	return headerField{
		label: label,
		value: styleDim.Render("‹ ") + text + styleDim.Render(" ›"),
	}
}

// sumField renders the sum of the frames in p — the ones the list shows, so the
// header adds up to the day totals below it. A running timer is deliberately
// not counted (it is not a frame yet) but is flagged, because the report and the
// overview do count it and the numbers would otherwise disagree without reason.
func (a *App) sumField(p period) headerField {
	value := formatDuration(sumInPeriod(a.frames, p, a.cfg.WeekStart))
	if runningInPeriod(a.state, p, a.cfg.WeekStart) {
		value += styleRunning.Render(" + läuft")
	}
	return headerField{label: "Summe", value: value}
}

// comparisonRow renders the neighbouring periods of p. Zero reads as a dash:
// "0m" invites the question whether it means nothing or no data.
func (a *App) comparisonRow(p period) []headerField {
	cs := comparisonPeriods(p)
	if len(cs) == 0 {
		return nil
	}
	fields := make([]headerField, 0, len(cs))
	for _, c := range cs {
		d := sumInPeriod(a.frames, c.per, a.cfg.WeekStart)
		value := "–"
		if d > 0 {
			value = formatDuration(d)
		}
		fields = append(fields, headerField{label: c.label, value: value})
	}
	return fields
}
```

`runningInPeriod` existiert schon in `report.go` und passt genau: es prüft, ob der Start des laufenden Timers in den Zeitraum fällt.

- [ ] **Step 4: `headerFields` ersetzen**

Die Vergleichszeile ist Zeile 2, damit `View()` sie als erste wegschneiden kann. Für `modeList`:

```go
	case modeList:
		n := 0
		for _, r := range a.list.rows {
			if !r.isHeader {
				n++
			}
		}
		projects := map[string]bool{}
		for _, r := range a.list.rows {
			if !r.isHeader {
				projects[r.frame.Project] = true
			}
		}
		filter := "—"
		if a.list.filtering {
			filter = a.list.filterInput.View()
		} else if a.list.filter != "" {
			filter = a.list.filter
		}
		return [][]headerField{
			{a.periodField("Zeitraum", a.list.per), a.sumField(a.list.per)},
			a.comparisonRow(a.list.per),
			{{"Filter", filter}, {"", fmt.Sprintf("%d Frames · %d Projekte", n, len(projects))}},
			{{}, timer},
		}
```

Für `modeReport` analog, mit `a.report.per` und `Report` als Label; die dritte Zeile trägt nur die Projektzahl:

```go
	case modeReport:
		lines, _ := aggregate(withRunning(a.frames, a.state, a.now), a.report.per, a.cfg.WeekStart)
		return [][]headerField{
			{a.periodField("Report", a.report.per), a.sumField(a.report.per)},
			a.comparisonRow(a.report.per),
			{{}, {"", fmt.Sprintf("%d Projekte", len(lines))}},
			{{}, timer},
		}
```

Für `modeOverview` bleiben zwei Zeilen plus Timer — feste Spalten, also kein `‹ ›` und kein Vergleich. `Summe gesamt` ist die Gesamtspalte:

```go
	case modeOverview:
		cols := overviewColumns(a.now, a.cfg.WeekStart)
		rows, totals := buildOverview(withRunning(a.frames, a.state, a.now), cols, a.cfg.WeekStart)
		grand := time.Duration(0)
		if len(totals) > 0 {
			grand = totals[len(totals)-1]
		}
		return [][]headerField{
			{{"Übersicht", "Abrechnung"}, {"Summe gesamt", formatDuration(grand)}},
			{{}, {"", fmt.Sprintf("%d Projekte", len(rows))}},
			{{}, timer},
		}
```

Die übrigen Modi bleiben, wie sie sind, mit dem Timer in der letzten Zeile.

- [ ] **Step 5: `View()` anpassen**

Vor dem Rendern die Vergleichszeile nach Budget entfernen und den Fuß mit der erlaubten Gruppenzahl bauen:

```go
	rows := a.headerFields()
	// Leere Zeilen (etwa die Vergleichszeile bei unitAll) fallen weg, bevor das
	// Budget greift — sonst kostet eine unsichtbare Zeile eine sichtbare.
	compact := make([][]headerField, 0, len(rows))
	for _, r := range rows {
		if len(r) == 0 {
			continue
		}
		compact = append(compact, r)
	}
	if budget := headerRowBudget(a.height); len(compact) > budget && budget > 0 {
		// Die Vergleichszeile ist Zeile 2 und weicht zuerst.
		if budget >= 3 && len(compact) >= 2 {
			compact = append(compact[:1], compact[2:]...)
		}
		if len(compact) > budget {
			compact = compact[:budget]
		}
	}
	header := renderHeader(a.width, a.height, a.version, compact)

	groups := footerHints(a.mode)
	if n := footerLines(a.height); len(groups) > n {
		if n == 1 {
			// Gruppen verketten statt die zweite zu verlieren: sonst
			// verschwinden ? und q, die man nicht nachschlagen kann.
			merged := make([]string, 0)
			for _, g := range groups {
				merged = append(merged, g...)
			}
			groups = [][]string{merged}
		} else {
			groups = groups[:n]
		}
	}
	footer := renderFooter(a.width, groups, a.errMsg)
```

Achtung bei `append(compact[:1], compact[2:]...)`: das schreibt in das zugrunde liegende Array. Da `compact` eine frische Slice ist, ist das hier unkritisch — halte es so, aber nutze nicht `rows` direkt.

- [ ] **Step 6: Tests laufen grün**

Run: `go test ./internal/tui/ -v 2>&1 | tail -25`
Expected: PASS. Die Golden-Datei `internal/tui/testdata/list-100x30.golden` zeigt jetzt den neuen Kopf und Fuß — mit `go test ./internal/tui/ -run TestGoldenListView -update` neu erzeugen, **die Datei lesen und beurteilen**, und nur committen, wenn sie stimmt.

- [ ] **Step 7: Zähne prüfen**

Drei Mutationen, jede einzeln, mit Ausgabe im Bericht:
1. `periodField` gibt die Winkel auch bei `unitAll` aus → `TestPeriodFieldShowsItIsShiftable` rot.
2. `sumField` nutzt `withRunning(a.frames, a.state, a.now)` statt `a.frames` → `TestSumFieldMatchesTheListAndFlagsTheTimer` rot.
3. Im `View()`-Block schneidet das Budget von hinten (`compact = compact[:budget]`) statt die Vergleichszeile → `TestViewDropsComparisonRowBeforeTheFrame` rot.

- [ ] **Step 8: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: show period, sum and neighbours in the header"
```

---

### Task 4: Hilfe und README nachziehen

**Files:**
- Modify: `internal/tui/app.go` (`helpView`)
- Modify: `README.md`
- Test: `internal/tui/chrome_test.go` (Höhen-Test der Hilfe anpassen)

**Interfaces:**
- Consumes: alles aus Tasks 1–3

- [ ] **Step 1: Hilfe prüfen und anpassen**

`helpView` ist auf zwölf Zeilen ausgelegt, weil das Chrome früher fünf Zeilen brauchte. Jetzt sind es acht, also passt die Hilfe erst ab Höhe 24 vollständig (`24 - 8 = 16` Zeilen Rumpf, davon zwei für den Panelrahmen). Prüfe mit dem bestehenden Höhen-Test, ob die letzte Zeile (`q beenden`) bei 24 Zeilen noch da ist; wenn nicht, kürze die Hilfe auf zehn Zeilen, indem du `R neu laden` und `? diese Hilfe` in eine Zeile zusammenlegst:

```go
  R / ?        neu laden · diese Hilfe
```

Passe den Test an, der das Budget aus `chromeHeight` ableitet — er muss die neuen acht Zeilen berücksichtigen.

- [ ] **Step 2: Tests laufen grün**

Run: `go test ./internal/tui/ -v 2>&1 | tail -10`
Expected: PASS

- [ ] **Step 3: README ergänzen**

Im Abschnitt „Bedienung", nach der Tastentabelle, den Absatz über die Oberfläche um den Kopf erweitern:

```markdown
Der Kopf zeigt, welchen Zeitraum du betrachtest — die Winkel `‹ ›` bedeuten,
dass `[` und `]` ihn verschieben —, die Summe dieses Zeitraums, den Vergleich
zum Vorzeitraum und zur nächstgrößeren Einheit sowie einen laufenden Timer.
Die Summe im Kopf entspricht der Liste darunter und zählt einen laufenden
Timer deshalb nicht mit; steht `+ läuft` dahinter, ist genau das der Grund,
warum Report und Übersicht für denselben Zeitraum mehr anzeigen.
```

- [ ] **Step 4: Manuelle Abnahme**

```bash
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Durchgehen: `n` zwei Frames in verschiedenen Wochen anlegen, dann `[` — der Kopf muss den neuen Zeitraum und die neue Summe zeigen, die Vergleichszeile die Nachbarn. `s` starten: `+ läuft` erscheint hinter der Summe. Terminal auf 22 Zeilen verkleinern: Vergleichszeile weg, Rahmen bleibt. Auf 16: Kopf einzeilig, ein Hinweiszeile. Auf 12: Kopf weg.

- [ ] **Step 5: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/ README.md
git commit -m "docs: explain the context header and fit the help to it"
```

---

## Self-Review

**Spec-Abdeckung:** Zeitraum mit `‹ ›` → Task 3 (`periodField`). Vergleichszeile mit den Einheiten-Tabellen → Task 1 (`comparisonPeriods`) und Task 3 (`comparisonRow`). Summe ohne laufenden Timer mit `+ läuft` → Task 3 (`sumField`), Regel in den Global Constraints. Frame- und Projektzahl → Task 3. Felder pro Modus → Task 3, Steps 4. Zweizeiliger Fuß mit abgesetzten Tasten und ergänzten Zeitraum-Hinweisen → Task 2 (`footerHints`, `shedHints`, `styleKey`). Zusammenführung der Gruppen bei einer Zeile → Task 2 (Test) und Task 3 (`View`). Höhenbudget mit vier Stufen → Task 2 (`chromeHeight`, `headerRowBudget`, `footerLines`). Neue Datei `periodsums.go` → Task 1. `styleKey` → Task 2, Step 4. Tests laut Spec → Tasks 1–3. Non-Goals verletzt keine Task: keine neue Taste (`unitYear` ist über keine Taste erreichbar), kein verschiebbarer Zeitraum in der Übersicht, kein Theme-Switch, `t/w/m` setzen weiter auf heute zurück.

**Platzhalter:** keine. Task 1 lässt bewusst zwei Wege offen (`unitYear` als Einheit versus rohe Grenzen in `comparison`) — mit klarer Empfehlung, vollständigem Code für den empfohlenen Weg und der Auflage, eine Abweichung zu begründen. Das ist eine Entscheidung mit Begründungspflicht, keine offene Stelle.

**Typkonsistenz:** `comparison{label string; per period}` in Task 1 definiert, in Task 3 als `c.label`/`c.per` genutzt. `sumInPeriod(frames, p, weekStart)` überall gleich aufgerufen. `renderFooter(width int, groups [][]string, errMsg string)` in Task 2 definiert, in Task 3 so aufgerufen; `footerHints(m mode) [][]string` ebenso. `headerRowBudget`/`footerLines` in Task 2 definiert, in Task 3 genutzt. `periodField(label string, p period) headerField`, `sumField(p period) headerField`, `comparisonRow(p period) []headerField` in Task 3 definiert und dort verwendet. `runningInPeriod(state, p, weekStart)` ist vorhandener Code aus `report.go`, keine Neudefinition.
