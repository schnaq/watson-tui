package watson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
