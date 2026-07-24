package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/schnaq/watson-tui/internal/watson"
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeReport
	modeConfirmDelete
	modeStartTimer
	modeConfirmCancel
	modeHelp
	modeFatal
)

// App is the Bubble Tea root model.
type App struct {
	store   *watson.Store
	version string
	cfg     watson.Config

	frames []watson.Frame
	state  *watson.State

	mode     mode
	now      time.Time
	errMsg   string // transient, shown in status bar, cleared on next key
	fatalMsg string

	width, height int
}

func NewApp(store *watson.Store, version string) *App {
	return &App{store: store, version: version, mode: modeList, now: time.Now(), width: 80, height: 24}
}

func (a *App) Init() tea.Cmd {
	a.cfg = a.store.Config()
	a.reload()
	return tickCmd()
}

// reload re-reads frames and state. Read errors are fatal: we must not
// write on top of a file we cannot parse (protects the .bak generation).
func (a *App) reload() {
	frames, err := a.store.Frames()
	if err != nil {
		a.fatal(fmt.Sprintf("frames-Datei nicht lesbar: %v\nBackup: %s/frames.bak", err, a.store.Dir()))
		return
	}
	state, err := a.store.State()
	if err != nil {
		a.fatal(fmt.Sprintf("state-Datei nicht lesbar: %v\nBackup: %s/state.bak", err, a.store.Dir()))
		return
	}
	a.frames = frames
	a.state = state
}

func (a *App) fatal(msg string) {
	a.fatalMsg = msg
	a.mode = modeFatal
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, nil
	case tickMsg:
		a.now = time.Time(msg)
		return a, tickCmd()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		a.errMsg = ""
		switch a.mode {
		case modeFatal:
			return a, tea.Quit
		case modeHelp:
			a.mode = modeList
			return a, nil
		case modeList:
			return a.updateList(msg)
		}
	}
	return a, nil
}

// updateList handles keys in the frame list view. Later tasks extend this.
func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "?":
		a.mode = modeHelp
	}
	return a, nil
}

// statusLeft is the view-specific left segment of the status bar.
// Task 9 extends this with period, count and filter info.
func (a *App) statusLeft() string { return "" }

func (a *App) View() string {
	var body string
	switch a.mode {
	case modeFatal:
		body = styleError.Render("Fehler") + "\n\n" + a.fatalMsg + "\n\nBeliebige Taste beendet."
	case modeHelp:
		body = helpView()
	default:
		body = fmt.Sprintf("%d Frames geladen — ? für Hilfe", len(a.frames))
	}
	return body + "\n" + renderStatus(a.width, a.state, a.now, a.statusLeft(), a.errMsg)
}

func helpView() string {
	return styleTitle.Render("Tasten") + `

  j/k, ↓/↑      navigieren
  enter         Frame editieren
  n             neuer Frame
  d             Frame löschen
  s             Timer starten/stoppen
  S             Timer verwerfen (cancel)
  /             filtern
  [ / ]         Zeitraum zurück/vor
  t/w/m/a       Tag/Woche/Monat/alles
  r             Report
  R             neu laden
  ?             diese Hilfe
  q             beenden

Beliebige Taste schließt die Hilfe.`
}
