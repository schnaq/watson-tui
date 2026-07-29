//go:build ignore

// gen-demo-data writes a fabricated Watson data directory (./demo-data by
// default) for taking screenshots without exposing real tracked time.
// Re-run any time — it always anchors on the real current week, so "today"
// stays today and the running timer stays fresh.
//
//	go run scripts/gen-demo-data.go [target-dir]
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/schnaq/watson-tui/internal/watson"
)

// session is one booked slot within a Mon–Fri day template.
type session struct {
	project, tag   string
	startH, startM int
	durMin         int
}

// dayTemplates are cycled deterministically across weeks/weekdays so the
// data varies without needing real randomness.
var dayTemplates = [][]session{
	{{"schnaq", "dev", 9, 0, 180}, {"kunde-a", "meeting", 13, 0, 60}},
	{{"kunde-b", "call", 9, 15, 45}, {"schnaq", "dev", 10, 15, 165}, {"schnaq", "review", 14, 0, 90}},
	{{"kunde-a", "onsite", 9, 0, 480}},
	{{"kunde-c", "dev", 9, 0, 120}, {"kunde-c", "standup", 11, 0, 15}, {"kunde-b", "support", 13, 0, 180}},
	{{"schnaq", "planning", 9, 0, 90}, {"kunde-a", "meeting", 11, 0, 60}},
	{{"kunde-b", "support", 9, 0, 210}, {"kunde-b", "call", 13, 30, 30}},
	{{"kunde-c", "dev", 9, 0, 360}},
	{{"schnaq", "dev", 9, 0, 120}, {"kunde-a", "meeting", 11, 15, 45}, {"kunde-a", "onsite", 13, 0, 240}},
	{{"kunde-b", "call", 9, 0, 30}, {"kunde-c", "dev", 10, 0, 180}, {"kunde-c", "standup", 13, 0, 15}},
	{{"kunde-a", "onsite", 9, 0, 240}},
	{{"schnaq", "dev", 9, 0, 150}, {"kunde-a", "meeting", 13, 0, 240}},
	{{"kunde-b", "call", 10, 0, 195}, {"schnaq", "review", 14, 0, 90}},
	{{"kunde-c", "dev", 9, 0, 180}, {"kunde-c", "standup", 12, 0, 15}, {"kunde-a", "onsite", 13, 0, 180}},
	{{"kunde-a", "meeting", 9, 0, 30}, {"kunde-a", "onsite", 10, 0, 300}},
}

const historyWeeks = 4 // full Mon–Fri weeks before the current one

// emptyDay marks a deliberate gap (e.g. a day off) so the summary shows the
// "worked but nothing today" dimmed state too, not just weekends.
type dayKey struct{ week, weekday int } // week: weeks-ago (0 = current), weekday: 0=Mon..4=Fri

var emptyDays = map[dayKey]bool{
	{week: 2, weekday: 2}: true, // a Wednesday off, two weeks back
}

func newID() string {
	u := uuid.New()
	return fmt.Sprintf("%x", u[:])
}

func main() {
	dir := "demo-data"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	now := time.Now()
	// Monday of the current week, local midnight.
	todayIdx := (int(now.Weekday()) + 6) % 7 // Mon=0..Sun=6
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	currentMonday := todayMidnight.AddDate(0, 0, -todayIdx)

	var frames []watson.Frame
	addSession := func(day time.Time, s session) {
		start := time.Date(day.Year(), day.Month(), day.Day(), s.startH, s.startM, 0, 0, time.Local)
		stop := start.Add(time.Duration(s.durMin) * time.Minute)
		frames = append(frames, watson.Frame{
			Start: start.UTC(), Stop: stop.UTC(), Project: s.project,
			ID: newID(), Tags: []string{s.tag}, UpdatedAt: stop.UTC(),
		})
	}

	template := func(weeksAgo, weekday int) []session {
		return dayTemplates[(weeksAgo*5+weekday)%len(dayTemplates)]
	}

	// Past full weeks: Mon–Fri only, weekends stay empty.
	for w := historyWeeks; w >= 1; w-- {
		monday := currentMonday.AddDate(0, 0, -7*w)
		for d := 0; d < 5; d++ {
			if emptyDays[dayKey{week: w, weekday: d}] {
				continue
			}
			day := monday.AddDate(0, 0, d)
			for _, s := range template(w, d) {
				addSession(day, s)
			}
		}
	}

	// Current week up to yesterday.
	for d := 0; d < todayIdx; d++ {
		day := currentMonday.AddDate(0, 0, d)
		for _, s := range template(0, d) {
			addSession(day, s)
		}
	}

	// Today, if it's a weekday: one finished morning session plus a
	// currently running timer, so the header shows a live timer too.
	var state *watson.State
	if todayIdx < 5 {
		addSession(todayMidnight, session{"schnaq", "dev", 9, 0, 150})
		runningStart := now.Add(-45 * time.Minute)
		state = &watson.State{Project: "kunde-b", Start: runningStart.UTC(), Tags: []string{"support"}}
	}

	watson.SortFrames(frames)
	framesData, err := watson.MarshalFrames(frames)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(dir, "frames"), framesData, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	stateData, err := watson.MarshalState(state)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(dir, "state"), stateData, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("wrote %d frames to %s (state: %v)\n", len(frames), dir, state != nil)
}
