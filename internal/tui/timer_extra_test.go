package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

// TestTimerStartPromptFocusAndInput covers the tab focus toggle (both
// directions) and the text-input default branch for both fields.
func TestTimerStartPromptFocusAndInput(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("s"))
	if app.start.focus != 0 {
		t.Fatalf("prompt must start on project field, focus = %d", app.start.focus)
	}
	app.Update(key("x")) // typing routes to the project field
	if app.start.project.Value() != "x" {
		t.Errorf("typing must update the project field, got %q", app.start.project.Value())
	}
	app.Update(key("tab"))
	if app.start.focus != 1 {
		t.Fatalf("tab must move focus to tags, focus = %d", app.start.focus)
	}
	app.Update(key("y")) // typing now routes to the tags field
	if app.start.tags.Value() != "y" {
		t.Errorf("typing must update the tags field, got %q", app.start.tags.Value())
	}
	app.Update(key("tab"))
	if app.start.focus != 0 {
		t.Errorf("tab must toggle focus back to project, focus = %d", app.start.focus)
	}
}

// TestTimerStartPromptView covers startModel.view() including the error branch.
func TestTimerStartPromptView(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("s"))
	app.start.errMsg = "Projekt fehlt"
	out := app.View()
	if !strings.Contains(out, "Timer starten") {
		t.Error("start prompt view missing title")
	}
	if !strings.Contains(out, "Projekt fehlt") {
		t.Error("start prompt view must show the error message")
	}
}

// TestTimerStartAlreadyRunningError covers the store.Start error path: the
// prompt must keep its message and stay open instead of crashing.
func TestTimerStartAlreadyRunningError(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("s")) // open prompt while idle
	app.start.project.SetValue("proj")
	if err := store.Start("other", nil, time.Now()); err != nil { // external start
		t.Fatal(err)
	}
	app.Update(key("enter"))
	if app.mode != modeStartTimer || !strings.Contains(app.start.errMsg, "Start fehlgeschlagen") {
		t.Errorf("start error must stay in prompt with a prefixed message, mode=%v errMsg=%q", app.mode, app.start.errMsg)
	}
}

// TestTimerStopError covers the store.Stop error path in updateList.
func TestTimerStopError(t *testing.T) {
	app := newTestApp(t)
	// state claims a running timer but the store has none → Stop errors.
	app.state = &watson.State{Project: "ghost", Start: time.Now(), Tags: []string{}}
	app.Update(key("s"))
	if !strings.Contains(app.errMsg, "Stop fehlgeschlagen") {
		t.Errorf("stop error must surface in status bar, errMsg = %q", app.errMsg)
	}
	if app.mode != modeList {
		t.Error("must stay in list after stop error")
	}
}

// TestTimerCancelOtherKeyAborts covers the "andere Taste" branch of the
// confirm-cancel dialog: it returns to the list and keeps the timer.
func TestTimerCancelOtherKeyAborts(t *testing.T) {
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
	app.Update(key("n")) // any other key aborts
	if app.mode != modeList {
		t.Fatal("other key must return to list")
	}
	if app.state == nil {
		t.Error("abort must keep the running timer")
	}
	frames, _ := store.Frames()
	if len(frames) != 0 {
		t.Errorf("abort must not create a frame: %+v", frames)
	}
}

// TestTimerCancelError covers the store.Cancel error path.
func TestTimerCancelError(t *testing.T) {
	app := newTestApp(t)
	app.mode = modeConfirmCancel // no running timer → Cancel returns ErrNotRunning
	app.Update(key("y"))
	if app.mode != modeList {
		t.Fatal("must return to list even on cancel error")
	}
	if !strings.Contains(app.errMsg, "Verwerfen fehlgeschlagen") {
		t.Errorf("cancel error must surface in status bar, errMsg = %q", app.errMsg)
	}
}

// TestTimerCapitalSIdleNoop covers the S branch when no timer runs.
func TestTimerCapitalSIdleNoop(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("S"))
	if app.mode != modeList {
		t.Errorf("S while idle must do nothing, mode = %v", app.mode)
	}
}
