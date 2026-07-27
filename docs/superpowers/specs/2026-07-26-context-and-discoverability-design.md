# watson-tui — Kontext und Entdeckbarkeit

**Datum:** 2026-07-26
**Status:** Approved
**Basis:** Branch `feature/tui-chrome` (Chrome-Redesign, noch nicht gemergt)

## Anlass

Der Nutzer wusste nicht, dass `[` und `]` den Zeitraum verschieben. Der Grund ist belegbar: `footerHints(modeList)` listet zehn Hinweise, **Zeitraum-Navigation ist keiner davon** — weder `[ ]` noch `t/w/m/a`. Die Hilfe (`?`) nennt beides, aber im laufenden Bild deutete nichts darauf hin, dass es die Funktion gibt. Dazu wirkt der Kopf inhaltsarm: er zeigt den Zeitraum als Zustand, nicht als Bedienelement, und nennt keine Summen.

## Ziel

Drei Dinge, rein in der Darstellung:

1. Der Zeitraum ist als verschiebbar erkennbar.
2. Der Kopf beantwortet „wo bin ich, wie viel ist das, was war davor" auf einen Blick.
3. Die Tastenhinweise sind vollständig genug, dass man die Kernfunktionen ohne `?` findet.

Kein Verhalten, keine neue Taste, keine Änderung an Datenlayer oder Berechnung.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Kopfdichte | Zeitraum mit `‹ ›`, Summe, Vergleich zum Vorzeitraum, Frame- und Projektzahl, Filter, Timer |
| Tastenhinweise | Zwei Zeilen, gruppiert, Tasten farblich abgesetzt, `[ ] Zeitraum` und `t/w/m/a` ergänzt |
| Summe in der Liste | Zählt den laufenden Timer **nicht** mit, weist ihn aber mit `+ läuft` aus |

## Layout

```
╭─ watson-tui 0.1.0 ──────────────────────────────────────────────╮
│ Zeitraum  ‹ Woche 20.07.–26.07.2026 ›       Summe  12h 30m + läuft │
│ Vorwoche  8h 00m · Monat  40h 15m                                │
│ Filter    —                              3 Frames · 2 Projekte   │
│                                            ▶ schnaq [tui] 1:23:45 │
╰──────────────────────────────────────────────────────────────────╯
╭─ Frames ─────────────────────────────────────────────────────────╮
│ …                                                                │
╰──────────────────────────────────────────────────────────────────╯
 j/k bewegen   [ ] Zeitraum   t/w/m/a Tag/Woche/Monat/alles
 enter bearbeiten  n neu  d löschen  s timer  / filtern  r report  ? hilfe  q ende
```

### Zeitraum-Feld

Der Wert wird als `‹ <label> ›` gerendert, die Winkel in der gedämpften Farbe, das Label normal. Sie sind die Andeutung für `[` und `]` — deshalb entfallen sie bei `unitAll`, wo Verschieben nichts tut (`period.bounds` liefert dort `ok == false`).

### Vergleichszeile

Zwei Werte, passend zur aktiven Einheit. Grundlage ist jeweils die Summe der Frames, deren Start im betreffenden Zeitraum liegt — dieselbe Zuordnungsregel wie überall sonst.

| Einheit | Feld 1 | Feld 2 |
|---|---|---|
| Tag | `Vortag` | `Woche` (die Woche des angezeigten Tages) |
| Woche | `Vorwoche` | `Monat` (der Monat des Wochenstarts) |
| Monat | `Vormonat` | `Jahr` (das Kalenderjahr des Monats) |
| alles | Zeile entfällt | — |

Ein Wert von null wird als `–` gezeigt, nicht als `0m`.

### Summe

Die Summe der Frames im Zeitraum — genau die, die in der Liste stehen, damit die Kopfzeile mit den Tagessummen darunter nachrechenbar übereinstimmt. Fällt der Start eines laufenden Timers in den Zeitraum, folgt ` + läuft` in der Timer-Farbe. Report und Übersicht rechnen den laufenden Timer weiterhin ein; die Kopfsumme der Liste tut es nicht, und `+ läuft` ist der Hinweis, woher die Differenz kommt.

### Felder pro Modus

| Modus | Zeile 1 | Zeile 2 | Zeile 3 | Zeile 4 |
|---|---|---|---|---|
| Liste | `Zeitraum ‹…›` \| `Summe` | Vergleich | `Filter` \| `n Frames · m Projekte` | Timer (rechts) |
| Report | `Report ‹…›` \| `Summe` | Vergleich | — \| `m Projekte` | Timer (rechts) |
| Übersicht | `Übersicht Abrechnung` \| `Summe gesamt` | — | — \| `m Projekte` | Timer (rechts) |
| Formular | `Frame bearbeiten (<id>)` bzw. `neu` | — | — | Timer (rechts) |
| Timer-Prompt, Dialoge, Hilfe, Fatal | wie bisher | — | — | Timer (rechts) |

Die Übersicht hat feste Spalten, also kein `‹ ›` und keine Vergleichszeile. `Summe gesamt` ist die Gesamtspalte der Tabelle.

### Tastenhinweise

Zwei Gruppen, jede eine Zeile. Innerhalb einer Gruppe wird wie bisher von hinten weggelassen, wenn die Breite nicht reicht — nie mitten im Wort. Die Taste steht in der Akzentfarbe, die Beschreibung gedämpft.

- **Zeile 1 (Navigation und Zeitraum):** `j/k bewegen`, `[ ] Zeitraum`, `t/w/m/a Tag/Woche/Monat/alles`
- **Zeile 2 (Aktionen und Ansichten):** `enter bearbeiten`, `n neu`, `d löschen`, `s timer`, `/ filtern`, `r report`, `o übersicht`, `R neu laden`, `? hilfe`, `q ende`

`? hilfe` und `q ende` stehen in Zeile 2 so weit vorn, dass sie bis 80 Spalten überleben — es sind die zwei Tasten, die man ohne sie nicht mehr nachschlagen kann. Die anderen Modi behalten je eine Zeile mit ihren bisherigen Hinweisen.

Steht nur eine Fußzeile zur Verfügung (Terminalhöhe unter 24), werden die Gruppen **nicht** zusammengeführt, sondern in eine Zeile gegossen und von hinten gekürzt: erst `j/k bewegen`, `[ ] Zeitraum`, dann `? hilfe`, `q ende`, dann der Rest in der Reihenfolge von Zeile 2. So bleiben Zeitraum-Navigation und die beiden Notausgänge auch auf niedrigen Terminals sichtbar — das ist der Zweck der ganzen Änderung.

## Höhenbudget

Das Chrome wächst von 5 auf 8 Zeilen (6 Kopf, 2 Fuß), deshalb ein gestufter Kollaps:

| Terminalhöhe | Kopf | Fuß | Chrome |
|---|---|---|---|
| ≥ 24 | gerahmt, 4 Feldzeilen | 2 Zeilen | 8 |
| 20–23 | gerahmt, ohne Vergleichszeile (3 Feldzeilen) | 1 Zeile | 6 |
| 14–19 | eine Zeile, ungerahmt | 1 Zeile | 2 |
| < 14 | entfällt | 1 Zeile | 1 |

Die einzeilige Stufe zeigt Zeitraum-Label und Timer, wie bisher.

## Komponenten

- **`internal/tui/chrome.go`**: `chromeHeight` bekommt die neuen Stufen; `renderHeader` rendert je nach Stufe drei oder vier Feldzeilen; `renderFooter` nimmt `[]string`-Gruppen statt einer Liste und rendert eine Zeile pro Gruppe; `footerHints` liefert `[][]string`.
- **`internal/tui/app.go`**: `headerFields` baut die neuen Zeilen; neue Helfer `periodField`, `comparisonFields`, `sumField`.
- **`internal/tui/period_sums.go`** (neu): `sumInPeriod(frames []watson.Frame, p period, weekStart time.Weekday) time.Duration` und `comparisonPeriods(p period) []struct{label string; per period}` — die Vergleichslogik gehört nicht in `app.go`, das schon groß ist.
- **`internal/tui/styles.go`**: `styleKey` für die abgesetzte Taste.

## Tests

- `sumInPeriod` gegen die Zuordnungsregel: ein Frame, der im Zeitraum beginnt, zählt vollständig; einer, der davor beginnt, gar nicht.
- `comparisonPeriods` je Einheit: die richtigen Labels und Grenzen; bei `unitAll` leer.
- Die Kopfsumme der Liste stimmt mit der Summe der Tagessummen der angezeigten Frames überein — und weicht bei laufendem Timer von der des Reports ab, mit `+ läuft` im Kopf.
- `‹ ›` erscheint bei Tag/Woche/Monat und nicht bei `alles`.
- Zeitraum-Navigation ist in den Hinweisen der Liste sichtbar: `[ ]` und `t/w/m/a` überleben bei 80 Spalten, ebenso `? hilfe` und `q ende`.
- `chromeHeight` an den Grenzen 24/23/20/19/14/13.
- Die bestehenden Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in den Abrechnungsansichten.

## Non-Goals

- Keine neue Taste, kein verschiebbarer Zeitraum in der Übersicht (eigenes Thema)
- Kein rotierender Tipp, keine Onboarding-Tour
- Keine Farbkonfiguration, kein Theme-Switch
- `t`/`w`/`m` setzen den Zeitraum weiterhin auf heute zurück — das bleibt, bis der Nutzer es anders will
