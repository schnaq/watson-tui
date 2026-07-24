package watson

import (
	"context"
	"errors"
	"io/fs"
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
		// The destination exists: rotate it to .bak before replacing.
		_ = os.Remove(path + ".bak")
		if err := os.Rename(path, path+".bak"); err != nil {
			_ = os.Remove(tmpName)
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		// A stat error other than "does not exist" is a real problem; do not
		// silently skip rotation. Clean up the temp file first.
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// lockTimeout bounds how long withLock waits to acquire the advisory lock
// before treating the situation as contention. It is a package var so tests
// can shorten it.
var lockTimeout = 3 * time.Second

// withLock runs fn while holding an advisory lock on <dir>/.watson-tui.lock.
// Watson itself has no locking; this protects concurrent watson-tui writers.
// If the filesystem does not support flock, fn runs unlocked (best effort).
func withLock(dir string, fn func() error) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fl := flock.New(filepath.Join(dir, ".watson-tui.lock"))
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	ok, err := fl.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		// Timeout or cancellation means the lock is genuinely held by another
		// process: refuse rather than run unlocked.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return errors.New("watson-Verzeichnis ist von anderem Prozess gesperrt")
		}
		// Any other error (e.g. a filesystem that does not support flock) is
		// treated as best-effort: run without the lock.
		return fn()
	}
	if !ok {
		// Defensive fallback: flock should return DeadlineExceeded on timeout,
		// but guard against (false, nil) just in case.
		return errors.New("watson-Verzeichnis ist von anderem Prozess gesperrt")
	}
	defer func() { _ = fl.Unlock() }()
	return fn()
}
