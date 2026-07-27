package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	keybind "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

const (
	dtLayout    = "2006-01-02 15:04"
	dtLayoutSec = "2006-01-02 15:04:05"
)

// editLayout picks the pre-fill layout for an existing frame. Watson timestamps
// carry seconds, so a minute-precision pre-fill would be re-parsed on save and
// silently shorten the frame by up to a minute — even on a pure project rename.
func editLayout(start, stop time.Time) string {
	if start.Second() != 0 || stop.Second() != 0 {
		return dtLayoutSec
	}
	return dtLayout
}

// parseDateTime accepts "2006-01-02 15:04[:05]" and "15:04" (= today), local time.
func parseDateTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	loc := now.Location()
	for _, layout := range []string{dtLayoutSec, dtLayout} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	if t, err := time.ParseInLocation("15:04", s, loc); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, loc), nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q (YYYY-MM-DD HH:MM or HH:MM)", s)
}

// splitTags parses a comma separated tag list.
func splitTags(s string) []string {
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// buildFrame validates form fields into a Frame; ID and UpdatedAt stay unset.
func buildFrame(project, startStr, stopStr, tagsStr string, now time.Time) (watson.Frame, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return watson.Frame{}, errors.New("project is missing")
	}
	start, err := parseDateTime(startStr, now)
	if err != nil {
		return watson.Frame{}, err
	}
	stop, err := parseDateTime(stopStr, now)
	if err != nil {
		return watson.Frame{}, err
	}
	if !stop.After(start) {
		return watson.Frame{}, errors.New("stop must be after start")
	}
	return watson.Frame{Start: start.UTC(), Stop: stop.UTC(), Project: project, Tags: splitTags(tagsStr)}, nil
}

// overlaps reports whether f overlaps any other frame (excludeID = frame being edited).
func overlaps(f watson.Frame, frames []watson.Frame, excludeID string) bool {
	for _, other := range frames {
		if other.ID == excludeID {
			continue
		}
		if f.Start.Before(other.Stop) && other.Start.Before(f.Stop) {
			return true
		}
	}
	return false
}

// projectNames returns the sorted unique project names for autocompletion.
func projectNames(frames []watson.Frame) []string {
	seen := map[string]bool{}
	var names []string
	for _, f := range frames {
		if !seen[f.Project] {
			seen[f.Project] = true
			names = append(names, f.Project)
		}
	}
	sort.Strings(names)
	return names
}

const (
	fieldProject = iota
	fieldStart
	fieldStop
	fieldTags
	fieldCount
)

type formModel struct {
	editing bool
	frameID string
	inputs  [fieldCount]textinput.Model
	focus   int
	errMsg  string
	warned  bool // overlap warning shown; next submit saves anyway
}

// newFormModel builds the form; existing == nil means "new frame".
func newFormModel(existing *watson.Frame, frames []watson.Frame, now time.Time) formModel {
	var m formModel
	placeholders := [fieldCount]string{"Project", "YYYY-MM-DD HH:MM", "YYYY-MM-DD HH:MM", "tag1, tag2"}
	for i := range m.inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholders[i]
		ti.Width = 40
		m.inputs[i] = ti
	}
	m.inputs[fieldProject].ShowSuggestions = true
	m.inputs[fieldProject].SetSuggestions(projectNames(frames))
	m.inputs[fieldProject].KeyMap.AcceptSuggestion = keybind.NewBinding(keybind.WithKeys("right", "ctrl+e"))
	if existing != nil {
		m.editing = true
		m.frameID = existing.ID
		m.inputs[fieldProject].SetValue(existing.Project)
		start, stop := existing.Start.Local(), existing.Stop.Local()
		layout := editLayout(start, stop)
		m.inputs[fieldStart].SetValue(start.Format(layout))
		m.inputs[fieldStop].SetValue(stop.Format(layout))
		m.inputs[fieldTags].SetValue(strings.Join(existing.Tags, ", "))
	} else {
		m.inputs[fieldStart].SetValue(now.Add(-time.Hour).Format(dtLayout))
		m.inputs[fieldStop].SetValue(now.Format(dtLayout))
	}
	m.inputs[fieldProject].Focus()
	return m
}

func (m *formModel) setFocus(i int) {
	m.focus = (i + fieldCount) % fieldCount
	for j := range m.inputs {
		if j == m.focus {
			m.inputs[j].Focus()
		} else {
			m.inputs[j].Blur()
		}
	}
}

func (m formModel) view() string {
	labels := [fieldCount]string{"Project", "Start  ", "Stop   ", "Tags   "}
	var b strings.Builder
	for i := range m.inputs {
		cursor := "  "
		if i == m.focus {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, labels[i], m.inputs[i].View())
	}
	if m.errMsg != "" {
		b.WriteString("\n" + styleError.Render(m.errMsg))
	}
	return b.String()
}
