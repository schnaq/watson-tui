# TUI-Chrome Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** watson-tui bekommt das visuelle Gerüst moderner Terminal-Tools — Header-Panel mit Kontext, gerahmte Panels, Key-Hint-Footer, farbiger Selektionsbalken.

**Architecture:** Ein neues `internal/tui/chrome.go` hält Header, Footer und Panel-Helper; `styles.go` wird zum benannten Theme. Die View-Funktionen der Modi bleiben unverändert und liefern nur ihren Rumpf — `App.View()` klammert Chrome darum und rechnet das Höhenbudget. `renderStatus`/`statusLeft` entfallen, ihre Aufgabe übernehmen Header und Footer.

**Tech Stack:** Go, charmbracelet/lipgloss (bereits vorhanden), keine neuen Dependencies.

**Spec:** `docs/superpowers/specs/2026-07-26-tui-chrome-design.md`

**Ausgangspunkt:** `main` bei v0.1.0. Gate nach jedem Task: `gofmt -l .` leer, `go build ./... && go vet ./... && go test ./... && golangci-lint run ./...` grün.

## Global Constraints

- Nur ANSI-Farben 0–15 (`lipgloss.Color("0")` … `("15")`) — keine Hex-Werte, kein `AdaptiveColor`
- Keine neuen Dependencies
- UI-Texte deutsch, Code/Bezeichner englisch
- Kein Verhalten, keine Tastenbelegung, keine Berechnung ändert sich — rein visuell
- Rahmen kosten 2 Spalten: gerahmte Views bekommen `width - 2`, die Übersicht die volle `width`
- Kollaps-Schwellen: ab 20 Zeilen voller Header (Chrome 5 Zeilen), 12–19 einzeilig (Chrome 2), unter 12 kein Header (Chrome 1)
- TDD: Test zuerst, RED verifizieren, dann implementieren
- Commit-Präfixe `feat:`/`refactor:`/`test:`, Trailer:
  `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` plus die `Claude-Session:`-Zeile der Session

---

### Task 1: Theme und Selektionsbalken

**Files:**
- Modify: `internal/tui/styles.go` (komplett ersetzen)
- Modify: `internal/tui/frameslist.go:278-294` (`renderRow`)
- Test: `internal/tui/styles_test.go` (neu)

**Interfaces:**
- Produces (Namen bleiben, Definitionen wechseln): `styleTitle`, `styleDayHeader`, `styleSelected`, `styleDim`, `styleRunning`, `styleError`
- Produces (neu, von Task 2/3 genutzt): `styleBorder`, `styleAccent`, `styleFocus`, `styleWarn`, `styleHeaderLabel`
- Produces: `selectionMarker = "▌"`

- [ ] **Step 1: Failing Test schreiben (`internal/tui/styles_test.go`)**

```go
package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestThemeUsesANSIOnly: the theme must stay inside ANSI 0-15 so it adapts to
// whatever palette the user's terminal defines.
func TestThemeUsesANSIOnly(t *testing.T) {
	styles := map[string]lipgloss.Style{
		"title": styleTitle, "dayHeader": styleDayHeader, "selected": styleSelected,
		"dim": styleDim, "running": styleRunning, "error": styleError,
		"border": styleBorder, "accent": styleAccent, "focus": styleFocus,
		"warn": styleWarn, "headerLabel": styleHeaderLabel,
	}
	for name, s := range styles {
		for what, c := range map[string]string{"fg": s.GetForeground().(lipgloss.TerminalColor).(lipgloss.Color).String(), "bg": string(mustColor(s.GetBackground()))} {
			if c == "" {
				continue
			}
			if len(c) > 2 || c[0] < '0' || c[0] > '9' {
				t.Errorf("%s %s = %q, want a plain ANSI index 0-15", name, what, c)
			}
		}
	}
}

func mustColor(c lipgloss.TerminalColor) lipgloss.Color {
	if col, ok := c.(lipgloss.Color); ok {
		return col
	}
	return lipgloss.Color("")
}

// TestSelectedRowCarriesMarker: the cursor row is marked by a bar in the first
// column, not by reverse video.
func TestSelectedRowCarriesMarker(t *testing.T) {
	now := time.Now()
	l := listModel{
		per:    period{unit: unitAll, ref: now},
		cursor: 1,
		rows: []row{
			{isHeader: true, title: "Montag"},
			{frame: watson.Frame{
				ID: "a1111111111111111111111111111111", Project: "alpha",
				Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Tags: []string{},
			}},
		},
	}
	selected := l.renderRow(1)
	if !strings.Contains(selected, selectionMarker) {
		t.Errorf("selected row lacks %q marker:\n%s", selectionMarker, selected)
	}
	l.cursor = -1
	if plain := l.renderRow(1); strings.Contains(plain, selectionMarker) {
		t.Errorf("unselected row must not carry the marker:\n%s", plain)
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -run 'TestTheme|TestSelectedRow' -v`
Expected: FAIL — `undefined: styleBorder`, `undefined: selectionMarker`

- [ ] **Step 3: `internal/tui/styles.go` ersetzen**

```go
// Package-level theme. Everything stays inside ANSI 0-15 so the UI adopts the
// palette of whatever terminal theme the user runs — the same choice k9s and
// lazygit make by default.
package tui

import "github.com/charmbracelet/lipgloss"

// ANSI colour roles. Named so a change of taste happens in one place.
const (
	colDim    = lipgloss.Color("8") // bright black: borders, secondary text
	colAccent = lipgloss.Color("6") // cyan: panel titles, day headers
	colFocus  = lipgloss.Color("4") // blue: focused border, selection
	colOK     = lipgloss.Color("2") // green: running timer
	colWarn   = lipgloss.Color("3") // yellow: warnings
	colErr    = lipgloss.Color("1") // red: errors
)

// selectionMarker sits in the first column of the cursor row. A bar plus a
// background reads as a selection at a glance; reverse video does not.
const selectionMarker = "▌"

var (
	styleTitle       = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleDayHeader   = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleSelected    = lipgloss.NewStyle().Bold(true).Foreground(colFocus)
	styleDim         = lipgloss.NewStyle().Foreground(colDim)
	styleRunning     = lipgloss.NewStyle().Bold(true).Foreground(colOK)
	styleError       = lipgloss.NewStyle().Bold(true).Foreground(colErr)
	styleBorder      = lipgloss.NewStyle().Foreground(colDim)
	styleAccent      = lipgloss.NewStyle().Foreground(colAccent)
	styleFocus       = lipgloss.NewStyle().Foreground(colFocus)
	styleWarn        = lipgloss.NewStyle().Foreground(colWarn)
	styleHeaderLabel = lipgloss.NewStyle().Foreground(colDim)
)
```

- [ ] **Step 4: `renderRow` anpassen**

In `internal/tui/frameslist.go`, `renderRow` ersetzen. Das Format bleibt, nur die ersten zwei Spalten tragen jetzt den Marker:

```go
func (l *listModel) renderRow(i int) string {
	r := l.rows[i]
	if r.isHeader {
		return styleDayHeader.Render(r.title)
	}
	fr := r.frame
	prefix := "  "
	if i == l.cursor {
		prefix = selectionMarker + " "
	}
	line := fmt.Sprintf("%s%s–%s  %7s  %-24s %-28s %s",
		prefix,
		fr.Start.Local().Format("15:04"),
		fr.Stop.Local().Format("15:04"),
		formatDuration(fr.Duration()),
		truncate(fr.Project, 24),
		truncate(strings.Join(fr.Tags, ", "), 28),
		fr.ShortID())
	if i == l.cursor {
		return styleSelected.Render(line)
	}
	return line
}
```

- [ ] **Step 5: Tests laufen grün**

Run: `go test ./internal/tui/ -v 2>&1 | tail -20`
Expected: PASS. Falls ein bestehender Test auf Reverse-Video oder die alten zwei Leerzeichen am Zeilenanfang prüft: den Test auf den Marker umstellen, nicht die Implementierung zurückdrehen.

- [ ] **Step 6: Gate und Commit**

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: give the TUI a named ANSI theme and a selection bar"
```

---

### Task 2: Panel und Footer

**Files:**
- Create: `internal/tui/chrome.go`
- Test: `internal/tui/chrome_test.go`

**Interfaces:**
- Consumes: `styleBorder`, `styleFocus`, `styleAccent`, `styleDim`, `styleError` (Task 1)
- Produces:
  - `func panel(title, body string, width int, focused bool) string` — gerahmtes Panel, Titel in der oberen Rahmenlinie; `width` ist die Gesamtbreite inklusive Rahmen
  - `func renderFooter(width int, hints, errMsg string) string` — Key-Hints gedämpft; `errMsg` ersetzt sie in Fehlerfarbe
  - `func footerHints(m mode) string` — Hints pro Modus

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/chrome_test.go`)**

```go
package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestPanelFramesBodyAndTitle: the panel draws a border, carries its title in
// the top line and never exceeds the width it was given.
func TestPanelFramesBodyAndTitle(t *testing.T) {
	out := panel("Frames", "erste Zeile\nzweite Zeile", 40, false)
	if !strings.Contains(out, "Frames") {
		t.Errorf("panel lost its title:\n%s", out)
	}
	for _, want := range []string{"erste Zeile", "zweite Zeile"} {
		if !strings.Contains(out, want) {
			t.Errorf("panel lost body line %q:\n%s", want, out)
		}
	}
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 40 {
			t.Errorf("line %d is %d wide, want <= 40: %q", i, w, line)
		}
	}
}

// TestPanelTruncatesLongTitle: a title wider than the frame must not push the
// border past the given width.
func TestPanelTruncatesLongTitle(t *testing.T) {
	out := panel(strings.Repeat("sehr langer titel ", 5), "body", 30, false)
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 30 {
			t.Errorf("line %d is %d wide, want <= 30: %q", i, w, line)
		}
	}
}

// TestFooterShowsHintsOrError: the error replaces the hints, it does not append.
func TestFooterShowsHintsOrError(t *testing.T) {
	hints := renderFooter(80, "j/k bewegen", "")
	if !strings.Contains(hints, "j/k bewegen") {
		t.Errorf("footer lost its hints: %q", hints)
	}
	failed := renderFooter(80, "j/k bewegen", "Speichern fehlgeschlagen")
	if !strings.Contains(failed, "Speichern fehlgeschlagen") {
		t.Errorf("footer lost the error: %q", failed)
	}
	if strings.Contains(failed, "j/k bewegen") {
		t.Errorf("error must replace the hints, got %q", failed)
	}
}

// TestFooterHintsPerMode: every mode names the keys that actually work there.
func TestFooterHintsPerMode(t *testing.T) {
	cases := map[mode][]string{
		modeList:          {"j/k", "enter", "n", "d", "s", "o", "r", "?"},
		modeForm:          {"tab", "enter", "esc"},
		modeReport:        {"t/w/m", "esc"},
		modeOverview:      {"esc"},
		modeStartTimer:    {"enter", "esc"},
		modeConfirmDelete: {"y", "abbrechen"},
		modeConfirmCancel: {"y", "abbrechen"},
		modeHelp:          {"Taste"},
		modeFatal:         {"beendet"},
	}
	for m, wants := range cases {
		got := footerHints(m)
		if got == "" {
			t.Errorf("mode %d has no hints", m)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Errorf("mode %d hints %q missing %q", m, got, want)
			}
		}
	}
}
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestPanel|TestFooter' -v`
Expected: FAIL — `undefined: panel`

- [ ] **Step 3: `internal/tui/chrome.go` schreiben**

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// panel frames body and writes title into the top border line, the way k9s
// labels its views. width is the total width including the border, so callers
// pass the terminal width and the content gets width-2.
func panel(title, body string, width int, focused bool) string {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	border := styleBorder
	if focused {
		border = styleFocus
	}

	// The title sits in the top line: "╭─ title ─...─╮". Two dashes and two
	// spaces frame it, plus the two corners.
	label := ""
	if title != "" {
		label = truncate(title, max(inner-4, 1))
	}
	top := "╭─"
	if label != "" {
		top += " " + styleAccent.Render(label) + " "
	}
	fill := inner - (lipgloss.Width(top) - 1) // -1: the corner is not content
	if fill < 0 {
		fill = 0
	}
	top = border.Render("╭─")
	if label != "" {
		top += " " + styleAccent.Render(label) + " "
	}
	used := 2
	if label != "" {
		used += lipgloss.Width(label) + 2
	}
	top += border.Render(strings.Repeat("─", max(inner-used, 0)) + "╮")

	var b strings.Builder
	b.WriteString(top + "\n")
	for _, line := range strings.Split(body, "\n") {
		line = truncate(line, inner-2)
		pad := inner - 2 - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(border.Render("│") + " " + line + strings.Repeat(" ", pad) + " " + border.Render("│") + "\n")
	}
	b.WriteString(border.Render("╰" + strings.Repeat("─", inner) + "╯"))
	return b.String()
}

// renderFooter draws the key hints. An error replaces them: a failed write is
// more urgent than a reminder of which key moves the cursor.
func renderFooter(width int, hints, errMsg string) string {
	if errMsg != "" {
		return " " + styleError.Render(truncate(errMsg, max(width-2, 1)))
	}
	return " " + styleDim.Render(truncate(hints, max(width-2, 1)))
}

// footerHints lists the keys that work in the given mode.
func footerHints(m mode) string {
	switch m {
	case modeForm:
		return "tab Feld · → Vorschlag · enter speichern · esc abbrechen"
	case modeReport:
		return "t/w/m Zeitraum · [ ] verschieben · esc zurück"
	case modeOverview:
		return "esc zurück"
	case modeStartTimer:
		return "tab Feld · → Vorschlag · enter starten · esc abbrechen"
	case modeConfirmDelete:
		return "y löschen · andere Taste abbrechen"
	case modeConfirmCancel:
		return "y verwerfen · andere Taste abbrechen"
	case modeHelp:
		return "beliebige Taste schließt die Hilfe"
	case modeFatal:
		return "beliebige Taste beendet watson-tui"
	default:
		return "j/k bewegen · enter bearbeiten · n neu · d löschen · " +
			"s timer · / filtern · r report · o übersicht · ? hilfe · q ende"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```

Hinweis zum obigen `panel`: der erste `top`-Aufbau ist Dopplung — beim Implementieren die erste `top := "╭─"`-Zuweisung samt `fill`-Berechnung weglassen und direkt mit der `border.Render("╭─")`-Variante beginnen. Der finale Code hat genau einen Aufbau der Titelzeile.

Falls Go ≥ 1.21 die eigene `max`-Funktion als Redeclaration ablehnt (Builtin), die lokale Definition streichen und das Builtin nutzen.

- [ ] **Step 4: Tests laufen grün**

Run: `go test ./internal/tui/ -run 'TestPanel|TestFooter' -v`
Expected: PASS (4 Tests)

- [ ] **Step 5: Gate und Commit**

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: add panel and footer chrome helpers"
```

---

### Task 3: Header mit Kontextfeldern

**Files:**
- Modify: `internal/tui/chrome.go` (Header ergänzen)
- Test: `internal/tui/chrome_test.go` (Tests ergänzen)

**Interfaces:**
- Consumes: `panel`, `max` (Task 2), `styleHeaderLabel`, `styleDim` (Task 1)
- Produces:
  - `type headerField struct { label, value string }`
  - `func renderHeader(width, height int, version string, rows [][]headerField) string` — leerer String, wenn kein Header ins Höhenbudget passt; ab 20 Zeilen gerahmt, darunter einzeilig
  - `func chromeHeight(height int) int` — 5 / 2 / 1 nach Kollaps-Schwellen

- [ ] **Step 1: Failing Tests ergänzen (`internal/tui/chrome_test.go`)**

```go
// TestChromeHeightCollapses: the chrome must not eat the list on short
// terminals, so it sheds the header in two steps.
func TestChromeHeightCollapses(t *testing.T) {
	cases := map[int]int{30: 5, 20: 5, 19: 2, 12: 2, 11: 1, 5: 1}
	for height, want := range cases {
		if got := chromeHeight(height); got != want {
			t.Errorf("chromeHeight(%d) = %d, want %d", height, got, want)
		}
	}
}

// TestHeaderShowsFieldsFramed: at full height the header is framed, carries the
// version and every field label and value.
func TestHeaderShowsFieldsFramed(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07.–26.07."}, {"Frames", "12"}},
		{{"Filter", "—"}, {"", "▶ schnaq 1:23:45"}},
	}
	out := renderHeader(100, 30, "0.1.0", rows)
	for _, want := range []string{"watson-tui", "0.1.0", "Zeitraum", "Woche 20.07.–26.07.", "Frames", "12", "Filter", "▶ schnaq 1:23:45"} {
		if !strings.Contains(out, want) {
			t.Errorf("framed header missing %q:\n%s", want, out)
		}
	}
	if lines := strings.Count(out, "\n") + 1; lines != 4 {
		t.Errorf("framed header has %d lines, want 4:\n%s", lines, out)
	}
}

// TestHeaderCollapsesToOneLine: between 12 and 19 lines the header keeps the
// values but drops the frame.
func TestHeaderCollapsesToOneLine(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", "Woche 20.07."}, {"Frames", "12"}},
		{{"Filter", "—"}, {"", "kein Timer"}},
	}
	out := renderHeader(100, 15, "0.1.0", rows)
	if lines := strings.Count(out, "\n") + 1; lines != 1 {
		t.Errorf("collapsed header has %d lines, want 1: %q", lines, out)
	}
	if !strings.Contains(out, "Woche 20.07.") || !strings.Contains(out, "kein Timer") {
		t.Errorf("collapsed header lost context: %q", out)
	}
	if strings.Contains(out, "╭") {
		t.Errorf("collapsed header must not draw a frame: %q", out)
	}
}

// TestHeaderVanishesOnTinyTerminals: below 12 lines every row belongs to the body.
func TestHeaderVanishesOnTinyTerminals(t *testing.T) {
	rows := [][]headerField{{{"Zeitraum", "Woche"}, {"Frames", "1"}}}
	if out := renderHeader(100, 10, "0.1.0", rows); out != "" {
		t.Errorf("header must be empty at height 10, got %q", out)
	}
}

// TestHeaderRespectsWidth: no rendered line may exceed the terminal width.
func TestHeaderRespectsWidth(t *testing.T) {
	rows := [][]headerField{
		{{"Zeitraum", strings.Repeat("lang ", 30)}, {"Frames", "999"}},
		{{"Filter", strings.Repeat("filter ", 20)}, {"", "▶ projekt 1:23:45"}},
	}
	for _, width := range []int{40, 80, 100} {
		for _, height := range []int{30, 15} {
			out := renderHeader(width, height, "0.1.0", rows)
			for i, line := range strings.Split(out, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d height %d line %d is %d wide: %q", width, height, i, w, line)
				}
			}
		}
	}
}
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestChromeHeight|TestHeader' -v`
Expected: FAIL — `undefined: chromeHeight`, `undefined: headerField`

- [ ] **Step 3: Header in `internal/tui/chrome.go` ergänzen**

```go
// headerField is one labelled value in the header panel.
type headerField struct {
	label string
	value string
}

// Collapse thresholds. The chrome must never grow so tall that the list it
// frames has no room left.
const (
	headerFullMinHeight = 20 // framed header (4 lines) + footer
	headerLineMinHeight = 12 // single-line header + footer
)

// chromeHeight reports how many lines the chrome occupies at this terminal
// height, so App.View can hand the rest to the body.
func chromeHeight(height int) int {
	switch {
	case height >= headerFullMinHeight:
		return 5
	case height >= headerLineMinHeight:
		return 2
	default:
		return 1
	}
}

// renderField formats "label value"; a field without a label is just its value.
func renderField(f headerField) string {
	if f.label == "" {
		return f.value
	}
	return styleHeaderLabel.Render(f.label) + "  " + f.value
}

// spread puts left at the start and right at the end of a line of the given
// width, with at least one space between them.
func spread(left, right string, width int) string {
	if right == "" {
		return truncate(left, width)
	}
	rightW := lipgloss.Width(right)
	if rightW >= width {
		return truncate(right, width)
	}
	left = truncate(left, max(width-rightW-1, 1))
	gap := width - lipgloss.Width(left) - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// renderHeader draws the context panel. Above headerFullMinHeight it is framed
// and shows every row; between the thresholds it collapses to a single
// unframed line carrying the first field of each row; below, it disappears.
func renderHeader(width, height int, version string, rows [][]headerField) string {
	switch {
	case height >= headerFullMinHeight:
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
		return panel("watson-tui "+version, strings.Join(lines, "\n"), width, false)

	case height >= headerLineMinHeight:
		var left, right string
		for _, r := range rows {
			for _, f := range r {
				if f.value == "" {
					continue
				}
				// The last value of the last row is the timer; it goes right.
				if left == "" {
					left = f.value
				} else {
					right = f.value
				}
			}
		}
		return " " + spread(styleDim.Render(left), right, max(width-2, 1))

	default:
		return ""
	}
}
```

Hinweis: der einzeilige Zweig sammelt bewusst nur den ersten und den letzten nichtleeren Wert — auf 12 bis 19 Zeilen ist Platz für Kontext plus Timer, nicht für vier Felder. Wenn beim Implementieren auffällt, dass `right` dabei von einem mittleren Feld überschrieben wird: die Schleife so umbauen, dass `right` erst nach dem Durchlauf aus dem letzten nichtleeren Wert gesetzt wird.

- [ ] **Step 4: Tests laufen grün**

Run: `go test ./internal/tui/ -run 'TestChromeHeight|TestHeader' -v`
Expected: PASS (5 Tests)

- [ ] **Step 5: Gate und Commit**

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: add the context header with collapse thresholds"
```

---

### Task 4: Chrome in App.View verdrahten

**Files:**
- Modify: `internal/tui/app.go` (`View()`, `statusLeft()` → `headerFields()`)
- Delete: `renderStatus` aus `internal/tui/statusbar.go:39-60`
- Modify: `internal/tui/statusbar_test.go` (Tests für `renderStatus` entfernen)
- Modify: `internal/tui/frameslist_extra_test.go:99-120` (`TestStatusLeft` auf `headerFields` umstellen)
- Modify: `internal/tui/overview_layout_test.go:125-140` (`statusLeft`-Aufrufe umstellen)
- Test: `internal/tui/app_chrome_test.go` (neu)

**Interfaces:**
- Consumes: `renderHeader`, `renderFooter`, `footerHints`, `chromeHeight`, `panel`, `headerField` (Tasks 2–3)
- Produces:
  - `func (a *App) headerFields() [][]headerField` — Felder nach Modus, wie in der Spec tabelliert
  - `func (a *App) timerField() headerField` — laufender Timer oder `kein Timer`
  - `App.View()` liefert Header + Body + Footer; `panelTitle(m mode) string` benennt das Body-Panel

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/app_chrome_test.go`)**

```go
package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestViewHasHeaderBodyFooter: the full frame carries all three parts.
func TestViewHasHeaderBodyFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	out := app.View()
	for _, want := range []string{"watson-tui", "Zeitraum", "j/k bewegen"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q:\n%s", want, out)
		}
	}
}

// TestViewNeverExceedsWidth: no line of any mode may be wider than the
// terminal — the whole point of the layout arithmetic.
func TestViewNeverExceedsWidth(t *testing.T) {
	now := time.Now()
	modes := []mode{
		modeList, modeForm, modeReport, modeOverview, modeStartTimer,
		modeConfirmDelete, modeConfirmCancel, modeHelp, modeFatal,
	}
	for _, width := range []int{60, 80, 100} {
		for _, height := range []int{30, 15, 10} {
			for _, m := range modes {
				app := newTestApp(t)
				app.Update(tea.WindowSizeMsg{Width: width, Height: height})
				app.frames = []watson.Frame{
					mkFrame("a1111111111111111111111111111111", "ein ziemlich langer projektname", now.Add(-2*time.Hour), time.Hour, "tag-eins", "tag-zwei"),
				}
				app.list.per = period{unit: unitAll, ref: now}
				app.list.refresh(app.frames, time.Monday)
				app.form = newFormModel(&app.frames[0], app.frames, now)
				app.start = newStartModel(app.frames)
				app.report = newReportModel(now)
				app.pendingDelete = app.frames[0]
				app.fatalMsg = "frames-Datei nicht lesbar"
				app.mode = m
				for i, line := range strings.Split(app.View(), "\n") {
					if w := lipgloss.Width(line); w > width {
						t.Errorf("mode %d at %dx%d: line %d is %d wide: %q", m, width, height, i, w, line)
					}
				}
			}
		}
	}
}

// TestViewBudgetsHeight: the rendered frame must fit the terminal height.
func TestViewBudgetsHeight(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	var frames []watson.Frame
	for i := 0; i < 40; i++ {
		frames = append(frames, mkFrame(
			strings.Repeat("a", 31)+string(rune('a'+i%26)), "projekt",
			now.Add(-time.Duration(i*30)*time.Hour), time.Hour))
	}
	app.frames = frames
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)
	for _, height := range []int{30, 20, 15, 10} {
		app.Update(tea.WindowSizeMsg{Width: 100, Height: height})
		if lines := strings.Count(app.View(), "\n") + 1; lines > height {
			t.Errorf("height %d: view has %d lines", height, lines)
		}
	}
}

// TestErrorGoesToFooter: a failed write shows up in the footer, replacing the hints.
func TestErrorGoesToFooter(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.errMsg = "Löschen fehlgeschlagen: kaputt"
	out := app.View()
	if !strings.Contains(out, "Löschen fehlgeschlagen: kaputt") {
		t.Errorf("error missing from view:\n%s", out)
	}
	if strings.Contains(out, "j/k bewegen") {
		t.Errorf("error must replace the hints:\n%s", out)
	}
}

// TestHeaderFieldsPerMode: each mode labels its own context.
func TestHeaderFieldsPerMode(t *testing.T) {
	now := time.Now()
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.state = &watson.State{Project: "laufend", Start: now.Add(-time.Minute), Tags: []string{}}

	app.mode = modeList
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Zeitraum") || !strings.Contains(got, "Frames") {
		t.Errorf("list header = %q", got)
	}
	app.mode = modeOverview
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Übersicht") {
		t.Errorf("overview header = %q", got)
	}
	app.mode = modeReport
	app.report = newReportModel(now)
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Report") {
		t.Errorf("report header = %q", got)
	}
	// The timer belongs to every mode.
	for _, m := range []mode{modeList, modeOverview, modeReport, modeForm} {
		app.mode = m
		if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "laufend") {
			t.Errorf("mode %d header lost the timer: %q", m, got)
		}
	}
}

func renderFieldsFlat(rows [][]headerField) string {
	var parts []string
	for _, r := range rows {
		for _, f := range r {
			parts = append(parts, f.label+" "+f.value)
		}
	}
	return strings.Join(parts, " | ")
}
```

- [ ] **Step 2: Tests laufen rot**

Run: `go test ./internal/tui/ -run 'TestView|TestError|TestHeaderFields' -v`
Expected: FAIL — `undefined: (*App).headerFields`

- [ ] **Step 3: `statusLeft` in `app.go` durch `headerFields` ersetzen**

`statusLeft()` (ab `internal/tui/app.go:356`) komplett ersetzen:

```go
// timerField renders the running timer for the header; every mode shows it.
func (a *App) timerField() headerField {
	if a.state == nil {
		return headerField{value: styleDim.Render("kein Timer")}
	}
	label := a.state.Project
	if len(a.state.Tags) > 0 {
		label += " [" + strings.Join(a.state.Tags, ", ") + "]"
	}
	return headerField{value: styleRunning.Render(
		fmt.Sprintf("▶ %s %s", label, formatClock(a.now.Sub(a.state.Start))))}
}

// headerFields returns the context rows of the header for the active mode.
func (a *App) headerFields() [][]headerField {
	timer := a.timerField()
	switch a.mode {
	case modeList:
		n := 0
		for _, r := range a.list.rows {
			if !r.isHeader {
				n++
			}
		}
		filter := "—"
		if a.list.filtering {
			filter = a.list.filterInput.View()
		} else if a.list.filter != "" {
			filter = a.list.filter
		}
		return [][]headerField{
			{{"Zeitraum", a.list.per.label(a.cfg.WeekStart)}, {"Frames", fmt.Sprintf("%d", n)}},
			{{"Filter", filter}, timer},
		}
	case modeReport:
		lines, _ := aggregate(withRunning(a.frames, a.state, a.now), a.report.per, a.cfg.WeekStart)
		return [][]headerField{
			{{"Report", a.report.per.label(a.cfg.WeekStart)}, {"Projekte", fmt.Sprintf("%d", len(lines))}},
			{{}, timer},
		}
	case modeOverview:
		rows, _ := buildOverview(withRunning(a.frames, a.state, a.now),
			overviewColumns(a.now, a.cfg.WeekStart), a.cfg.WeekStart)
		return [][]headerField{
			{{"Übersicht", "Abrechnung"}, {"Projekte", fmt.Sprintf("%d", len(rows))}},
			{{}, timer},
		}
	case modeForm:
		what := "neu"
		if a.form.editing {
			what = "bearbeiten (" + watson.Frame{ID: a.form.frameID}.ShortID() + ")"
		}
		return [][]headerField{{{"Frame", what}}, {{}, timer}}
	case modeStartTimer:
		return [][]headerField{{{"Timer", "starten"}}, {{}, timer}}
	case modeConfirmDelete:
		return [][]headerField{{{"Frame", "löschen"}}, {{}, timer}}
	case modeConfirmCancel:
		return [][]headerField{{{"Timer", "verwerfen"}}, {{}, timer}}
	case modeHelp:
		return [][]headerField{{{"Hilfe", "Tastenbelegung"}}, {{}, timer}}
	default: // modeFatal
		return [][]headerField{{{"Fehler", "watson-tui kann nicht weiterarbeiten"}}}
	}
}

// panelTitle names the body panel of the active mode.
func panelTitle(m mode) string {
	switch m {
	case modeForm:
		return "Frame"
	case modeReport:
		return "Report"
	case modeStartTimer:
		return "Timer"
	case modeConfirmDelete, modeConfirmCancel:
		return "Bestätigen"
	case modeHelp:
		return "Hilfe"
	case modeFatal:
		return "Fehler"
	default:
		return "Frames"
	}
}
```

- [ ] **Step 4: `View()` in `app.go` ersetzen**

```go
func (a *App) View() string {
	chrome := chromeHeight(a.height)
	bodyHeight := a.height - chrome
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	framed := true
	switch a.mode {
	case modeFatal:
		body = a.fatalMsg
	case modeHelp:
		body = helpView()
	case modeForm:
		body = a.form.view()
	case modeReport:
		body = a.report.view(a.frames, a.state, a.cfg.WeekStart, a.now)
	case modeOverview:
		// The billing table needs every column it can get, so it renders
		// without side borders.
		body = overviewView(a.frames, a.state, a.cfg.WeekStart, a.now, a.width)
		framed = false
	case modeStartTimer:
		body = a.start.view()
	case modeConfirmCancel:
		body = "Laufenden Timer verwerfen?"
	case modeConfirmDelete:
		f := a.pendingDelete
		body = fmt.Sprintf("Frame löschen?\n\n  %s  %s–%s  %s",
			f.Project,
			f.Start.Local().Format("2006-01-02 15:04"),
			f.Stop.Local().Format("15:04"),
			f.ShortID())
	default:
		// The list already scrolls to a height; the panel adds two border lines.
		body = a.list.view(bodyHeight - 2)
	}

	var parts []string
	if header := renderHeader(a.width, a.height, a.version, a.headerFields()); header != "" {
		parts = append(parts, header)
	}
	if framed {
		parts = append(parts, panel(panelTitle(a.mode), body, a.width, false))
	} else {
		parts = append(parts, body)
	}
	parts = append(parts, renderFooter(a.width, footerHints(a.mode), a.errMsg))
	return strings.Join(parts, "\n")
}
```

- [ ] **Step 5: `renderStatus` löschen und Tests umstellen**

`renderStatus` aus `internal/tui/statusbar.go` entfernen (Zeilen 39–60) samt der dann unbenutzten Imports `strings`, `lipgloss` und `watson` — `go vet` zeigt sie an. In `internal/tui/statusbar_test.go` die drei `renderStatus`-Tests (`TestRenderStatusRunning`, `TestRenderStatusIdleAndError` und was sonst darauf zugreift) löschen; `TestFormatDuration`/`TestFormatClock` bleiben. In `frameslist_extra_test.go` `TestStatusLeft` ersetzen:

```go
// TestHeaderFieldsListBranches covers the filter states of the list header.
func TestHeaderFieldsListBranches(t *testing.T) {
	app := newTestApp(t)
	app.mode = modeList
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Frames 0") {
		t.Errorf("plain header = %q", got)
	}
	app.list.filter = "foo"
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "foo") {
		t.Errorf("filter header = %q", got)
	}
	app.list.filtering = true
	app.list.filterInput.SetValue("bar")
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "bar") {
		t.Errorf("filtering header = %q", got)
	}
}
```

In `overview_layout_test.go` die beiden `statusLeft()`-Aufrufe auf `renderFieldsFlat(app.headerFields())` umstellen und die erwarteten Teilstrings beibehalten.

- [ ] **Step 6: Tests laufen grün**

Run: `go test ./internal/tui/ -v 2>&1 | tail -25`
Expected: PASS. `TestViewNeverExceedsWidth` und `TestViewBudgetsHeight` sind die scharfen Tests — wenn sie rot sind, stimmt die Layout-Arithmetik nicht; dann `bodyHeight`/`inner` prüfen, nicht die Tests aufweichen.

- [ ] **Step 7: Gate und Commit**

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/
git commit -m "feat: wire header, panel and footer into the app view"
```

---

### Task 5: Golden-Test und manuelle Abnahme

**Files:**
- Test: `internal/tui/golden_test.go` (neu)
- Create: `internal/tui/testdata/list-100x30.golden` (aus der Implementierung erzeugt)
- Modify: `README.md` (Screenshot-Abschnitt)

**Interfaces:**
- Consumes: alles aus Tasks 1–4
- Produces: `-update`-Flag zum Neuschreiben der Golden-Datei

- [ ] **Step 1: Golden-Test schreiben (`internal/tui/golden_test.go`)**

```go
package tui

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// TestGoldenListView pins the whole rendered frame. It exists to make layout
// regressions visible in a diff — run with -update after an intentional change
// and read the diff before committing it.
func TestGoldenListView(t *testing.T) {
	// A fixed instant keeps the frame deterministic.
	now := time.Date(2026, 7, 22, 15, 4, 5, 0, time.UTC)
	app := NewApp(watson.NewStore(t.TempDir()), "0.1.0")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.now = now
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "schnaq", now.Add(-30*time.Hour), 2*time.Hour+30*time.Minute, "dev"),
		mkFrame("b2222222222222222222222222222222", "kunde-a", now.Add(-26*time.Hour), 4*time.Hour, "meeting"),
		mkFrame("c3333333333333333333333333333333", "kunde-b", now.Add(-5*time.Hour), 3*time.Hour+15*time.Minute, "call"),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)
	got := app.View()

	path := filepath.Join("testdata", "list-100x30.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v — run: go test ./internal/tui/ -run TestGoldenListView -update", err)
	}
	if got != string(want) {
		t.Errorf("frame changed.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// TestGoldenIsStable: rendering twice must produce the same frame, otherwise
// the golden file would flap.
func TestGoldenIsStable(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.now = time.Date(2026, 7, 22, 15, 4, 5, 0, time.UTC)
	if first, second := app.View(), app.View(); first != second {
		t.Errorf("view is not deterministic:\n%s\n---\n%s", first, second)
	}
	if strings.TrimSpace(app.View()) == "" {
		t.Error("view is empty")
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -run TestGolden -v`
Expected: FAIL — `open testdata/list-100x30.golden: no such file or directory`

- [ ] **Step 3: Golden erzeugen und ansehen**

```bash
go test ./internal/tui/ -run TestGoldenListView -update
cat internal/tui/testdata/list-100x30.golden
```

Die Datei prüfen: Header gerahmt mit `watson-tui 0.1.0`, Kontextfelder gefüllt, Tagesheader mit Tagessumme, Selektionsbalken in der ersten Datenzeile, Footer mit Key-Hints. Passt etwas nicht, ist das ein Fund für den Code — nicht die Golden-Datei „passend" machen.

- [ ] **Step 4: Tests laufen grün**

Run: `go test ./internal/tui/ -run TestGolden -v`
Expected: PASS (2 Tests)

- [ ] **Step 5: Manuelle Abnahme im echten Terminal**

```bash
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Durchgehen: `n` legt einen Frame an (Formular im Panel), `enter` bearbeitet, `j`/`k` bewegen den Selektionsbalken, `s` startet den Timer (Header zeigt ihn tickend), `r` Report, `o` Übersicht (volle Breite, kein Seitenrahmen), `?` Hilfe, `q` beendet. Terminal auf ~15 Zeilen verkleinern: Header wird einzeilig. Auf ~10 Zeilen: Header weg, Liste bleibt nutzbar.

- [ ] **Step 6: README ergänzen**

Im Abschnitt „Bedienung" nach der Tastentabelle einfügen:

```markdown
Die Oberfläche nutzt die ANSI-Farben deines Terminal-Themes: Kontext-Header
oben, gerahmtes Hauptpanel, Tastenhinweise unten. Auf niedrigen Terminals
klappt der Header ein, damit die Liste Platz behält.
```

- [ ] **Step 7: Gate und Commit**

```bash
gofmt -l . && go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
git add internal/tui/ README.md
git commit -m "test: pin the rendered frame with a golden file"
```

---

## Self-Review

**Spec-Abdeckung:** Layout-Skizze → Tasks 2–4. Header-Felder pro Modus (Spec-Tabelle) → Task 4, Step 3, plus Test `TestHeaderFieldsPerMode`. Theme-Tabelle → Task 1. Rahmenlose Übersicht → Task 4, `framed = false`. Höhen-/Breitenbudget inklusive Kollaps-Schwellen → Task 3 (`chromeHeight`) und Task 4 (`bodyHeight`), Tests `TestChromeHeightCollapses`/`TestViewBudgetsHeight`. `renderStatus` entfällt → Task 4, Step 5. Fehler im Footer → Task 2 (`renderFooter`) und Test `TestErrorGoesToFooter`. Tests laut Spec (Header/Footer/Panel, Breiten-Invariant über alle Modi, Kollaps-Schwellen, Golden) → Tasks 2, 3, 4, 5. Non-Goals verletzt keine Task.

**Platzhalter:** keine. Zwei Stellen tragen bewusst einen Umsetzungshinweis statt fertigen Codes — der doppelte `top`-Aufbau in `panel` und die `right`-Zuweisung im einzeiligen Header-Zweig. Beide sind mit der gewünschten Endform beschrieben, nicht offengelassen.

**Typkonsistenz:** `headerField{label, value}` durchgängig; `renderHeader(width, height int, version string, rows [][]headerField)` in Task 3 definiert und in Task 4 so aufgerufen; `panel(title, body string, width int, focused bool)` in Task 2 definiert, in Task 3 und 4 so genutzt; `chromeHeight(height int) int` einheitlich; `footerHints(m mode)` und `panelTitle(m mode)` beide über `mode`. Der Test-Helper `renderFieldsFlat` wird in Task 4 definiert und in den umgestellten Tests derselben Task genutzt.
