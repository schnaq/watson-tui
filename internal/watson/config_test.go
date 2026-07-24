package watson

import (
	"testing"
	"time"
)

func TestParseConfigWeekStart(t *testing.T) {
	cfg, err := ParseConfig([]byte("[options]\nweek_start = sunday\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WeekStart != time.Sunday {
		t.Errorf("got %v, want Sunday", cfg.WeekStart)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	for _, in := range []string{"", "[backend]\nurl = x\n", "[options]\nweek_start = kaputt\n"} {
		cfg, err := ParseConfig([]byte(in))
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if cfg.WeekStart != time.Monday {
			t.Errorf("%q: got %v, want Monday", in, cfg.WeekStart)
		}
	}
}
