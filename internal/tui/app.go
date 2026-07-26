// Package tui implements the Bubble Tea user interface: frame list, form,
// report, billing overview and the live timer.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	modeOverview
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
	report        reportModel

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
		case modeReport:
			switch msg.String() {
			case "esc", "q", "r":
				a.mode = modeList
			case "t":
				a.report.per = period{unit: unitDay, ref: time.Now()}
			case "w":
				a.report.per = period{unit: unitWeek, ref: time.Now()}
			case "m":
				a.report.per = period{unit: unitMonth, ref: time.Now()}
			case "[":
				a.report.per = a.report.per.shift(a.cfg.WeekStart, -1)
			case "]":
				a.report.per = a.report.per.shift(a.cfg.WeekStart, +1)
			}
			return a, nil
		case modeOverview:
			switch msg.String() {
			case "esc", "q", "o":
				a.mode = modeList
			}
			return a, nil
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
	case "r":
		a.report = newReportModel(time.Now())
		a.mode = modeReport
	case "o":
		a.mode = modeOverview
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
	before := a.form.inputs[a.form.focus].Value()
	var cmd tea.Cmd
	a.form.inputs[a.form.focus], cmd = a.form.inputs[a.form.focus].Update(msg)
	if a.form.inputs[a.form.focus].Value() != before {
		// A changed field invalidates the overlap warning: otherwise the next
		// enter would save times that were never checked for overlap.
		a.form.warned = false
		a.form.errMsg = ""
	}
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
			a.start.errMsg = "Start fehlgeschlagen: " + err.Error()
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

// timerField renders the running timer for the header; every mode shows it, so
// stopping a timer is never more than one glance away.
func (a *App) timerField() headerField {
	if a.state == nil {
		return headerField{value: styleDim.Render("kein Timer")}
	}
	label := a.state.Project
	if len(a.state.Tags) > 0 {
		label += " [" + strings.Join(a.state.Tags, ", ") + "]"
	}
	return headerField{value: styleRunning.Render(
		fmt.Sprintf("▶ %s %s", label, formatClock(a.now.Sub(a.state.Start))))}
}

// headerFields returns the context rows of the header for the active mode.
// Always two rows: chromeHeight budgets four lines for the framed header, which
// is two field rows plus the border, and a mode with fewer would leave the body
// a line it never uses.
func (a *App) headerFields() [][]headerField {
	timer := a.timerField()
	switch a.mode {
	case modeList:
		n := 0
		for _, r := range a.list.rows {
			if !r.isHeader {
				n++
			}
		}
		filter := "—"
		if a.list.filtering {
			filter = a.list.filterInput.View()
		} else if a.list.filter != "" {
			filter = a.list.filter
		}
		return [][]headerField{
			{{"Zeitraum", a.list.per.label(a.cfg.WeekStart)}, {"Frames", fmt.Sprintf("%d", n)}},
			{{"Filter", filter}, timer},
		}
	case modeReport:
		lines, _ := aggregate(withRunning(a.frames, a.state, a.now), a.report.per, a.cfg.WeekStart)
		return [][]headerField{
			{{"Report", a.report.per.label(a.cfg.WeekStart)}, {"Projekte", fmt.Sprintf("%d", len(lines))}},
			{{}, timer},
		}
	case modeOverview:
		rows, _ := buildOverview(withRunning(a.frames, a.state, a.now),
			overviewColumns(a.now, a.cfg.WeekStart), a.cfg.WeekStart)
		return [][]headerField{
			{{"Übersicht", "Abrechnung"}, {"Projekte", fmt.Sprintf("%d", len(rows))}},
			{{}, timer},
		}
	case modeForm:
		what := "neu"
		if a.form.editing {
			// ShortID instead of a slice: a foreign frames file may carry an ID
			// shorter than 7 chars, and a panic in View() strands the alt-screen.
			what = "bearbeiten (" + (watson.Frame{ID: a.form.frameID}).ShortID() + ")"
		}
		return [][]headerField{{{"Frame", what}}, {{}, timer}}
	case modeStartTimer:
		return [][]headerField{{{"Timer", "starten"}}, {{}, timer}}
	case modeConfirmDelete:
		return [][]headerField{{{"Frame", "löschen"}}, {{}, timer}}
	case modeConfirmCancel:
		return [][]headerField{{{"Timer", "verwerfen"}}, {{}, timer}}
	case modeHelp:
		return [][]headerField{{{"Hilfe", "Tastenbelegung"}}, {{}, timer}}
	default: // modeFatal
		// The timer row belongs here too: fatal is reached from reload(), which
		// returns before it overwrites a.state, so a timer that was running when
		// the file went bad is still shown — and it is the one thing the user may
		// want to act on before restarting.
		return [][]headerField{
			{{"Fehler", "watson-tui kann nicht weiterarbeiten"}}, {{}, timer},
		}
	}
}

// panelBorder colours the frame around the body. The fatal screen gets the
// error colour the spec asks for; a red frame is what makes it read as a stop
// sign rather than as one more view.
func panelBorder(m mode) lipgloss.Style {
	if m == modeFatal {
		return styleError
	}
	return styleBorder
}

// panelTitle names the body panel of the active mode.
func panelTitle(m mode) string {
	switch m {
	case modeForm:
		return "Frame"
	case modeReport:
		return "Report"
	case modeStartTimer:
		return "Timer"
	case modeConfirmDelete, modeConfirmCancel:
		return "Bestätigen"
	case modeHelp:
		return "Hilfe"
	case modeFatal:
		return "Fehler"
	default:
		return "Frames"
	}
}

func (a *App) View() string {
	// The chrome takes its lines off the top and bottom; the rest is the body's.
	bodyHeight := max(a.height-chromeHeight(a.height), 1)
	// Two exceptions to the frame: the billing table needs every column it can
	// get, so it renders without side borders, and a body of one line has no room
	// left for a border either — on a terminal that short the height promise
	// outranks the decoration.
	framed := a.mode != modeOverview && bodyHeight > 2
	content, bodyWidth := bodyHeight, a.width
	if framed {
		// panel keeps two border lines and, per line, two border columns plus a
		// space of gutter on either side.
		content, bodyWidth = bodyHeight-2, max(a.width-4, 1)
	}

	var body string
	switch a.mode {
	case modeFatal:
		// Wrapped, not clipped: the message names the backup file, and a cut
		// would drop exactly the path the user has to go and look at. In error
		// colour, inside a panel whose border panelBorder colours to match.
		body = styleError.Width(bodyWidth).Render(a.fatalMsg)
	case modeHelp:
		body = helpView()
	case modeForm:
		body = a.form.view()
	case modeReport:
		// bodyWidth, not a.width: the report is framed, so the columns it lays out
		// have to fit inside the panel — handing it the terminal width would put
		// four columns of border and gutter back under fitBody's knife.
		body = a.report.view(a.frames, a.state, a.cfg.WeekStart, a.now, bodyWidth)
	case modeOverview:
		body = overviewView(a.frames, a.state, a.cfg.WeekStart, a.now, a.width)
	case modeStartTimer:
		body = a.start.view()
	case modeConfirmCancel:
		body = "Laufenden Timer verwerfen?"
	case modeConfirmDelete:
		f := a.pendingDelete
		body = fmt.Sprintf("Frame löschen?\n\n  %s  %s–%s  %s",
			f.Project,
			f.Start.Local().Format("2006-01-02 15:04"),
			f.Stop.Local().Format("15:04"),
			f.ShortID())
	default:
		// The list scrolls itself to a height; the others are cut by fitBody.
		body = a.list.view(content)
	}
	body = fitBody(body, content, bodyWidth)

	var parts []string
	if header := renderHeader(a.width, a.height, a.version, a.headerFields()); header != "" {
		parts = append(parts, header)
	}
	if framed {
		parts = append(parts, panel(panelTitle(a.mode), body, a.width, panelBorder(a.mode)))
	} else {
		parts = append(parts, body)
	}
	parts = append(parts, renderFooter(a.width, footerHints(a.mode), a.errMsg))
	return strings.Join(parts, "\n")
}

func helpView() string {
	// Twelve lines, one per key group: the help screen does not scroll, and at 20
	// terminal lines the panel around it fits thirteen. A longer list would lose
	// its tail to fitBody — and the tail is where the key that quits sits. That
	// is also why s/S and r/o share a line instead of taking two each.
	return `  j/k, ↓/↑     navigieren
  enter        Frame editieren
  n            neuer Frame
  d            Frame löschen
  s / S        Timer starten/stoppen · verwerfen
  /            filtern
  [ / ]        Zeitraum zurück/vor
  t/w/m/a      Tag/Woche/Monat/alles
  r / o        Report · Übersicht (Abrechnung)
  R            neu laden
  ?            diese Hilfe
  q            beenden`
}
