package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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

	list listModel
	form formModel

	pendingDelete watson.Frame
	start         startModel

	mode     mode
	now      time.Time
	errMsg   string // transient, shown in status bar, cleared on next key
	fatalMsg string

	width, height int
}

func NewApp(store *watson.Store, version string) *App {
	return &App{
		store: store, version: version, mode: modeList, now: time.Now(),
		width: 80, height: 24, list: newListModel(time.Now()),
	}
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
	a.list.refresh(a.frames, a.cfg.WeekStart)
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
		case modeForm:
			return a.updateForm(msg)
		case modeStartTimer:
			return a.updateStartTimer(msg)
		case modeConfirmCancel:
			if s := msg.String(); s == "y" || s == "enter" {
				if err := a.store.Cancel(); err != nil {
					a.errMsg = "Verwerfen fehlgeschlagen: " + err.Error()
				}
				a.reload()
			}
			a.mode = modeList
			return a, nil
		case modeConfirmDelete:
			if s := msg.String(); s == "y" || s == "enter" {
				if err := a.store.Delete(a.pendingDelete.ID); err != nil {
					a.errMsg = "Löschen fehlgeschlagen: " + err.Error()
				}
				a.reload()
			}
			a.mode = modeList
			return a, nil
		}
	}
	return a, nil
}

// updateList handles keys in the frame list view. Later tasks extend this.
func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.list.filtering {
		switch msg.String() {
		case "enter":
			a.list.filtering = false
			a.list.filter = a.list.filterInput.Value()
		case "esc":
			a.list.filtering = false
			a.list.filterInput.SetValue("")
			a.list.filter = ""
		default:
			var cmd tea.Cmd
			a.list.filterInput, cmd = a.list.filterInput.Update(msg)
			a.list.filter = a.list.filterInput.Value()
			a.list.refresh(a.frames, a.cfg.WeekStart)
			return a, cmd
		}
		a.list.refresh(a.frames, a.cfg.WeekStart)
		return a, nil
	}
	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "?":
		a.mode = modeHelp
	case "enter":
		if f, ok := a.list.selected(); ok {
			a.form = newFormModel(&f, a.frames, time.Now())
			a.mode = modeForm
			return a, textinput.Blink
		}
	case "n":
		a.form = newFormModel(nil, a.frames, time.Now())
		a.mode = modeForm
		return a, textinput.Blink
	case "j", "down":
		a.list.move(+1)
	case "k", "up":
		a.list.move(-1)
	case "g":
		a.list.cursor = firstFrameRow(a.list.rows)
	case "G":
		if i := nextFrameRow(a.list.rows, len(a.list.rows), -1); i < len(a.list.rows) {
			a.list.cursor = i
		}
	case "/":
		a.list.filtering = true
		a.list.filterInput.Focus()
		return a, textinput.Blink
	case "[":
		a.list.per = a.list.per.shift(a.cfg.WeekStart, -1)
		a.list.refresh(a.frames, a.cfg.WeekStart)
	case "]":
		a.list.per = a.list.per.shift(a.cfg.WeekStart, +1)
		a.list.refresh(a.frames, a.cfg.WeekStart)
	case "t":
		a.setPeriodUnit(unitDay)
	case "w":
		a.setPeriodUnit(unitWeek)
	case "m":
		a.setPeriodUnit(unitMonth)
	case "a":
		a.setPeriodUnit(unitAll)
	case "d":
		if f, ok := a.list.selected(); ok {
			a.pendingDelete = f
			a.mode = modeConfirmDelete
		}
	case "s":
		if a.state != nil {
			if _, err := a.store.Stop(time.Now()); err != nil {
				a.errMsg = "Stop fehlgeschlagen: " + err.Error()
			}
			a.reload()
		} else {
			a.start = newStartModel(a.frames)
			a.mode = modeStartTimer
			return a, textinput.Blink
		}
	case "S":
		if a.state != nil {
			a.mode = modeConfirmCancel
		}
	case "R":
		a.reload()
	}
	return a, nil
}

func (a *App) setPeriodUnit(u periodUnit) {
	a.list.per = period{unit: u, ref: time.Now()}
	a.list.refresh(a.frames, a.cfg.WeekStart)
}

func (a *App) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeList
		return a, nil
	case "tab", "down":
		a.form.setFocus(a.form.focus + 1)
		return a, nil
	case "shift+tab", "up":
		a.form.setFocus(a.form.focus - 1)
		return a, nil
	case "enter", "ctrl+s":
		return a.submitForm()
	}
	var cmd tea.Cmd
	a.form.inputs[a.form.focus], cmd = a.form.inputs[a.form.focus].Update(msg)
	return a, cmd
}

func (a *App) submitForm() (tea.Model, tea.Cmd) {
	now := time.Now()
	frame, err := buildFrame(
		a.form.inputs[fieldProject].Value(),
		a.form.inputs[fieldStart].Value(),
		a.form.inputs[fieldStop].Value(),
		a.form.inputs[fieldTags].Value(),
		now,
	)
	if err != nil {
		a.form.errMsg = err.Error()
		return a, nil
	}
	if !a.form.warned && overlaps(frame, a.frames, a.form.frameID) {
		a.form.warned = true
		a.form.errMsg = "Überlappt mit anderem Frame — enter speichert trotzdem"
		return a, nil
	}
	if a.form.editing {
		frame.ID = a.form.frameID
		err = a.store.Update(frame, now)
	} else {
		_, err = a.store.Add(frame, now)
	}
	if err != nil {
		a.form.errMsg = "Speichern fehlgeschlagen: " + err.Error()
		return a, nil
	}
	a.reload()
	a.mode = modeList
	return a, nil
}

func (a *App) updateStartTimer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.mode = modeList
		return a, nil
	case "tab", "shift+tab", "down", "up":
		a.start.focus = 1 - a.start.focus
		if a.start.focus == 0 {
			a.start.project.Focus()
			a.start.tags.Blur()
		} else {
			a.start.tags.Focus()
			a.start.project.Blur()
		}
		return a, nil
	case "enter":
		project := strings.TrimSpace(a.start.project.Value())
		if project == "" {
			a.start.errMsg = "Projekt fehlt"
			return a, nil
		}
		if err := a.store.Start(project, splitTags(a.start.tags.Value()), time.Now()); err != nil {
			a.start.errMsg = err.Error()
			return a, nil
		}
		a.reload()
		a.mode = modeList
		return a, nil
	}
	var cmd tea.Cmd
	if a.start.focus == 0 {
		a.start.project, cmd = a.start.project.Update(msg)
	} else {
		a.start.tags, cmd = a.start.tags.Update(msg)
	}
	return a, cmd
}

// statusLeft is the view-specific left segment of the status bar.
func (a *App) statusLeft() string {
	n := 0
	for _, r := range a.list.rows {
		if !r.isHeader {
			n++
		}
	}
	left := fmt.Sprintf("%s · %d Frames", a.list.per.label(a.cfg.WeekStart), n)
	if a.list.filtering {
		return left + " · " + a.list.filterInput.View()
	}
	if a.list.filter != "" {
		left += " · Filter: " + a.list.filter
	}
	return left
}

func (a *App) View() string {
	var body string
	switch a.mode {
	case modeFatal:
		body = styleError.Render("Fehler") + "\n\n" + a.fatalMsg + "\n\nBeliebige Taste beendet."
	case modeHelp:
		body = helpView()
	case modeForm:
		body = a.form.view()
	case modeStartTimer:
		body = a.start.view()
	case modeConfirmCancel:
		body = styleTitle.Render("Laufenden Timer verwerfen?") + "\n\n" +
			styleDim.Render("y/enter: verwerfen · andere Taste: abbrechen")
	case modeConfirmDelete:
		f := a.pendingDelete
		body = styleTitle.Render("Frame löschen?") + fmt.Sprintf("\n\n  %s  %s–%s  %s\n\n%s",
			f.Project,
			f.Start.Local().Format("2006-01-02 15:04"),
			f.Stop.Local().Format("15:04"),
			f.ShortID(),
			styleDim.Render("y/enter: löschen · andere Taste: abbrechen"))
	default:
		body = a.list.view(a.height - 1)
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
