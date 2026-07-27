package tui

// This file pins the whole rendered frame in a file, the way the chrome tests
// pin single pieces of it. Those answer "is the width invariant held"; this one
// answers "does the screen still look like the screen", which no assertion
// about a substring can.

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// TestGoldenListView pins the whole rendered frame. It exists to make layout
// regressions visible in a diff — run with -update after an intentional change
// and read the diff before committing it.
//
// Everything the frame is built from is fixed: a frozen clock and three frames
// spelled out with explicit dates, so no boundary of the real calendar can move
// a row. The frames are built in time.Local because the views format in local
// time — building them in UTC would make the rendered clock times depend on the
// TZ of whoever runs the test. The dates sit in July for the same kind of
// reason: no zone changes its offset then, so "09:00 local" is 09:00 everywhere.
//
// Two periods, because the header reads differently in each and one file cannot
// show both. "all frames" cannot be shifted, so it carries neither the ‹ ›
// affordance nor a comparison row and the header comes out one row short of its
// four-row budget — that blank row and where it lands is worth pinning. The week
// carries both, and all three frames fall inside it, so the body below is the
// same in either file and the diff between them is exactly the header.
//
// And both bodies, because f switches between two views of the same frames and
// only one of them can be the one the app opens on. The summary is what the two
// period files show; the third pins the frame list, so a change to either
// rendering has a file to diff against instead of only the view that happens to
// be the default.
func TestGoldenListView(t *testing.T) {
	// Under go test stdout is a pipe, so lipgloss falls back to the ASCII profile
	// and the frame comes out as plain text — which is what makes the file
	// readable and its diffs worth looking at. CLICOLOR_FORCE would put escape
	// sequences into every styled cell; saying so beats an unreadable diff.
	if strings.Contains(styleDim.Render("x"), "\x1b") {
		t.Fatal("colour is forced on (CLICOLOR_FORCE?); the golden file holds the plain frame")
	}

	now := time.Date(2026, 7, 22, 15, 4, 5, 0, time.Local)
	cases := []struct {
		name string
		// per takes the week start because the week case has to be built the way
		// the app builds it — normalised to the period's start, see period.
		per     func(time.Weekday) period
		compact bool
	}{
		{"list-100x30", func(time.Weekday) period { return period{unit: unitAll, ref: now} }, true},
		{"list-week-100x30", func(ws time.Weekday) period {
			return period{unit: unitWeek, ref: now}.shift(ws, 0)
		}, true},
		{"frames-100x30", func(time.Weekday) period { return period{unit: unitAll, ref: now} }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := NewApp(watson.NewStore(t.TempDir()), "0.1.0")
			app.Init()
			app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			app.now = now
			app.frames = []watson.Frame{
				mkFrame("a1111111111111111111111111111111", "schnaq",
					time.Date(2026, 7, 21, 9, 0, 0, 0, time.Local), 2*time.Hour+30*time.Minute, "dev"),
				mkFrame("b2222222222222222222222222222222", "kunde-a",
					time.Date(2026, 7, 21, 13, 0, 0, 0, time.Local), 4*time.Hour, "meeting"),
				mkFrame("c3333333333333333333333333333333", "kunde-b",
					time.Date(2026, 7, 22, 10, 0, 0, 0, time.Local), 3*time.Hour+15*time.Minute, "call"),
			}
			app.list.per = tc.per(app.cfg.WeekStart)
			app.list.compact = tc.compact
			app.list.refresh(app.frames, app.cfg.WeekStart, app.state, app.now)
			got := app.View()

			path := filepath.Join("testdata", tc.name+".golden")
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("wrote %s", path)
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v — run: go test ./internal/tui/ -run TestGoldenListView -update", err)
			}
			if got != string(want) {
				t.Errorf("frame changed.\n--- want ---\n%s\n--- got ---\n%s", want, got)
			}
		})
	}
}

// TestGoldenIsStable: rendering twice must produce the same frame, otherwise
// the golden file would flap.
func TestGoldenIsStable(t *testing.T) {
	app := newTestApp(t)
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app.now = time.Date(2026, 7, 22, 15, 4, 5, 0, time.Local)
	if first, second := app.View(), app.View(); first != second {
		t.Errorf("view is not deterministic:\n%s\n---\n%s", first, second)
	}
	if strings.TrimSpace(app.View()) == "" {
		t.Error("view is empty")
	}
}
