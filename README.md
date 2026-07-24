# watson-tui

Schlanke Terminal-UI für [Watson](https://jazzband.github.io/Watson/)-Zeiterfassung,
in Go. Liest und schreibt Watsons Datenformat direkt — eine Watson-Installation
ist nicht nötig, vorhandene Daten funktionieren sofort weiter.

## Installation

```sh
brew install --cask schnaq/tap/watson-tui
```

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

Datenverzeichnis: `$WATSON_DIR`, sonst OS-Standard
(macOS: `~/Library/Application Support/watson`). Override: `--dir PFAD`.

### Report und Übersicht

Der Report (`r`) summiert den gewählten Zeitraum pro Projekt, darunter die
Tag-Summen. Tags werden dabei einzeln gezählt: ein Frame mit zwei Tags zählt in
beide — die Tag-Summen können deshalb größer sein als die Projektsumme. Sie
beantworten „wie viel Zeit pro Tag", nicht „wie teilt sich das Projekt auf".

Die Übersicht (`o`) ist für die Abrechnung: Summen pro Projekt über diese
Woche, letzte Woche, diesen Monat, letzten Monat und gesamt. Ein laufender
Timer zählt erst mit, wenn er gestoppt ist.

## Entwicklung

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Releases: Git-Tag `v*` pushen — GitHub Actions baut mit GoReleaser und
aktualisiert den Homebrew-Cask in [schnaq/homebrew-tap](https://github.com/schnaq/homebrew-tap).
