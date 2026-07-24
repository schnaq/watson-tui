package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

// key builds a KeyMsg from a readable name ("q", "enter", "ctrl+c", ...).
func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	app := NewApp(watson.NewStore(t.TempDir()), "test")
	app.Init()
	app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return app
}

func TestQKeyQuits(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(key("q"))
	if cmd == nil {
		t.Fatal("q must return tea.Quit")
	}
}

func TestHelpToggle(t *testing.T) {
	app := newTestApp(t)
	app.Update(key("?"))
	if app.mode != modeHelp {
		t.Fatal("? must open help")
	}
	if !strings.Contains(app.View(), "Tasten") {
		t.Error("help view missing title")
	}
	app.Update(key("x"))
	if app.mode != modeList {
		t.Error("any key must close help")
	}
}

func TestFatalOnCorruptFrames(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frames"), []byte("{ kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(watson.NewStore(dir), "test")
	app.Init()
	if app.mode != modeFatal {
		t.Fatal("corrupt frames must switch to fatal mode")
	}
	if !strings.Contains(app.View(), "frames.bak") {
		t.Error("fatal view must mention the backup")
	}
}

func TestStatusShowsRunningTimer(t *testing.T) {
	app := newTestApp(t)
	app.state = &watson.State{Project: "proj", Start: time.Now().Add(-90 * time.Second), Tags: []string{}}
	app.now = time.Now()
	if !strings.Contains(app.View(), "▶ proj") {
		t.Error("running timer missing in status bar")
	}
}
