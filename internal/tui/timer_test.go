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
