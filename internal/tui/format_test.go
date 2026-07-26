package tui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{45 * time.Minute, "45m"},
		{time.Hour + 5*time.Minute, "1h 05m"},
		{6*time.Hour + 30*time.Minute, "6h 30m"},
		{0, "0m"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestFormatClock(t *testing.T) {
	if got := formatClock(90 * time.Second); got != "0:01:30" {
		t.Errorf("got %q", got)
	}
	if got := formatClock(3*time.Hour + 62*time.Second); got != "3:01:02" {
		t.Errorf("got %q", got)
	}
}
