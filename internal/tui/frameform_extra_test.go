package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

// parseDateTime also accepts the seconds layout; the minute layout drops seconds.
func TestParseDateTimeSeconds(t *testing.T) {
	now := time.Date(2026, 7, 24, 14, 0, 0, 0, time.Local)
	got, err := parseDateTime("2026-07-20 09:30:45", now)
	if err != nil || got.Second() != 45 || got.Minute() != 30 || got.Hour() != 9 {
		t.Errorf("seconds layout: %v, %v", got, err)
	}
}

// buildFrame propagates parse errors from either time field.
func TestBuildFrameParseErrors(t *testing.T) {
	now := time.Now()
	if _, err := buildFrame("p", "kaputt", "10:00", "", now); err == nil {
		t.Error("invalid start must fail")
	}
	if _, err := buildFrame("p", "09:00", "kaputt", "", now); err == nil {
		t.Error("invalid stop must fail")
	}
}

// splitTags yields no tags for empty or whitespace-only input.
func TestSplitTagsEmpty(t *testing.T) {
	if got := splitTags(""); len(got) != 0 {
		t.Errorf("empty = %v", got)
	}
	if got := splitTags("  ,  , "); len(got) != 0 {
		t.Errorf("whitespace-only = %v", got)
	}
}

// projectNames deduplicates and sorts.
func TestProjectNames(t *testing.T) {
	frames := []watson.Frame{
		{Project: "beta"}, {Project: "alpha"}, {Project: "beta"}, {Project: "gamma"},
	}
	got := projectNames(frames)
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// updateForm navigates fields (tab/shift+tab with wrap-around) and cancels on esc.
func TestUpdateFormNavigationAndEsc(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("n"))
	if app.mode != modeForm || app.form.focus != fieldProject {
		t.Fatalf("new form must focus project, mode=%v focus=%d", app.mode, app.form.focus)
	}
	app.Update(key("tab"))
	if app.form.focus != fieldStart {
		t.Errorf("tab focus = %d, want %d", app.form.focus, fieldStart)
	}
	app.Update(key("shift+tab"))
	app.Update(key("shift+tab")) // wrap past project (0) to last field
	if app.form.focus != fieldTags {
		t.Errorf("wrap focus = %d, want %d", app.form.focus, fieldTags)
	}
	app.Update(key("esc"))
	if app.mode != modeList {
		t.Errorf("esc must return to list, mode=%v", app.mode)
	}
}

// updateForm forwards ordinary key presses to the focused input.
func TestUpdateFormTypesIntoField(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("n"))
	app.Update(key("z"))
	if v := app.form.inputs[fieldProject].Value(); v != "z" {
		t.Errorf("typed value = %q, want %q", v, "z")
	}
}

// submitForm surfaces validation errors and stays in the form.
func TestSubmitFormValidationError(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("n"))
	app.form.inputs[fieldProject].SetValue("") // empty project is invalid
	app.Update(key("enter"))
	if app.mode != modeForm {
		t.Fatal("validation error must keep the form open")
	}
	if app.form.errMsg == "" {
		t.Error("validation error must set errMsg")
	}
}

// TestEditRoundTripKeepsSeconds: Watson timestamps carry seconds. A pure
// project rename must leave the times untouched — a minute-precision pre-fill
// would be re-parsed on save and silently shorten the frame, which then lands
// on an invoice.
func TestEditRoundTripKeepsSeconds(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	added, err := store.Add(watson.Frame{
		Start:   time.Date(2026, 7, 20, 9, 0, 17, 0, time.Local),
		Stop:    time.Date(2026, 7, 20, 10, 30, 42, 0, time.Local),
		Project: "alt", Tags: []string{},
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(store, "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.Update(key("a")) // Zeitraum "alles"
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
		t.Fatalf("frames = %+v", frames)
	}
	if !frames[0].Start.Equal(added.Start) || !frames[0].Stop.Equal(added.Stop) {
		t.Errorf("edit shifted the times: %v–%v, want %v–%v",
			frames[0].Start, frames[0].Stop, added.Start, added.Stop)
	}
	if got, want := frames[0].Duration(), added.Duration(); got != want {
		t.Errorf("duration = %v, want %v", got, want)
	}
}

// view renders both titles and the error line.
func TestFormView(t *testing.T) {
	newForm := newFormModel(nil, nil, time.Now())
	out := newForm.view()
	if !strings.Contains(out, "Neuer Frame") {
		t.Error("new form view missing title")
	}
	if !strings.Contains(out, "esc: abbrechen") {
		t.Error("view missing key hint")
	}

	existing := watson.Frame{
		ID: "abcdef01234567890123456789012345", Project: "p",
		Start: time.Now().Add(-time.Hour), Stop: time.Now(), Tags: []string{"x"},
	}
	editForm := newFormModel(&existing, nil, time.Now())
	editForm.errMsg = "kaputt"
	out = editForm.view()
	if !strings.Contains(out, "Frame bearbeiten (abcdef0") {
		t.Errorf("edit form view missing title: %q", out)
	}
	if !strings.Contains(out, "kaputt") {
		t.Error("view missing errMsg")
	}
}
