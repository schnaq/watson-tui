# Cloud-Handover: watson-tui

> Übergabe-Dokument für die Fortsetzung in einer neuen Claude-Session (claude.ai/code oder lokal).
> Stand: 2026-07-24. Nach Projektabschluss löschen.

## Projekt

Native Go-TUI für [Watson](https://jazzband.github.io/Watson/)-Zeiterfassung. Liest/schreibt Watsons Datenformat direkt (byte-kompatibel), keine Watson-Installation nötig. Bubble Tea. Release via GoReleaser auf self-hosted Runner, Distribution über Homebrew-Tap `schnaq/homebrew-tap`.

- **Spec:** `docs/superpowers/specs/2026-07-24-watson-tui-design.md`
- **Plan (maßgeblich, enthält kompletten Code pro Task):** `docs/superpowers/plans/2026-07-24-watson-tui.md`
- **Branch:** `feature/watson-tui` (von `main`, wird am Ende gemergt)
- **Linear:** Projekt [watson-tui](https://linear.app/schnaq/project/watson-tui-1e6d5128b26f/overview), Team `schnaq`

## Arbeitsweise (bitte beibehalten)

Subagent-driven Development nach `superpowers:subagent-driven-development`:
1. Pro Task: BASE-Commit merken → Task-Text aus dem Plan extrahieren (Abschnitt `### Task N:`) → **Implementer-Subagent (Modell: Opus)** mit Task-Brief dispatchen (TDD: Test zuerst, RED verifizieren, dann implementieren; Extra-Tests für offensichtliche Fehlerpfad-Lücken erwünscht)
2. Danach **Task-Review-Subagent (Sonnet; Opus bei Integrationsrisiko)** über den Diff BASE..HEAD; Critical/Important-Findings → Fix-Subagent → Re-Review
3. Task fertig → Commit liegt vor → **pushen** → **Linear-Issue auf Done** setzen (nächstes auf In Progress)
4. Commit-Messages: conventional (`feat:`/`test:`/`ci:`/`docs:`), Trailer: `Co-Authored-By: <Modell der Session> <noreply@anthropic.com>` plus `Claude-Session:`-Zeile der jeweiligen Session
   – Tasks 16, 14 und 15 liefen in einer Session, in der Subagents per Session-Policy gesperrt waren: TDD inline, kein separates Review-Subagent. Alles nach `e504482` (`e504482..HEAD`, aktuell bis `e5eb9da`) ist unreviewed — beim Final-Review mit abdecken.
5. UI-Texte deutsch, Code/Bezeichner englisch. Erlaubte Deps: charm-Stack (bubbletea/bubbles/lipgloss), gofrs/flock, google/uuid, ini.v1 — sonst nichts.

## Status: 15 von 16 Tasks fertig

| Task | Inhalt | Linear | Status |
|---|---|---|---|
| 1 | Scaffold (go.mod, LICENSE, --version) | SNQ-575 | ✅ Done |
| 2 | paths.go (Dir-Auflösung) | SNQ-576 | ✅ Done |
| 3 | frames.go (byte-kompatible Serialisierung) | SNQ-577 | ✅ Done |
| 4 | save.go (safe-save + flock) | SNQ-578 | ✅ Done (inkl. Fix: fail-open bei Lock-Contention behoben) |
| 5 | state.go | SNQ-579 | ✅ Done (inkl. Test-Härtung) |
| 6 | config.go (week_start) | SNQ-580 | ✅ Done (inkl. Fehlerpfad-Test) |
| 7 | store.go (gelockte Mutationen) | SNQ-581 | ✅ Done — Milestone 1 komplett |
| 8 | TUI-Grundgerüst (App/Statusbar/Hilfe/Fatal) | SNQ-582 | ✅ Done |
| 9 | Frames-Liste (Haupt-View) | SNQ-583 | ✅ Done |
| 10 | Frame-Formular (Edit/Neu) | SNQ-584 | ✅ Done |
| 11 | Löschen + Reload | SNQ-585 | ✅ Done |
| 12 | Live-Timer (s/S) | SNQ-586 | ✅ Done |
| 13 | Report-View (r) | SNQ-587 | ✅ Done |
| 16 | Abrechnungs-Übersicht (o) | SNQ-590 | ✅ Done (inkl. Extra-Tests; kein Subagent-Review, s. u.) |
| 14 | CI-Workflow | SNQ-588 | ✅ Done — CI grün auf `b3d878a` (Jobs `test` + `lint`) |
| **15** | **Release-Pipeline + README** | **SNQ-589** | 🟡 **Vorarbeit committet, Release blockiert (s. u.)** |

Letzter Commit auf `feature/watson-tui`: `500761b` (Task 15, Vorarbeit).

## PR und CI

Draft-PR [#1](https://github.com/schnaq/watson-tui/pull/1) `feature/watson-tui` → `main` ist offen; `.github/workflows/ci.yml` triggert auf `push: main` und `pull_request`, läuft also bei jedem Push auf den Branch. Der Lauf zu `b3d878a` war grün (test + lint auf dem self-hosted Runner). Der PR ist bewusst Draft — echter Merge erst nach Final-Review.

- Lokal ist das Gate grün: `go build ./... && go test ./... && go vet ./... && golangci-lint run ./...` (0 Findings).
- `.golangci.yml` (v2-Format) aktiviert staticcheck mit `all` minus `ST1005` (deutsche Fehlertexte sind UI-Text, Substantive groß). Weil `all` auch künftige Checks einschaltet, ist die Lint-Version im Workflow auf `v2.12.2` gepinnt (lokal verifizierte Version) — beim Hochziehen `golangci-lint run ./...` lokal gegenläufig prüfen. `all` ist strenger als der golangci-lint-Default — deshalb kamen ST1000/QF1012 dazu und sind in `93d2eff` gefixt (Package-Docs, `fmt.Fprintf` statt `WriteString(fmt.Sprintf(...))`).
- Runner-Check: `gh api repos/schnaq/watson-tui/actions/runners` liefert 0 (repo-level); Org-Ebene ist mit dem aktuellen Token nicht abfragbar (403, braucht `admin:org`). Laut User-Vorgabe existieren die Runner auf Org-Ebene — deshalb nicht blockiert.
- Runner: `runs-on: self-hosted` — matcht **gimli (macOS)** oder **OpenSuse (Linux)**. Workflows OS-agnostisch halten (actions/setup-go, kein brew in CI).

## Task 15: Stand und offene Schritte

Fertig und committet (`500761b`): `.goreleaser.yaml`, `.github/workflows/release.yml`, `README.md`.

- `goreleaser check` valide; Snapshot-Build lokal verifiziert (4 Archive darwin/linux × amd64/arm64, Cask generiert, `--version` gibt die injizierte Version aus).
- **Plan-Abweichung:** Der Plan nutzt `brews:` — GoReleaser hat das in v2.16 entfernt. Stattdessen `homebrew_casks:` mit `binaries: [watson-tui]` und `postflight`-Hook, der das Quarantine-Flag entfernt (Binary ist unsigniert). Folgen: Installation ist `brew install --cask schnaq/tap/watson-tui`, Casks sind **macOS-only** (Linux → Release-Archiv), und die `test`-Stanza aus dem Plan fällt weg (Casks haben keine). goreleaser-action ist auf `~> v2.16` gepinnt.
- Tap-Repo `schnaq/homebrew-tap` existiert, ist aber **leer (kein initialer Commit, kein default branch)**. Vor dem ersten Release prüfen, ob GoReleaser dorthin pushen kann — sonst einmal mit README initialisieren.

Offene Schritte (Plan Task 15, Steps 6 und 8–10):

1. **USER ACTION, blockierend:** Fine-grained PAT für `schnaq/homebrew-tap` (Contents: Read+Write) erstellen und als Secret setzen: `gh secret set TAP_GITHUB_TOKEN --repo schnaq/watson-tui`. `gh secret list --repo schnaq/watson-tui` ist derzeit leer. STOPP bis Secret da ist.
2. Tag `v0.1.0` pushen → `gh run watch --exit-status` → Release-Assets prüfen (4 tar.gz + checksums.txt) → `watson-tui.rb` im Tap unter `Casks/`.
3. Smoke-Test: `brew install --cask schnaq/tap/watson-tui && watson-tui --version` → `watson-tui 0.1.0`.

## Nach Task 15: Abschluss

1. **Final-Review** (whole-branch, fähigstes verfügbares Modell) über `git merge-base main HEAD`..HEAD — dabei die unten gesammelten Minor-Findings triagieren
2. Findings fixen (EIN Fix-Subagent mit kompletter Liste)
3. Merge nach `main` (superpowers:finishing-a-development-branch), Linear-Issues final prüfen

## Gesammelte Minor-Findings für den Final-Review

Behoben nach dem ersten Durchgang (Commit siehe `git log --oneline`): Übersicht ist breitenadaptiv, laufender Timer zählt mit und wird ausgewiesen, Statusbar zeigt in Report und Übersicht die eigene View, `truncate` hat einen Guard für `max<=0`, Tag-Semantik des Reports steht im README.

- Task 3: null-stop-Pfad und `Duration()` ohne Testabdeckung (frames_test.go)
- Task 7: UTC-Constraint ohne Test-Teeth im Store; „never overwrite" nicht via Byte-Erhalt getestet; corrupt-config→Default ungetestet
- Task 9: Scroll-Test rows>height fehlt; q-in-Filter-Regressionstest fehlt; g/G ohne Integrationstest
- Task 10: `warned`-Flag ist One-Way-Latch (kein Re-Check nach Feldänderung); `frameID[:7]`-Panic theoretisch möglich; Sekunden gehen beim Edit-Roundtrip verloren (dtLayout minutengenau)
- Task 12: Start-Fehler ohne deutschen Prefix in Prompt-errMsg; Autocomplete-Accept (right/ctrl+e) ungetestet
- Task 13/16: Übersicht rendert die Spaltenüberschriften erst ab ~94 Spalten voll; darunter greifen Kurztitel (`Vorwoche`/`Vormonat`) und ab ~70 Spalten schrumpft die Projektspalte. Unter 70 Spalten bleibt die Tabelle zu breit — kein Scrolling.
- Task 14: CI prüft kein `gofmt`/`gci` — Formatverstöße fallen nur lokal auf (`gofmt -l .`)
- Task 14/15: CI-Annotations melden Node-20-Deprecation für `actions/checkout@v4` und `actions/setup-go@v5` (werden auf Node 24 gezwungen) sowie einen fehlgeschlagenen Cache-Restore (`/usr/bin/tar` exit 2) auf dem Runner — beides nur Warnungen, Actions-Versionen beim nächsten Anlass hochziehen
- Task 15: Release läuft auf `runs-on: self-hosted`, also auf gimli **oder** OpenSuse; `CGO_ENABLED=0` macht die Builds plattformunabhängig, aber der Cask-Push hängt am Runner-Netzzugang zu GitHub

## Verifikation nach jedem Task

```sh
go build ./... && go test ./... && go vet ./...
```

Manueller Smoke-Test der TUI: `WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui`
