# watson-tui — kompakte Tagesübersicht

**Datum:** 2026-07-27
**Status:** Approved
**Basis:** `main` bei `ad2d8a1` (englische Oberfläche)

## Anlass

Die Frameliste zeigt jede erfasste Sitzung einzeln. Was den Nutzer beim Öffnen interessiert, ist aber die Verdichtung, die `watson aggregate` liefert: pro Tag, welche Projekte, und darin welche Tags. Die einzelnen Frames braucht man erst, wenn man etwas ändern will.

## Ziel

Eine kompakte Tagesübersicht als Startbild, mit `f` umschaltbar auf die bisherige Frameliste. Keine neue Berechnung — die Zahlen kommen aus derselben Aggregation wie der Report.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Genauigkeit | Minuten, wie überall sonst in der Anwendung |
| Startbild | kompakte Ansicht |
| Umschalten | `f` |
| Leere Tage | werden gezeigt, mit `–` als Summe |
| Struktur | zweite Darstellung der Liste, keine eigene Betriebsart |

Der letzte Punkt ist der tragende: Zeitraum-Navigation, Filter, Kopf und Höhenbudget bleiben dadurch unverändert gültig. Eine eigene Betriebsart müsste sie duplizieren.

## Layout

```
╭─ Summary ────────────────────────────────────────────────────────────╮
│ Friday, 2026-07-24 — 6h 20m                                          │
│ ▌  moneymoney — 3h 43m                                               │
│       [monitoring  3h 43m]                                           │
│       [docs           30m]                                           │
│                                                                      │
│    rheinbahn — 42m                                                   │
│       [docs           42m]                                           │
│                                                                      │
│ Saturday, 2026-07-25 — –                                             │
│                                                                      │
│ Monday, 2026-07-27 — 1h 58m                                          │
│    ewr — 1h 30m                                                      │
╰──────────────────────────────────────────────────────────────────────╯
```

- **Tageszeile:** `<Wochentag>, <ISO-Datum> — <Summe>`, in der Akzentfarbe wie die bisherigen Tagesheader. Ohne Erfassung steht `–` statt `0m`.
- **Projektzeile:** zwei Leerzeichen eingerückt, `<projekt> — <summe>`. Sortiert nach Dauer absteigend, bei Gleichstand alphabetisch — wie im Report.
- **Tagzeile:** sechs Leerzeichen eingerückt, `[<tag>  <summe>]` in der gedämpften Farbe. Sortierung wie beim Projekt.
- **Leerzeile** nach jedem Projektblock und vor jeder Tageszeile, außer am Anfang.
- **Panel-Titel:** `Summary` statt `Frames`.

**Ausrichtung:** die Dauern stehen in einer Spalte, die sich nach dem längsten Namen der Ansicht richtet — nicht am rechten Rand des Terminals. Auf einem breiten Terminal soll die Zahl neben dem Namen stehen, nicht sechzig Spalten entfernt. `report.go` löst dasselbe Problem bereits mit `reportLayout` und `reportRow`; die Zusammenfassung folgt diesem Muster.

Die Zahl gibt nie nach: passt eine Zeile nicht, wird der Projekt- oder Tagname mit `…` gekürzt, nie die Dauer — dieselbe Regel wie in Liste, Report und Übersicht.

## Verhalten

- `f` schaltet um. Der Fuß zeigt in der kompakten Ansicht `f frames`, in der Frameliste `f summary`.
- Der Cursor läuft über die **Projektzeilen**. Damit funktionieren `j`/`k` und das Scrollen unverändert.
- `enter` und `d` brauchen einen Frame und sind in der kompakten Ansicht wirkungslos; der Fuß nennt sie dort nicht. `n` bleibt, es braucht keine Auswahl.
- Alles andere verhält sich wie bisher: `←`/`→`, `t`/`w`/`m`/`a`, `/`, `s`/`S`, `r`, `o`, `R`, `?`, `q`.
- Der Filter wirkt auch hier: gefiltert wird über dieselbe `filterFrames`, die die Frameliste nutzt.

## Aggregation

Pro Tag des Zeitraums wird `aggregate` über die Frames dieses Tages gerufen — dieselbe Funktion wie im Report, also automatisch dieselben Regeln:

- Ein Frame zählt vollständig in den Tag, an dem er **beginnt**.
- Tags werden einzeln gezählt; ein Frame mit zwei Tags zählt in beide, die Tag-Summen können also die Projektsumme übersteigen.
- Ein laufender Timer zählt bis jetzt mit, wie im Report und in der Übersicht.

Die Tageszeile zeigt die Summe der Frames dieses Tages, nicht die Summe der Tag-Zeilen darunter — sonst stünde bei Mehrfach-Tags eine zu große Zahl.

Welche Tage der Zeitraum umfasst, liefert `period.bounds`. Bei `unitAll` gibt es keine Grenzen: dort werden nur Tage mit Erfassung gezeigt, vom ältesten bis zum jüngsten Frame — sonst entstünden Zeilen für jeden Tag seit Projektbeginn.

## Kopfzeile

**Korrigiert nach der Umsetzung.** Der ursprüngliche Entwurf ließ `Total` auch in der kompakten Ansicht den laufenden Timer aussparen und mit `+ running` markieren — das war falsch. Die Tageszeilen der Zusammenfassung rechnen ihn ein, weil sie aus `aggregate` kommen; ein Kopf ohne ihn widerspräche also seinem eigenen Body. Genau dieser Fehler wurde im Report schon einmal behoben.

Deshalb gilt: in der **Frameliste** bleibt `Total` ohne laufenden Timer, mit `+ running` — dort schließen die Tagessummen ihn auch aus. In der **Zusammenfassung** zählt `Total` ihn mit und trägt keine Markierung, wie im Report und in der Übersicht. Der Kopf stimmt damit in beiden Darstellungen mit dem überein, was darunter steht.

Der Zähler `n frames · m projects` zählt weiterhin Frames, nicht Zeilen.

## Komponenten

- **`internal/tui/summary.go`** (neu): `buildSummaryRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string, state *watson.State, now time.Time) []row` und die Zeilenformatierung. Die Aggregation gehört nicht in `frameslist.go`, das schon groß ist.
- **`internal/tui/frameslist.go`**: `row` bekommt statt `isHeader bool` ein Feld `kind rowKind` mit den Werten `rowFrame`, `rowDayHeader`, `rowProject`, `rowTag`, `rowBlank`; `listModel` bekommt `compact bool`; `refresh` wählt den Zeilenbauer; `renderRow` rendert die neuen Arten; `firstFrameRow`/`nextFrameRow`/`selected` arbeiten über die Art statt über `isHeader`.
- **`internal/tui/app.go`**: `f` in `updateList`; `panelTitle` liefert `Summary` bzw. `Frames`.
- **`internal/tui/chrome.go`**: die Hinweisgruppen der Liste bekommen `f frames` bzw. `f summary`; `enter edit` und `d delete` entfallen in der kompakten Ansicht.

## Tests

- `buildSummaryRows`: Reihenfolge (Tage aufsteigend, Projekte nach Dauer absteigend mit alphabetischem Gleichstand, Tags ebenso), leere Tage mit `–`, Filter wirkt, laufender Timer zählt mit.
- Die Tageszeile entspricht der Summe der Frames dieses Tages, auch wenn ein Frame mehrere Tags trägt — der Fall, in dem die Tag-Summen größer sind.
- `unitAll` erzeugt keine Zeilen für Tage ohne Erfassung.
- Cursor: `j`/`k` bewegen sich über Projektzeilen, nie über Tag-, Tages- oder Leerzeilen; `selected()` liefert in der kompakten Ansicht keinen Frame.
- `f` schaltet um und zurück; der Fuß nennt jeweils das Gegenstück; `enter`/`d` ändern in der kompakten Ansicht nichts.
- Die bestehenden Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in Liste, Report und Übersicht — und keine gekürzte Zahl in der kompakten Ansicht.
- Golden-Datei für die kompakte Ansicht bei 100×30.

## Non-Goals

- Kein Sekundenformat
- Kein Aufklappen einzelner Tage, keine Auswahl von Projekten zum Filtern
- Keine dritte Darstellung, kein Rotieren über mehrere Ansichten
- Keine Änderung an Report, Übersicht, Datenlayer oder Watson-Dateiformat
