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
	app.Update(key("a")) // period "all", independent of week boundaries
	app.Update(key("f")) // enter edits a frame, and only the frame list selects one
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
		Start:   time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local),
		Stop:    time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local),
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
