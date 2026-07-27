package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestPeriodLabel pins the exact status-bar strings for every unit.
// label() is an exported-contract item (Produces, Task 9) that the eight
// brief tests never exercise; Tasks 10-16 read these strings.
func TestPeriodLabel(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local) // Wednesday
	cases := []struct {
		unit periodUnit
		want string
	}{
		{unitDay, "Wednesday, 2026-07-22"},
		{unitWeek, "Week 2026-07-20 – 2026-07-26"},
		{unitMonth, "July 2026"},
		{unitAll, "all frames"},
	}
	for _, c := range cases {
		if got := (period{unit: c.unit, ref: ref}).label(time.Monday); got != c.want {
			t.Errorf("unit %d: label = %q, want %q", c.unit, got, c.want)
		}
	}
}

// TestPeriodShiftAllNoop covers the !ok early return in shift(): unitAll is
// unbounded, so shifting must leave the period untouched.
func TestPeriodShiftAllNoop(t *testing.T) {
	ref := time.Date(2026, 7, 22, 15, 0, 0, 0, time.Local)
	p := period{unit: unitAll, ref: ref}
	got := p.shift(time.Monday, 3)
	if got.unit != unitAll || !got.ref.Equal(ref) {
		t.Errorf("shift on unitAll changed period: %+v", got)
	}
}

// TestTruncate covers both branches: short strings pass through unchanged,
// long strings are cut rune-safely with an ellipsis.
func TestTruncate(t *testing.T) {
	if got := truncate("abcd", 4); got != "abcd" {
		t.Errorf("equal-length: got %q", got)
	}
	if got := truncate("abc", 10); got != "abc" {
		t.Errorf("short: got %q", got)
	}
	// Unicode: each umlaut is one rune, so the cut must not split bytes.
	if got := truncate("äöüxyz", 4); got != "äöü…" {
		t.Errorf("long unicode: got %q, want %q", got, "äöü…")
	}
}

// TestFirstAndNextFrameRowEmpty covers the "no frame" paths of the row
// helpers used for navigation.
func TestFirstAndNextFrameRowEmpty(t *testing.T) {
	if firstFrameRow(nil) != -1 {
		t.Error("firstFrameRow(nil) must be -1")
	}
	headers := []row{{kind: rowDayHeader}, {kind: rowDayHeader}}
	if firstFrameRow(headers) != -1 {
		t.Error("firstFrameRow(all headers) must be -1")
	}
	// No frame in the requested direction: return the starting index.
	if got := nextFrameRow(headers, 0, +1); got != 0 {
		t.Errorf("nextFrameRow with no target = %d, want 0", got)
	}
}

// TestPeriodRefStartsAtThePeriodStart: ref must be the period's start from the
// moment the period is built, not only after a shift. Until it was, the same
// view answered two ways: the week of Monday 31.08.2026, looked at on Sunday
// 06.09., resolved to September — because ref was the arbitrary instant of
// construction — while after ] and [ back it resolved to August. Whoever asks
// "which month is this week in" has to get one answer, and the header does ask.
func TestPeriodRefStartsAtThePeriodStart(t *testing.T) {
	sunday := time.Date(2026, 9, 6, 17, 30, 0, 0, time.Local) // last day of that week
	want := time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local)    // its Monday
	if got := newListModel(sunday, time.Monday).per.ref; !got.Equal(want) {
		t.Errorf("newListModel ref = %v, want the week's start %v", got, want)
	}
	if got := newReportModel(sunday, time.Monday).per.ref; !got.Equal(want) {
		t.Errorf("newReportModel ref = %v, want the week's start %v", got, want)
	}
	// The same period, one shift out and back: the invariant is that these two
	// ways of arriving at a week cannot disagree.
	shifted := newListModel(sunday, time.Monday).per.shift(time.Monday, +1).shift(time.Monday, -1)
	if got := newListModel(sunday, time.Monday).per; got.ref != shifted.ref {
		t.Errorf("constructed ref %v, but ] then [ gives %v", got.ref, shifted.ref)
	}
}

// TestAppPeriodKeysNormaliseRef: every key that builds a period has to leave it
// normalised too, or the invariant holds only until the user presses w. Against
// bounds() rather than a literal, so the assertion does not depend on the day
// the suite runs on.
func TestAppPeriodKeysNormaliseRef(t *testing.T) {
	normalised := func(t *testing.T, what string, p period, weekStart time.Weekday) {
		t.Helper()
		from, _, ok := p.bounds(weekStart)
		if !ok {
			return // unitAll has no bounds to normalise to
		}
		if !p.ref.Equal(from) {
			t.Errorf("%s: ref = %v, want the period's start %v", what, p.ref, from)
		}
	}
	for _, k := range []string{"t", "w", "m", "a"} {
		app := newTestApp(t)
		app.Update(key(k))
		normalised(t, "list key "+k, app.list.per, app.cfg.WeekStart)
	}
	for _, k := range []string{"t", "w", "m"} {
		app := newTestApp(t)
		app.Update(key("r")) // opens the report, which builds a period of its own
		normalised(t, "report", app.report.per, app.cfg.WeekStart)
		app.Update(key(k))
		normalised(t, "report key "+k, app.report.per, app.cfg.WeekStart)
	}
}

// TestListMoveEmpty ensures move() and selected() are safe on an empty list
// (cursor == -1).
func TestListMoveEmpty(t *testing.T) {
	l := newListModel(time.Now(), time.Monday)
	if _, ok := l.selected(); ok {
		t.Error("empty list must not report a selection")
	}
	l.move(+1) // must not panic and must not leave -1
	if l.cursor != -1 {
		t.Errorf("move on empty list changed cursor to %d", l.cursor)
	}
}

// TestListViewEmptyMessages covers view()'s two empty-state messages.
func TestListViewEmptyMessages(t *testing.T) {
	l := newListModel(time.Now(), time.Monday)
	if got := l.view(10, 80); !strings.Contains(got, "no frames in this period") {
		t.Errorf("empty view = %q", got)
	}
	l.filter = "zzz"
	if got := l.view(10, 80); !strings.Contains(got, "no match for filter “zzz”") {
		t.Errorf("no-match view = %q", got)
	}
}

// TestHeaderFieldsListBranches covers the three filter branches of the list
// header: no filter (the field says so instead of vanishing), a set filter, and
// the live input while typing, which has to replace the stored filter rather
// than sit next to it.
func TestHeaderFieldsListBranches(t *testing.T) {
	app := newTestApp(t)
	app.mode = modeList
	app.list.per = period{unit: unitAll, ref: time.Now()}
	app.list.refresh(app.frames, time.Monday, app.state, app.now)

	// "0 frames", not "Frames 0": the count shares its row with the number of
	// projects, so it carries its own unit instead of a label.
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "0 frames") ||
		!strings.Contains(got, "Filter —") {
		t.Errorf("plain header = %q", got)
	}

	app.list.filter = "foo"
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "Filter foo") {
		t.Errorf("filter header = %q", got)
	}

	app.list.filtering = true
	app.list.filterInput.Focus()
	app.list.filterInput.SetValue("bar")
	if got := renderFieldsFlat(app.headerFields()); !strings.Contains(got, "bar") ||
		strings.Contains(got, "Filter foo") {
		t.Errorf("filtering header = %q", got)
	}
}

// TestListViewRendersRows covers the non-empty view/renderRow path (header +
// frame line), which the App.View default branch reaches at runtime but the
// brief tests do not.
func TestListViewRendersRows(t *testing.T) {
	day := time.Date(2026, 7, 20, 9, 0, 0, 0, time.Local) // Monday
	l := newListModel(day, time.Monday)
	l.per = period{unit: unitWeek, ref: day}
	l.compact = false // the frame list, which is what this test reads back
	l.refresh([]watson.Frame{
		mkFrame("a1111111111111111111111111111111", "alpha", day, time.Hour, "tag1"),
	}, time.Monday, nil, time.Time{})
	out := l.view(10, 80)
	if !strings.Contains(out, "alpha") {
		t.Errorf("view missing project: %q", out)
	}
	if !strings.Contains(out, "Monday, 2026-07-20") { // day header
		t.Errorf("view missing day header: %q", out)
	}
}

// TestLabelsAreEnglishISO: the display format matches what the frame form
// accepts as input, so a date read off the screen can be typed straight back in.
// The day label goes through formatDay, so it carries the weekday too.
func TestLabelsAreEnglishISO(t *testing.T) {
	// Wednesday 2026-07-22; the week (Mon) runs 20.–26.
	ref := time.Date(2026, 7, 22, 12, 0, 0, 0, time.Local)
	cases := map[periodUnit]string{
		unitDay:   "Wednesday, 2026-07-22",
		unitWeek:  "Week 2026-07-20 – 2026-07-26",
		unitMonth: "July 2026",
		unitYear:  "Year 2026",
		unitAll:   "all frames",
	}
	for unit, want := range cases {
		if got := (period{unit: unit, ref: ref}).label(time.Monday); got != want {
			t.Errorf("unit %d label = %q, want %q", unit, got, want)
		}
	}
	if got := formatDay(ref); got != "Wednesday, 2026-07-22" {
		t.Errorf("formatDay = %q, want %q", got, "Wednesday, 2026-07-22")
	}
}

// TestRowSelectability: the cursor rests on frames in the list and on projects
// in the summary — never on a header, a tag line or a blank.
func TestRowSelectability(t *testing.T) {
	cases := map[rowKind]bool{
		rowFrame: true, rowProject: true,
		rowDayHeader: false, rowTag: false, rowBlank: false,
	}
	for kind, want := range cases {
		if got := (row{kind: kind}).selectable(); got != want {
			t.Errorf("kind %d selectable = %v, want %v", kind, got, want)
		}
	}
}

// TestSelectedOnlyReturnsFrames: a project row is selectable but is not a
// frame, so selected() must refuse it — otherwise enter would edit a zero frame.
func TestSelectedOnlyReturnsFrames(t *testing.T) {
	l := listModel{rows: []row{{kind: rowProject, title: "alpha"}}, cursor: 0}
	if _, ok := l.selected(); ok {
		t.Error("selected() must not return a frame for a project row")
	}
}
