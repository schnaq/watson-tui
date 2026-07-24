package watson

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// safeSave writes data atomically, keeping one .bak generation of the
// previous content. Mirrors Watson's safe_save (watson/utils.py).
func safeSave(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if _, err := os.Stat(path); err == nil {
		_ = os.Remove(path + ".bak")
		if err := os.Rename(path, path+".bak"); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
	}
	return os.Rename(tmpName, path)
}

// withLock runs fn while holding an advisory lock on <dir>/.watson-tui.lock.
// Watson itself has no locking; this protects concurrent watson-tui writers.
// If the filesystem does not support flock, fn runs unlocked (best effort).
func withLock(dir string, fn func() error) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fl := flock.New(filepath.Join(dir, ".watson-tui.lock"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fn()
	}
	if !ok {
		return errors.New("watson-Verzeichnis ist von anderem Prozess gesperrt")
	}
	defer func() { _ = fl.Unlock() }()
	return fn()
}
