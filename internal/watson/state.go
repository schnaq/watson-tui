package watson

import (
	"bytes"
	"encoding/json"
	"time"
)

// State is the currently running timer (Watson's `state` file).
type State struct {
	Project string
	Start   time.Time
	Tags    []string
}

type stateJSON struct {
	Project string   `json:"project"`
	Start   int64    `json:"start"`
	Tags    []string `json:"tags"`
}

// ParseState returns nil when no timer is running ({} or empty file).
func ParseState(data []byte) (*State, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var raw stateJSON
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, err
	}
	// An empty or missing project means no timer is running: treat it as idle
	// (nil, nil) rather than an error, matching the `{}` / empty-file semantics.
	// Defensive against a malformed state file that carries a start but no project.
	if raw.Project == "" {
		return nil, nil
	}
	tags := raw.Tags
	if tags == nil {
		tags = []string{}
	}
	return &State{Project: raw.Project, Start: time.Unix(raw.Start, 0).UTC(), Tags: tags}, nil
}

// MarshalState encodes the state file (indent=1, no trailing newline, {} when idle).
func MarshalState(s *State) ([]byte, error) {
	if s == nil {
		return []byte("{}"), nil
	}
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(stateJSON{Project: s.Project, Start: s.Start.Unix(), Tags: tags}); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
