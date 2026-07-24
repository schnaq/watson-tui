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
   – Task 16 lief in einer Session, in der Subagents per Session-Policy gesperrt waren: TDD inline, kein separates Review-Subagent. Diff `fd3d7f6..2f363dd` beim Final-Review mit abdecken.
5. UI-Texte deutsch, Code/Bezeichner englisch. Erlaubte Deps: charm-Stack (bubbletea/bubbles/lipgloss), gofrs/flock, google/uuid, ini.v1 — sonst nichts.

## Status: 14 von 16 Tasks fertig

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
| **14** | **CI-Workflow** | **SNQ-588** | ⬜ **NÄCHSTER SCHRITT** |
| 15 | Release-Pipeline + README | SNQ-589 | ⬜ offen |

Letzter Commit auf `feature/watson-tui`: `2f363dd` (Task 16). Reihenfolge der Rest-Tasks: **14 → 15**.

## Nächster Schritt: Task 14 (CI), danach Task 15 (Release)

- Runner: `runs-on: self-hosted` — matcht **gimli (macOS)** oder **OpenSuse (Linux)**. Workflows OS-agnostisch halten (actions/setup-go, kein brew in CI).
- Task 15 enthält **eine USER ACTION**: Fine-grained PAT für `schnaq/homebrew-tap` (Contents: Read+Write) erstellen und als Secret setzen: `gh secret set TAP_GITHUB_TOKEN --repo schnaq/watson-tui`. Davor Tap-Repo anlegen (`gh repo create schnaq/homebrew-tap --public`). STOPP bis Secret da ist.
- Release: Tag `v0.1.0` pushen → GoReleaser → Release + Formel im Tap → `brew install schnaq/tap/watson-tui` verifizieren.

## Nach Task 15: Abschluss

1. **Final-Review** (whole-branch, fähigstes verfügbares Modell) über `git merge-base main HEAD`..HEAD — dabei die unten gesammelten Minor-Findings triagieren
2. Findings fixen (EIN Fix-Subagent mit kompletter Liste)
3. Merge nach `main` (superpowers:finishing-a-development-branch), Linear-Issues final prüfen

## Gesammelte Minor-Findings für den Final-Review

- Task 3: null-stop-Pfad und `Duration()` ohne Testabdeckung (frames_test.go)
- Task 7: UTC-Constraint ohne Test-Teeth im Store; „never overwrite" nicht via Byte-Erhalt getestet; corrupt-config→Default ungetestet
- Task 9: Scroll-Test rows>height fehlt; q-in-Filter-Regressionstest fehlt; g/G ohne Integrationstest; `truncate` panict bei max<=0 (defensive guard einbauen)
- Task 10: `warned`-Flag ist One-Way-Latch (kein Re-Check nach Feldänderung); `frameID[:7]`-Panic theoretisch möglich; Sekunden gehen beim Edit-Roundtrip verloren (dtLayout minutengenau)
- Task 12: Start-Fehler ohne deutschen Prefix in Prompt-errMsg; Autocomplete-Accept (right/ctrl+e) ungetestet
- Task 13: Statusbar zeigt im Report die Listen-Periode; Tag-Summen > Projektsumme bei Mehrfach-Tags (als „Zeit pro Tag" dokumentieren, z. B. im README)
- Task 16: Zeilenbreite der Übersicht ist fix ~94 Spalten (24 + 5×14), ohne Umbruch-/Scroll-Handling — bricht auf 80-Spalten-Terminals um (`App`-Default ist width 80)
- Task 16: laufender Timer fehlt in den Summen (`overviewView` liest nur `a.frames`, nicht `a.state`) — in der Abrechnung ggf. überraschend
- Task 16: Statusbar zeigt in der Übersicht weiter die Listen-Periode (gleiches Muster wie Task 13)
- Vorbestehend: `internal/tui/frameform_test.go` ist nicht gofmt-konform (`gofmt -l .`), seit `4b22852`

## Verifikation nach jedem Task

```sh
go build ./... && go test ./... && go vet ./...
```

Manueller Smoke-Test der TUI: `WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui`
