# watson-tui — visuelles Redesign (Chrome & Theme)

**Datum:** 2026-07-26
**Status:** Approved
**Basis:** v0.1.0 (`main`)

## Ziel

Die TUI soll aussehen wie ein modernes Terminal-Tool (k9s, lazygit) statt wie ein Skript-Output: Header-Panel mit Kontext, gerahmte Panels, gestylter Key-Hint-Footer, farbiger Selektionsbalken. Rein visuell — kein Verhalten, keine Tastenbelegung und keine Berechnung ändern sich.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Chrome-Level | Voll k9s-artig: Header-Panel, gerahmte Panels, Footer |
| Farben | Nur ANSI 0–15, also die Palette des Terminal-Themes (wie k9s/lazygit per Default) |
| Abrechnungs-Übersicht | Rahmenlos, behält die volle Breite; Header und Footer trotzdem wie überall |

Begründung Farben: passt sich jedem Terminal-Theme an, funktioniert über SSH und in jedem Terminal-Emulator. Die bestehenden Farbrollen (cyan Tagesheader, grüner Timer, roter Fehler) bleiben damit erhalten.

Begründung rahmenlose Übersicht: Die Tabelle braucht bereits ~94 Spalten für volle Spaltentitel; zwei weitere für Rahmen würden Kurztitel früher erzwingen. Header und Footer bilden die visuelle Klammer, der fehlende Seitenrahmen fällt kaum auf.

## Layout

```
╭─ watson-tui 0.1.0 ───────────────────────────────────────────╮
│ Zeitraum  Woche 20.07.–26.07.        Frames  12              │
│ Filter    —                          ▶ schnaq 1:23:45        │
╰──────────────────────────────────────────────────────────────╯
╭─ Frames ─────────────────────────────────────────────────────╮
│ Montag, 20.07.2026                                    6h 30m │
│   09:00–11:30   2h 30m  schnaq        dev       a1b2c3d      │
│▌  13:00–17:00   4h 00m  kunde-a       meeting   e4f5g6h      │
╰──────────────────────────────────────────────────────────────╯
 j/k bewegen · enter bearbeiten · n neu · d löschen · ? hilfe
```

- **Header** (4 Zeilen): Rahmen mit Titel `watson-tui <version>`; innen zwei Zeilen mit je zwei Feldern — links der View-Kontext, rechts Zählwerte und der laufende Timer. Die Felder pro Modus:

| Modus | Feld 1 (links oben) | Feld 2 (links unten) | Feld 3 (rechts oben) |
|---|---|---|---|
| Liste | `Zeitraum` = Perioden-Label | `Filter` = Filtertext oder `—`; im Filtermodus das Eingabefeld | `Frames` = Anzahl im Zeitraum |
| Report | `Report` = Perioden-Label | `Filter` entfällt | `Projekte` = Anzahl Zeilen |
| Übersicht | `Übersicht` = `Abrechnung` | — | `Projekte` = Anzahl Zeilen |
| Formular | `Frame` = `bearbeiten (<kurz-id>)` oder `neu` | — | — |
| Timer-Prompt | `Timer` = `starten` | — | — |
| Dialoge, Hilfe, Fatal | Titel des Dialogs | — | — |

  Feld 4 (rechts unten) ist in jedem Modus der laufende Timer (`▶ projekt H:MM:SS`) bzw. `kein Timer`. Leere Felder werden weggelassen, nicht als Leerzeile gerendert.
- **Body**: bei Liste, Report, Formular, Timer-Prompt, Hilfe und Dialogen in ein Panel mit Titel gewrappt. Die Übersicht rendert direkt, ohne Seitenrahmen.
- **Footer** (1 Zeile): Key-Hints des aktiven Modus, gedämpft. Fehler ersetzen die Hints, in Fehlerfarbe.

## Komponenten

**Neu: `internal/tui/chrome.go`** — das gesamte Gerüst an einer Stelle:

- `renderHeader(width, height int, title string, fields []headerField) string` — Header-Panel; kollabiert nach Höhenbudget (siehe unten)
- `renderFooter(width int, hints string, errMsg string) string` — Key-Hints oder Fehler
- `panel(title, body string, width int, focused bool) string` — gerahmtes Panel; `focused` färbt den Rahmen in der Akzentfarbe
- `type headerField struct { label, value string }`
- `chromeHeight(height int) int` — belegte Zeilen des Chrome bei gegebener Terminalhöhe
- `footerHints(m mode) string` — Hints pro Modus

**Geändert: `internal/tui/styles.go`** — benanntes Theme statt flacher Vars:

| Rolle | ANSI | Verwendung |
|---|---|---|
| Rahmen inaktiv | 8 | Panel-Rahmen, gedämpfter Text |
| Akzent | 6 | Panel-Titel, Tagesheader |
| Fokus | 4 | Rahmen des aktiven Panels, Selektionshintergrund |
| Erfolg | 2 | laufender Timer |
| Warnung | 3 | Overlap-Warnung |
| Fehler | 1 | Fehlermeldungen, Fatal-Screen |

Die bisherigen Namen (`styleTitle`, `styleDayHeader`, `styleSelected`, `styleDim`, `styleRunning`, `styleError`) bleiben als Bezeichner erhalten, damit die View-Dateien unverändert bleiben; nur ihre Definitionen wechseln. `styleSelected` wechselt von `Reverse(true)` auf Fokus-Hintergrund; der `▌`-Marker in der ersten Spalte kommt aus `listModel.renderRow`.

**Geändert: `internal/tui/app.go`** — `View()` setzt Header, Body und Footer zusammen und rechnet das Höhenbudget aus. Die View-Funktionen der einzelnen Modi liefern weiterhin nur ihren Rumpf und wissen nichts vom Chrome. `statusLeft()` wird zu `headerFields()`, das die Felder strukturiert liefert statt eine vorformatierte Zeile.

**Entfällt:** `renderStatus` in `statusbar.go` — der Header übernimmt Kontext und Timer, der Footer die Hints. `formatDuration`/`formatClock`/`tickCmd` bleiben unverändert.

## Höhen- und Breitenbudget

- Rahmen kosten zwei Spalten; die Panel-Breite ist `width − 2`.
- Body-Höhe: `height − chromeHeight(height)`.
- **Kollaps-Schwellen**, damit das Chrome auf kleinen Terminals nicht die Liste frisst:
  - ab 20 Zeilen: voller Header (4 Zeilen) + Footer (1) = 5
  - 12–19 Zeilen: Header auf eine Zeile ohne Rahmen (1) + Footer (1) = 2
  - unter 12 Zeilen: kein Header, nur Footer = 1
- Die Übersicht erhält die volle `width`, alle anderen Views `width − 2`.

## Fehlerbehandlung

Unverändert in der Sache: Lesefehler bleiben fatal, Schreibfehler landen in `errMsg`. Neu ist nur der Ort — `errMsg` erscheint im Footer statt in der alten Statuszeile, in Fehlerfarbe und anstelle der Key-Hints. Der Fatal-Screen bekommt ein Panel in Fehlerfarbe.

## Tests

Die bestehenden Tests prüfen Inhalte über `strings.Contains` und bleiben grün, weil Rahmen den Text nicht verändern. Neu:

- `chrome_test.go`: Header enthält Titel und alle Feldwerte; Footer zeigt die Hints des Modus bzw. bei gesetztem `errMsg` die Fehlermeldung statt der Hints; `panel` setzt den Titel in die obere Rahmenlinie.
- Breiten-Invariant über den kompletten Frame: bei 80 und 100 Spalten ist keine gerenderte Zeile breiter als das Terminal — in allen Modi. Erweitert die Idee aus `overview_layout_test.go` auf `App.View()`.
- Kollaps-Schwellen: `chromeHeight` liefert 5 / 2 / 1 an den Grenzen 20, 19, 12, 11; `App.View()` bei Höhe 10 enthält keinen Header-Rahmen.
- Golden-Test: kompletter Frame bei 100×30 im Listen-Modus gegen eine erwartete Zeichenkette, damit Layout-Regressionen auffallen.

## Non-Goals

- Kein Theme-Switch, keine Farb-Konfiguration
- Keine Truecolor-Palette
- Keine Maus, keine Tabs, keine neuen Views
- Keine Änderung an Tastenbelegung, Datenlayer oder Berechnungen
