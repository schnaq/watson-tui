package tui

// This file holds the frame list's half of the rule the overview and the report
// already follow: when the width does not fit, a label gives way, never a number
// and never an identifier — and a shortened value must never look complete.

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/schnaq/watson-tui/internal/watson"
)

// wideAndNarrowFrames are two frames of the same day whose IDs share the seven
// characters ShortID renders, one of them long enough to need a three-digit hour
// count. Both defects this file pins need exactly that pair: the wide duration
// shifts the columns behind it, and the shared prefix is what makes a shortened
// ID dangerous rather than merely ugly — watson resolves IDs by prefix
// (watson.FindByIDPrefix), so "a11111" read off the screen is ambiguous between
// these two frames and "watson edit a11111" either refuses or edits the wrong one.
func wideAndNarrowFrames(day time.Time) []watson.Frame {
	return []watson.Frame{
		// Three weeks on one frame: what the a (all) period shows for a project that
		// has been running for a while.
		mkFrame("a11111122222222222222222222222222", "schnaq", day, 130*time.Hour, "dev"),
		mkFrame("a11111199999999999999999999999999", "kunde-a", day.Add(2*time.Hour),
			30*time.Minute, "meeting"),
	}
}

// TestListKeepsColumnsAlignedPastNinetyNineHours: the duration field was a fixed
// %7s, but "130h 00m" is eight columns, so every row with a three-digit hour
// count pushed the columns behind it one to the right — and the ID sits at the
// end of the row, which is the column fitBody cuts first. The field is now as
// wide as the widest duration the list actually renders, the way report.go sizes
// its number column, and it stays right-aligned so the numbers line up.
func TestListKeepsColumnsAlignedPastNinetyNineHours(t *testing.T) {
	day := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := wideAndNarrowFrames(day)
	l := newListModel(day, time.Monday)
	l.per = period{unit: unitAll, ref: day}
	l.compact = false // this file is the frame list's half of the width rule
	l.refresh(frames, time.Monday, nil, time.Time{})
	// No row is the cursor row, so nothing is styled and the assertions can count
	// columns in the plain line instead of stepping over escape sequences.
	l.cursor = -1
	durW := listDurWidth(l.rows)

	var starts []int
	for i, r := range l.rows {
		if r.kind != rowFrame {
			continue
		}
		fr := r.frame
		line := l.renderRow(i, 100, viewLayout{durW: durW})
		times := fr.Start.Local().Format("15:04") + "–" + fr.Stop.Local().Format("15:04")
		_, rest, ok := strings.Cut(line, times)
		if !ok {
			t.Fatalf("row %d does not show its times %q: %q", i, times, line)
		}
		// Everything between the times and the project name is the duration cell,
		// padding included. Trimmed it has to be the duration verbatim: a cut number
		// would still be a substring of the line, so Contains would not catch it.
		cell, _, ok := strings.Cut(rest, fr.Project)
		if !ok {
			t.Fatalf("row %d does not show its project %q: %q", i, fr.Project, line)
		}
		if got, want := strings.TrimSpace(cell), formatDuration(fr.Duration()); got != want {
			t.Errorf("row %d duration = %q, want %q: %q", i, got, want, line)
		}
		starts = append(starts, lipgloss.Width(line[:strings.Index(line, fr.Project)]))
	}
	if len(starts) != 2 {
		t.Fatalf("got %d frame rows, want 2", len(starts))
	}
	if starts[0] != starts[1] {
		t.Errorf("project column starts at column %d in the 130h row and at %d in the 30m row",
			starts[0], starts[1])
	}
}

// ansiSeq matches the colour sequences lipgloss wraps a styled line in. The
// assertions below tokenize rendered lines, and a token that came back as
// "\x1b[1;34ma111111" would slip through the prefix check — so the escapes come
// off first, rather than the test refusing to run when colour is forced on.
var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// plainFields splits a rendered line into its visible tokens, with the colour
// sequences and the panel's border columns and gutter removed.
func plainFields(line string) []string {
	return strings.Fields(strings.Trim(ansiSeq.ReplaceAllString(line, ""), "│ "))
}

// assertWholeIDs fails for any token that is a prefix of a real frame ID without
// being the whole of what ShortID renders. That is the dangerous case rather than
// merely the ugly one: watson resolves frame IDs by prefix (watson.FindByIDPrefix,
// and the watson CLI the same way), so "a11111" read off the screen is ambiguous
// between the two fixture frames — "watson edit a11111" either refuses or, with
// other data, silently edits the wrong frame. A truncated identifier that looks
// complete is worse than no identifier, so the column is shown whole or not at all.
//
// minTok is the shortest token worth looking at. The list's own lines are checked
// down to a single character; the whole screen is checked from two up, because the
// chrome clips its labels — on a 25-column terminal the period label "alle Frames"
// comes back as "a", which is a label cut short and not an identifier.
func assertWholeIDs(t *testing.T, where, line string, frames []watson.Frame, minTok int) {
	t.Helper()
	for _, tok := range plainFields(line) {
		// An ellipsis does not make a partial ID acceptable, so it comes off before
		// the comparison: seven columns are not enough to spell out that "a11…"
		// cannot be typed at watson, and the rule is whole column or none. Stripping
		// it can leave nothing at all — a project name cut to one column is a bare
		// ellipsis — and every ID starts with the empty string, hence the length
		// guard afterwards.
		tok = strings.TrimSuffix(tok, "…")
		if len(tok) < minTok {
			continue
		}
		for _, fr := range frames {
			if strings.HasPrefix(fr.ID, tok) && tok != fr.ShortID() {
				t.Errorf("%s: %q is a prefix of frame %s but not its ShortID %q — "+
					"read off the screen it resolves to the wrong frame or to none: %q",
					where, tok, fr.ID, fr.ShortID(), line)
			}
		}
	}
}

// TestListNeverShowsAPartialFrameID sweeps every width a terminal may have.
// renderRow used to build the row from a fixed format string and never negotiated
// against the width at all; fitBody then cut the right edge, which is exactly
// where the ID sits. With the two fixture frames that gave "a111111" and "a111111"
// at 100 columns, "a111111" and "a11111" at 89 — the second one short of a
// character and no way to tell — "a11" and "a1" at 85, and nothing at all below 82.
func TestListNeverShowsAPartialFrameID(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 4, 5, 0, time.Local)
	day := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := wideAndNarrowFrames(day)
	l := newListModel(day, time.Monday)
	l.per = period{unit: unitAll, ref: day}
	l.compact = false // this file is the frame list's half of the width rule
	l.refresh(frames, time.Monday, nil, time.Time{})

	for width := 20; width <= 140; width++ {
		body := l.view(len(l.rows), width)
		for i, line := range strings.Split(body, "\n") {
			// The row is laid out against the width instead of relying on fitBody to
			// cut it, so it has to fit on its own.
			if w := lipgloss.Width(line); w > width {
				t.Errorf("width %d: line %d is %d columns wide: %q", width, i, w, line)
			}
			assertWholeIDs(t, fmt.Sprintf("body width %d", width), line, frames, 1)
		}
	}

	// The same sweep through App.View, because that is where the characters
	// actually went: the list gets the terminal width minus the panel, and fitBody
	// cuts whatever still comes back too wide.
	for width := 20; width <= 140; width++ {
		app := chromeApp(t, now, width, 24, frames)
		app.now = now
		for _, line := range strings.Split(app.View(), "\n") {
			assertWholeIDs(t, fmt.Sprintf("%dx24", width), line, frames, 2)
		}
	}

	// Teeth: never rendering the column at all would satisfy the rule above, so
	// the IDs have to stand there in full wherever there is room for them.
	wide := l.view(len(l.rows), 96)
	for _, fr := range frames {
		if !strings.Contains(wide, fr.ShortID()) {
			t.Errorf("at 96 columns the list must show %q in full:\n%s", fr.ShortID(), wide)
		}
	}
}

// TestListLayoutGivesWayInPriorityOrder pins the order the columns give way in,
// the way TestOverviewDropOrderIsPriority pins the overview's. Walking down from
// a width where everything fits, each column has to start giving way in the
// documented sequence — and the duration, the number the row is billed from, must
// still be whole while anything else has room left.
func TestListLayoutGivesWayInPriorityOrder(t *testing.T) {
	const durW = 8 // "130h 00m"
	// The widest terminal at which each column has begun to give way.
	var tagsCut, tagsGone, projCut, idGone, timesGone int
	note := func(dst *int, cond bool, width int) {
		if *dst == 0 && cond {
			*dst = width
		}
	}
	for width := 200; width >= listMarkerWidth; width-- {
		rl := listLayout(width, durW)
		if rl.columns() > width {
			t.Fatalf("width %d: layout %+v needs %d columns", width, rl, rl.columns())
		}
		if rl.durW < durW && (rl.times || rl.id || rl.projW > 0 || rl.tagW > 0) {
			t.Errorf("width %d: the duration gave way while other columns still stood: %+v",
				width, rl)
		}
		note(&tagsCut, rl.tagW < listTagsMax, width)
		note(&tagsGone, rl.tagW == 0, width)
		note(&projCut, rl.projW < listProjMax, width)
		note(&idGone, !rl.id, width)
		note(&timesGone, !rl.times, width)
	}
	inOrder := tagsCut > tagsGone && tagsGone >= projCut &&
		projCut > idGone && idGone > timesGone
	if !inOrder {
		t.Errorf("give-way order is tags cut %d, tags gone %d, project cut %d, ID dropped %d, "+
			"times dropped %d — want them in that descending order",
			tagsCut, tagsGone, projCut, idGone, timesGone)
	}
}

// TestListLastResortCutsTheNumberVisibly covers the bottom of the ladder: below
// the marker column plus the duration there is nothing left to give up, so the
// duration itself is cut — and then it must carry an ellipsis, because "130h 00m"
// that came back as "130h" is a plausible wrong number. Unreachable through
// App.View (a 20-column terminal still leaves the list a body of 16), so the row
// is rendered directly.
func TestListLastResortCutsTheNumberVisibly(t *testing.T) {
	day := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	l := newListModel(day, time.Monday)
	l.per = period{unit: unitAll, ref: day}
	l.compact = false // this file is the frame list's half of the width rule
	l.refresh(wideAndNarrowFrames(day), time.Monday, nil, time.Time{})
	l.cursor = -1
	const durW = 8 // "130h 00m", the widest duration in the fixture

	for _, width := range []int{5, 8} {
		line := l.renderRow(1, width, viewLayout{durW: durW})
		if w := lipgloss.Width(line); w > width {
			t.Errorf("width %d: line is %d columns wide: %q", width, w, line)
		}
		if strings.Contains(line, "130h 00m") {
			t.Errorf("width %d: the full duration cannot fit yet stands there: %q", width, line)
		}
		if !strings.Contains(line, "…") {
			t.Errorf("width %d: the cut duration reads as a complete number: %q", width, line)
		}
	}
}

// TestListKeepsTheDayTotalWhole pins the same rule for the day header, which the
// sweep above found breaking it too: the header was one pre-built string of 31
// columns ("Dienstag, 21.07.2026 — 130h 30m"), so on a narrow terminal fitBody cut
// it from the right — where the day's total sits. At a body of 28 columns it read
// "Dienstag, 21.07.2026 — 130h", a plausible wrong number for a day that is 130h
// 30m long. The day is a label and gives way with an ellipsis; the total does not.
func TestListKeepsTheDayTotalWhole(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 4, 5, 0, time.Local)
	day := time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local)
	frames := wideAndNarrowFrames(day)
	l := newListModel(day, time.Monday)
	l.per = period{unit: unitAll, ref: day}
	l.compact = false // this file is the frame list's half of the width rule
	l.refresh(frames, time.Monday, nil, time.Time{})
	total := formatDuration(130*time.Hour + 30*time.Minute)

	for width := 20; width <= 140; width++ {
		header := strings.Split(l.view(len(l.rows), width), "\n")[0]
		if got := strings.TrimRight(ansiSeq.ReplaceAllString(header, ""), " "); !strings.HasSuffix(got, total) {
			t.Errorf("body width %d: day header does not end in the day's total %q: %q",
				width, total, got)
		}
	}
	for width := 20; width <= 140; width++ {
		app := chromeApp(t, now, width, 24, frames)
		app.now = now
		found := false
		for _, line := range strings.Split(app.View(), "\n") {
			// Some line has to end in the day's total. Looking the day header up by its
			// day would not do — that is the half of the line which gives way, and on a
			// narrow terminal it reads "Dien…". The panel closes every body line with
			// padding and a border column, so those come off first.
			plain := strings.TrimRight(ansiSeq.ReplaceAllString(line, ""), " │")
			if strings.HasSuffix(plain, total) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%dx24: the day header lost its total %q:\n%s", width, total, app.View())
		}
	}
}
