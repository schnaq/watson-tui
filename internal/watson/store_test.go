package watson

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func TestStoreEmptyDir(t *testing.T) {
	s := testStore(t)
	frames, err := s.Frames()
	if err != nil || len(frames) != 0 {
		t.Fatalf("frames = %v, %v", frames, err)
	}
	state, err := s.State()
	if err != nil || state != nil {
		t.Fatalf("state = %v, %v", state, err)
	}
	if s.Config().WeekStart != time.Monday {
		t.Error("config default missing")
	}
}

func TestStoreAddUpdateDelete(t *testing.T) {
	s := testStore(t)
	now := time.Unix(1658000000, 0).UTC()
	added, err := s.Add(Frame{
		Start: now.Add(-time.Hour), Stop: now, Project: "p1", Tags: []string{"t"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(added.ID) {
		t.Errorf("id = %q, want 32 hex chars", added.ID)
	}
	if !added.UpdatedAt.Equal(now) {
		t.Errorf("updated_at = %v", added.UpdatedAt)
	}

	added.Project = "p2"
	later := now.Add(time.Minute)
	if err := s.Update(added, later); err != nil {
		t.Fatal(err)
	}
	frames, _ := s.Frames()
	if len(frames) != 1 || frames[0].Project != "p2" || !frames[0].UpdatedAt.Equal(later) {
		t.Fatalf("after update: %+v", frames)
	}

	if err := s.Update(Frame{ID: "ffffffffffffffffffffffffffffffff"}, later); err != ErrNotFound {
		t.Errorf("update unknown id: %v", err)
	}
	if err := s.Delete(added.ID); err != nil {
		t.Fatal(err)
	}
	frames, _ = s.Frames()
	if len(frames) != 0 {
		t.Fatalf("after delete: %+v", frames)
	}
	if err := s.Delete(added.ID); err != ErrNotFound {
		t.Errorf("double delete: %v", err)
	}
}

func TestStoreStartStopCancel(t *testing.T) {
	s := testStore(t)
	t0 := time.Unix(1658000000, 0).UTC()
	t1 := t0.Add(30 * time.Minute)

	if _, err := s.Stop(t1); err != ErrNotRunning {
		t.Errorf("stop idle: %v", err)
	}
	if err := s.Cancel(); err != ErrNotRunning {
		t.Errorf("cancel idle: %v", err)
	}
	if err := s.Start("proj", []string{"a"}, t0); err != nil {
		t.Fatal(err)
	}
	if err := s.Start("other", nil, t0); err != ErrAlreadyRunning {
		t.Errorf("double start: %v", err)
	}
	state, _ := s.State()
	if state == nil || state.Project != "proj" || !state.Start.Equal(t0) {
		t.Fatalf("state = %+v", state)
	}

	frame, err := s.Stop(t1)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Project != "proj" || !frame.Start.Equal(t0) || !frame.Stop.Equal(t1) {
		t.Errorf("stopped frame = %+v", frame)
	}
	state, _ = s.State()
	if state != nil {
		t.Errorf("state after stop = %+v", state)
	}
	frames, _ := s.Frames()
	if len(frames) != 1 {
		t.Errorf("frames after stop = %+v", frames)
	}

	if err := s.Start("proj2", nil, t1); err != nil {
		t.Fatal(err)
	}
	if err := s.Cancel(); err != nil {
		t.Fatal(err)
	}
	frames, _ = s.Frames()
	if len(frames) != 1 {
		t.Errorf("cancel must not create a frame: %+v", frames)
	}
}

func TestStoreWritesBak(t *testing.T) {
	s := testStore(t)
	now := time.Unix(1658000000, 0).UTC()
	_, _ = s.Add(Frame{Start: now, Stop: now.Add(time.Hour), Project: "a"}, now)
	_, _ = s.Add(Frame{Start: now, Stop: now.Add(time.Hour), Project: "b"}, now)
	if _, err := os.Stat(filepath.Join(s.Dir(), "frames.bak")); err != nil {
		t.Errorf("frames.bak missing: %v", err)
	}
}

func TestStoreCorruptFrames(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir(), "frames"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Frames(); err == nil {
		t.Error("want parse error for corrupt frames file")
	}
}

// --- Extra error-branch tests (added per task instruction) ---

// TestStoreCorruptState mirrors TestStoreCorruptFrames for the state file:
// invalid JSON must surface as a parse error from State(), not as idle (nil).
func TestStoreCorruptState(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir(), "state"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.State(); err == nil {
		t.Error("want parse error for corrupt state file")
	}
}

// TestStoreAddCorruptFrames pins Add's load-error branch: a corrupt frames file
// must abort the mutation instead of silently overwriting it.
func TestStoreAddCorruptFrames(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir(), "frames"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1658000000, 0).UTC()
	if _, err := s.Add(Frame{Start: now, Stop: now, Project: "p"}, now); err == nil {
		t.Error("want error when adding onto a corrupt frames file")
	}
}

// TestStoreStopCorruptState pins Stop's state-load-error branch: a corrupt state
// file must yield the parse error, not the idle ErrNotRunning path.
func TestStoreStopCorruptState(t *testing.T) {
	s := testStore(t)
	if err := os.WriteFile(filepath.Join(s.Dir(), "state"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1658000000, 0).UTC()
	if _, err := s.Stop(now); err == nil || err == ErrNotRunning {
		t.Errorf("want parse error, got %v", err)
	}
}
