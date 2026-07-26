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

// TestFormViewShortFrameID: a foreign frames file may carry an ID shorter than
// the seven chars the title shows. View() must not panic on it — a panic there
// leaves the terminal in the alt-screen. The title lives in the chrome now, so
// the short ID is asserted where it is rendered: the header of the form mode.
func TestFormViewShortFrameID(t *testing.T) {
	now := time.Now()
	existing := watson.Frame{ID: "abc", Project: "p", Start: now.Add(-time.Hour), Stop: now, Tags: []string{}}
	app := newTestApp(t)
	app.form = newFormModel(&existing, nil, now)
	app.mode = modeForm
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "bearbeiten (abc)") {
		t.Errorf("short id must render verbatim: %q", got)
	}
	if out := app.View(); !strings.Contains(out, "bearbeiten (abc)") {
		t.Errorf("assembled view lost the frame it edits:\n%s", out)
	}
}

// TestOverlapWarningResetsOnEdit: after the warning the user may change the
// frame. The next enter must re-check instead of saving a new, never-checked
// overlap on the latched flag.
func TestOverlapWarningResetsOnEdit(t *testing.T) {
	store := watson.NewStore(t.TempDir())
	if _, err := store.Add(watson.Frame{
		Start:   time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local),
		Stop:    time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local),
		Project: "bestehend", Tags: []string{},
	}, time.Now()); err != nil {
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
	for i := 0; i < 3; i++ {
		app.Update(key("tab")) // Projekt → Tags
	}
	app.Update(key("x")) // Feld geändert
	if app.form.warned || app.form.errMsg != "" {
		t.Errorf("changed field must re-arm the check, warned=%v errMsg=%q", app.form.warned, app.form.errMsg)
	}
	app.Update(key("enter"))
	if app.mode != modeForm || !app.form.warned {
		t.Fatalf("the still-overlapping frame must warn again, mode=%v", app.mode)
	}
	frames, _ := store.Frames()
	if len(frames) != 1 {
		t.Errorf("nothing must be saved yet, frames = %+v", frames)
	}
}

// TestFormView: the body carries the fields and the error line; the chrome
// carries what it used to repeat — the header says which frame is being edited,
// the footer names the keys. Both are asserted, so the coverage moved with the
// text instead of disappearing.
func TestFormView(t *testing.T) {
	app := newTestApp(t)
	app.form = newFormModel(nil, nil, time.Now())
	app.mode = modeForm
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Frame neu") {
		t.Errorf("header must name the new frame: %q", got)
	}
	if got := strings.Join(footerHints(modeForm), hintSep); !strings.Contains(got, "esc abbrechen") {
		t.Errorf("footer must carry the abort key: %q", got)
	}
	out := app.form.view()
	if strings.Contains(out, "Neuer Frame") || strings.Contains(out, "esc:") {
		t.Errorf("body must not repeat the chrome's title or hints:\n%s", out)
	}

	existing := watson.Frame{
		ID: "abcdef01234567890123456789012345", Project: "p",
		Start: time.Now().Add(-time.Hour), Stop: time.Now(), Tags: []string{"x"},
	}
	app.form = newFormModel(&existing, nil, time.Now())
	app.form.errMsg = "kaputt"
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Frame bearbeiten (abcdef0") {
		t.Errorf("header must name the edited frame: %q", got)
	}
	if out := app.form.view(); !strings.Contains(out, "kaputt") {
		t.Errorf("body must keep the errMsg:\n%s", out)
	}
}
