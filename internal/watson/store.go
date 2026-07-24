// Package watson reads and writes Watson's data directory (frames, state,
// config) byte-compatibly, so a Watson installation stays interchangeable.
package watson

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAlreadyRunning = errors.New("es läuft bereits ein Timer")
	ErrNotRunning     = errors.New("kein Timer aktiv")
)

// Store reads and writes a Watson data directory.
type Store struct {
	dir string
}

func NewStore(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Dir() string        { return s.dir }
func (s *Store) framesPath() string { return filepath.Join(s.dir, "frames") }
func (s *Store) statePath() string  { return filepath.Join(s.dir, "state") }
func (s *Store) configPath() string { return filepath.Join(s.dir, "config") }

func newID() string {
	u := uuid.New()
	return hex.EncodeToString(u[:])
}

// Frames reads the frames file; a missing file means no frames yet.
func (s *Store) Frames() ([]Frame, error) {
	data, err := os.ReadFile(s.framesPath())
	if errors.Is(err, os.ErrNotExist) {
		return []Frame{}, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseFrames(data)
}

// State reads the state file; a missing file means no running timer.
func (s *Store) State() (*State, error) {
	data, err := os.ReadFile(s.statePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ParseState(data)
}

// Config reads the config file; on any error it returns defaults.
func (s *Store) Config() Config {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		return DefaultConfig()
	}
	cfg, err := ParseConfig(data)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

func (s *Store) writeFrames(frames []Frame) error {
	data, err := MarshalFrames(frames)
	if err != nil {
		return err
	}
	return safeSave(s.framesPath(), data)
}

func (s *Store) writeState(state *State) error {
	data, err := MarshalState(state)
	if err != nil {
		return err
	}
	return safeSave(s.statePath(), data)
}

// Add assigns a fresh ID and UpdatedAt, appends and persists the frame.
func (s *Store) Add(f Frame, now time.Time) (Frame, error) {
	f.ID = newID()
	f.UpdatedAt = now.UTC()
	err := withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		frames = append(frames, f)
		return s.writeFrames(frames)
	})
	return f, err
}

// Update replaces the frame with f.ID and stamps UpdatedAt.
func (s *Store) Update(f Frame, now time.Time) error {
	f.UpdatedAt = now.UTC()
	return withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		for i := range frames {
			if frames[i].ID == f.ID {
				frames[i] = f
				return s.writeFrames(frames)
			}
		}
		return ErrNotFound
	})
}

// Delete removes the frame with the given full ID.
func (s *Store) Delete(id string) error {
	return withLock(s.dir, func() error {
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		for i := range frames {
			if frames[i].ID == id {
				return s.writeFrames(append(frames[:i], frames[i+1:]...))
			}
		}
		return ErrNotFound
	})
}

// Start begins a new timer; fails if one is already running.
func (s *Store) Start(project string, tags []string, now time.Time) error {
	if tags == nil {
		tags = []string{}
	}
	return withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state != nil {
			return ErrAlreadyRunning
		}
		return s.writeState(&State{Project: project, Start: now.UTC(), Tags: tags})
	})
}

// Stop ends the running timer, persists it as a frame and clears the state.
func (s *Store) Stop(now time.Time) (Frame, error) {
	var frame Frame
	err := withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state == nil {
			return ErrNotRunning
		}
		frames, err := s.Frames()
		if err != nil {
			return err
		}
		frame = Frame{
			Start: state.Start, Stop: now.UTC(), Project: state.Project,
			ID: newID(), Tags: state.Tags, UpdatedAt: now.UTC(),
		}
		if err := s.writeFrames(append(frames, frame)); err != nil {
			return err
		}
		return s.writeState(nil)
	})
	return frame, err
}

// Cancel discards the running timer without recording a frame.
func (s *Store) Cancel() error {
	return withLock(s.dir, func() error {
		state, err := s.State()
		if err != nil {
			return err
		}
		if state == nil {
			return ErrNotRunning
		}
		return s.writeState(nil)
	})
}
