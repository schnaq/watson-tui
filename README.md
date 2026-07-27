# watson-tui

Schlanke Terminal-UI für [Watson](https://jazzband.github.io/Watson/)-Zeiterfassung,
in Go. Liest und schreibt Watsons Datenformat direkt — eine Watson-Installation
ist nicht nötig, vorhandene Daten funktionieren sofort weiter.

## Installation

```sh
brew tap schnaq/tap
brew install --cask watson-tui
```

Der Einzeiler `brew install --cask schnaq/tap/watson-tui` funktioniert auch, ist
aber in Homebrew abgekündigt (siehe [Tap-Trust](https://docs.brew.sh/Tap-Trust)).

Updates kommen über `brew upgrade`. Homebrew-Casks sind macOS-only; unter Linux
das passende Archiv vom [Release](https://github.com/schnaq/watson-tui/releases)
laden (`linux_amd64` oder `linux_arm64`) und `watson-tui` in den `PATH` legen.

## Bedienung

`watson-tui` starten — zeigt die aktuelle Woche.

| Taste | Aktion |
|---|---|
| `j`/`k` | navigieren |
| `enter` | Frame editieren |
| `n` | neuer Frame |
| `d` | Frame löschen |
| `s` | Timer starten/stoppen |
| `S` | Timer verwerfen |
| `/` | filtern (Projekt, Tag, ID) |
| `←` / `→` (oder `[` / `]`) | Zeitraum zurück/vor |
| `t`/`w`/`m`/`a` | Tag/Woche/Monat/alles |
| `r` | Report |
| `o` | Übersicht (Abrechnung) |
| `R` | neu laden |
| `?` | Hilfe |
| `q` | beenden |

Der Kopf zeigt, welchen Zeitraum du betrachtest — die Winkel `‹ ›` bedeuten,
dass `←` und `→` ihn verschieben (`[` und `]` tun dasselbe) —, dazu die Summe
dieses Zeitraums, den Vergleich zum Vorzeitraum und zur nächstgrößeren Einheit
(bei einer Woche also Vorwoche und Monat), die Zahl der Frames und Projekte
sowie einen laufenden Timer.
`alle Frames` lässt sich nicht verschieben und bekommt deshalb keine Winkel und
keinen Vergleich. Die Summe im Kopf der Liste entspricht der Liste darunter und
zählt einen laufenden Timer deshalb nicht mit; steht `+ läuft` dahinter, ist
genau das der Grund, warum Report und Übersicht für denselben Zeitraum mehr
anzeigen — deren Köpfe zählen ihn mit, weil ihre Tabellen es auch tun.

Die Oberfläche nutzt die ANSI-Farben deines Terminal-Themes: Kontext-Header
oben, gerahmtes Hauptpanel, Tastenhinweise unten. Das Chrome gibt seinen Platz
in vier Stufen an die Liste zurück: ab 24 Terminalzeilen ist der Header gerahmt
und zeigt vier Feldzeilen über zwei Hinweiszeilen, zwischen 20 und 23 Zeilen
bleibt er gerahmt mit drei Feldzeilen und einer Hinweiszeile, zwischen 14 und 19
klappt er auf eine Zeile ein, unter 14 verschwindet er ganz. Passen nicht alle
Tastenhinweise in die Breite, fallen die hinteren ganz weg statt mitten im Wort
zu enden; bleibt nur eine Hinweiszeile, rücken die Zeitraum-Tasten und die
beiden Ausgänge `?` und `q` in sie zusammen. `?` zeigt die vollständige
Belegung; sie passt in jedes Terminal, das noch einen Kopf hat (ab 14 Zeilen),
darunter fehlt ihr Ende.

Auch die Frameliste richtet sich nach der Breite, und zwar in dieser Reihenfolge:
ab etwa 88 Spalten schrumpfen die Tags (mit `…`), unter etwa 60 sind sie weg und
der Projektname schrumpft, unter etwa 42 fällt die ID-Spalte **ganz** weg, unter
etwa 24 auch Start- und Endzeit. Gekürzt wird nur, was ein Label ist. Die Dauer
bleibt immer vollständig, und die ID steht entweder mit allen sieben Zeichen da
oder gar nicht: Watson löst IDs über ihr Präfix auf, eine gekürzte ID würde also
mehrere oder den falschen Frame treffen. Vollständig steht sie im
Bearbeiten-Dialog (`enter`) und in der Löschabfrage (`d`). Die genauen Schwellen
hängen an der breitesten Dauer der Liste — `130h 00m` braucht eine Spalte mehr
als `6h 30m`.

Datenverzeichnis: `$WATSON_DIR`, sonst OS-Standard
(macOS: `~/Library/Application Support/watson`). Override: `--dir PFAD`.

### Report und Übersicht

Der Report (`r`) summiert den gewählten Zeitraum pro Projekt, darunter die
Tag-Summen. Tags werden dabei einzeln gezählt: ein Frame mit zwei Tags zählt in
beide — die Tag-Summen können deshalb größer sein als die Projektsumme. Sie
beantworten „wie viel Zeit pro Tag", nicht „wie teilt sich das Projekt auf".

Die Übersicht (`o`) ist für die Abrechnung: Summen pro Projekt über diese
Woche, letzte Woche, diesen Monat, letzten Monat und gesamt. Die Tabelle passt
sich der Terminalbreite an: unter etwa 94 Spalten werden die Spaltentitel
gekürzt (`Vorwoche`, `Vormonat`), unter etwa 70 schrumpft zusätzlich die
Projektspalte. Unter etwa 62 Spalten fallen ganze Wert-Spalten weg — welche,
steht über der Tabelle; `gesamt` bleibt immer stehen. So wird keine Dauer
abgeschnitten.

**Wie Zeit einem Zeitraum zugeordnet wird** — relevant, wenn du daraus
Rechnungen schreibst:

- Ein Frame zählt vollständig in den Zeitraum, in dem er **beginnt**. Eine
  Sitzung vom 31.07. 23:00 bis 01.08. 02:00 zählt also mit drei Stunden in den
  Juli, nicht gesplittet. Watson selbst rechnet genauso.
- Ein laufender Timer zählt bis jetzt mit und wird unter der Tabelle
  ausgewiesen — in der Übersicht und im Report gleich, und in deren Kopfsummen
  ebenso. Nur die Frameliste lässt ihn aus ihrer Kopfsumme heraus, damit diese
  mit den Tagessummen darunter zusammengeht, und schreibt `+ läuft` dahinter.
- Grundlage ist immer die lokale Zeitzone; der Wochenstart kommt aus Watsons
  `config` (`[options] week_start`, Standard Montag).

## Entwicklung

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Releases: Git-Tag `v*` pushen — GitHub Actions baut mit GoReleaser und
aktualisiert den Homebrew-Cask in [schnaq/homebrew-tap](https://github.com/schnaq/homebrew-tap).
