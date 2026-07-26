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
| `[` / `]` | Zeitraum zurück/vor |
| `t`/`w`/`m`/`a` | Tag/Woche/Monat/alles |
| `r` | Report |
| `o` | Übersicht (Abrechnung) |
| `R` | neu laden |
| `?` | Hilfe |
| `q` | beenden |

Die Oberfläche nutzt die ANSI-Farben deines Terminal-Themes: Kontext-Header
oben, gerahmtes Hauptpanel, Tastenhinweise unten. Ab 20 Terminalzeilen ist der
Header gerahmt, zwischen 12 und 19 Zeilen klappt er auf eine Zeile ein, unter 12
verschwindet er — die Liste behält den Platz. Passen nicht alle Tastenhinweise in
die Breite, fallen die hinteren ganz weg statt mitten im Wort zu enden; `?` zeigt
immer die vollständige Belegung.

Auch die Frameliste richtet sich nach der Breite, und zwar in dieser Reihenfolge:
zuerst schrumpfen die Tags, dann der Projektname (beide mit `…`, unter etwa 60
bzw. 58 Spalten), unter etwa 44 Spalten fällt die ID-Spalte **ganz** weg, unter
etwa 26 auch Start- und Endzeit. Gekürzt wird nur, was ein Label ist. Die Dauer
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
  ausgewiesen — in der Übersicht und im Report gleich.
- Grundlage ist immer die lokale Zeitzone; der Wochenstart kommt aus Watsons
  `config` (`[options] week_start`, Standard Montag).

## Entwicklung

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Releases: Git-Tag `v*` pushen — GitHub Actions baut mit GoReleaser und
aktualisiert den Homebrew-Cask in [schnaq/homebrew-tap](https://github.com/schnaq/homebrew-tap).
