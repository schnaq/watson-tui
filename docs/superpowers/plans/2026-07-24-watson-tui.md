# watson-tui Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> **Modell-Vorgabe des Users:** Implementierungs-Subagents mit **Opus** ausführen.

**Goal:** Native Go-TUI für Watson-Zeiterfassungsdaten — Frames anschauen/editieren/anlegen/löschen, Live-Timer, Report — plus Release-Pipeline (GoReleaser, self-hosted Runner, Homebrew-Tap).

**Architecture:** Zwei Schichten: `internal/watson` (Datenlayer, liest/schreibt Watsons `frames`/`state`/`config` byte-kompatibel, kein TUI-Import) und `internal/tui` (Bubble-Tea-Models). Jede Mutation geht synchron durch den Store (Load → Mutate → Save unter flock). Kein Daemon, kein File-Watching.

**Tech Stack:** Go ≥ 1.24, charmbracelet/bubbletea + bubbles + lipgloss, gofrs/flock, google/uuid, gopkg.in/ini.v1, GoReleaser v2, GitHub Actions (self-hosted).

**Spec:** `docs/superpowers/specs/2026-07-24-watson-tui-design.md`

**Tracking:** Linear-Projekt [watson-tui](https://linear.app/schnaq/project/watson-tui-1e6d5128b26f/overview) — pro Task ein Issue. Nach Abschluss eines Tasks (Tests grün, committed) das zugehörige Linear-Issue auf Done setzen.

## Global Constraints

- Modul: `github.com/schnaq/watson-tui`, Go ≥ 1.24
- Erlaubte Dependencies (direkt): `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`, `github.com/gofrs/flock`, `github.com/google/uuid`, `gopkg.in/ini.v1` — sonst nichts
- Watson-Format byte-kompatibel: JSON mit Indent = **1 Space**, **kein** HTML-Escaping (`<`,`>`,`&`,`ü` bleiben roh), **kein** trailing Newline, Frames als Array-of-Arrays `[start, stop, project, id, tags, updated_at]`, Unix-Sekunden UTC
- Zeiten intern `time.Time` in UTC, Anzeige lokal
- IDs: UUID v4 als 32-stelliger lowercase-Hex ohne Bindestriche; Anzeige 7 Zeichen
- UI-Texte deutsch, Code/Bezeichner englisch
- Nach jedem Task: `go build ./... && go test ./...` grün; Commit-Präfixe `feat:`/`test:`/`chore:`/`ci:`/`docs:`
- Fehlende Datendateien = leer; korrupte Dateien = Fatal-Screen ohne jeden Schreibzugriff

---

### Task 1: Projekt-Scaffold

**Files:**
- Create: `go.mod`, `.gitignore`, `LICENSE`, `cmd/watson-tui/main.go`

**Interfaces:**
- Produces: Modulpfad `github.com/schnaq/watson-tui`, `main.version`-Variable (ldflags-Hook für GoReleaser)

- [ ] **Step 1: Modul initialisieren**

```bash
cd /Users/n2o/repos/schnaq/watson-tui
go mod init github.com/schnaq/watson-tui
```

- [ ] **Step 2: `.gitignore` schreiben**

```gitignore
/watson-tui
/dist/
*.test
```

- [ ] **Step 3: `LICENSE` schreiben (MIT)**

MIT-Standardtext, Copyright-Zeile: `Copyright (c) 2026 schnaq GmbH`

- [ ] **Step 4: `cmd/watson-tui/main.go` schreiben**

```go
package main

import (
	"flag"
	"fmt"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("watson-tui", version)
		return
	}
	fmt.Println("watson-tui: TUI kommt in Task 8")
}
```

- [ ] **Step 5: Build prüfen**

Run: `go build ./... && go run ./cmd/watson-tui --version`
Expected: `watson-tui dev`

- [ ] **Step 6: Commit**

```bash
git add go.mod .gitignore LICENSE cmd/
git commit -m "chore: scaffold Go module with version flag"
```

---

### Task 2: Datenverzeichnis-Auflösung (`paths.go`)

**Files:**
- Create: `internal/watson/paths.go`
- Test: `internal/watson/paths_test.go`

**Interfaces:**
- Produces: `watson.Dir(override string) (string, error)` — Auflösung: `override` > `$WATSON_DIR` > `os.UserConfigDir()/watson` (macOS: `~/Library/Application Support/watson`, Linux: `$XDG_CONFIG_HOME/watson` bzw. `~/.config/watson` — deckt sich mit Pythons `click.get_app_dir`)

- [ ] **Step 1: Failing Test schreiben**

```go
package watson

import (
	"path/filepath"
	"testing"
)

func TestDirOverrideWins(t *testing.T) {
	t.Setenv("WATSON_DIR", "/env/watson")
	got, err := Dir("/explicit")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/explicit" {
		t.Errorf("got %q, want /explicit", got)
	}
}

func TestDirEnvVar(t *testing.T) {
	t.Setenv("WATSON_DIR", "/env/watson")
	got, err := Dir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/env/watson" {
		t.Errorf("got %q, want /env/watson", got)
	}
}

func TestDirDefaultEndsWithWatson(t *testing.T) {
	t.Setenv("WATSON_DIR", "")
	got, err := Dir("")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "watson" {
		t.Errorf("got %q, want path ending in /watson", got)
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/watson/ -run TestDir -v`
Expected: FAIL — `undefined: Dir`

- [ ] **Step 3: Implementieren**

```go
package watson

import (
	"os"
	"path/filepath"
)

// Dir resolves the Watson data directory: explicit override > $WATSON_DIR > OS app dir.
func Dir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if d := os.Getenv("WATSON_DIR"); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "watson"), nil
}
```

- [ ] **Step 4: Test läuft grün**

Run: `go test ./internal/watson/ -run TestDir -v`
Expected: PASS (3 Tests)

- [ ] **Step 5: Commit**

```bash
git add internal/watson/
git commit -m "feat: resolve watson data directory (override > WATSON_DIR > app dir)"
```

---

### Task 3: Frame-Typ & Serialisierung (`frames.go`)

**Files:**
- Create: `internal/watson/frames.go`
- Test: `internal/watson/frames_test.go`

**Interfaces:**
- Produces:
  - `type Frame struct { Start, Stop time.Time; Project, ID string; Tags []string; UpdatedAt time.Time }`
  - `func (f Frame) Duration() time.Duration`
  - `func (f Frame) ShortID() string` — erste 7 Zeichen
  - `func ParseFrames(data []byte) ([]Frame, error)`
  - `func MarshalFrames(frames []Frame) ([]byte, error)` — byte-kompatibel zu Python `json.dumps(..., indent=1, ensure_ascii=False)`, kein trailing Newline
  - `func SortFrames(frames []Frame)` — nach Start aufsteigend, stabil
  - `func FindByIDPrefix(frames []Frame, prefix string) (int, error)`
  - `var ErrNotFound, ErrAmbiguous error`

- [ ] **Step 1: Failing Tests schreiben**

Golden-Fixture ist exakt der Output von Python `json.dumps(rows, indent=1, ensure_ascii=False)`:

```go
package watson

import (
	"testing"
	"time"
)

const goldenFrames = `[
 [
  1658000000,
  1658003600,
  "watson-tui",
  "9f3a2cf607e34b1c8f0e0c1234567890",
  [
   "cli",
   "docs"
  ],
  1658003600
 ],
 [
  1658007200,
  1658010800,
  "büro & R&D",
  "0aa2b3c4d5e6f708192a3b4c5d6e7f80",
  [],
  1658010800
 ]
]`

func TestParseFramesGolden(t *testing.T) {
	frames, err := ParseFrames([]byte(goldenFrames))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	f := frames[0]
	if !f.Start.Equal(time.Unix(1658000000, 0)) {
		t.Errorf("start = %v", f.Start)
	}
	if !f.Stop.Equal(time.Unix(1658003600, 0)) {
		t.Errorf("stop = %v", f.Stop)
	}
	if f.Project != "watson-tui" {
		t.Errorf("project = %q", f.Project)
	}
	if f.ID != "9f3a2cf607e34b1c8f0e0c1234567890" {
		t.Errorf("id = %q", f.ID)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "cli" || f.Tags[1] != "docs" {
		t.Errorf("tags = %v", f.Tags)
	}
	if frames[1].Project != "büro & R&D" {
		t.Errorf("unicode/html project = %q", frames[1].Project)
	}
	if len(frames[1].Tags) != 0 {
		t.Errorf("empty tags = %v", frames[1].Tags)
	}
}

func TestMarshalFramesRoundTripBytes(t *testing.T) {
	frames, err := ParseFrames([]byte(goldenFrames))
	if err != nil {
		t.Fatal(err)
	}
	out, err := MarshalFrames(frames)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenFrames {
		t.Errorf("round trip mismatch:\ngot:\n%s\nwant:\n%s", out, goldenFrames)
	}
}

func TestMarshalFramesEmpty(t *testing.T) {
	out, err := MarshalFrames(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "[]" {
		t.Errorf("got %q, want []", out)
	}
}

func TestParseFramesErrors(t *testing.T) {
	if _, err := ParseFrames([]byte("{ nope")); err == nil {
		t.Error("want error for corrupt json")
	}
	if _, err := ParseFrames([]byte(`[[1, 2, "p"]]`)); err == nil {
		t.Error("want error for short row")
	}
}

func TestFindByIDPrefix(t *testing.T) {
	frames := []Frame{
		{ID: "9f3a2cf607e34b1c8f0e0c1234567890"},
		{ID: "9f9999f607e34b1c8f0e0c1234567890"},
		{ID: "0aa2b3c4d5e6f708192a3b4c5d6e7f80"},
	}
	if i, err := FindByIDPrefix(frames, "0aa2b3c"); err != nil || i != 2 {
		t.Errorf("got (%d, %v), want (2, nil)", i, err)
	}
	if _, err := FindByIDPrefix(frames, "9f"); err != ErrAmbiguous {
		t.Errorf("got %v, want ErrAmbiguous", err)
	}
	if _, err := FindByIDPrefix(frames, "ffff"); err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestSortFrames(t *testing.T) {
	frames := []Frame{
		{ID: "b", Start: time.Unix(200, 0)},
		{ID: "a", Start: time.Unix(100, 0)},
	}
	SortFrames(frames)
	if frames[0].ID != "a" {
		t.Errorf("not sorted by start: %v", frames)
	}
}

func TestShortID(t *testing.T) {
	f := Frame{ID: "9f3a2cf607e34b1c8f0e0c1234567890"}
	if f.ShortID() != "9f3a2cf" {
		t.Errorf("got %q", f.ShortID())
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/watson/ -run 'TestParse|TestMarshal|TestFind|TestSort|TestShort' -v`
Expected: FAIL — `undefined: ParseFrames` etc.

- [ ] **Step 3: Implementieren**

Zwei Fallstricke, deshalb genau so bauen: (a) `encoding/json` escapt per Default `<>&` — deshalb überall Encoder mit `SetEscapeHTML(false)`; (b) Output von Custom-`MarshalJSON` wird vom äußeren Encoder korrekt re-indentiert, innen darf also kompakt gebaut werden.

```go
package watson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound  = errors.New("frame not found")
	ErrAmbiguous = errors.New("frame id prefix is ambiguous")
)

// Frame is one completed tracking entry. Times are UTC.
type Frame struct {
	Start     time.Time
	Stop      time.Time
	Project   string
	ID        string // 32-char lowercase hex UUID v4, no dashes
	Tags      []string
	UpdatedAt time.Time
}

func (f Frame) Duration() time.Duration { return f.Stop.Sub(f.Start) }

func (f Frame) ShortID() string {
	if len(f.ID) < 7 {
		return f.ID
	}
	return f.ID[:7]
}

// marshalNoEscape is json.Marshal without HTML escaping and trailing newline.
func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// MarshalJSON emits Watson's array form [start, stop, project, id, tags, updated_at].
func (f Frame) MarshalJSON() ([]byte, error) {
	tags := f.Tags
	if tags == nil {
		tags = []string{}
	}
	return marshalNoEscape([]any{
		f.Start.Unix(),
		f.Stop.Unix(),
		f.Project,
		f.ID,
		tags,
		f.UpdatedAt.Unix(),
	})
}

// UnmarshalJSON reads Watson's array form. stop may be null (defensive).
func (f *Frame) UnmarshalJSON(data []byte) error {
	var row []json.RawMessage
	if err := json.Unmarshal(data, &row); err != nil {
		return err
	}
	if len(row) != 6 {
		return fmt.Errorf("frame row has %d fields, want 6", len(row))
	}
	var start, updated int64
	var stop *int64
	fields := []struct {
		dst  any
		name string
	}{
		{&start, "start"}, {&stop, "stop"}, {&f.Project, "project"},
		{&f.ID, "id"}, {&f.Tags, "tags"}, {&updated, "updated_at"},
	}
	for i, fd := range fields {
		if err := json.Unmarshal(row[i], fd.dst); err != nil {
			return fmt.Errorf("frame field %s: %w", fd.name, err)
		}
	}
	f.Start = time.Unix(start, 0).UTC()
	if stop != nil {
		f.Stop = time.Unix(*stop, 0).UTC()
	} else {
		f.Stop = time.Time{}
	}
	f.UpdatedAt = time.Unix(updated, 0).UTC()
	if f.Tags == nil {
		f.Tags = []string{}
	}
	return nil
}

// ParseFrames decodes the frames file content.
func ParseFrames(data []byte) ([]Frame, error) {
	var frames []Frame
	if err := json.Unmarshal(data, &frames); err != nil {
		return nil, err
	}
	if frames == nil {
		frames = []Frame{}
	}
	return frames, nil
}

// MarshalFrames encodes frames in Watson's on-disk format:
// indent=1, no HTML escaping, no trailing newline (Python json.dumps equivalent).
func MarshalFrames(frames []Frame) ([]byte, error) {
	if len(frames) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(frames); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// SortFrames sorts by start time ascending (stable).
func SortFrames(frames []Frame) {
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].Start.Before(frames[j].Start) })
}

// FindByIDPrefix returns the index of the frame whose ID starts with prefix.
func FindByIDPrefix(frames []Frame, prefix string) (int, error) {
	idx := -1
	for i, fr := range frames {
		if strings.HasPrefix(fr.ID, prefix) {
			if idx != -1 {
				return -1, ErrAmbiguous
			}
			idx = i
		}
	}
	if idx == -1 {
		return -1, ErrNotFound
	}
	return idx, nil
}
```

- [ ] **Step 4: Test läuft grün**

Run: `go test ./internal/watson/ -v`
Expected: PASS — insbesondere `TestMarshalFramesRoundTripBytes` (byte-genau)

- [ ] **Step 5: Commit**

```bash
git add internal/watson/
git commit -m "feat: parse and serialize watson frames byte-compatible"
```

---

### Task 4: Atomares Speichern & Locking (`save.go`)

**Files:**
- Create: `internal/watson/save.go`
- Test: `internal/watson/save_test.go`

**Interfaces:**
- Consumes: —
- Produces (paketintern, vom Store in Task 7 genutzt):
  - `func safeSave(path string, data []byte) error` — temp-Datei im Zielverzeichnis → alte Datei nach `<path>.bak` rotieren (eine Generation) → Rename; legt Verzeichnis mit `0700` an
  - `func withLock(dir string, fn func() error) error` — flock auf `<dir>/.watson-tui.lock`, Timeout 3 s; wenn Locking vom FS nicht unterstützt: Best-Effort ohne Lock

- [ ] **Step 1: Dependency holen**

```bash
go get github.com/gofrs/flock@latest
```

- [ ] **Step 2: Failing Tests schreiben**

```go
package watson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeSaveCreatesFileAndDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	path := filepath.Join(dir, "frames")
	if err := safeSave(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "v1" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSafeSaveRotatesBak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frames")
	if err := safeSave(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if err := safeSave(path, []byte("v2")); err != nil {
		t.Fatal(err)
	}
	if err := safeSave(path, []byte("v3")); err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(path)
	bak, _ := os.ReadFile(path + ".bak")
	if string(cur) != "v3" || string(bak) != "v2" {
		t.Errorf("cur=%q bak=%q, want v3/v2", cur, bak)
	}
}

func TestSafeSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frames")
	_ = safeSave(path, []byte("v1"))
	_ = safeSave(path, []byte("v2"))
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
}

func TestWithLockRuns(t *testing.T) {
	ran := false
	if err := withLock(t.TempDir(), func() error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("fn did not run")
	}
}
```

- [ ] **Step 3: Test läuft rot**

Run: `go test ./internal/watson/ -run 'TestSafeSave|TestWithLock' -v`
Expected: FAIL — `undefined: safeSave`

- [ ] **Step 4: Implementieren**

```go
package watson

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// safeSave writes data atomically, keeping one .bak generation of the
// previous content. Mirrors Watson's safe_save (watson/utils.py).
func safeSave(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if _, err := os.Stat(path); err == nil {
		_ = os.Remove(path + ".bak")
		if err := os.Rename(path, path+".bak"); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
	}
	return os.Rename(tmpName, path)
}

// withLock runs fn while holding an advisory lock on <dir>/.watson-tui.lock.
// Watson itself has no locking; this protects concurrent watson-tui writers.
// If the filesystem does not support flock, fn runs unlocked (best effort).
func withLock(dir string, fn func() error) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fl := flock.New(filepath.Join(dir, ".watson-tui.lock"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fn()
	}
	if !ok {
		return errors.New("watson-Verzeichnis ist von anderem Prozess gesperrt")
	}
	defer func() { _ = fl.Unlock() }()
	return fn()
}
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./internal/watson/ -v && go mod tidy`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/watson/ go.mod go.sum
git commit -m "feat: atomic safe-save with .bak rotation and flock"
```

---

### Task 5: State-Datei — laufender Timer (`state.go`)

**Files:**
- Create: `internal/watson/state.go`
- Test: `internal/watson/state_test.go`

**Interfaces:**
- Produces:
  - `type State struct { Project string; Start time.Time; Tags []string }`
  - `func ParseState(data []byte) (*State, error)` — `nil, nil` bei `{}`/leerer Datei (= kein Timer)
  - `func MarshalState(s *State) ([]byte, error)` — `{}` bei nil, sonst `{"project", "start", "tags"}` mit Indent 1, kein trailing Newline

- [ ] **Step 1: Failing Tests schreiben**

```go
package watson

import (
	"testing"
	"time"
)

const goldenState = `{
 "project": "watson-tui",
 "start": 1658000000,
 "tags": [
  "cli"
 ]
}`

func TestParseStateRunning(t *testing.T) {
	s, err := ParseState([]byte(goldenState))
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("got nil, want running state")
	}
	if s.Project != "watson-tui" || !s.Start.Equal(time.Unix(1658000000, 0)) || len(s.Tags) != 1 {
		t.Errorf("state = %+v", s)
	}
}

func TestParseStateEmpty(t *testing.T) {
	for _, in := range []string{"{}", "", "  \n"} {
		s, err := ParseState([]byte(in))
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if s != nil {
			t.Errorf("%q: got %+v, want nil", in, s)
		}
	}
}

func TestParseStateCorrupt(t *testing.T) {
	if _, err := ParseState([]byte("{ nope")); err == nil {
		t.Error("want error")
	}
}

func TestMarshalStateRoundTrip(t *testing.T) {
	s, err := ParseState([]byte(goldenState))
	if err != nil {
		t.Fatal(err)
	}
	out, err := MarshalState(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenState {
		t.Errorf("got:\n%s\nwant:\n%s", out, goldenState)
	}
}

func TestMarshalStateNil(t *testing.T) {
	out, err := MarshalState(nil)
	if err != nil || string(out) != "{}" {
		t.Errorf("got %q, %v", out, err)
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/watson/ -run TestParseState -v`
Expected: FAIL — `undefined: ParseState`

- [ ] **Step 3: Implementieren**

```go
package watson

import (
	"bytes"
	"encoding/json"
	"time"
)

// State is the currently running timer (Watson's `state` file).
type State struct {
	Project string
	Start   time.Time
	Tags    []string
}

type stateJSON struct {
	Project string   `json:"project"`
	Start   int64    `json:"start"`
	Tags    []string `json:"tags"`
}

// ParseState returns nil when no timer is running ({} or empty file).
func ParseState(data []byte) (*State, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var raw stateJSON
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, err
	}
	if raw.Project == "" {
		return nil, nil
	}
	tags := raw.Tags
	if tags == nil {
		tags = []string{}
	}
	return &State{Project: raw.Project, Start: time.Unix(raw.Start, 0).UTC(), Tags: tags}, nil
}

// MarshalState encodes the state file (indent=1, no trailing newline, {} when idle).
func MarshalState(s *State) ([]byte, error) {
	if s == nil {
		return []byte("{}"), nil
	}
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(stateJSON{Project: s.Project, Start: s.Start.Unix(), Tags: tags}); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
```

- [ ] **Step 4: Test läuft grün**

Run: `go test ./internal/watson/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/watson/
git commit -m "feat: parse and serialize watson state file"
```

---

### Task 6: Config lesen (`config.go`)

**Files:**
- Create: `internal/watson/config.go`
- Test: `internal/watson/config_test.go`

**Interfaces:**
- Produces:
  - `type Config struct { WeekStart time.Weekday }` — v1 nutzt nur `week_start`; `date_format`/`time_format` bleiben ungelesen (Anzeige-Formate der TUI sind fix)
  - `func DefaultConfig() Config` — `WeekStart: time.Monday` (Watson-Default)
  - `func ParseConfig(data []byte) (Config, error)` — INI, Section `[options]`, Key `week_start` (lowercase englischer Wochentag); unbekannter Wert → Default

- [ ] **Step 1: Dependency holen**

```bash
go get gopkg.in/ini.v1@latest
```

- [ ] **Step 2: Failing Tests schreiben**

```go
package watson

import (
	"testing"
	"time"
)

func TestParseConfigWeekStart(t *testing.T) {
	cfg, err := ParseConfig([]byte("[options]\nweek_start = sunday\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WeekStart != time.Sunday {
		t.Errorf("got %v, want Sunday", cfg.WeekStart)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	for _, in := range []string{"", "[backend]\nurl = x\n", "[options]\nweek_start = kaputt\n"} {
		cfg, err := ParseConfig([]byte(in))
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if cfg.WeekStart != time.Monday {
			t.Errorf("%q: got %v, want Monday", in, cfg.WeekStart)
		}
	}
}
```

- [ ] **Step 3: Test läuft rot**

Run: `go test ./internal/watson/ -run TestParseConfig -v`
Expected: FAIL — `undefined: ParseConfig`

- [ ] **Step 4: Implementieren**

```go
package watson

import (
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// Config holds the subset of Watson's config the TUI reads.
type Config struct {
	WeekStart time.Weekday
}

func DefaultConfig() Config {
	return Config{WeekStart: time.Monday}
}

var weekdays = map[string]time.Weekday{
	"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
	"sunday": time.Sunday,
}

// ParseConfig reads Watson's INI config; unknown values fall back to defaults.
func ParseConfig(data []byte) (Config, error) {
	cfg := DefaultConfig()
	file, err := ini.Load(data)
	if err != nil {
		return cfg, err
	}
	raw := strings.ToLower(strings.TrimSpace(file.Section("options").Key("week_start").String()))
	if wd, ok := weekdays[raw]; ok {
		cfg.WeekStart = wd
	}
	return cfg, nil
}
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./internal/watson/ -v && go mod tidy`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/watson/ go.mod go.sum
git commit -m "feat: read week_start from watson config"
```

---

### Task 7: Store — alles zusammen (`store.go`)

**Files:**
- Create: `internal/watson/store.go`
- Test: `internal/watson/store_test.go`

**Interfaces:**
- Consumes: `ParseFrames`, `MarshalFrames`, `ParseState`, `MarshalState`, `ParseConfig`, `safeSave`, `withLock`, `FindByIDPrefix` (Tasks 3–6)
- Produces (das komplette API für die TUI):
  - `func NewStore(dir string) *Store`, `func (s *Store) Dir() string`
  - `func (s *Store) Frames() ([]Frame, error)` — fehlende Datei → leere Liste
  - `func (s *Store) State() (*State, error)` — fehlende Datei → `nil, nil`
  - `func (s *Store) Config() Config` — fehlende/kaputte Datei → `DefaultConfig()`
  - `func (s *Store) Add(f Frame, now time.Time) (Frame, error)` — vergibt ID + UpdatedAt
  - `func (s *Store) Update(f Frame, now time.Time) error` — ersetzt per voller ID, setzt UpdatedAt; `ErrNotFound` wenn ID fehlt
  - `func (s *Store) Delete(id string) error` — volle ID; `ErrNotFound` wenn fehlt
  - `func (s *Store) Start(project string, tags []string, now time.Time) error` — `ErrAlreadyRunning` wenn Timer läuft
  - `func (s *Store) Stop(now time.Time) (Frame, error)` — `ErrNotRunning` wenn kein Timer; erzeugt Frame, leert State
  - `func (s *Store) Cancel() error` — `ErrNotRunning` wenn kein Timer; leert State ohne Frame
  - `var ErrAlreadyRunning, ErrNotRunning error`

- [ ] **Step 1: Dependency holen**

```bash
go get github.com/google/uuid@latest
```

- [ ] **Step 2: Failing Tests schreiben**

```go
package watson

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func TestStoreEmptyDir(t *testing.T) {
	s := testStore(t)
	frames, err := s.Frames()
	if err != nil || len(frames) != 0 {
		t.Fatalf("frames = %v, %v", frames, err)
	}
	state, err := s.State()
	if err != nil || state != nil {
		t.Fatalf("state = %v, %v", state, err)
	}
	if s.Config().WeekStart != time.Monday {
		t.Error("config default missing")
	}
}

func TestStoreAddUpdateDelete(t *testing.T) {
	s := testStore(t)
	now := time.Unix(1658000000, 0).UTC()
	added, err := s.Add(Frame{
		Start: now.Add(-time.Hour), Stop: now, Project: "p1", Tags: []string{"t"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(added.ID) {
		t.Errorf("id = %q, want 32 hex chars", added.ID)
	}
	if !added.UpdatedAt.Equal(now) {
		t.Errorf("updated_at = %v", added.UpdatedAt)
	}

	added.Project = "p2"
	later := now.Add(time.Minute)
	if err := s.Update(added, later); err != nil {
		t.Fatal(err)
	}
	frames, _ := s.Frames()
	if len(frames) != 1 || frames[0].Project != "p2" || !frames[0].UpdatedAt.Equal(later) {
		t.Fatalf("after update: %+v", frames)
	}

	if err := s.Update(Frame{ID: "ffffffffffffffffffffffffffffffff"}, later); err != ErrNotFound {
		t.Errorf("update unknown id: %v", err)
	}
	if err := s.Delete(added.ID); err != nil {
		t.Fatal(err)
	}
	frames, _ = s.Frames()
	if len(frames) != 0 {
		t.Fatalf("after delete: %+v", frames)
	}
	if err := s.Delete(added.ID); err != ErrNotFound {
		t.Errorf("double delete: %v", err)
	}
}

func TestStoreStartStopCancel(t *testing.T) {
	s := testStore(t)
	t0 := time.Unix(1658000000, 0).UTC()
	t1 := t0.Add(30 * time.Minute)

	if _, err := s.Stop(t1); err != ErrNotRunning {
		t.Errorf("stop idle: %v", err)
	}
	if err := s.Cancel(); err != ErrNotRunning {
		t.Errorf("cancel idle: %v", err)
	}
	if err := s.Start("proj", []string{"a"}, t0); err != nil {
		t.Fatal(err)
	}
	if err := s.Start("other", nil, t0); err != ErrAlreadyRunning {
		t.Errorf("double start: %v", err)
	}
	state, _ := s.State()
	if state == nil || state.Project != "proj" || !state.Start.Equal(t0) {
		t.Fatalf("state = %+v", state)
	}

	frame, err := s.Stop(t1)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Project != "proj" || !frame.Start.Equal(t0) || !frame.Stop.Equal(t1) {
		t.Errorf("stopped frame = %+v", frame)
	}
	state, _ = s.State()
	if state != nil {
		t.Errorf("state after stop = %+v", state)
	}
	frames, _ := s.Frames()
	if len(frames) != 1 {
		t.Errorf("frames after stop = %+v", frames)
	}

	if err := s.Start("proj2", nil, t1); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	frames, _ = s.Frames()
	if len(frames) != 1 {
		t.Errorf("cancel must not create a frame: %+v", frames)
	}
}

func TestStoreWritesBak(t *testing.T) {
	s := testStore(t)
	now := time.Unix(1658000000, 0).UTC()
	_, _ = s.Add(Frame{Start: now, Stop: now.Add(time.Hour), Project: "a"}, now)
	_, _ = s.Add(Frame{Start: now, Stop: now.Add(time.Hour), Project: "b"}, now)
	if _, err := os.Stat(filepath.Join(s.Dir(), "frames.bak")); err != nil {
		t.Errorf("frames.bak missing: %v", err)
	}
}

func TestStoreCorruptFrames(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir(), "frames"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Frames(); err == nil {
		t.Error("want parse error for corrupt frames file")
	}
}
```

- [ ] **Step 3: Test läuft rot**

Run: `go test ./internal/watson/ -run TestStore -v`
Expected: FAIL — `undefined: NewStore`

- [ ] **Step 4: Implementieren**

Wichtig: Mutationen nehmen **einen** Lock über die ganze Load→Mutate→Save-Sequenz. `safeSave` selbst lockt nicht (sonst Deadlock durch verschachteltes flock).

```go
package watson

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAlreadyRunning = errors.New("es läuft bereits ein Timer")
	ErrNotRunning     = errors.New("kein Timer aktiv")
)

// Store reads and writes a Watson data directory.
type Store struct {
	dir string
}

func NewStore(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Dir() string        { return s.dir }
func (s *Store) framesPath() string { return filepath.Join(s.dir, "frames") }
func (s *Store) statePath() string  { return filepath.Join(s.dir, "state") }
func (s *Store) configPath() string { return filepath.Join(s.dir, "config") }

func newID() string {
	u := uuid.New()
	return hex.EncodeToString(u[:])
}

// Frames reads the frames file; a missing file means no frames yet.
func (s *Store) Frames() ([]Frame, error) {
	data, err := os.ReadFile(s.framesPath())
	if errors.Is(err, os.ErrNotExist) {
		return []Frame{}, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseFrames(data)
}

// State reads the state file; a missing file means no running timer.
func (s *Store) State() (*State, error) {
	data, err := os.ReadFile(s.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseState(data)
}

// Config reads the config file; on any error it returns defaults.
func (s *Store) Config() Config {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		return DefaultConfig()
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

func (s *Store) writeFrames(frames []Frame) error {
	data, err := MarshalFrames(frames)
	if err != nil {
		return err
	}
	return safeSave(s.framesPath(), data)
}

func (s *Store) writeState(state *State) error {
	data, err := MarshalState(state)
	if err != nil {
		return err
	}
	return safeSave(s.statePath(), data)
}

// Add assigns a fresh ID and UpdatedAt, appends and persists the frame.
func (s *Store) Add(f Frame, now time.Time) (Frame, error) {
	f.ID = newID()
	f.UpdatedAt = now.UTC()
	err := withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		frames = append(frames, f)
		return s.writeFrames(frames)
	})
	return f, err
}

// Update replaces the frame with f.ID and stamps UpdatedAt.
func (s *Store) Update(f Frame, now time.Time) error {
	f.UpdatedAt = now.UTC()
	return withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		for i := range frames {
			if frames[i].ID == f.ID {
				frames[i] = f
				return s.writeFrames(frames)
			}
		}
		return ErrNotFound
	})
}

// Delete removes the frame with the given full ID.
func (s *Store) Delete(id string) error {
	return withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		for i := range frames {
			if frames[i].ID == id {
				return s.writeFrames(append(frames[:i], frames[i+1:]...))
			}
		}
		return ErrNotFound
	})
}

// Start begins a new timer; fails if one is already running.
func (s *Store) Start(project string, tags []string, now time.Time) error {
	if tags == nil {
		tags = []string{}
	}
	return withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state != nil {
			return ErrAlreadyRunning
		}
		return s.writeState(&State{Project: project, Start: now.UTC(), Tags: tags})
	})
}

// Stop ends the running timer, persists it as a frame and clears the state.
func (s *Store) Stop(now time.Time) (Frame, error) {
	var frame Frame
	err := withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state == nil {
			return ErrNotRunning
		}
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		frame = Frame{
			Start: state.Start, Stop: now.UTC(), Project: state.Project,
			ID: newID(), Tags: state.Tags, UpdatedAt: now.UTC(),
		}
		if err := s.writeFrames(append(frames, frame)); err != nil {
			return err
		}
		return s.writeState(nil)
	})
	return frame, err
}

// Cancel discards the running timer without recording a frame.
func (s *Store) Cancel() error {
	return withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state == nil {
			return ErrNotRunning
		}
		return s.writeState(nil)
	})
}
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./internal/watson/ -v && go mod tidy && go vet ./...`
Expected: PASS, vet sauber

- [ ] **Step 6: Watson-Kompat-Smoketest (optional, wenn Python vorhanden)**

```bash
python3 - <<'EOF'
import json
data = json.load(open('/tmp/wtest/frames'))
print("OK", len(data), "frames") if all(len(r) == 6 for r in data) else print("FORMAT KAPUTT")
EOF
```
Vorher mit `WATSON_DIR=/tmp/wtest` per Store-Test-Binary oder kleinem Go-Snippet zwei Frames anlegen. Schritt überspringen, wenn kein python3 da ist — die Golden-Tests decken das Format ab.

- [ ] **Step 7: Commit**

```bash
git add internal/watson/ go.mod go.sum
git commit -m "feat: store with locked mutations (add/update/delete, start/stop/cancel)"
```

---

### Task 8: TUI-Grundgerüst (App, Statusbar, Hilfe, main-Wiring)

**Files:**
- Create: `internal/tui/styles.go`, `internal/tui/statusbar.go`, `internal/tui/app.go`
- Modify: `cmd/watson-tui/main.go`
- Test: `internal/tui/statusbar_test.go`, `internal/tui/app_test.go`

**Interfaces:**
- Consumes: `watson.Store`, `watson.State`, `watson.Config` (Task 7)
- Produces:
  - `func NewApp(store *watson.Store, version string) *App` — Bubble-Tea-Root-Model
  - `mode`-Enum: `modeList, modeForm, modeReport, modeConfirmDelete, modeStartTimer, modeConfirmCancel, modeHelp, modeFatal`
  - `App`-Felder für spätere Tasks: `store`, `cfg`, `frames`, `state`, `mode`, `now`, `errMsg`
  - `func (a *App) reload()` — liest frames+state neu; Parse-Fehler → `modeFatal` (nie schreiben!)
  - `tickMsg`, `func tickCmd() tea.Cmd` — Sekundentick
  - `func formatDuration(d time.Duration) string` ("6h 30m"), `func formatClock(d time.Duration) string` ("1:23:45")
  - `func renderStatus(width int, state *watson.State, now time.Time, left, errMsg string) string`
  - Test-Helper `func key(s string) tea.KeyMsg`, `func newTestApp(t *testing.T) *App` (von allen späteren TUI-Tests genutzt)

- [ ] **Step 1: Dependencies holen**

```bash
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Failing Tests schreiben**

`internal/tui/statusbar_test.go`:

```go
package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Minute, "45m"},
		{time.Hour + 5*time.Minute, "1h 05m"},
		{6*time.Hour + 30*time.Minute, "6h 30m"},
		{0, "0m"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFormatClock(t *testing.T) {
	if got := formatClock(90 * time.Second); got != "0:01:30" {
		t.Errorf("got %q", got)
	}
	if got := formatClock(3*time.Hour + 62*time.Second); got != "3:01:02" {
		t.Errorf("got %q", got)
	}
}

func TestRenderStatusRunning(t *testing.T) {
	start := time.Now().Add(-90 * time.Second)
	s := &watson.State{Project: "proj", Start: start, Tags: []string{"a"}}
	out := renderStatus(80, s, time.Now(), "links", "")
	if !strings.Contains(out, "▶ proj [a]") || !strings.Contains(out, "links") {
		t.Errorf("status = %q", out)
	}
}

func TestRenderStatusIdleAndError(t *testing.T) {
	out := renderStatus(80, nil, time.Now(), "links", "")
	if !strings.Contains(out, "kein Timer") {
		t.Errorf("status = %q", out)
	}
	out = renderStatus(80, nil, time.Now(), "links", "kaputt")
	if !strings.Contains(out, "kaputt") || strings.Contains(out, "links") {
		t.Errorf("error must replace left segment: %q", out)
	}
}
```

`internal/tui/app_test.go`:

```go
package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

// key builds a KeyMsg from a readable name ("q", "enter", "ctrl+c", ...).
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp(watson.NewStore(t.TempDir()), "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return app
}

func TestQKeyQuits(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(key("q"))
	if cmd == nil {
		t.Fatal("q must return tea.Quit")
	}
}

func TestHelpToggle(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("?"))
	if app.mode != modeHelp {
		t.Fatal("? must open help")
	}
	if !strings.Contains(app.View(), "Tasten") {
		t.Error("help view missing title")
	}
	app.Update(key("x"))
	if app.mode != modeList {
		t.Error("any key must close help")
	}
}

func TestFatalOnCorruptFrames(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frames"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(watson.NewStore(dir), "test")
	app.Init()
	if app.mode != modeFatal {
		t.Fatal("corrupt frames must switch to fatal mode")
	}
	if !strings.Contains(app.View(), "frames.bak") {
		t.Error("fatal view must mention the backup")
	}
}

func TestStatusShowsRunningTimer(t *testing.T) {
	app := newTestApp(t)
	app.state = &watson.State{Project: "proj", Start: time.Now().Add(-90 * time.Second), Tags: []string{}}
	app.now = time.Now()
	if !strings.Contains(app.View(), "▶ proj") {
		t.Error("running timer missing in status bar")
	}
}
```

- [ ] **Step 3: Test läuft rot**

Run: `go test ./internal/tui/ -v`
Expected: FAIL — `undefined: NewApp` etc.

- [ ] **Step 4: `internal/tui/styles.go` schreiben**

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle     = lipgloss.NewStyle().Bold(true)
	styleDayHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	styleSelected  = lipgloss.NewStyle().Reverse(true)
	styleDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleRunning   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	styleError     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
)
```

- [ ] **Step 5: `internal/tui/statusbar.go` schreiben**

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// formatDuration renders "6h 30m" / "45m", rounded to minutes.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}

// formatClock renders a live timer as H:MM:SS.
func formatClock(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// renderStatus renders the global one-line status bar. left carries
// view-specific info (period, counts, filter); errMsg replaces left.
func renderStatus(width int, state *watson.State, now time.Time, left, errMsg string) string {
	if errMsg != "" {
		left = styleError.Render(errMsg)
	}
	var right string
	if state != nil {
		label := state.Project
		if len(state.Tags) > 0 {
			label += " [" + strings.Join(state.Tags, ", ") + "]"
		}
		right = styleRunning.Render(fmt.Sprintf("▶ %s %s", label, formatClock(now.Sub(state.Start))))
	} else {
		right = styleDim.Render("kein Timer")
	}
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
```

- [ ] **Step 6: `internal/tui/app.go` schreiben**

```go
package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeReport
	modeConfirmDelete
	modeStartTimer
	modeConfirmCancel
	modeHelp
	modeFatal
)

// App is the Bubble Tea root model.
type App struct {
	store   *watson.Store
	version string
	cfg     watson.Config

	frames []watson.Frame
	state  *watson.State

	mode     mode
	now      time.Time
	errMsg   string // transient, shown in status bar, cleared on next key
	fatalMsg string

	width, height int
}

func NewApp(store *watson.Store, version string) *App {
	return &App{store: store, version: version, mode: modeList, now: time.Now(), width: 80, height: 24}
}

func (a *App) Init() tea.Cmd {
	a.cfg = a.store.Config()
	a.reload()
	return tickCmd()
}

// reload re-reads frames and state. Read errors are fatal: we must not
// write on top of a file we cannot parse (protects the .bak generation).
func (a *App) reload() {
	frames, err := a.store.Frames()
	if err != nil {
		a.fatal(fmt.Sprintf("frames-Datei nicht lesbar: %v\nBackup: %s/frames.bak", err, a.store.Dir()))
		return
	}
	state, err := a.store.State()
	if err != nil {
		a.fatal(fmt.Sprintf("state-Datei nicht lesbar: %v\nBackup: %s/state.bak", err, a.store.Dir()))
		return
	}
	a.frames = frames
	a.state = state
}

func (a *App) fatal(msg string) {
	a.fatalMsg = msg
	a.mode = modeFatal
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case tickMsg:
		a.now = time.Time(msg)
		return a, tickCmd()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		a.errMsg = ""
		switch a.mode {
		case modeFatal:
			return a, tea.Quit
		case modeHelp:
			a.mode = modeList
			return a, nil
		case modeList:
			return a.updateList(msg)
		}
	}
	return a, nil
}

// updateList handles keys in the frame list view. Later tasks extend this.
func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "?":
		a.mode = modeHelp
	}
	return a, nil
}

// statusLeft is the view-specific left segment of the status bar.
// Task 9 extends this with period, count and filter info.
func (a *App) statusLeft() string { return "" }

func (a *App) View() string {
	var body string
	switch a.mode {
	case modeFatal:
		body = styleError.Render("Fehler") + "\n\n" + a.fatalMsg + "\n\nBeliebige Taste beendet."
	case modeHelp:
		body = helpView()
	default:
		body = fmt.Sprintf("%d Frames geladen — ? für Hilfe", len(a.frames))
	}
	return body + "\n" + renderStatus(a.width, a.state, a.now, a.statusLeft(), a.errMsg)
}

func helpView() string {
	return styleTitle.Render("Tasten") + `

  j/k, ↓/↑      navigieren
  enter         Frame editieren
  n             neuer Frame
  d             Frame löschen
  s             Timer starten/stoppen
  S             Timer verwerfen (cancel)
  /             filtern
  [ / ]         Zeitraum zurück/vor
  t/w/m/a       Tag/Woche/Monat/alles
  r             Report
  R             neu laden
  ?             diese Hilfe
  q             beenden

Beliebige Taste schließt die Hilfe.`
}
```

- [ ] **Step 7: `cmd/watson-tui/main.go` ersetzen**

```go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/tui"
	"github.com/schnaq/watson-tui/internal/watson"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	dirFlag := flag.String("dir", "", "watson data directory (default: $WATSON_DIR or OS app dir)")
	flag.Parse()
	if *showVersion {
		fmt.Println("watson-tui", version)
		return
	}
	dir, err := watson.Dir(*dirFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	app := tui.NewApp(watson.NewStore(dir), version)
	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 8: Test läuft grün**

Run: `go mod tidy && go test ./... && go build ./...`
Expected: PASS

- [ ] **Step 9: Manueller Smoke-Test (optional, interaktives Terminal nötig)**

Run: `WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui`
Expected: Alt-Screen mit "0 Frames geladen", Statusbar "kein Timer", `?` zeigt Hilfe, `q` beendet. In CI/Headless überspringen.

- [ ] **Step 10: Commit**

```bash
git add internal/tui/ cmd/ go.mod go.sum
git commit -m "feat: bubbletea app skeleton with status bar, help and fatal screen"
```

---

### Task 9: Frames-Liste (Haupt-View)

**Files:**
- Create: `internal/tui/frameslist.go`
- Modify: `internal/tui/app.go` (Felder, `updateList`, `statusLeft`, `View`)
- Test: `internal/tui/frameslist_test.go`

**Interfaces:**
- Consumes: `App`, `key()`/`newTestApp()` (Task 8), `watson.Frame`, `watson.SortFrames`
- Produces (von Tasks 10–13 genutzt):
  - `type period struct { unit periodUnit; ref time.Time }` mit `unitWeek, unitMonth, unitDay, unitAll`
  - `func (p period) bounds(weekStart time.Weekday) (from, to time.Time, ok bool)` — `[from, to)`, lokale Zeit
  - `func (p period) shift(weekStart time.Weekday, delta int) period`
  - `func (p period) label(weekStart time.Weekday) string`
  - `type row struct { isHeader bool; title string; frame watson.Frame }`
  - `func buildRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string) []row`
  - `listModel` mit `refresh(frames, weekStart)`, `selected() (watson.Frame, bool)`, `move(dir int)`, `view(height int) string`
  - `func firstFrameRow(rows []row) int`, `func nextFrameRow(rows []row, from, dir int) int`, `func truncate(s string, max int) string`
  - App-Feld `list listModel`; `a.reload()` ruft am Ende `a.list.refresh(a.frames, a.cfg.WeekStart)` auf

- [ ] **Step 1: Failing Tests schreiben**

```go
package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func mkFrame(id, project string, start time.Time, dur time.Duration, tags ...string) watson.Frame {
	return watson.Frame{
		ID: id, Project: project, Start: start.UTC(), Stop: start.Add(dur).UTC(),
		Tags: tags, UpdatedAt: start.UTC(),
	}
}

func TestPeriodBoundsWeek(t *testing.T) {
	// Mittwoch 2026-07-22, Wochenstart Montag → 20.07. bis (exkl.) 27.07.
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	from, to, ok := period{unit: unitWeek, ref: ref}.bounds(time.Monday)
	if !ok || from.Day() != 20 || to.Day() != 27 {
		t.Errorf("from=%v to=%v ok=%v", from, to, ok)
	}
}

func TestPeriodBoundsWeekSundayStart(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	from, _, _ := period{unit: unitWeek, ref: ref}.bounds(time.Sunday)
	if from.Day() != 19 {
		t.Errorf("from=%v, want 19.07.", from)
	}
}

func TestPeriodShiftMonthFromJan31(t *testing.T) {
	ref := time.Date(2026, 1, 31, 12, 0, 0, 0, time.Local)
	p := period{unit: unitMonth, ref: ref}.shift(time.Monday, 1)
	from, _, _ := p.bounds(time.Monday)
	if int(from.Month()) != 2 {
		t.Errorf("shift from Jan 31 must land in February, got %v", from)
	}
}

func TestPeriodAllUnbounded(t *testing.T) {
	if _, _, ok := (period{unit: unitAll, ref: time.Now()}).bounds(time.Monday); ok {
		t.Error("unitAll must be unbounded")
	}
}

func TestBuildRowsGroupsAndFilters(t *testing.T) {
	day1 := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	day2 := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day1, time.Hour, "x"),
		mkFrame("b2222222222222222222222222222222", "beta", day2, 30*time.Minute),
		mkFrame("c3333333333333333333333333333333", "alpha", day2, time.Hour),
	}
	p := period{unit: unitWeek, ref: day1}
	rows := buildRows(frames, p, time.Monday, "")
	if len(rows) != 5 { // 2 Tages-Header + 3 Frames
		t.Fatalf("got %d rows, want 5", len(rows))
	}
	if !rows[0].isHeader || !rows[2].isHeader {
		t.Errorf("headers at wrong positions")
	}

	rows = buildRows(frames, p, time.Monday, "beta")
	if len(rows) != 2 || rows[1].frame.Project != "beta" {
		t.Errorf("filter beta: %+v", rows)
	}

	rows = buildRows(frames, p, time.Monday, "x") // Tag-Filter
	if len(rows) != 2 || rows[1].frame.Project != "alpha" {
		t.Errorf("filter tag x: %+v", rows)
	}

	rows = buildRows(frames, period{unit: unitDay, ref: day1}, time.Monday, "")
	if len(rows) != 2 {
		t.Errorf("day period: %+v", rows)
	}
}

func TestListNavigationSkipsHeaders(t *testing.T) {
	app := newTestApp(t)
	now := time.Now()
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-50*time.Hour), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(-2*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)
	if f, ok := app.list.selected(); !ok || f.Project != "alpha" {
		t.Fatalf("initial selection: %+v", f)
	}
	app.Update(key("j"))
	if f, _ := app.list.selected(); f.Project != "beta" {
		t.Errorf("after j: %v", f.Project)
	}
	app.Update(key("j")) // am Ende: bleibt stehen
	if f, _ := app.list.selected(); f.Project != "beta" {
		t.Errorf("j at end moved cursor")
	}
	app.Update(key("k"))
	if f, _ := app.list.selected(); f.Project != "alpha" {
		t.Errorf("after k: %v", f.Project)
	}
}

func TestFilterKeyFlow(t *testing.T) {
	app := newTestApp(t)
	now := time.Now()
	app.frames = []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", now.Add(-2*time.Hour), time.Hour),
		mkFrame("b2222222222222222222222222222222", "beta", now.Add(-4*time.Hour), time.Hour),
	}
	app.list.per = period{unit: unitAll, ref: now}
	app.list.refresh(app.frames, time.Monday)
	app.Update(key("/"))
	if !app.list.filtering {
		t.Fatal("/ must enter filter mode")
	}
	app.Update(key("b"))
	app.Update(key("enter"))
	if app.list.filtering || app.list.filter != "b" {
		t.Fatalf("filter state: %q filtering=%v", app.list.filter, app.list.filtering)
	}
	if f, ok := app.list.selected(); !ok || f.Project != "beta" {
		t.Errorf("selected = %+v", f)
	}
	app.Update(key("/"))
	app.Update(key("esc"))
	if app.list.filter != "" {
		t.Error("esc must clear filter")
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -v`
Expected: FAIL — `undefined: period` etc.

- [ ] **Step 3: `internal/tui/frameslist.go` schreiben**

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

type periodUnit int

const (
	unitWeek periodUnit = iota
	unitMonth
	unitDay
	unitAll
)

// period is a display time range: a unit plus a reference time inside it.
type period struct {
	unit periodUnit
	ref  time.Time
}

// bounds returns the half-open range [from, to) in local time; ok=false for unitAll.
func (p period) bounds(weekStart time.Weekday) (from, to time.Time, ok bool) {
	if p.unit == unitAll {
		return time.Time{}, time.Time{}, false
	}
	y, m, d := p.ref.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, p.ref.Location())
	switch p.unit {
	case unitDay:
		return day, day.AddDate(0, 0, 1), true
	case unitWeek:
		diff := (int(day.Weekday()) - int(weekStart) + 7) % 7
		start := day.AddDate(0, 0, -diff)
		return start, start.AddDate(0, 0, 7), true
	default: // unitMonth
		start := time.Date(y, m, 1, 0, 0, 0, 0, p.ref.Location())
		return start, start.AddDate(0, 1, 0), true
	}
}

// shift moves the period by delta units (delta -1 = one week/month/day back).
// Normalizes ref to the period start first, so Jan 31 + 1 month = February.
func (p period) shift(weekStart time.Weekday, delta int) period {
	from, _, ok := p.bounds(weekStart)
	if !ok {
		return p
	}
	switch p.unit {
	case unitDay:
		p.ref = from.AddDate(0, 0, delta)
	case unitWeek:
		p.ref = from.AddDate(0, 0, 7*delta)
	default:
		p.ref = from.AddDate(0, delta, 0)
	}
	return p
}

var germanDays = map[time.Weekday]string{
	time.Monday: "Montag", time.Tuesday: "Dienstag", time.Wednesday: "Mittwoch",
	time.Thursday: "Donnerstag", time.Friday: "Freitag", time.Saturday: "Samstag",
	time.Sunday: "Sonntag",
}

var germanMonths = [...]string{"", "Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember"}

func formatDay(t time.Time) string {
	return fmt.Sprintf("%s, %02d.%02d.%d", germanDays[t.Weekday()], t.Day(), int(t.Month()), t.Year())
}

// label describes the period for the status bar.
func (p period) label(weekStart time.Weekday) string {
	from, to, ok := p.bounds(weekStart)
	if !ok {
		return "alle Frames"
	}
	switch p.unit {
	case unitDay:
		return formatDay(from)
	case unitWeek:
		last := to.AddDate(0, 0, -1)
		return fmt.Sprintf("Woche %02d.%02d. – %02d.%02d.%d",
			from.Day(), int(from.Month()), last.Day(), int(last.Month()), last.Year())
	default:
		return fmt.Sprintf("%s %d", germanMonths[int(from.Month())], from.Year())
	}
}

// row is one display line: a day header or a frame.
type row struct {
	isHeader bool
	title    string
	frame    watson.Frame
}

func matchesFilter(f watson.Frame, needle string) bool {
	if strings.Contains(strings.ToLower(f.Project), needle) {
		return true
	}
	for _, tag := range f.Tags {
		if strings.Contains(strings.ToLower(tag), needle) {
			return true
		}
	}
	return strings.HasPrefix(f.ID, needle)
}

// buildRows filters frames to period+filter, sorts by start, groups by local day.
func buildRows(frames []watson.Frame, p period, weekStart time.Weekday, filter string) []row {
	from, to, bounded := p.bounds(weekStart)
	needle := strings.ToLower(strings.TrimSpace(filter))
	var sel []watson.Frame
	for _, fr := range frames {
		if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
			continue
		}
		if needle != "" && !matchesFilter(fr, needle) {
			continue
		}
		sel = append(sel, fr)
	}
	watson.SortFrames(sel)

	type group struct {
		day    time.Time
		frames []watson.Frame
		total  time.Duration
	}
	var groups []group
	for _, fr := range sel {
		y, m, d := fr.Start.Local().Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		if len(groups) == 0 || !groups[len(groups)-1].day.Equal(day) {
			groups = append(groups, group{day: day})
		}
		g := &groups[len(groups)-1]
		g.frames = append(g.frames, fr)
		g.total += fr.Duration()
	}
	var rows []row
	for _, g := range groups {
		rows = append(rows, row{isHeader: true, title: fmt.Sprintf("%s — %s", formatDay(g.day), formatDuration(g.total))})
		for _, fr := range g.frames {
			rows = append(rows, row{frame: fr})
		}
	}
	return rows
}

// firstFrameRow returns the index of the first non-header row, -1 if none.
func firstFrameRow(rows []row) int {
	for i, r := range rows {
		if !r.isHeader {
			return i
		}
	}
	return -1
}

// nextFrameRow returns the next non-header row index in direction dir (+1/-1),
// or from when there is none.
func nextFrameRow(rows []row, from, dir int) int {
	for i := from + dir; i >= 0 && i < len(rows); i += dir {
		if !rows[i].isHeader {
			return i
		}
	}
	return from
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

type listModel struct {
	rows        []row
	cursor      int // index into rows; always on a frame row; -1 when empty
	offset      int // first visible row
	per         period
	filter      string
	filtering   bool
	filterInput textinput.Model
}

func newListModel(now time.Time) listModel {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "Projekt, Tag oder ID"
	return listModel{per: period{unit: unitWeek, ref: now}, cursor: -1, filterInput: ti}
}

// refresh rebuilds rows and keeps the selection on the same frame if possible.
func (l *listModel) refresh(frames []watson.Frame, weekStart time.Weekday) {
	prevID := ""
	if f, ok := l.selected(); ok {
		prevID = f.ID
	}
	l.rows = buildRows(frames, l.per, weekStart, l.filter)
	l.cursor = firstFrameRow(l.rows)
	if prevID != "" {
		for i, r := range l.rows {
			if !r.isHeader && r.frame.ID == prevID {
				l.cursor = i
				break
			}
		}
	}
}

func (l *listModel) selected() (watson.Frame, bool) {
	if l.cursor < 0 || l.cursor >= len(l.rows) || l.rows[l.cursor].isHeader {
		return watson.Frame{}, false
	}
	return l.rows[l.cursor].frame, true
}

func (l *listModel) move(dir int) {
	if l.cursor >= 0 {
		l.cursor = nextFrameRow(l.rows, l.cursor, dir)
	}
}

// view renders `height` rows, scrolling so the cursor stays visible.
func (l *listModel) view(height int) string {
	if height < 1 {
		height = 1
	}
	if len(l.rows) == 0 {
		empty := "keine Frames im Zeitraum — n legt einen neuen an"
		if l.filter != "" {
			empty = "kein Treffer für Filter »" + l.filter + "«"
		}
		return styleDim.Render(empty)
	}
	if l.cursor >= 0 && l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+height {
		l.offset = l.cursor - height + 1
	}
	if l.offset > len(l.rows)-height {
		l.offset = len(l.rows) - height
	}
	if l.offset < 0 {
		l.offset = 0
	}
	var b strings.Builder
	end := l.offset + height
	if end > len(l.rows) {
		end = len(l.rows)
	}
	for i := l.offset; i < end; i++ {
		b.WriteString(l.renderRow(i))
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (l *listModel) renderRow(i int) string {
	r := l.rows[i]
	if r.isHeader {
		return styleDayHeader.Render(r.title)
	}
	fr := r.frame
	line := fmt.Sprintf("  %s–%s  %7s  %-24s %-28s %s",
		fr.Start.Local().Format("15:04"),
		fr.Stop.Local().Format("15:04"),
		formatDuration(fr.Duration()),
		truncate(fr.Project, 24),
		truncate(strings.Join(fr.Tags, ", "), 28),
		fr.ShortID())
	if i == l.cursor {
		return styleSelected.Render(line)
	}
	return line
}
```

- [ ] **Step 4: `internal/tui/app.go` erweitern**

Struct-Feld ergänzen (nach `state  *watson.State`):

```go
	list listModel
```

`NewApp` ersetzen:

```go
func NewApp(store *watson.Store, version string) *App {
	return &App{
		store: store, version: version, mode: modeList, now: time.Now(),
		width: 80, height: 24, list: newListModel(time.Now()),
	}
}
```

Ans Ende von `reload()` (nach `a.state = state`):

```go
	a.list.refresh(a.frames, a.cfg.WeekStart)
```

`updateList` ersetzen:

```go
func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.list.filtering {
		switch msg.String() {
		case "enter":
			a.list.filtering = false
			a.list.filter = a.list.filterInput.Value()
		case "esc":
			a.list.filtering = false
			a.list.filterInput.SetValue("")
			a.list.filter = ""
		default:
			var cmd tea.Cmd
			a.list.filterInput, cmd = a.list.filterInput.Update(msg)
			a.list.filter = a.list.filterInput.Value()
			a.list.refresh(a.frames, a.cfg.WeekStart)
			return a, cmd
		}
		a.list.refresh(a.frames, a.cfg.WeekStart)
		return a, nil
	}
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "?":
		a.mode = modeHelp
	case "j", "down":
		a.list.move(+1)
	case "k", "up":
		a.list.move(-1)
	case "g":
		a.list.cursor = firstFrameRow(a.list.rows)
	case "G":
		if i := nextFrameRow(a.list.rows, len(a.list.rows), -1); i < len(a.list.rows) {
			a.list.cursor = i
		}
	case "/":
		a.list.filtering = true
		a.list.filterInput.Focus()
		return a, textinput.Blink
	case "[":
		a.list.per = a.list.per.shift(a.cfg.WeekStart, -1)
		a.list.refresh(a.frames, a.cfg.WeekStart)
	case "]":
		a.list.per = a.list.per.shift(a.cfg.WeekStart, +1)
		a.list.refresh(a.frames, a.cfg.WeekStart)
	case "t":
		a.setPeriodUnit(unitDay)
	case "w":
		a.setPeriodUnit(unitWeek)
	case "m":
		a.setPeriodUnit(unitMonth)
	case "a":
		a.setPeriodUnit(unitAll)
	}
	return a, nil
}

func (a *App) setPeriodUnit(u periodUnit) {
	a.list.per = period{unit: u, ref: time.Now()}
	a.list.refresh(a.frames, a.cfg.WeekStart)
}
```

Import ergänzen: `"github.com/charmbracelet/bubbles/textinput"`.

`statusLeft` ersetzen:

```go
func (a *App) statusLeft() string {
	n := 0
	for _, r := range a.list.rows {
		if !r.isHeader {
			n++
		}
	}
	left := fmt.Sprintf("%s · %d Frames", a.list.per.label(a.cfg.WeekStart), n)
	if a.list.filtering {
		return left + " · " + a.list.filterInput.View()
	}
	if a.list.filter != "" {
		left += " · Filter: " + a.list.filter
	}
	return left
}
```

Im `View()` den `default`-Zweig ersetzen:

```go
	default:
		body = a.list.view(a.height - 1)
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./... && go vet ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: frame list view with day grouping, periods and filter"
```

---

### Task 10: Frame-Formular (Edit & Neu)

**Files:**
- Create: `internal/tui/frameform.go`
- Modify: `internal/tui/app.go` (Feld `form`, Update-Case, View-Case, `updateList`-Cases `enter`/`n`)
- Test: `internal/tui/frameform_test.go`

**Interfaces:**
- Consumes: `App`, `listModel.selected()`, `store.Add/Update`, `period` (Tasks 7–9)
- Produces (Task 12 nutzt `splitTags`, `projectNames`):
  - `func parseDateTime(s string, now time.Time) (time.Time, error)` — Layouts `2006-01-02 15:04[:05]` und `15:04` (= heute), lokale Zeit
  - `func splitTags(s string) []string` — kommagetrennt, getrimmt, leere raus
  - `func buildFrame(project, startStr, stopStr, tagsStr string, now time.Time) (watson.Frame, error)` — ID/UpdatedAt bleiben leer
  - `func overlaps(f watson.Frame, frames []watson.Frame, excludeID string) bool`
  - `func projectNames(frames []watson.Frame) []string` — unique, sortiert
  - `formModel` mit Feld-Konstanten `fieldProject, fieldStart, fieldStop, fieldTags, fieldCount`, `newFormModel(existing *watson.Frame, frames []watson.Frame, now time.Time) formModel`, `setFocus(i int)`, `view() string`
  - App-Feld `form formModel`, Methoden `updateForm`, `submitForm`

- [ ] **Step 1: Failing Tests schreiben**

```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

func TestParseDateTime(t *testing.T) {
	now := time.Date(2026, 7, 24, 14, 0, 0, 0, time.Local)
	got, err := parseDateTime("2026-07-20 09:30", now)
	if err != nil || got.Day() != 20 || got.Hour() != 9 || got.Minute() != 30 {
		t.Errorf("full layout: %v, %v", got, err)
	}
	got, err = parseDateTime("09:15", now)
	if err != nil || got.Day() != 24 || got.Hour() != 9 {
		t.Errorf("time-only layout: %v, %v", got, err)
	}
	if _, err := parseDateTime("gestern", now); err == nil {
		t.Error("want error for garbage")
	}
}

func TestBuildFrameValidation(t *testing.T) {
	now := time.Now()
	if _, err := buildFrame("", "09:00", "10:00", "", now); err == nil {
		t.Error("empty project must fail")
	}
	if _, err := buildFrame("p", "10:00", "09:00", "", now); err == nil {
		t.Error("stop before start must fail")
	}
	f, err := buildFrame("p", "09:00", "10:00", "a, b, ", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "a" || f.Tags[1] != "b" {
		t.Errorf("tags = %v", f.Tags)
	}
	if f.Duration() != time.Hour {
		t.Errorf("duration = %v", f.Duration())
	}
}

func TestOverlaps(t *testing.T) {
	base := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC)
	existing := []watson.Frame{
		{ID: "x", Start: base, Stop: base.Add(time.Hour)},
	}
	f := watson.Frame{Start: base.Add(30 * time.Minute), Stop: base.Add(2 * time.Hour)}
	if !overlaps(f, existing, "") {
		t.Error("want overlap")
	}
	if overlaps(f, existing, "x") {
		t.Error("excludeID must skip the frame being edited")
	}
	f = watson.Frame{Start: base.Add(time.Hour), Stop: base.Add(2 * time.Hour)}
	if overlaps(f, existing, "") {
		t.Error("touching frames do not overlap")
	}
}

func TestNewFrameFlow(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("n"))
	if app.mode != modeForm || app.form.editing {
		t.Fatal("n must open the new-frame form")
	}
	app.form.inputs[fieldProject].SetValue("proj")
	app.form.inputs[fieldStart].SetValue("2026-07-20 09:00")
	app.form.inputs[fieldStop].SetValue("2026-07-20 10:00")
	app.form.inputs[fieldTags].SetValue("a, b")
	app.Update(key("enter"))
	if app.mode != modeList {
		t.Fatalf("save must return to list, errMsg=%q", app.form.errMsg)
	}
	frames, _ := store.Frames()
	if len(frames) != 1 || frames[0].Project != "proj" || len(frames[0].Tags) != 2 || len(frames[0].ID) != 32 {
		t.Errorf("frames = %+v", frames)
	}
}

func TestEditFlowChangesProject(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	now := time.Now()
	if _, err := store.Add(watson.Frame{
		Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Project: "alt", Tags: []string{},
	}, now); err != nil {
		t.Fatal(err)
	}
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("a")) // Zeitraum "alles", unabhängig von Wochengrenzen
	app.Update(key("enter"))
	if app.mode != modeForm || !app.form.editing {
		t.Fatal("enter must open the edit form")
	}
	app.form.inputs[fieldProject].SetValue("neu")
	app.Update(key("enter"))
	if app.mode != modeList {
		t.Fatalf("save failed: %q", app.form.errMsg)
	}
	frames, _ := store.Frames()
	if len(frames) != 1 || frames[0].Project != "neu" {
		t.Errorf("frames = %+v", frames)
	}
}

func TestOverlapWarnsThenSaves(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	now := time.Now()
	if _, err := store.Add(watson.Frame{
		Start: time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local),
		Stop:  time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local),
		Project: "bestehend", Tags: []string{},
	}, now); err != nil {
		t.Fatal(err)
	}
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("n"))
	app.form.inputs[fieldProject].SetValue("p")
	app.form.inputs[fieldStart].SetValue("2026-07-20 09:30")
	app.form.inputs[fieldStop].SetValue("2026-07-20 10:30")
	app.Update(key("enter"))
	if app.mode != modeForm || !app.form.warned {
		t.Fatal("first submit must warn about overlap")
	}
	app.Update(key("enter"))
	if app.mode != modeList {
		t.Fatal("second submit must save anyway")
	}
	frames, _ := store.Frames()
	if len(frames) != 2 {
		t.Errorf("got %d frames, want 2", len(frames))
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -v`
Expected: FAIL — `undefined: parseDateTime` etc.

- [ ] **Step 3: `internal/tui/frameform.go` schreiben**

```go
package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

const dtLayout = "2006-01-02 15:04"

// parseDateTime accepts "2006-01-02 15:04[:05]" and "15:04" (= today), local time.
func parseDateTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	loc := now.Location()
	for _, layout := range []string{"2006-01-02 15:04:05", dtLayout} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation("15:04", s, loc); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc), nil
	}
	return time.Time{}, fmt.Errorf("ungültige Zeit %q (YYYY-MM-DD HH:MM oder HH:MM)", s)
}

// splitTags parses a comma separated tag list.
func splitTags(s string) []string {
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// buildFrame validates form fields into a Frame; ID and UpdatedAt stay unset.
func buildFrame(project, startStr, stopStr, tagsStr string, now time.Time) (watson.Frame, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return watson.Frame{}, errors.New("Projekt fehlt")
	}
	start, err := parseDateTime(startStr, now)
	if err != nil {
		return watson.Frame{}, err
	}
	stop, err := parseDateTime(stopStr, now)
	if err != nil {
		return watson.Frame{}, err
	}
	if !stop.After(start) {
		return watson.Frame{}, errors.New("Stop muss nach Start liegen")
	}
	return watson.Frame{Start: start.UTC(), Stop: stop.UTC(), Project: project, Tags: splitTags(tagsStr)}, nil
}

// overlaps reports whether f overlaps any other frame (excludeID = frame being edited).
func overlaps(f watson.Frame, frames []watson.Frame, excludeID string) bool {
	for _, other := range frames {
		if other.ID == excludeID {
			continue
		}
		if f.Start.Before(other.Stop) && other.Start.Before(f.Stop) {
			return true
		}
	}
	return false
}

// projectNames returns the sorted unique project names for autocompletion.
func projectNames(frames []watson.Frame) []string {
	seen := map[string]bool{}
	var names []string
	for _, f := range frames {
		if !seen[f.Project] {
			seen[f.Project] = true
			names = append(names, f.Project)
		}
	}
	sort.Strings(names)
	return names
}

const (
	fieldProject = iota
	fieldStart
	fieldStop
	fieldTags
	fieldCount
)

type formModel struct {
	editing bool
	frameID string
	inputs  [fieldCount]textinput.Model
	focus   int
	errMsg  string
	warned  bool // overlap warning shown; next submit saves anyway
}

// newFormModel builds the form; existing == nil means "new frame".
func newFormModel(existing *watson.Frame, frames []watson.Frame, now time.Time) formModel {
	var m formModel
	placeholders := [fieldCount]string{"Projekt", "YYYY-MM-DD HH:MM", "YYYY-MM-DD HH:MM", "tag1, tag2"}
	for i := range m.inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholders[i]
		ti.Width = 40
		m.inputs[i] = ti
	}
	m.inputs[fieldProject].ShowSuggestions = true
	m.inputs[fieldProject].SetSuggestions(projectNames(frames))
	m.inputs[fieldProject].KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("right", "ctrl+e"))
	if existing != nil {
		m.editing = true
		m.frameID = existing.ID
		m.inputs[fieldProject].SetValue(existing.Project)
		m.inputs[fieldStart].SetValue(existing.Start.Local().Format(dtLayout))
		m.inputs[fieldStop].SetValue(existing.Stop.Local().Format(dtLayout))
		m.inputs[fieldTags].SetValue(strings.Join(existing.Tags, ", "))
	} else {
		m.inputs[fieldStart].SetValue(now.Add(-time.Hour).Format(dtLayout))
		m.inputs[fieldStop].SetValue(now.Format(dtLayout))
	}
	m.inputs[fieldProject].Focus()
	return m
}

func (m *formModel) setFocus(i int) {
	m.focus = (i + fieldCount) % fieldCount
	for j := range m.inputs {
		if j == m.focus {
			m.inputs[j].Focus()
		} else {
			m.inputs[j].Blur()
		}
	}
}

func (m formModel) view() string {
	title := "Neuer Frame"
	if m.editing {
		title = "Frame bearbeiten (" + m.frameID[:7] + ")"
	}
	labels := [fieldCount]string{"Projekt", "Start  ", "Stop   ", "Tags   "}
	var b strings.Builder
	b.WriteString(styleTitle.Render(title) + "\n\n")
	for i := range m.inputs {
		cursor := "  "
		if i == m.focus {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, labels[i], m.inputs[i].View()))
	}
	if m.errMsg != "" {
		b.WriteString("\n" + styleError.Render(m.errMsg))
	}
	b.WriteString("\n" + styleDim.Render("tab: Feld · →: Vorschlag übernehmen · enter: speichern · esc: abbrechen"))
	return b.String()
}
```

- [ ] **Step 4: `internal/tui/app.go` erweitern**

Struct-Feld ergänzen (nach `list listModel`):

```go
	form formModel
```

In `Update`, im `switch a.mode`-Block ergänzen:

```go
		case modeForm:
			return a.updateForm(msg)
```

In `updateList` (nicht-filternder Zweig) ergänzen:

```go
	case "enter":
		if f, ok := a.list.selected(); ok {
			a.form = newFormModel(&f, a.frames, time.Now())
			a.mode = modeForm
			return a, textinput.Blink
		}
	case "n":
		a.form = newFormModel(nil, a.frames, time.Now())
		a.mode = modeForm
		return a, textinput.Blink
```

Neue Methoden:

```go
func (a *App) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeList
		return a, nil
	case "tab", "down":
		a.form.setFocus(a.form.focus + 1)
		return a, nil
	case "shift+tab", "up":
		a.form.setFocus(a.form.focus - 1)
		return a, nil
	case "enter", "ctrl+s":
		return a.submitForm()
	}
	var cmd tea.Cmd
	a.form.inputs[a.form.focus], cmd = a.form.inputs[a.form.focus].Update(msg)
	return a, cmd
}

func (a *App) submitForm() (tea.Model, tea.Cmd) {
	now := time.Now()
	frame, err := buildFrame(
		a.form.inputs[fieldProject].Value(),
		a.form.inputs[fieldStart].Value(),
		a.form.inputs[fieldStop].Value(),
		a.form.inputs[fieldTags].Value(),
		now,
	)
	if err != nil {
		a.form.errMsg = err.Error()
		return a, nil
	}
	if !a.form.warned && overlaps(frame, a.frames, a.form.frameID) {
		a.form.warned = true
		a.form.errMsg = "Überlappt mit anderem Frame — enter speichert trotzdem"
		return a, nil
	}
	if a.form.editing {
		frame.ID = a.form.frameID
		err = a.store.Update(frame, now)
	} else {
		_, err = a.store.Add(frame, now)
	}
	if err != nil {
		a.form.errMsg = "Speichern fehlgeschlagen: " + err.Error()
		return a, nil
	}
	a.reload()
	a.mode = modeList
	return a, nil
}
```

Im `View()`-Switch ergänzen:

```go
	case modeForm:
		body = a.form.view()
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./... && go vet ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: frame form for create and edit with validation and overlap warning"
```

---

### Task 11: Löschen mit Confirm + manueller Reload

**Files:**
- Modify: `internal/tui/app.go` (Feld `pendingDelete`, Update-Case, View-Case, `updateList`-Cases `d`/`R`)
- Test: `internal/tui/app_test.go` (Tests ergänzen)

**Interfaces:**
- Consumes: `store.Delete` (Task 7), `listModel.selected()` (Task 9)
- Produces: App-Feld `pendingDelete watson.Frame`; Verhalten: `d` → Confirm, `y`/`enter` löscht, jede andere Taste bricht ab; `R` lädt von Disk neu

- [ ] **Step 1: Failing Tests ergänzen (`internal/tui/app_test.go`)**

```go
func TestDeleteFlowWithConfirm(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	now := time.Now()
	added, err := store.Add(watson.Frame{
		Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Project: "weg", Tags: []string{},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("a"))
	app.Update(key("d"))
	if app.mode != modeConfirmDelete || app.pendingDelete.ID != added.ID {
		t.Fatal("d must ask for confirmation")
	}
	app.Update(key("x")) // abbrechen
	if app.mode != modeList {
		t.Fatal("other key must cancel")
	}
	frames, _ := store.Frames()
	if len(frames) != 1 {
		t.Fatal("cancel must not delete")
	}
	app.Update(key("d"))
	app.Update(key("y"))
	frames, _ = store.Frames()
	if len(frames) != 0 {
		t.Errorf("y must delete, frames = %+v", frames)
	}
	if app.mode != modeList {
		t.Error("must return to list")
	}
}

func TestReloadPicksUpExternalChanges(t *testing.T) {
	dir := t.TempDir()
	store := watson.NewStore(dir)
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if len(app.frames) != 0 {
		t.Fatal("start empty")
	}
	// externe Änderung simulieren (z. B. watson-CLI parallel)
	now := time.Now()
	if _, err := store.Add(watson.Frame{
		Start: now.Add(-time.Hour), Stop: now, Project: "extern", Tags: []string{},
	}, now); err != nil {
		t.Fatal(err)
	}
	app.Update(key("R"))
	if len(app.frames) != 1 || app.frames[0].Project != "extern" {
		t.Errorf("R must reload, frames = %+v", app.frames)
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -run 'TestDelete|TestReload' -v`
Expected: FAIL — `undefined` bzw. Mode wechselt nicht

- [ ] **Step 3: `internal/tui/app.go` erweitern**

Struct-Feld ergänzen (nach `form formModel`):

```go
	pendingDelete watson.Frame
```

In `Update`, `switch a.mode` ergänzen:

```go
		case modeConfirmDelete:
			if s := msg.String(); s == "y" || s == "enter" {
				if err := a.store.Delete(a.pendingDelete.ID); err != nil {
					a.errMsg = "Löschen fehlgeschlagen: " + err.Error()
				}
				a.reload()
			}
			a.mode = modeList
			return a, nil
```

In `updateList` ergänzen:

```go
	case "d":
		if f, ok := a.list.selected(); ok {
			a.pendingDelete = f
			a.mode = modeConfirmDelete
		}
	case "R":
		a.reload()
```

Im `View()`-Switch ergänzen:

```go
	case modeConfirmDelete:
		f := a.pendingDelete
		body = styleTitle.Render("Frame löschen?") + fmt.Sprintf("\n\n  %s  %s–%s  %s\n\n%s",
			f.Project,
			f.Start.Local().Format("2006-01-02 15:04"),
			f.Stop.Local().Format("15:04"),
			f.ShortID(),
			styleDim.Render("y/enter: löschen · andere Taste: abbrechen"))
```

- [ ] **Step 4: Test läuft grün**

Run: `go test ./... `
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat: delete frame with confirmation and manual reload"
```

---

### Task 12: Live-Timer (Start/Stop/Cancel)

**Files:**
- Create: `internal/tui/timer.go`
- Modify: `internal/tui/app.go` (Feld `start`, Update-Cases, View-Cases, `updateList`-Cases `s`/`S`)
- Test: `internal/tui/timer_test.go`

**Interfaces:**
- Consumes: `store.Start/Stop/Cancel` (Task 7), `splitTags`, `projectNames` (Task 10), Statusbar zeigt laufenden Timer bereits (Task 8)
- Produces: `startModel` mit `newStartModel(frames []watson.Frame) startModel`, `view() string`; App-Feld `start startModel`, Methode `updateStartTimer`

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/timer_test.go`)**

```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

func TestTimerStartStopFlow(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	app.Update(key("s"))
	if app.mode != modeStartTimer {
		t.Fatal("s must open the start prompt when idle")
	}
	app.Update(key("esc"))
	if app.mode != modeList {
		t.Fatal("esc must cancel the prompt")
	}

	app.Update(key("s"))
	app.start.project.SetValue("proj")
	app.start.tags.SetValue("a, b")
	app.Update(key("enter"))
	if app.mode != modeList {
		t.Fatalf("start failed: %q", app.start.errMsg)
	}
	if app.state == nil || app.state.Project != "proj" || len(app.state.Tags) != 2 {
		t.Fatalf("state = %+v", app.state)
	}

	app.Update(key("s")) // laufender Timer → stop
	if app.state != nil {
		t.Fatal("s must stop the running timer")
	}
	frames, _ := store.Frames()
	if len(frames) != 1 || frames[0].Project != "proj" {
		t.Errorf("frames = %+v", frames)
	}
}

func TestTimerStartRequiresProject(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("s"))
	app.Update(key("enter"))
	if app.mode != modeStartTimer || app.start.errMsg == "" {
		t.Error("empty project must show error and stay in prompt")
	}
}

func TestTimerCancelDiscardsFrame(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	if err := store.Start("proj", nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("S"))
	if app.mode != modeConfirmCancel {
		t.Fatal("S must ask for confirmation")
	}
	app.Update(key("y"))
	if app.state != nil {
		t.Fatal("cancel must clear the timer")
	}
	frames, _ := store.Frames()
	if len(frames) != 0 {
		t.Errorf("cancel must not create a frame: %+v", frames)
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -run TestTimer -v`
Expected: FAIL

- [ ] **Step 3: `internal/tui/timer.go` schreiben**

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

// startModel is the prompt for starting a new timer.
type startModel struct {
	project textinput.Model
	tags    textinput.Model
	focus   int // 0 = project, 1 = tags
	errMsg  string
}

func newStartModel(frames []watson.Frame) startModel {
	p := textinput.New()
	p.Prompt = ""
	p.Placeholder = "Projekt"
	p.Width = 40
	p.ShowSuggestions = true
	p.SetSuggestions(projectNames(frames))
	p.KeyMap.AcceptSuggestion = key.NewBinding(key.WithKeys("right", "ctrl+e"))
	p.Focus()
	tg := textinput.New()
	tg.Prompt = ""
	tg.Placeholder = "tag1, tag2 (optional)"
	tg.Width = 40
	return startModel{project: p, tags: tg}
}

func (m startModel) view() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Timer starten") + "\n\n")
	cursors := [2]string{"  ", "  "}
	cursors[m.focus] = "> "
	b.WriteString(cursors[0] + "Projekt " + m.project.View() + "\n")
	b.WriteString(cursors[1] + "Tags    " + m.tags.View() + "\n")
	if m.errMsg != "" {
		b.WriteString("\n" + styleError.Render(m.errMsg))
	}
	b.WriteString("\n" + styleDim.Render("enter: starten · tab: Feld · →: Vorschlag · esc: abbrechen"))
	return b.String()
}
```

- [ ] **Step 4: `internal/tui/app.go` erweitern**

Struct-Feld ergänzen (nach `pendingDelete watson.Frame`):

```go
	start startModel
```

In `updateList` ergänzen:

```go
	case "s":
		if a.state != nil {
			if _, err := a.store.Stop(time.Now()); err != nil {
				a.errMsg = "Stop fehlgeschlagen: " + err.Error()
			}
			a.reload()
		} else {
			a.start = newStartModel(a.frames)
			a.mode = modeStartTimer
			return a, textinput.Blink
		}
	case "S":
		if a.state != nil {
			a.mode = modeConfirmCancel
		}
```

In `Update`, `switch a.mode` ergänzen:

```go
		case modeStartTimer:
			return a.updateStartTimer(msg)
		case modeConfirmCancel:
			if s := msg.String(); s == "y" || s == "enter" {
				if err := a.store.Cancel(); err != nil {
					a.errMsg = "Verwerfen fehlgeschlagen: " + err.Error()
				}
				a.reload()
			}
			a.mode = modeList
			return a, nil
```

Neue Methode:

```go
func (a *App) updateStartTimer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeList
		return a, nil
	case "tab", "shift+tab", "down", "up":
		a.start.focus = 1 - a.start.focus
		if a.start.focus == 0 {
			a.start.project.Focus()
			a.start.tags.Blur()
		} else {
			a.start.tags.Focus()
			a.start.project.Blur()
		}
		return a, nil
	case "enter":
		project := strings.TrimSpace(a.start.project.Value())
		if project == "" {
			a.start.errMsg = "Projekt fehlt"
			return a, nil
		}
		if err := a.store.Start(project, splitTags(a.start.tags.Value()), time.Now()); err != nil {
			a.start.errMsg = err.Error()
			return a, nil
		}
		a.reload()
		a.mode = modeList
		return a, nil
	}
	var cmd tea.Cmd
	if a.start.focus == 0 {
		a.start.project, cmd = a.start.project.Update(msg)
	} else {
		a.start.tags, cmd = a.start.tags.Update(msg)
	}
	return a, cmd
}
```

Import `"strings"` in app.go ergänzen. Im `View()`-Switch ergänzen:

```go
	case modeStartTimer:
		body = a.start.view()
	case modeConfirmCancel:
		body = styleTitle.Render("Laufenden Timer verwerfen?") + "\n\n" +
			styleDim.Render("y/enter: verwerfen · andere Taste: abbrechen")
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./... && go vet ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/tui/
git commit -m "feat: live timer start/stop/cancel with project prompt"
```

---

### Task 13: Report-View

**Files:**
- Create: `internal/tui/report.go`
- Modify: `internal/tui/app.go` (Feld `report`, Update-Case, View-Case, `updateList`-Case `r`)
- Test: `internal/tui/report_test.go`

**Interfaces:**
- Consumes: `period` (Task 9), `formatDuration` (Task 8), `truncate` (Task 9)
- Produces: `func aggregate(frames []watson.Frame, p period, weekStart time.Weekday) ([]reportLine, time.Duration)`; `reportModel` mit `newReportModel(now time.Time)`, `view(frames, weekStart) string`

- [ ] **Step 1: Failing Tests schreiben (`internal/tui/report_test.go`)**

```go
package tui

import (
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

func TestAggregate(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local)
	frames := []watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, 2*time.Hour, "code"),
		mkFrame("b2222222222222222222222222222222", "alpha", day.Add(3*time.Hour), time.Hour, "review"),
		mkFrame("c3333333333333333333333333333333", "beta", day.Add(5*time.Hour), 30*time.Minute),
		// außerhalb des Zeitraums:
		mkFrame("d4444444444444444444444444444444", "alpha", day.AddDate(0, 0, 14), time.Hour),
	}
	lines, grand := aggregate(frames, period{unit: unitWeek, ref: day}, time.Monday)
	if grand != 3*time.Hour+30*time.Minute {
		t.Errorf("grand = %v", grand)
	}
	if len(lines) != 2 || lines[0].project != "alpha" || lines[1].project != "beta" {
		t.Fatalf("lines = %+v", lines)
	}
	if lines[0].total != 3*time.Hour {
		t.Errorf("alpha total = %v", lines[0].total)
	}
	if len(lines[0].tags) != 2 || lines[0].tags[0].tag != "code" {
		t.Errorf("alpha tags = %+v", lines[0].tags)
	}
}

func TestAggregateEmpty(t *testing.T) {
	lines, grand := aggregate(nil, period{unit: unitWeek, ref: time.Now()}, time.Monday)
	if len(lines) != 0 || grand != 0 {
		t.Errorf("lines=%v grand=%v", lines, grand)
	}
}

func TestReportKeyFlow(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("r"))
	if app.mode != modeReport {
		t.Fatal("r must open report")
	}
	app.Update(key("m"))
	if app.report.per.unit != unitMonth {
		t.Error("m must switch report to month")
	}
	before := app.report.per
	app.Update(key("["))
	if app.report.per == before {
		t.Error("[ must shift the period")
	}
	app.Update(key("esc"))
	if app.mode != modeList {
		t.Error("esc must return to list")
	}
}
```

- [ ] **Step 2: Test läuft rot**

Run: `go test ./internal/tui/ -run 'TestAggregate|TestReport' -v`
Expected: FAIL — `undefined: aggregate`

- [ ] **Step 3: `internal/tui/report.go` schreiben**

```go
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

type tagLine struct {
	tag string
	d   time.Duration
}

type reportLine struct {
	project string
	total   time.Duration
	tags    []tagLine
}

// aggregate sums durations per project (with tag breakdown) for frames whose
// start lies in the period. Sorted by duration desc, ties alphabetically.
func aggregate(frames []watson.Frame, p period, weekStart time.Weekday) ([]reportLine, time.Duration) {
	from, to, bounded := p.bounds(weekStart)
	totals := map[string]time.Duration{}
	tagTotals := map[string]map[string]time.Duration{}
	var grand time.Duration
	for _, fr := range frames {
		if bounded && (fr.Start.Before(from) || !fr.Start.Before(to)) {
			continue
		}
		d := fr.Duration()
		totals[fr.Project] += d
		grand += d
		for _, tag := range fr.Tags {
			if tagTotals[fr.Project] == nil {
				tagTotals[fr.Project] = map[string]time.Duration{}
			}
			tagTotals[fr.Project][tag] += d
		}
	}
	projects := make([]string, 0, len(totals))
	for name := range totals {
		projects = append(projects, name)
	}
	sort.Slice(projects, func(i, j int) bool {
		if totals[projects[i]] != totals[projects[j]] {
			return totals[projects[i]] > totals[projects[j]]
		}
		return projects[i] < projects[j]
	})
	lines := make([]reportLine, 0, len(projects))
	for _, name := range projects {
		rl := reportLine{project: name, total: totals[name]}
		tags := make([]string, 0, len(tagTotals[name]))
		for tg := range tagTotals[name] {
			tags = append(tags, tg)
		}
		sort.Slice(tags, func(i, j int) bool {
			ti, tj := tagTotals[name][tags[i]], tagTotals[name][tags[j]]
			if ti != tj {
				return ti > tj
			}
			return tags[i] < tags[j]
		})
		for _, tg := range tags {
			rl.tags = append(rl.tags, tagLine{tag: tg, d: tagTotals[name][tg]})
		}
		lines = append(lines, rl)
	}
	return lines, grand
}

type reportModel struct {
	per period
}

func newReportModel(now time.Time) reportModel {
	return reportModel{per: period{unit: unitWeek, ref: now}}
}

func (m reportModel) view(frames []watson.Frame, weekStart time.Weekday) string {
	lines, grand := aggregate(frames, m.per, weekStart)
	var b strings.Builder
	b.WriteString(styleTitle.Render("Report — "+m.per.label(weekStart)) + "\n\n")
	if len(lines) == 0 {
		b.WriteString(styleDim.Render("keine Frames im Zeitraum") + "\n")
	}
	for _, l := range lines {
		b.WriteString(fmt.Sprintf("%-32s %10s\n", truncate(l.project, 32), formatDuration(l.total)))
		for _, tl := range l.tags {
			b.WriteString(styleDim.Render(fmt.Sprintf("  [%s]", tl.tag)) +
				fmt.Sprintf("%*s\n", 42-len("  []")-len([]rune(tl.tag)), formatDuration(tl.d)))
		}
	}
	b.WriteString("\n" + styleTitle.Render(fmt.Sprintf("%-32s %10s", "Gesamt", formatDuration(grand))))
	b.WriteString("\n\n" + styleDim.Render("t/w/m: Zeitraum · [ / ]: verschieben · esc: zurück"))
	return b.String()
}
```

Hinweis zur Tag-Zeile: Falls das `%*s`-Padding mit Umlauten hakelig wird, ist `fmt.Sprintf("  [%-28s] %10s", ...)`-Ausrichtung genauso ok — Optik zählt, kein Test hängt an der exakten Spalte.

- [ ] **Step 4: `internal/tui/app.go` erweitern**

Struct-Feld ergänzen (nach `start startModel`):

```go
	report reportModel
```

In `updateList` ergänzen:

```go
	case "r":
		a.report = newReportModel(time.Now())
		a.mode = modeReport
```

In `Update`, `switch a.mode` ergänzen:

```go
		case modeReport:
			switch msg.String() {
			case "esc", "q", "r":
				a.mode = modeList
			case "t":
				a.report.per = period{unit: unitDay, ref: time.Now()}
			case "w":
				a.report.per = period{unit: unitWeek, ref: time.Now()}
			case "m":
				a.report.per = period{unit: unitMonth, ref: time.Now()}
			case "[":
				a.report.per = a.report.per.shift(a.cfg.WeekStart, -1)
			case "]":
				a.report.per = a.report.per.shift(a.cfg.WeekStart, +1)
			}
			return a, nil
```

Im `View()`-Switch ergänzen:

```go
	case modeReport:
		body = a.report.view(a.frames, a.cfg.WeekStart)
```

- [ ] **Step 5: Test läuft grün**

Run: `go test ./... && go vet ./...`
Expected: PASS

- [ ] **Step 6: Manueller Smoke-Test (optional, interaktives Terminal)**

Run: `WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui`
Durchklicken: `n` → Frame anlegen → `r` Report zeigt Summe → `s` Timer → Statusbar tickt → `s` Stop → `d` löschen. `q`.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/
git commit -m "feat: report view with per-project and per-tag totals"
```

---

### Task 14: CI-Workflow (self-hosted Runner)

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: grüne Testsuite (Tasks 1–13)
- Produces: CI-Gate auf `main` + PRs (build, vet, test, golangci-lint)

- [ ] **Step 1: Runner prüfen**

```bash
gh api repos/schnaq/watson-tui/actions/runners --jq '.runners[] | "\(.name) \(.status)"'
```
Expected: mindestens ein Runner `online`. Wenn leer: Runner können auch auf Org-Ebene hängen — `gh api orgs/schnaq/actions/runners --jq '.runners[] | "\(.name) \(.status)"'`. Wenn beides leer → STOPP, User fragen (self-hosted Runner ist Voraussetzung).

- [ ] **Step 2: `.github/workflows/ci.yml` schreiben**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - run: go build ./...
      - run: go vet ./...
      - run: go test ./...

  lint:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - uses: golangci/golangci-lint-action@v8
        with:
          version: latest
```

- [ ] **Step 3: Lint lokal vorziehen**

```bash
command -v golangci-lint >/dev/null || brew install golangci-lint
golangci-lint run ./...
```
Expected: keine Findings (sonst fixen, bevor CI rot wird).

- [ ] **Step 4: Commit + Push + CI beobachten**

```bash
git add .github/
git commit -m "ci: build, vet, test and lint on self-hosted runner"
git push -u origin main
gh run watch --exit-status
```
Expected: beide Jobs grün. Wenn der Runner kein Go/keine Toolchain hat, zieht `actions/setup-go` sie selbst — kein Handlungsbedarf.

---

### Task 15: Release-Pipeline (GoReleaser + Homebrew-Tap) & README

**Files:**
- Create: `.goreleaser.yaml`, `.github/workflows/release.yml`, `README.md`

**Interfaces:**
- Consumes: `main.version` (Task 1), CI grün (Task 14)
- Produces: GitHub Release mit Binaries (darwin/linux × amd64/arm64), Formel in `schnaq/homebrew-tap`, installierbar via `brew install schnaq/tap/watson-tui`

- [ ] **Step 1: `.goreleaser.yaml` schreiben**

```yaml
version: 2

project_name: watson-tui

before:
  hooks:
    - go mod tidy

builds:
  - main: ./cmd/watson-tui
    env:
      - CGO_ENABLED=0
    goos: [darwin, linux]
    goarch: [amd64, arm64]
    flags: [-trimpath]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - formats: [tar.gz]

checksum:
  name_template: checksums.txt

changelog:
  use: github-native

brews:
  - repository:
      owner: schnaq
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    directory: Formula
    homepage: https://github.com/schnaq/watson-tui
    description: Schlanke Terminal-UI für Watson-Zeiterfassung
    license: MIT
    test: |
      system "#{bin}/watson-tui", "--version"
```

- [ ] **Step 2: Config validieren**

```bash
command -v goreleaser >/dev/null || brew install goreleaser
goreleaser check
```
Expected: `1 configuration file(s) validated`. Bei Schema-Fehlern (goreleaser-Versionen ändern gern Keys wie `formats` vs. `format`): Fehlermeldung lesen, Key anpassen, erneut prüfen.

- [ ] **Step 3: `.github/workflows/release.yml` schreiben**

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

- [ ] **Step 4: `README.md` schreiben**

```markdown
# watson-tui

Schlanke Terminal-UI für [Watson](https://jazzband.github.io/Watson/)-Zeiterfassung,
in Go. Liest und schreibt Watsons Datenformat direkt — eine Watson-Installation
ist nicht nötig, vorhandene Daten funktionieren sofort weiter.

## Installation

```sh
brew install schnaq/tap/watson-tui
```

Updates kommen über `brew upgrade`. Alternativ: Binary vom
[Release](https://github.com/schnaq/watson-tui/releases) laden.

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
| `R` | neu laden |
| `?` | Hilfe |
| `q` | beenden |

Datenverzeichnis: `$WATSON_DIR`, sonst OS-Standard
(macOS: `~/Library/Application Support/watson`). Override: `--dir PFAD`.

## Entwicklung

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

Releases: Git-Tag `v*` pushen — GitHub Actions baut mit GoReleaser und
aktualisiert die Homebrew-Formel.
```

- [ ] **Step 5: Tap-Repo anlegen**

```bash
gh repo view schnaq/homebrew-tap >/dev/null 2>&1 || gh repo create schnaq/homebrew-tap --public --description "Homebrew formulae for schnaq tools"
```

- [ ] **Step 6: USER ACTION — `TAP_GITHUB_TOKEN` Secret**

Braucht ein PAT (fine-grained, Repo `schnaq/homebrew-tap`, Permission `Contents: Read and write`). Kann nur der User erzeugen: <https://github.com/settings/personal-access-tokens/new>. Danach:

```bash
gh secret set TAP_GITHUB_TOKEN --repo schnaq/watson-tui
```
STOPP bis Secret gesetzt ist (`gh secret list --repo schnaq/watson-tui` zeigt es).

- [ ] **Step 7: Commit + Push**

```bash
git add .goreleaser.yaml .github/ README.md
git commit -m "ci: goreleaser release pipeline with homebrew tap publishing"
git push
```

- [ ] **Step 8: Release v0.1.0 taggen**

```bash
git tag v0.1.0
git push origin v0.1.0
gh run watch --exit-status
```
Expected: Release-Workflow grün.

- [ ] **Step 9: Release verifizieren**

```bash
gh release view v0.1.0 --json assets --jq '.assets[].name'
gh api repos/schnaq/homebrew-tap/contents/Formula --jq '.[].name'
```
Expected: 4 tar.gz + checksums.txt; `watson-tui.rb` im Tap.

- [ ] **Step 10: Brew-Smoke-Test**

```bash
brew install schnaq/tap/watson-tui
watson-tui --version
```
Expected: `watson-tui 0.1.0`
