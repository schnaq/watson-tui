# Englisch-Sweep — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Die gesamte Oberfläche spricht Englisch, Datumsangaben im ISO-Format, ohne dass sich Verhalten, Tastenbelegung oder eine Berechnung ändert.

**Architecture:** Die Übersetzung folgt der Datenflussrichtung: erst die Kalendernamen und Perioden-Labels in `frameslist.go`/`periodsums.go`, weil alle anderen Ansichten sie anzeigen, dann die Ansichten selbst, dann Kopf und Fuß, zuletzt der Datenlayer. So bleibt nach jedem Task eine übersetzte Schicht vollständig statt halb.

**Tech Stack:** Go — keine neuen Dependencies.

**Spec:** `docs/superpowers/specs/2026-07-27-english-ui-design.md` (enthält das verbindliche Glossar und die Tabelle der vollständigen Meldungen — **beim Übersetzen nachschlagen, nicht raten**)

**Ausgangspunkt:** `main` bei `c965a7c`.

## Global Constraints

- Nur die Sprache ändert sich: keine neue Taste, kein Layout, keine Berechnung, kein Datenformat
- Datumsanzeige ISO (`Monday, 2026-07-20`, `Week 2026-07-20 – 2026-07-26`, `July 2026`, `Year 2026`); Uhrzeiten und das Eingabeformat des Formulars bleiben
- Begriffe **ausschließlich** aus dem Glossar der Spec
- Tests werden **umgestellt, nicht gelöscht** — sie haben in diesem Projekt dreimal Zahlenfehler gefangen
- Bestehende Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in Liste, Report und Übersicht
- Gate vor jedem Commit: `gofmt -l .` als eigener Schritt (endet mit 0, auch wenn er Dateien listet), dann `go build ./... && go vet ./... && go test ./... && golangci-lint run ./...`
- `commit.gpgsign=true` und Signieren funktioniert — Commits signiert lassen
- Commit-Trailer: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` plus die `Claude-Session:`-Zeile der Session

## Verteilung

| Datei | deutsche Stellen |
|---|---|
| `internal/tui/chrome.go` | 30 (Tastenhinweise, Kopf-Kollaps) |
| `internal/tui/app.go` | 27 (Kopf-Felder, Dialoge, Hilfe, Fehlermeldungen) |
| `internal/tui/overview.go` | 11 (Spaltentitel, Drop-Hinweis, Timer-Notiz) |
| `internal/tui/frameslist.go` | 4 (Wochentage, Monate, Leermeldungen) |
| `internal/tui/report.go` | 3 |
| `internal/tui/periodsums.go` | 2 (Vergleichs-Labels) |
| `internal/tui/frameform.go` | 2 |
| `internal/tui/timer.go` | 1 |
| `internal/watson/store.go` | 2 (Sentinel-Fehler) |

14 Testdateien prüfen auf deutsche Zeichenketten.

---

### Task 1: Kalender, Perioden-Labels und Vergleiche

**Files:**
- Modify: `internal/tui/frameslist.go` (`germanDays`, `germanMonths`, `formatDay`, `period.label`, Leermeldungen in `listModel.view`)
- Modify: `internal/tui/periodsums.go` (`comparisonPeriods`-Labels)
- Test: `internal/tui/frameslist_test.go`, `frameslist_extra_test.go`, `periodsums_test.go`

**Interfaces:**
- Produces: `weekdayNames map[time.Weekday]string` und `monthNames [...]string` (ersetzen `germanDays`/`germanMonths`); `formatDay(t) string` → `Monday, 2026-07-20`; `period.label(weekStart) string` → `Week 2026-07-20 – 2026-07-26` / `July 2026` / `Year 2026` / `all frames`; Vergleichslabels `Prev day`/`Week`, `Prev week`/`Month`, `Prev month`/`Year`

- [ ] **Step 1: Tests auf die neuen Texte umstellen**

In `frameslist_test.go` und `frameslist_extra_test.go` alle Erwartungen mit `Montag`, `Woche `, `Juli`, `alle Frames`, `keine Frames`, `kein Treffer für Filter` auf die englischen Entsprechungen ändern. Die Assertions selbst bleiben, nur die Zeichenketten wechseln. In `periodsums_test.go` `Vorwoche`/`Monat`/`Vortag`/`Woche`/`Vormonat`/`Jahr` auf `Prev week`/`Month`/`Prev day`/`Week`/`Prev month`/`Year`.

Ein neuer Test pinnt das Format, damit es nicht zurückrutscht:

```go
// TestLabelsAreEnglishISO: the display format matches what the frame form
// accepts as input, so a date read off the screen can be typed straight back in.
func TestLabelsAreEnglishISO(t *testing.T) {
	// Wednesday 2026-07-22; the week (Mon) runs 20.–26.
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	cases := map[periodUnit]string{
		unitDay:   "2026-07-22",
		unitWeek:  "Week 2026-07-20 – 2026-07-26",
		unitMonth: "July 2026",
		unitYear:  "Year 2026",
		unitAll:   "all frames",
	}
	for unit, want := range cases {
		if got := (period{unit: unit, ref: ref}).label(time.Monday); got != want {
			t.Errorf("unit %d label = %q, want %q", unit, got, want)
		}
	}
	if got := formatDay(ref); got != "Wednesday, 2026-07-22" {
		t.Errorf("formatDay = %q, want %q", got, "Wednesday, 2026-07-22")
	}
}
```

Der Tages-Label-Fall (`unitDay`) nutzt `formatDay`; falls `label` dort heute etwas anderes liefert, den erwarteten Wert an die tatsächliche Implementierung anpassen — aber ISO muss es sein.

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestPeriod|TestLabels|TestBuildRows|TestComparison|TestList' -v 2>&1 | tail -20`
Expected: FAIL auf den deutschen Texten

- [ ] **Step 3: `frameslist.go` übersetzen**

```go
var weekdayNames = map[time.Weekday]string{
	time.Monday: "Monday", time.Tuesday: "Tuesday", time.Wednesday: "Wednesday",
	time.Thursday: "Thursday", time.Friday: "Friday", time.Saturday: "Saturday",
	time.Sunday: "Sunday",
}

var monthNames = [...]string{"", "January", "February", "March", "April", "May",
	"June", "July", "August", "September", "October", "November", "December"}

// formatDay renders a day header in ISO, so a date read off the screen can be
// typed straight into the frame form, which expects the same layout.
func formatDay(t time.Time) string {
	return fmt.Sprintf("%s, %s", weekdayNames[t.Weekday()], t.Format("2006-01-02"))
}
```

`period.label` entsprechend:

```go
	case unitDay:
		return formatDay(from)
	case unitWeek:
		last := to.AddDate(0, 0, -1)
		return fmt.Sprintf("Week %s – %s", from.Format("2006-01-02"), last.Format("2006-01-02"))
	case unitYear:
		return fmt.Sprintf("Year %d", from.Year())
	default: // unitMonth
		return fmt.Sprintf("%s %d", monthNames[int(from.Month())], from.Year())
```

und der unbegrenzte Fall `return "all frames"`.

Die Leermeldungen in `listModel.view`:

```go
		empty := "no frames in this period — press n to add one"
		if l.filter != "" {
			empty = "no match for filter “" + l.filter + "”"
		}
```

- [ ] **Step 4: `periodsums.go` übersetzen**

Die Labels in `comparisonPeriods`: `"Vortag"`→`"Prev day"`, `"Woche"`→`"Week"`, `"Vorwoche"`→`"Prev week"`, `"Monat"`→`"Month"`, `"Vormonat"`→`"Prev month"`, `"Jahr"`→`"Year"`.

- [ ] **Step 5: Tests laufen grün**

Run: `go test ./internal/tui/ 2>&1 | tail -5`
Expected: die Tests dieser Dateien grün; Tests anderer Dateien dürfen noch auf deutschen Texten scheitern — die kommen in Task 2 und 3. Notiere im Bericht, welche das sind.

- [ ] **Step 6: Gate und Commit**

Das Gate kann hier noch rot sein, weil andere Dateien folgen. Committe trotzdem, wenn `go build` und `go vet` sauber sind und nur Texterwartungen anderer Dateien scheitern:

```bash
gofmt -l .
go build ./... && go vet ./...
git add internal/tui/
git commit -m "refactor: say the calendar and the period labels in English"
```

---

### Task 2: Ansichten — Liste, Report, Übersicht, Formular, Timer

**Files:**
- Modify: `internal/tui/overview.go`, `report.go`, `frameform.go`, `timer.go`
- Test: `internal/tui/overview_test.go`, `overview_extra_test.go`, `overview_layout_test.go`, `report_test.go`, `report_extra_test.go`, `frameform_test.go`, `frameform_extra_test.go`

**Interfaces:**
- Consumes: die englischen Labels aus Task 1
- Produces: `overviewColumns` mit englischen Titeln und Kurzformen; die Spaltenreihenfolge und die Drop-Reihenfolge bleiben unverändert

- [ ] **Step 1: Tests umstellen**

Erwartete Zeichenketten in den sieben Testdateien austauschen: `Übersicht — Summen pro Projekt` (Titel entfällt ohnehin, der Kopf trägt ihn), `zu schmal für:`→`too narrow for:`, `Gesamt`→`Total`, `diese Woche`→`this week`, `letzte Woche`→`last week`, `dieser Monat`→`this month`, `letzter Monat`→`last month`, `gesamt`→`all`, `Vorwoche`→`prev wk`, `Vormonat`→`prev mo`, `keine Frames im Zeitraum`→`no frames in this period`, `keine Frames vorhanden`→`no frames yet`, `läuft (…) und ist eingerechnet`→`running (…), included`, `eingerechnet in:`→`counted in:`, Formularmeldungen laut Spec-Tabelle.

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestOverview|TestReport|TestFrameForm|TestBuildFrame|TestParseDateTime' -v 2>&1 | tail -20`
Expected: FAIL auf den deutschen Texten

- [ ] **Step 3: `overview.go` übersetzen**

Die Spaltendefinition — Titel und Kurzform, die Kurzform greift, wenn der Titel nicht passt:

```go
	return []overviewColumn{
		{"this week", "week", week},
		{"last week", "prev wk", week.shift(weekStart, -1)},
		{"this month", "month", month},
		{"last month", "prev mo", month.shift(weekStart, -1)},
		{"all", "all", period{unit: unitAll, ref: now}},
	}
```

Beachte: die Kurzformen sind kürzer als die Titel, sonst greift der Mechanismus nicht. `prev wk` (7) < `last week` (9) ✓, `prev mo` (7) < `last month` (10) ✓, `week` (4) < `this week` (9) ✓, `month` (5) < `this month` (10) ✓, `all` = `all` ✓ (unverändert, wie bisher bei `gesamt`).

Weiter: `"Projekt"`→`"Project"`, `"Gesamt"`→`"Total"`, `"zu schmal für: "`→`"too narrow for: "`, die Timer-Notiz `"▶ %s läuft (%s)"`→`"▶ %s running (%s)"` mit der Folgezeile `"  eingerechnet in: %s"`→`"  counted in: %s"`, und `"keine Frames vorhanden"`→`"no frames yet"`.

- [ ] **Step 4: `report.go`, `frameform.go`, `timer.go` übersetzen**

`report.go`: `"keine Frames im Zeitraum"`→`"no frames in this period"`, `"Gesamt"`→`"Total"`, die Timer-Notiz `"▶ %s läuft (%s) und ist eingerechnet"`→`"▶ %s running (%s), included"`.

`frameform.go`: Feldbeschriftungen `"Projekt"`→`"Project"` (Start/Stop/Tags bleiben), Platzhalter `"Projekt"`→`"Project"`, `"tag1, tag2"` bleibt, Panel-Titel `"Neuer Frame"`→`"New frame"` und `"Frame bearbeiten ("`→`"Edit frame ("`, Validierungsmeldungen `"Projekt fehlt"`→`"project is missing"`, `"Stop muss nach Start liegen"`→`"stop must be after start"`, `"ungültige Zeit %q (YYYY-MM-DD HH:MM oder HH:MM)"`→`"invalid time %q (YYYY-MM-DD HH:MM or HH:MM)"`, die Hinweiszeile im Formular entfällt bereits (der Fuß trägt sie).

`timer.go`: `"Timer starten"`→`"Start timer"`, Platzhalter `"Projekt"`→`"Project"` und `"tag1, tag2 (optional)"` bleibt.

- [ ] **Step 5: Tests laufen grün**

Run: `go test ./internal/tui/ 2>&1 | tail -5`
Expected: die Tests dieser Dateien grün; `app_test.go`, `app_chrome_test.go` und `chrome_test.go` dürfen noch scheitern — Task 3.

- [ ] **Step 6: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./...
git add internal/tui/
git commit -m "refactor: say the views in English"
```

---

### Task 3: Kopf, Fuß, Hilfe, Dialoge und der Datenlayer

**Files:**
- Modify: `internal/tui/chrome.go` (Tastenhinweise), `internal/tui/app.go` (Kopf-Felder, Dialoge, `helpView`, Fehlermeldungen), `internal/watson/store.go` (Sentinel-Fehler), `internal/watson/save.go` (Lock-Fehler)
- Test: `internal/tui/app_test.go`, `app_chrome_test.go`, `chrome_test.go`, `internal/watson/frames_test.go`
- Regenerate: `internal/tui/testdata/*.golden`

**Interfaces:**
- Consumes: alles aus Tasks 1 und 2
- Produces: die vollständig englische Oberfläche

- [ ] **Step 1: Tests umstellen**

Erwartete Zeichenketten in den vier Testdateien austauschen. Die wichtigsten: `Zeitraum`→`Period`, `Summe`→`Total`, `Frames`→`Frames` (unverändert), `Projekte`→`Projects`, `Filter`→`Filter` (unverändert), `kein Timer`→`no timer`, `+ läuft`→`+ running`, `← → Zeitraum`→`← → period`, `? hilfe`→`? help`, `q ende`→`q quit`, `Übersicht`→`Overview`, `Abrechnung`→`Billing`, `Hilfe`→`Help`, `Fehler`→`Error`, `Bestätigen`→`Confirm`, `beliebige Taste`→`any key`, `frames-Datei nicht lesbar`→`cannot read the frames file`, `Löschen fehlgeschlagen`→`delete failed`.

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ ./internal/watson/ 2>&1 | tail -10`
Expected: FAIL auf den deutschen Texten

- [ ] **Step 3: `chrome.go` übersetzen**

Die Hinweisgruppen — Reihenfolge unverändert, nur die Texte:

```go
	default:
		return [][]string{
			{"j/k move", "← → period", "t/w/m/a day/week/month/all"},
			{"enter edit", "n new", "? help", "q quit", "d delete", "s timer",
				"/ filter", "r report", "o overview", "R reload"},
		}
```

Die übrigen Modi laut Spec-Tabelle: `{"tab field", "→ suggestion", "enter save", "esc cancel"}`, `{"t/w/m period", "← → shift", "esc back"}`, `{"esc back"}`, `{"tab field", "→ suggestion", "enter start", "esc cancel"}`, `{"y delete", "any other key cancels"}`, `{"y discard", "any other key cancels"}`, `{"any key closes the help"}`, `{"any key quits watson-tui"}`.

Der Kommentar an `lead` nennt die ersten beiden Hinweise — auf `"j/k move" and "← → period"` anpassen.

Beachte den Deny-List-Mechanismus in `styleHint`: er verhindert, dass das erste Wort als Taste eingefärbt wird, wenn es keine ist. Die deutsche Liste enthält `andere` und `beliebige`; sie wird zu `any`.

- [ ] **Step 4: `app.go` übersetzen**

Kopf-Felder: `"Zeitraum"`→`"Period"`, `"Summe"`→`"Total"`, `"Filter"` bleibt, `"%d Frames · %d Projekte"`→`"%d frames · %d projects"`, `"%d Projekte"`→`"%d projects"`, `"Report"` bleibt, `"Übersicht"`/`"Abrechnung"`→`"Overview"`/`"Billing"`, `"Summe gesamt"`→`"Total, all"`, `"Frame"`/`"bearbeiten (…)"`/`"neu"`→`"Frame"`/`"editing (…)"`/`"new"`, `"Timer"`/`"starten"`→`"Timer"`/`"starting"`, `"Hilfe"`/`"Tastenbelegung"`→`"Help"`/`"key map"`, `"Fehler"`/`"watson-tui kann nicht weiterarbeiten"`→`"Error"`/`"watson-tui cannot continue"`.

Timer-Feld: `"kein Timer"`→`"no timer"`, `" + läuft"`→`" + running"`.

Panel-Titel (`panelTitle`): `"Bestätigen"`→`"Confirm"`, `"Hilfe"`→`"Help"`, `"Fehler"`→`"Error"`, die übrigen bleiben.

Dialoge: `"Frame löschen?"`→`"Delete frame?"`, `"Laufenden Timer verwerfen?"`→`"Discard the running timer?"`.

Fehlermeldungen laut Spec-Tabelle: `"frames-Datei nicht lesbar: %v\nBackup: %s/frames.bak"`→`"cannot read the frames file: %v\nbackup: %s/frames.bak"`, analog `state`, sowie `"Löschen fehlgeschlagen: "`→`"delete failed: "`, `"Stop fehlgeschlagen: "`→`"stop failed: "`, `"Verwerfen fehlgeschlagen: "`→`"discard failed: "`, `"Speichern fehlgeschlagen: "`→`"save failed: "`, `"Timer starten fehlgeschlagen: "`→`"starting the timer failed: "`.

`helpView` — zehn Zeilen, Beschreibung ab Spalte 15 (die Tastenspalte ist 13 Zeichen breit, davor zwei Leerzeichen):

```go
	return `  j/k, ↓/↑     move
  ← →, [ ]     previous/next period (‹ › in the header)
  t/w/m/a      day/week/month/all
  enter        edit frame
  n / d        new frame · delete frame
  s / S        start/stop timer · discard
  /            filter
  r / o        report · overview (billing)
  R / ?        reload · this help
  q            quit`
```

Prüfe die Ausrichtung nach: jede Beschreibung muss an derselben Spalte beginnen.

- [ ] **Step 5: `internal/watson` übersetzen**

`store.go`: `errors.New("es läuft bereits ein Timer")`→`errors.New("a timer is already running")`, `errors.New("kein Timer aktiv")`→`errors.New("no timer is running")`.

`save.go`: `errors.New("watson-Verzeichnis ist von anderem Prozess gesperrt")`→`errors.New("the watson directory is locked by another process")`.

Diese Texte erscheinen über `a.errMsg` in der Fußzeile, sind also Oberfläche.

- [ ] **Step 6: Golden-Dateien neu erzeugen und lesen**

```bash
go test ./internal/tui/ -run TestGolden -update
cat internal/tui/testdata/list-100x30.golden
cat internal/tui/testdata/list-week-100x30.golden
```

Beide Dateien **lesen**: Kopf englisch, ISO-Datum im Tagesheader und im Wochenlabel, Fußzeile englisch, nichts ragt über den Rahmen. Stimmt etwas nicht, ist das ein Fund im Code, nicht in der Datei.

- [ ] **Step 7: Breitenprüfung**

Das Wochenlabel ist drei Zeichen länger als vorher. Prüfe mit einem Wegwerf-Test, dass bei 80 Spalten weiterhin `← →`, `? help` und `q quit` in der Fußzeile stehen und keine Zeile zu breit ist:

```go
func TestZZWidth(t *testing.T) {
	app := newTestApp(t)
	for _, w := range []int{60, 80, 100} {
		app.Update(tea.WindowSizeMsg{Width: w, Height: 30})
		out := app.View()
		for _, want := range []string{"← →", "? help", "q quit"} {
			if w >= 80 && !strings.Contains(out, want) {
				t.Errorf("width %d lost %q:\n%s", w, want, out)
			}
		}
		for i, line := range strings.Split(out, "\n") {
			if lipgloss.Width(line) > w {
				t.Errorf("width %d: line %d too wide", w, i)
			}
		}
	}
}
```

Fällt einer der drei bei 80 Spalten weg, sortiere innerhalb der Hinweisgruppen um — kürze nicht. Danach den Wegwerf-Test löschen und, falls umsortiert wurde, den bestehenden `TestFooterKeepsPeriodAndExitsAt80` entsprechend anpassen.

- [ ] **Step 8: Test gegen deutsche Rückfälle**

Neu in `internal/tui/chrome_test.go`:

```go
// TestNoGermanLeftInTheInterface guards the sweep: the comments in this
// codebase are English, so a German word in a string literal is a leftover.
func TestNoGermanLeftInTheInterface(t *testing.T) {
	words := []string{"Zeitraum", "Summe", "Gesamt", "löschen", "läuft", "keine ",
		"kein ", "Fehler", "Datei", "Taste", "Woche", "Monat", "Projekte",
		"zurück", "beenden", "abbrechen", "speichern", "bearbeiten", "schmal"}
	for _, dir := range []string{".", "../watson"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(string(src), "\n") {
				code, _, _ := strings.Cut(line, "//")
				if !strings.Contains(code, `"`) {
					continue
				}
				for _, w := range words {
					if strings.Contains(code, w) {
						t.Errorf("%s/%s: German %q left in a string: %s",
							dir, name, w, strings.TrimSpace(line))
					}
				}
			}
		}
	}
}
```

Der `strings.Cut` am `//` lässt Kommentare in Ruhe; er schneidet auch an einem `//` innerhalb eines Strings, was hier keine Rolle spielt, weil kein UI-Text zwei Schrägstriche enthält. Prüfe die Zähne, indem du ein deutsches Wort testweise zurückschreibst und den Test rot siehst.

- [ ] **Step 9: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/ && git commit -m "refactor: say the chrome, the dialogs and the store errors in English"
```

---

### Task 4: README und manuelle Abnahme

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Den Hinweis entfernen**

Die Zeile `The interface itself is German.` unter dem Einleitungsabsatz streichen.

- [ ] **Step 2: Deutsche Beispiele im README prüfen**

Das README nennt an mehreren Stellen deutsche Oberflächentexte als Beispiele: `alle Frames`, `+ läuft`, `Vorwoche`, `Vormonat`, `gesamt`. Ersetze sie durch die englischen Entsprechungen (`all frames`, `+ running`, `prev wk`, `prev mo`, `all`), damit die Beschreibung die Oberfläche wieder trifft.

- [ ] **Step 3: Manuelle Abnahme**

```bash
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Durchgehen: `n` legt einen Frame an — Formular auf Englisch, Eingabeformat `2026-07-20 09:00`. `?` zeigt die Hilfe, alle Zeilen bündig. `r` und `o` zeigen englische Überschriften und ISO-Daten. `←`/`→` blättern. `s` startet den Timer, `+ running` erscheint hinter der Summe. `q` beendet. Notiere im Bericht, was du gesehen hast.

- [ ] **Step 4: Gate und Commit**

```bash
gofmt -l .
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add README.md && git commit -m "docs: drop the note that the interface is German"
```

---

## Self-Review

**Spec-Abdeckung:** Datum/Zeit-Tabelle → Task 1. Glossar → Tasks 1–3, jeweils mit Verweis auf die Spec. Vollständige Meldungen → Task 2 (Formular, Report, Übersicht) und Task 3 (Dialoge, Fehler, Store). Hilfe und Tastenhinweise → Task 3, Steps 3–4. Breitenfolge des längeren Wochenlabels → Task 3, Step 7. Tests umstellen statt löschen → Global Constraints und jeweils Step 1 der Tasks 1–3. Test gegen Rückfälle → Task 3, Step 8. Golden lesen → Task 3, Step 6. README → Task 4. Non-Goals verletzt keine Task.

**Platzhalter:** keine. Task 1, Step 1 lässt den `unitDay`-Erwartungswert an die tatsächliche Implementierung anpassen — mit der Auflage, dass es ISO sein muss; das ist eine benannte Prüfung, keine offene Stelle.

**Typkonsistenz:** `weekdayNames`/`monthNames` in Task 1 eingeführt und nur dort verwendet. `formatDay` und `period.label` behalten ihre Signaturen. `overviewColumn{title, short, per}` behält seine Felder, nur die Werte wechseln — die Kurzform muss kürzer als der Titel sein, sonst greift der Mechanismus nicht, was Task 2, Step 3 ausdrücklich nachrechnet. `footerHints(m mode) [][]string` unverändert. Keine Funktion wechselt Namen oder Signatur.
