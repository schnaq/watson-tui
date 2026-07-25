package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schnaq/watson-tui/internal/watson"
)

// TestFatalOnCorruptState covers the second error branch of reload():
// an unparseable state file must switch to fatal mode and reference the
// state.bak backup (symmetric to TestFatalOnCorruptFrames).
func TestFatalOnCorruptState(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(watson.NewStore(dir), "test")
	app.Init()
	if app.mode != modeFatal {
		t.Fatal("corrupt state must switch to fatal mode")
	}
	if !strings.Contains(app.View(), "state.bak") {
		t.Error("fatal view must mention the state backup")
	}
}

// TestCtrlCQuits covers the global ctrl+c handling, which returns tea.Quit
// regardless of the current mode.
func TestCtrlCQuits(t *testing.T) {
	app := newTestApp(t)
	if _, cmd := app.Update(key("ctrl+c")); cmd == nil {
		t.Fatal("ctrl+c must return tea.Quit")
	}
}
