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

// TestNewAppNormalisesThePeriodAgainstTheConfiguredWeekStart: NewApp reads the
// config itself rather than leaving it to Init, because it builds the list
// before Init runs and the list normalises its period against the week start
// right there. With the zero value the period is normalised against Sunday and
// then read back with the configured week start, and it resolves to the week
// before.
//
// Deliberately not asserted against app.cfg.WeekStart: under the bug both the
// period and the config field are the zero value, so they agree with each other
// and the assertion passes. The week start has to come from somewhere the app
// does not reach — hence a config file the test wrote and bounds() called
// directly with the weekday that file names.
//
// Wednesday, because it is neither the configured default (Monday) nor the zero
// value (Sunday), so neither can be mistaken for a pass. A Wednesday-started
// week and a Sunday-started week never begin on the same date — one day has one
// weekday — so there is no calendar position at which the bug would look right.
func TestNewAppNormalisesThePeriodAgainstTheConfiguredWeekStart(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config"),
		[]byte("[options]\nweek_start = wednesday\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// No Init: the point is that the list is already right when NewApp returns.
	app := NewApp(watson.NewStore(dir), "test")

	want, _, _ := period{unit: unitWeek, ref: time.Now()}.bounds(time.Wednesday)
	if got := app.list.per.ref; !got.Equal(want) {
		t.Errorf("NewApp built the list against the wrong week start:\n got %s\nwant %s",
			got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	// The same claim, immune to the two time.Now() calls above straddling
	// midnight — and Sunday under the bug, so it is damning on its own.
	if wd := app.list.per.ref.Weekday(); wd != time.Wednesday {
		t.Errorf("the list's period starts on a %s, want Wednesday", wd)
	}
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
	// The panel title names the view; the body is the key list itself.
	out := app.View()
	if !strings.Contains(out, panelTitle(modeHelp)) {
		t.Errorf("help panel missing its title:\n%s", out)
	}
	if !strings.Contains(out, "neuer Frame") {
		t.Errorf("help view missing the key list:\n%s", out)
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

// TestStatusShowsRunningTimer: the header carries marker, project, tag list and
// clock. The tag list is part of it on purpose — with an empty Tags slice the
// assertion passed even when timerField dropped the " [tags]" suffix, and the
// retired TestRenderStatusRunning was the only test that pinned it.
func TestStatusShowsRunningTimer(t *testing.T) {
	app := newTestApp(t)
	now := time.Now()
	app.state = &watson.State{
		Project: "proj", Start: now.Add(-90 * time.Second), Tags: []string{"dev", "ops"},
	}
	app.now = now
	if want := "▶ proj [dev, ops] 0:01:30"; !strings.Contains(app.View(), want) {
		t.Errorf("header must carry %q:\n%s", want, app.View())
	}
}

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

// TestDeleteErrorShowsStatus covers the Delete error path (ErrNotFound): the
// failure surfaces in a.errMsg and must not crash or leave a stuck mode.
func TestDeleteErrorShowsStatus(t *testing.T) {
	app := newTestApp(t)
	app.pendingDelete = watson.Frame{ID: "nonexistent"}
	app.mode = modeConfirmDelete
	app.Update(key("y"))
	if app.mode != modeList {
		t.Fatal("must return to list even on error")
	}
	if !strings.Contains(app.errMsg, "Löschen fehlgeschlagen") {
		t.Errorf("delete error must show in status bar, errMsg = %q", app.errMsg)
	}
}
