# watson-tui — die Zusammenfassung lesbar machen

**Datum:** 2026-07-28
**Status:** Approved
**Basis:** `main` bei `46d97d1` (Zahlenspalte nach den Namen bemessen)
**Vorgänger:** [2026-07-27-summary-view-design.md](2026-07-27-summary-view-design.md)

## Anlass

Die Zusammenfassung stimmt rechnerisch, aber sie sieht aus wie ein Ausdruck: drei
Einrückungsebenen, sonst nur Leerzeichen. Kein Strukturzeichen, kein Gewicht, und
auf einem breiten Terminal steht rechts der Zahlenspalte nichts. Der Nutzer
vergleicht sie mit der Frameliste, die durch ihre Zeitspannen von sich aus
Textur hat, und findet die Zusammenfassung dagegen leer.

Drei Dinge fehlen, und alle drei sind Lücken im Vorgänger-Entwurf, keine
Fehlentscheidungen darin:

1. **Kein Größenverhältnis.** Dass Montag der stärkste Tag der Woche ist, steht
   nur in Ziffern da. Es ist die Frage, mit der man eine Zeiterfassung öffnet.
2. **Heute ist nicht markiert.** Der laufende Tag sieht aus wie jeder andere.
3. **Tagesblöcke lesen sich nicht als Blöcke.** Tages- und Projektgrenze tragen
   dieselbe Leerzeile, also gibt es optisch keine Blöcke, nur eine Liste.

## Ziel

Dieselben Zahlen, dieselbe Bedienung, dieselbe Aggregation. Nur: ein
Anteilsbalken im ungenutzten Raum, eine Klammer um jeden Tagesblock, und drei
Gewichte für die Tageszeile.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Größenverhältnis | Anteilsbalken rechts der Zahl |
| Balkenskala | auf den Zeitraum: der stärkste Tag füllt die Breite |
| Tagesblock | Klammer-Rail in einer eigenen Spalte links |
| Tag-Zeilen | Baumzeichen `├─`/`└─` statt der eckigen Klammer |
| Heute | fette Tageszeile plus Rail in der Akzentfarbe |
| Leere Tage | bleiben sichtbar, gedimmt, ohne Rail |
| Projektfarben | nein, eine Farbe — die Palette behält ihre Bedeutungen |
| Zahlenspalte | unverändert nach den Namen bemessen |

Der letzte Punkt ist die Randbedingung, unter der alles andere steht: `46d97d1`
hat die Zahlen absichtlich neben die Namen gestellt statt an den rechten Rand.
Der Balken füllt den Raum, der dadurch rechts frei bleibt — er verschiebt die
Zahlenspalte nicht und wird erst berechnet, wenn Label- und Zahlenspalte stehen.

## Layout

```
╭─ Summary ──────────────────────────────────────────────────╮
│ ╭ Monday, 2026-07-27    4h 59m  ████████████████████████   │
│ ▌   unlock              2h 07m  ██████████▎░░░░░░░░░░░░░   │
│ │     └─ feed           2h 07m                             │
│ │   ewr                 1h 30m  ███████▎░░░░░░░░░░░░░░░░   │
│ │     └─ kiserver       1h 30m                             │
│ │   moneymoney          1h 21m  ██████▌░░░░░░░░░░░░░░░░░   │
│ │     ├─ conan             53m                             │
│ │     ├─ monitoring        53m                             │
│ ╰     └─ meeting           28m                             │
│                                                            │
│ ╭ Tuesday, 2026-07-28   2h 07m  ██████████▎░░░░░░░░░░░░░   │
│ │   ordino              1h 00m  ████▉░░░░░░░░░░░░░░░░░░░   │
│ │     └─ beratung       1h 00m                             │
│ │   unlock                 59m  ████▊░░░░░░░░░░░░░░░░░░░   │
│ │     └─ dev               59m                             │
│ │   moneymoney              8m  ▋░░░░░░░░░░░░░░░░░░░░░░░   │
│ │     ├─ conan              8m                             │
│ ╰     └─ monitoring         8m                             │
│                                                            │
│   Wednesday, 2026-07-29      –                             │
│   Thursday, 2026-07-30       –                             │
│   Friday, 2026-07-31         –                             │
╰────────────────────────────────────────────────────────────╯
```

Montag ist mit 4h 59m der stärkste Tag der Woche, sein Balken ist deshalb voll.
Dienstag hat 2h 07m, also 43 % davon, und bekommt 43 % Balken. Die
Projektbalken derselben Skala addieren sich damit exakt zum Tagesbalken darüber
— er ist ihre Summe, nicht eine zweite Skala.

### Spalten

Von links: zwei Spalten Gutter (Rail-Zeichen und ein Leerzeichen), die
Labelspalte, ein Trennzeichen, die Zahlenspalte, zwei Spalten Abstand, der
Balken. `panel` setzt selbst schon ein Leerzeichen hinter den Rahmen, ein
zusätzlicher Rand ist also nicht nötig.

Die Labelspalte wird wie bisher aus den Zeilen bemessen — jetzt über das
Maximum aus Tagesname, `2 + Projektname` und `4 + 3 + Tagname` — und
anschließend gegen `width - 2` an die Breite angepasst. Die eckige Klammer der
Tag-Zeile fällt weg, `summaryTail` damit auch: der Nachlauf ist überall leer.

### Rail

Eine Spalte, die den Tagesblock umklammert:

| Zeile | Zeichen |
|---|---|
| Tageszeile eines Tages mit Erfassung | `╭` |
| Projekt- und Tag-Zeilen darin | `│` |
| letzte Zeile des letzten Projekts des Tages | `╰` |
| Tageszeile eines leeren Tages | keins |
| Cursor-Zeile | `▌` |

Das Zeichen ist eine Funktion aus den Zeilen und dem Index, kein Feld auf `row`:
`╰` steht, wenn die nächste Zeile eine Leer- oder Tageszeile ist oder es keine
nächste gibt. Damit stimmt es auch für ein letztes Projekt ohne Tags, ohne dass
der Zeilenbauer davon wissen muss.

Zwei Kollisionen und wie sie ausgehen:

- **Cursor auf der Schlusszeile.** Der Cursor läuft über Projektzeilen; die
  Schlusszeile ist normalerweise eine Tag-Zeile. Trägt das letzte Projekt eines
  Tages keine Tags und steht der Cursor darauf, ersetzt `▌` das `╰` und die
  Klammer schließt optisch nicht. Der Cursor gewinnt — wo man steht ist die
  dringendere Information, und der Zustand vergeht mit der nächsten Taste.
- **Angeschnittener Block.** Scrollt der Blockanfang aus dem Bild, fehlt das
  `╭` und der Rail läuft oben aus. Das ist richtig so: ein `╭` an der oberen
  Kante würde behaupten, der Tag beginne dort.

### Balken

- **Skala:** Bezugsgröße ist die größte Tagessumme der Ansicht, über alle
  Zeilen berechnet, nicht nur die sichtbaren — wie `summaryColumns` und
  `listDurWidth`, damit Scrollen nichts unter dem Leser verschiebt. Ist sie 0
  (leerer Zeitraum, oder ein Filter ohne Treffer), gibt es keine Balken; damit
  ist auch die Division abgesichert.
- **Auflösung:** Achtelblöcke `▏▎▍▌▋▊▉`. Jede Dauer über 0 bekommt mindestens
  `▏` — sonst rendert ein 8-Minuten-Eintrag als nichts, und nichts ist eine
  Lüge. Im `all`-Zeitraum, wo ein starker Tag alles andere klein macht, hält
  genau diese Untergrenze die kleinen Tage sichtbar.
- **Track:** der Rest bis zur Balkenbreite mit `░`. Jede Zeile hat dadurch
  dieselbe optische Breite und der Balken liest sich als Anzeige statt als
  loser Strich.
- **Breite:** was hinter der Zahlenspalte übrig ist, nach oben auf 24 Spalten
  gekappt — sonst wird der Balken auf einem 200-Spalten-Terminal albern.
- **Wegfall:** unter 10 Spalten Rest gibt es keinen Balken. Bei weniger
  unterscheidet er 10 % nicht mehr von 40 %, und ein Balken, der falsche
  Verhältnisse zeigt, ist schlechter als keiner — dieselbe Begründung, aus der
  die ID-Spalte der Frameliste ganz oder gar nicht steht. Ein schmales Terminal
  zeigt damit genau das heutige Bild; es entsteht keine neue Breiten-Leiter.
- **Tag-Zeilen bekommen keinen Balken.** Tags werden einzeln gezählt, ein Frame
  mit zwei Tags zählt in beide. Ihre Balken wären zusammen länger als der
  Projektbalken darüber und würden eine Aufteilung behaupten, die es nicht
  gibt.

**`summaryBar` liefert reinen Text, gefüllten Teil und Track getrennt** —
`(filled, track string)`. Die Farbe setzt erst `renderRow`. Die Zeile besteht
sonst aus drei verschieden gefärbten Abschnitten, und dann rechnet jede Breite,
die über den zusammengesetzten String läuft, Escape-Bytes als Spalten mit: die
Zeile wird zu lang, `panel` schneidet sie mit `lipgloss.Width` ab, dabei fällt
die Reset-Sequenz weg und die Farbe blutet über den Rest des Bildschirms. Genau
davor warnt der Kommentar an `reportRow`. Regel: **jede Breitenrechnung
passiert auf dem unformatierten Text, vor dem ersten `Render`.**

**Die Balkenbreite hängt am längsten Namen der Ansicht.** Erst wird die
Labelspalte aus den Daten bemessen und gegen die Breite gekappt, dann die
Zahlenspalte, dann bekommt der Balken den Rest. Auf einem breiten Terminal mit
sehr langen Projektnamen kann der Balken deshalb wegfallen, obwohl Platz frei
wirkt. Das ist gewollt und nicht umgekehrt gelöst: der Name ist die
Information, der Balken die Zugabe — `46d97d1` hat diese Rangfolge gesetzt, und
ein Balken, der Namen kürzt, dreht sie um. Bei 100 Spalten müsste die
Labelspalte über 79 Spalten breit werden, damit das eintritt.

### Farben

Alles bleibt in ANSI 0–15, damit die Oberfläche weiter das Terminal-Thema
übernimmt.

| Element | Stil |
|---|---|
| Tageszeile heute | fett, Akzent (`styleDayHeader`, unverändert) |
| Tageszeile mit Erfassung | Akzent, nicht fett (`styleDayPlain`, neu) |
| Tageszeile leer | gedimmt |
| Rail im heutigen Block | Akzent |
| Rail sonst | gedimmt |
| Tagesbalken, gefüllt | Akzent |
| Projektbalken, gefüllt | Vordergrundfarbe des Terminals, wie die Projektzeile |
| Track | gedimmt |
| Tag-Zeile mit Baumzeichen | gedimmt, wie bisher |
| Cursor-Zeile | fett, Fokusfarbe — Balken eingeschlossen, Track bleibt gedimmt |

Der Tagesbalken in der Akzentfarbe und die Projektbalken in der neutralen
Vordergrundfarbe: damit liest sich die Summe als Summe, ohne dass ein zweites
Zeichen dafür nötig ist.

Heute trägt zwei Signale, fette Tageszeile und Akzent-Rail, und kein eigenes
Glyph. Ein Marker in der Gutter-Spalte hätte mit dem `╭` konkurriert.

`styleDayHeader` bleibt fett und behält seine Bedeutung, weil die **Frameliste**
es für ihre Tagesheader benutzt. Der nicht fette Fall der Zusammenfassung
braucht deshalb einen eigenen Stil und darf nicht durch Ändern des bestehenden
entstehen — sonst verliert die Frameliste ihre Header-Gewichtung mit.

Damit der Rail eines Blocks weiß, ob er zu heute gehört, trägt `row.today` nicht
nur die Tageszeile, sondern **jede Zeile des Blocks**. Sonst müsste der Renderer
zur nächsten Tageszeile zurücksuchen, um eine Farbe zu wählen.

### Vertikaler Rhythmus

- Keine Leerzeile mehr zwischen Projekten. Bisher trugen Tages- und
  Projektgrenze dieselbe Leerzeile; deshalb waren die Blöcke keine.
- Eine Leerzeile zwischen zwei Tagesblöcken — außer beide Tage sind leer. Eine
  ruhige Wochenhälfte kostet sonst zehn Zeilen für fünf Striche.

## Was der Vorgänger-Entwurf anders entschied

Drei Punkte werden umgekehrt, alle mit Grund:

**Die eckige Klammer der Tag-Zeile fällt.** Der Vorgänger begründete sie damit,
dass sie die einzige Spalte ist, die eine Tag-Zeile hat und Projekt- und
Tageszeile nicht — und dass genau das die Tag-Zeile als Aufschlüsselung lesbar
macht. Das Argument bleibt richtig, das Baumzeichen `├─`/`└─` erfüllt es aber
besser: es sagt zusätzlich, welche Tag-Zeile die letzte ist, und es hängt die
Zeile sichtbar an die darüber. `[docs 42m]` liest sich außerdem wie ein
Datenformat, nicht wie eine Oberfläche.

**Die Leerzeile nach jedem Projektblock fällt.** Sie war dort als Trennung
gedacht; die Trennung leistet jetzt der Rail, und zwar auf der Ebene, auf der
sie gebraucht wird.

**Der Gedankenstrich hinter Tages- und Projektnamen fällt.** Er setzte im
Vorgänger die Zahl vom Namen ab. Das leisten jetzt die feste Zahlenspalte und —
eine Ebene darüber — die Rail-Spalte; der Strich wiederholt es nur und kostet
zwei Spalten Labelbreite, die dem Balken fehlen. Er entfällt ersatzlos.

Alles Übrige bleibt unberührt: das ISO-Datum, die sichtbaren leeren Tage mit
`–`, die Sortierung, die Aggregationsregeln, die Kopfzeile und die Bedienung.

## Komponenten

- **`internal/tui/styles.go`**: neu `styleDayPlain` (Akzent, nicht fett) für die
  Tage der Zusammenfassung, die nicht heute sind; `styleDayHeader` bleibt
  unverändert. Balken-, Track- und Rail-Zeichen als Konstanten, dazu die beiden
  Balkenmaße (Kappung 24, Wegfall unter 10).
- **`internal/tui/summary.go`**: `summaryTail` entfällt. Neu: `summaryPeak`
  (größte Tagessumme), `summaryBarWidth` (Rest der Breite, gekappt, oder 0),
  `summaryBar` (Achtelblöcke plus Track), `summaryRail` (Zeichen aus Zeilen und
  Index). `summaryColumns` rechnet mit `width - 2` und ohne Nachlauf.
  `summaryTagIndent` wird 4, das Baumzeichen liegt im Label.
  `buildSummaryRows` setzt die neue Leerzeilen-Regel und `row.today`.
- **`internal/tui/frameslist.go`**: `row` bekommt `today bool`. `renderRow`
  setzt Gutter, Zeile und Balken getrennt zusammen, damit eine nicht
  ausgewählte Projektzeile keinen Balken in Terminal-Vordergrundfarbe und
  keinen Track in Vollfarbe bekommt. Die Frameliste bleibt unberührt.
- **`README.md`**: der Abschnitt über die Zusammenfassung beschreibt Balken,
  Rail, Gewichte und die neue Reihenfolge des Nachgebens.

## Tests

- `summaryRail`: `╭` auf der Tageszeile, `│` innen, `╰` auf der letzten Zeile
  des Tages — auch wenn das letzte Projekt keine Tags trägt; kein Zeichen auf
  einem leeren Tag.
- `summaryRail` mit einem Filter, der einen Tag **mitten im Zeitraum** leer
  macht: Woche, Filter trifft nur Montag und Mittwoch. Erwartet Montag `╭…╰`,
  Dienstag ohne Rail, Mittwoch `╭…╰`, Donnerstag bis Sonntag ohne Rail und ohne
  Leerzeilen zwischen sich. Der Fall unterscheidet sich von einem Zeitraum, in
  dem die leeren Tage am Ende liegen, und ist der einzige, in dem eine
  Tageszeile ohne Rail zwischen zwei Blöcken steht.
- Keine Zeile enthält eine Escape-Sequenz, bevor ihre Breite gerechnet ist:
  `lipgloss.Width` der fertigen Zeile ist gleich der Summe der Spalten, die die
  Layoutfunktionen zugeteilt haben.
- `summaryBar`: Achtel runden richtig; jede Dauer über 0 ergibt mindestens
  `▏`; der stärkste Tag füllt die Breite; Track füllt genau auf die Breite auf;
  Bezugsgröße 0 ergibt keinen Balken.
- `summaryBarWidth`: gekappt bei 24, 0 unter 10 Spalten Rest, und die Zahlen-
  und Labelspalte sind davon in keiner Breite betroffen.
- Leerzeilen-Regel: eine zwischen zwei Tagesblöcken, keine zwischen Projekten,
  keine zwischen zwei leeren Tagen.
- Drei Gewichte: heute fett, Tag mit Erfassung normal, leerer Tag gedimmt — und
  in einem Zeitraum ohne heute ist keine Zeile fett.
- Die bestehenden Invarianten bleiben unangetastet grün: keine gerenderte Zeile
  breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl in Liste,
  Report, Übersicht und Zusammenfassung.
- Die Golden-Datei der Zusammenfassung bei 100×30 wird neu erzeugt.
- Ansehen bei 100, 80, 60 und 40 Spalten: der Balken verschwindet, statt
  falsche Verhältnisse zu zeigen.
- **Im echten Terminal ansehen, nicht nur im Test.** Die Entwürfe zu diesem
  Spec sind gerechnet, nicht gerendert. `░` (U+2591) ist ein Schattierungs-, kein
  Blockelement und fällt in Terminal-Schriften unterschiedlicher aus als die
  Achtelblöcke. Liest es sich nicht als Track, ist der Ersatz ein gedimmtes `█`
  oder `·` — die Bedeutung bleibt, nur das Zeichen wechselt.

## Non-Goals

- Keine Projektfarben
- Kein Sekundenformat, keine andere Aggregation, kein anderer Datenlayer
- Keine Änderung an Frameliste, Report, Übersicht oder Kopfzeile
- Kein Aufklappen, kein Sortierwechsel, keine dritte Darstellung
