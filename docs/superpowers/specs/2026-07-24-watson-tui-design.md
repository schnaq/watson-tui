# watson-tui — Design-Spezifikation

**Datum:** 2026-07-24
**Status:** Approved
**Repo:** github.com/schnaq/watson-tui

## Ziel

Schlanke, schnelle TUI für [Watson](https://jazzband.github.io/Watson/)-Zeiterfassungsdaten in Go. Frames anschauen, editieren, anlegen, löschen; Live-Tracking (Timer starten/stoppen); einfacher Report-View. Watson selbst wird **nicht** benötigt — die TUI implementiert das Watson-Dateiformat nativ und bleibt 100 % kompatibel.

## Kontext & Recherche-Ergebnisse

- Watson (jazzband/Watson, Python) ist seit Juli 2022 unmaintained (letzter Commit 2022-07-15, letztes Release 2.1.0). Das Datenformat ist de facto eingefroren → natives Reimplementieren ist risikoarm.
- Lizenz Watson: MIT. Es wird kein Watson-Code übernommen, nur das Dateiformat.
- Homebrew-Formel `watson` (2.1.0) existiert, wird aber nicht als Dependency gebraucht.

### Watson-Datenformat (Referenz)

Speicherort: `$WATSON_DIR`, sonst OS-App-Dir:

| OS | Pfad |
|---|---|
| macOS | `~/Library/Application Support/watson/` |
| Linux | `$XDG_CONFIG_HOME/watson/` bzw. `~/.config/watson/` |

Dateien (ohne Endung):

- **`frames`** — JSON-Array von 6-elementigen Arrays, Feldreihenfolge exakt:
  `[start:int64, stop:int64, project:string, id:string, tags:[]string, updated_at:int64]`
  - Zeiten: Unix-Sekunden, UTC. Anzeige lokal.
  - `id`: UUID v4 als 32-stelliger lowercase-Hex-String ohne Bindestriche. CLI/TUI zeigen 7-Zeichen-Präfix; Lookup per Präfix-Match.
  - Serialisierung: `json.dumps(..., indent=1, ensure_ascii=False)`-Äquivalent (Indent = 1 Space), ganze Datei wird bei jedem Save neu geschrieben.
  - Zeilenreihenfolge = Einfügereihenfolge, nicht garantiert chronologisch.
- **`state`** — laufender Timer: `{"project": string, "start": int64, "tags": []string}` oder `{}`.
- **`config`** — INI (`[backend]`, `[options]`, `[default_tags]`). Relevant für TUI: `options.week_start`, `options.date_format`, `options.time_format` (nur lesen; fehlende Werte → Defaults).
- **Schreib-Pattern** (wie Watson `safe_save`): in Temp-Datei schreiben → vorhandene Datei nach `<name>.bak` rotieren (eine Generation) → Temp-Datei per Rename an Ziel. Watson hat **kein** Locking; watson-tui ergänzt flock (Advisory-Lock, Best-Effort) gegen parallele Schreiber.

## Architektur

**Stack:** Go ≥ 1.24, [Bubble Tea](https://github.com/charmbracelet/bubbletea) + Bubbles + Lipgloss. Keine Laufzeit-Dependencies außerhalb der Binary.

```
cmd/watson-tui/main.go    # Entry, Flag-Parsing (--version, --dir override)
internal/watson/          # Datenlayer — importiert kein TUI
  paths.go                # Verzeichnis-Auflösung: --dir > WATSON_DIR > OS-App-Dir
  frames.go               # Frame-Typ, Parse/Serialize, Prefix-Lookup, Sortierung
  state.go                # State lesen/schreiben, Start/Stop/Cancel-Logik
  config.go               # INI lesen (read-only)
  save.go                 # safe-save (temp + rename + .bak) + flock
internal/tui/
  app.go                  # Root-Model, View-Switching, globale Keys
  frameslist.go           # Haupt-View: Tabelle mit Filter + Zeitraum
  frameform.go            # Edit-/Neu-Formular (ein Model für beide)
  report.go               # Aggregations-View
  statusbar.go            # laufender Timer (tickt sekündlich), Fehler-/Statuszeile
  keys.go, styles.go      # zentrale Keymap + Lipgloss-Styles
```

Datenfluss: TUI-Models halten eine in-memory Kopie der Frames (`[]Frame`). Jede Mutation (edit/create/delete/start/stop) geht durch `internal/watson`, das sofort persistiert (Load → Mutate → Save unter flock). Kein Hintergrund-Daemon, kein File-Watching in v1; manueller Reload via Taste.

## Views & Bedienung

### 1. Frames-Liste (Start-View)

Tabelle: Datum, Start–Stop, Dauer, Projekt, Tags, Kurz-ID. Gruppierung nach Tag (Datumszeile als Separator mit Tagessumme). Standardzeitraum: aktuelle Woche.

| Key | Aktion |
|---|---|
| `j`/`k`, `↓`/`↑` | navigieren |
| `enter` | Frame editieren |
| `n` | neuer Frame |
| `d` | löschen (mit Confirm-Prompt) |
| `s` | Timer starten (Projekt-Prompt) / laufenden stoppen |
| `S` | Timer canceln (verwirft laufenden Frame, mit Confirm) |
| `/` | Filter: Projekt/Tag/Volltext |
| `[` / `]` | Zeitraum zurück/vor (Woche) |
| `w`/`m`/`a` | Zeitraum Woche/Monat/alles |
| `r` | Report-View |
| `R` | Reload von Disk |
| `?` | Hilfe-Overlay |
| `q` / `ctrl+c` | beenden |

### 2. Frame-Formular (Edit & Neu)

Felder: Projekt (Autocomplete aus vorhandenen Projekten), Start (Datetime), Stop (Datetime), Tags (kommagetrennt, Autocomplete). `enter`/`ctrl+s` speichern, `esc` abbrechen.

Validierung:
- `stop > start`, beide parsebar (Formate: `YYYY-MM-DD HH:MM`, `HH:MM` = heute)
- Overlap mit existierenden Frames → Warnung anzeigen, Speichern trotzdem erlaubt (Watson erlaubt Overlaps auch)
- Neuer Frame: `id = uuid4().hex`, `updated_at = now`. Edit: `updated_at = now`, `id` bleibt.

### 3. Report-View

Summen pro Projekt, darunter Tag-Breakdown (wie `watson report`). Zeitraum: Tag/Woche/Monat (`t`/`w`/`m` umschalten, `[`/`]` verschieben). Rein lesend, berechnet aus geladenen Frames. `esc` zurück zur Liste.

### 4. Statusbar (global)

Läuft ein Timer: `▶ projekt [tags] 01:23:45` — tickt sekündlich (Bubble Tea `tick`). Sonst Zeitraum + Frame-Anzahl + Filterstatus. Fehler (z. B. Save fehlgeschlagen) erscheinen hier, nicht als Crash.

## Fehlerbehandlung

- Watson-Verzeichnis fehlt → wie leer behandeln; beim ersten Schreiben anlegen (`0700`).
- `frames`/`state` korrupt (JSON-Parse-Fehler) → Fehlerscreen mit Pfad + Hinweis auf `.bak`; **kein** Schreibzugriff, damit `.bak` intakt bleibt.
- Save-Fehler (Permissions, Disk) → Fehler in Statusbar, in-memory State bleibt, Retry möglich.
- flock nicht verfügbar (z. B. Netzwerk-FS) → Warnung, weiter ohne Lock (Best-Effort wie Watson selbst).

## Tests

- **Datenlayer:** Round-Trip-Tests gegen Fixture-Dateien im echten Watson-Format (inkl. Unicode-Projekte, leere Tags, `.bak`-Rotation, Prefix-Lookup, leeres/fehlendes Verzeichnis, korruptes JSON). Byte-genaue Serialisierungs-Golden-Tests (Indent = 1).
- **TUI:** Update-Funktionen als reine Unit-Tests (Key-Event rein, Model-Zustand raus). Kern-Flows (Liste → Edit → Save, Start → Stop) mit `charmbracelet/x/exp/teatest`.
- **CI-Gate:** `go test ./...` + `golangci-lint run` müssen grün sein.

## CI/CD & Distribution

**Runner:** self-hosted (Label `self-hosted`), Workflows:

- `.github/workflows/ci.yml` — Push auf `main` + PRs: Checkout, Go-Setup, `go test ./...`, `golangci-lint`.
- `.github/workflows/release.yml` — Tag `v*`: GoReleaser.

**GoReleaser** (`.goreleaser.yaml`):
- Builds: `darwin`/`linux` × `amd64`/`arm64`, `CGO_ENABLED=0`, `-trimpath`, Version via `ldflags` (`main.version`).
- Archive (tar.gz) + `checksums.txt` → GitHub Release.
- `brews`-Section: Formel-Push nach `schnaq/homebrew-tap` (Repo muss existieren; Secret `TAP_GITHUB_TOKEN` = PAT mit `contents:write` auf das Tap-Repo).

**Nutzer-Flow:**
```
brew install schnaq/tap/watson-tui   # Installation
brew upgrade                          # Updates
```
Kein eingebauter Self-Updater — Homebrew ist der Update-Kanal (gängigster Weg für Go-CLIs; Binaries im Release decken Nicht-brew-Nutzer ab).

## Non-Goals (v1)

- Watson-Sync (`crick`-Backend)
- Bulk-Operationen `merge`/`rename`
- Windows-Support
- File-Watching / Auto-Reload
- Schreiben der `config`-Datei

## Offene Punkte für die Implementierungsphase

- `schnaq/homebrew-tap`-Repo anlegen + `TAP_GITHUB_TOKEN`-Secret setzen (einmalig, manuell bzw. per `gh`).
- Prüfen, ob der self-hosted Runner Go-Toolchain hat oder `actions/setup-go` nutzt.
