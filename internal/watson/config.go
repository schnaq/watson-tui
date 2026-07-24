package watson

import (
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// Config holds the subset of Watson's config the TUI reads.
type Config struct {
	WeekStart time.Weekday
}

// DefaultConfig returns the Config used when no config file is present or
// parsing fails; WeekStart defaults to time.Monday.
func DefaultConfig() Config {
	return Config{WeekStart: time.Monday}
}

var weekdays = map[string]time.Weekday{
	"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
	"sunday": time.Sunday,
}

// ParseConfig reads Watson's INI config; unknown values fall back to defaults.
func ParseConfig(data []byte) (Config, error) {
	cfg := DefaultConfig()
	file, err := ini.Load(data)
	if err != nil {
		return cfg, err
	}
	raw := strings.ToLower(strings.TrimSpace(file.Section("options").Key("week_start").String()))
	if wd, ok := weekdays[raw]; ok {
		cfg.WeekStart = wd
	}
	return cfg, nil
}
