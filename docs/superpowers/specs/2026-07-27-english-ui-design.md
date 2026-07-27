# watson-tui — Oberfläche auf Englisch

**Datum:** 2026-07-27
**Status:** Approved
**Basis:** `main` bei v0.2.1

## Ziel

Die gesamte Oberfläche spricht Englisch. Das Repository ist öffentlich, der Code, die Kommentare und die Commit-Historie sind ohnehin englisch — die deutschen Texte waren der einzige Bruch. Rein sprachlich: kein Verhalten, keine Taste, keine Berechnung ändert sich.

## Umfang

Neun Produktionsdateien enthalten deutschen Text: `internal/tui/{app,chrome,frameform,frameslist,overview,periodsums,report,timer}.go` und `internal/watson/store.go`. Zehn Testdateien prüfen auf deutsche Zeichenketten.

Nicht betroffen: das Watson-Dateiformat, Projektnamen und Tags des Nutzers, Kommentare (schon englisch), die README (schon englisch — nur der Hinweis „the interface itself is German" entfällt).

## Datum und Zeit

ISO, damit Anzeige und Eingabe dieselbe Sprache sprechen — das Frame-Formular erwartet schon jetzt `2006-01-02 15:04`.

| Stelle | Vorher | Nachher |
|---|---|---|
| Tagesheader | `Montag, 20.07.2026` | `Monday, 2026-07-20` |
| Wochenlabel | `Woche 20.07. – 26.07.2026` | `Week 2026-07-20 – 2026-07-26` |
| Monatslabel | `Juli 2026` | `July 2026` |
| Jahreslabel | `Jahr 2026` | `Year 2026` |
| Uhrzeiten | `09:00–11:30` | unverändert |
| Formulareingabe | `2026-07-20 09:00` | unverändert |

`germanDays` und `germanMonths` in `frameslist.go` werden zu `weekdayNames` und `monthNames` mit englischen Werten.

**Breitenfolge:** das Wochenlabel wächst von 25 auf 28 Zeichen. Der Kopf und die Fußzeile ringen bereits um Platz, deshalb ist nach der Umstellung zu prüfen, ob die Zeitraum-Tasten und die beiden Ausgänge `?`/`q` bei 80 Spalten noch überleben; falls nicht, wird innerhalb der Hinweisgruppen umsortiert, nicht gekürzt.

## Glossar

Verbindlich, damit dieselbe Sache überall gleich heißt.

| Deutsch | Englisch |
|---|---|
| Zeitraum | Period |
| Summe | Total |
| Gesamt | Total |
| Frames | Frames |
| Projekte | Projects |
| Filter | Filter |
| Übersicht | Overview |
| Abrechnung | Billing |
| Report | Report |
| Hilfe | Help |
| Fehler | Error |
| Vortag / Vorwoche / Vormonat | Prev day / Prev week / Prev month |
| Tag / Woche / Monat / Jahr / alles | Day / Week / Month / Year / all |
| alle Frames | all frames |
| diese Woche / letzte Woche | this week / last week |
| dieser Monat / letzter Monat | this month / last month |
| kein Timer | no timer |
| `+ läuft` | `+ running` |
| `… läuft (…) und ist eingerechnet` | `… running (…), included` |
| bewegen / bearbeiten / neu / löschen | move / edit / new / delete |
| filtern / neu laden / beenden | filter / reload / quit |
| starten / stoppen / verwerfen | start / stop / discard |
| speichern / abbrechen | save / cancel |
| Feld / Vorschlag / Taste | field / suggestion / key |
| zurück | back |
| Bestätigen | Confirm |
| zu schmal für: | too narrow for: |
| keine Frames im Zeitraum | no frames in this period |
| keine Frames vorhanden | no frames yet |
| kein Treffer für Filter »x« | no match for filter “x” |

Die Anführungszeichen `»…«` werden zu `“…”`.

## Vollständige Meldungen

| Vorher | Nachher |
|---|---|
| `keine Frames im Zeitraum — n legt einen neuen an` | `no frames in this period — press n to add one` |
| `Frame löschen?` | `Delete frame?` |
| `Laufenden Timer verwerfen?` | `Discard the running timer?` |
| `y löschen · andere Taste abbrechen` | `y delete · any other key cancels` |
| `y verwerfen · andere Taste abbrechen` | `y discard · any other key cancels` |
| `beliebige Taste schließt die Hilfe` | `any key closes the help` |
| `beliebige Taste beendet watson-tui` | `any key quits watson-tui` |
| `frames-Datei nicht lesbar: %v` + `Backup: %s/frames.bak` | `cannot read the frames file: %v` + `backup: %s/frames.bak` |
| `state-Datei nicht lesbar: …` | `cannot read the state file: …` |
| `Löschen fehlgeschlagen: %v` | `delete failed: %v` |
| `Stop fehlgeschlagen: %v` | `stop failed: %v` |
| `Verwerfen fehlgeschlagen: %v` | `discard failed: %v` |
| `Speichern fehlgeschlagen: %v` | `save failed: %v` |
| `Timer starten fehlgeschlagen: %v` | `starting the timer failed: %v` |
| `Projekt fehlt` | `project is missing` |
| `Stop muss nach Start liegen` | `stop must be after start` |
| `ungültige Zeit %q (YYYY-MM-DD HH:MM oder HH:MM)` | `invalid time %q (YYYY-MM-DD HH:MM or HH:MM)` |
| `Überlappt mit anderem Frame — enter speichert trotzdem` | `overlaps another frame — enter saves anyway` |
| `es läuft bereits ein Timer` (store.go) | `a timer is already running` |
| `kein Timer aktiv` (store.go) | `no timer is running` |
| `watson-Verzeichnis ist von anderem Prozess gesperrt` (save.go) | `the watson directory is locked by another process` |

Formularbeschriftungen: `Projekt` → `Project`, `Start` → `Start`, `Stop` → `Stop`, `Tags` → `Tags`. Panel-Titel: `Frames` bleibt, `Frame bearbeiten (…)` → `Edit frame (…)`, `Neuer Frame` → `New frame`, `Timer starten` → `Start timer`.

## Hilfe und Tastenhinweise

Die Hilfe behält ihre zehn Zeilen und die Spaltenausrichtung (Beschreibung ab Spalte 15). Die Fußzeilen-Hinweise werden übersetzt; ihre Reihenfolge bleibt, sofern die Breitenprüfung nichts anderes erzwingt. Englisch ist hier eher kürzer als Deutsch (`n new` statt `n neu` ist gleich lang, `d delete` statt `d löschen` spart eins), es sollte also kein Platz verloren gehen.

## Tests

Die zehn Testdateien werden auf die englischen Texte umgestellt — **umgestellt, nicht gelöscht**. Sie sind das Sicherheitsnetz, das in diesem Projekt dreimal Zahlenfehler gefangen hat; eine Assertion zu entfernen, weil ihr Text sich ändert, wäre der teuerste Fehler dieses Durchgangs.

Zusätzlich:

- Ein Test, der belegt, dass keine Produktionsdatei mehr deutschen Text enthält: eine Suche über `internal/` nach Umlauten und nach einer Handvoll verräterischer Wörter (`Zeitraum`, `Summe`, `löschen`, `keine`, `läuft`), die nur in Kommentaren erlaubt sind — praktikabel, weil die Kommentare bereits englisch sind.
- Die Golden-Dateien werden neu erzeugt und **gelesen**, nicht blind übernommen.
- Die bestehenden Invarianten bleiben grün: keine gerenderte Zeile breiter als das Terminal, kein Rahmen höher, keine gekürzte Zahl oder ID in Liste, Report und Übersicht.
- Die Breitenprüfung aus dem Abschnitt „Datum und Zeit": `←`/`→`, `? help` und `q quit` überleben bei 80 Spalten.

## Non-Goals

- Keine Mehrsprachigkeit, kein Katalog, keine Locale-Erkennung — ein Werkzeug, eine Sprache
- Keine Änderung am Watson-Dateiformat oder an Nutzerdaten
- Keine Änderung an Tastenbelegung, Layout-Arithmetik oder Berechnungen
- Kein neues Datumsformat für die Formulareingabe (war schon ISO)
