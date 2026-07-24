package watson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestSafeSaveCreatesFileAndDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	path := filepath.Join(dir, "frames")
	if err := safeSave(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "v1" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSafeSaveRotatesBak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frames")
	if err := safeSave(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if err := safeSave(path, []byte("v2")); err != nil {
		t.Fatal(err)
	}
	if err := safeSave(path, []byte("v3")); err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(path)
	bak, _ := os.ReadFile(path + ".bak")
	if string(cur) != "v3" || string(bak) != "v2" {
		t.Errorf("cur=%q bak=%q, want v3/v2", cur, bak)
	}
}

func TestSafeSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frames")
	_ = safeSave(path, []byte("v1"))
	_ = safeSave(path, []byte("v2"))
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
}

func TestWithLockRuns(t *testing.T) {
	ran := false
	if err := withLock(t.TempDir(), func() error { ran = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("fn did not run")
	}
}

func TestWithLockContention(t *testing.T) {
	dir := t.TempDir()

	// Hold the lock from an external flock handle to simulate another process.
	external := flock.New(filepath.Join(dir, ".watson-tui.lock"))
	if err := external.Lock(); err != nil {
		t.Fatalf("external lock: %v", err)
	}
	defer func() { _ = external.Unlock() }()

	// Shorten the acquisition timeout so the test stays fast and deterministic.
	prev := lockTimeout
	lockTimeout = 200 * time.Millisecond
	defer func() { lockTimeout = prev }()

	ran := false
	err := withLock(dir, func() error { ran = true; return nil })
	if err == nil {
		t.Fatal("expected contention error, got nil")
	}
	if ran {
		t.Error("fn ran despite lock contention")
	}
}
