// Package tui implements the Bubble Tea user interface: frame list, form,
// report, billing overview and the live timer.
package tui

import (
	"fmt"
	"slices"
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

// NewApp reads the config here rather than leaving it to Init, because the
// list it builds needs the week start: a period normalised against the zero
// value (Sunday) and then read back with the configured Monday resolves to the
// week before. Init reads it again — the file may have changed in between, and
// it costs one read.
func NewApp(store *watson.Store, version string) *App {
	cfg := store.Config()
	return &App{
		store: store, version: version, cfg: cfg, mode: modeList, now: time.Now(),
		width: 80, height: 24, list: newListModel(time.Now(), cfg.WeekStart),
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
			// shift(ws, 0) normalises ref to the period's start; see the
			// invariant on period. Without it the report's ‹month of this
			// week› would depend on the minute the key was pressed.
			case "t":
				a.report.per = period{unit: unitDay, ref: time.Now()}.shift(a.cfg.WeekStart, 0)
			case "w":
				a.report.per = period{unit: unitWeek, ref: time.Now()}.shift(a.cfg.WeekStart, 0)
			case "m":
				a.report.per = period{unit: unitMonth, ref: time.Now()}.shift(a.cfg.WeekStart, 0)
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
		a.report = newReportModel(time.Now(), a.cfg.WeekStart)
		a.mode = modeReport
	case "o":
		a.mode = modeOverview
	}
	return a, nil
}

func (a *App) setPeriodUnit(u periodUnit) {
	a.list.per = period{unit: u, ref: time.Now()}.shift(a.cfg.WeekStart, 0)
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

// periodField renders the period as "‹ label ›" so the brackets show that [ and
// ] move it. The keys were in nobody's way — they were simply invisible, because
// a bare date reads as a fact about the view and not as something one can walk
// along. unitAll cannot be shifted, so it gets no brackets: an affordance that
// does nothing when pressed is worse than none.
func (a *App) periodField(label string, p period) headerField {
	text := p.label(a.cfg.WeekStart)
	if p.unit == unitAll {
		return headerField{label: label, value: text}
	}
	return headerField{
		label: label,
		value: styleDim.Render("‹ ") + text + styleDim.Render(" ›"),
	}
}

// The header's "Summe" is a number the body underneath has to add up to, and
// whether a running timer belongs inside it is a property of the view, not of the
// frames a caller happens to have at hand. So there are two functions, one per
// case, and the ` + läuft` flag follows from which one is called:
//
//   - sumFieldWithoutRunning — the timer is outside the number, so it is flagged.
//   - sumFieldWithRunning — the timer is inside the number, so there is no flag.
//
// The third combination is the one this split exists to make unwritable. A single
// sumFieldOf() decided the flag on its own, from "does a timer run in p", so
// handing it withRunning() frames produced a header that flagged the running hour
// as uncounted while counting it — over a report body that said the opposite in
// words. A flag meaning "not counted" on a number that counts it is worse than no
// flag at all.

// sumFieldWithoutRunning sums the given frames over p and flags a timer running
// in p as not counted. The list uses it: its sum sits one row above the day
// totals, which are frames only, so a header of "9h 45m" over days adding up to
// "2h 30m" would read as a bug in the day totals. The caller narrows the frames
// to the ones the list shows — see headerFields.
func (a *App) sumFieldWithoutRunning(frames []watson.Frame, p period) headerField {
	value := formatDuration(sumInPeriod(frames, p, a.cfg.WeekStart))
	if runningInPeriod(a.state, p, a.cfg.WeekStart) {
		value += styleRunning.Render(" + läuft")
	}
	return headerField{label: "Summe", value: value}
}

// sumFieldWithRunning sums the given frames over p with the running timer counted
// up to now, and carries no flag. The report uses it, and takes the number from
// aggregate() — the report body's own arithmetic — so the header cannot drift
// from the "Gesamt" it sits above: one function computes both. The body already
// says in words that the timer is counted.
func (a *App) sumFieldWithRunning(frames []watson.Frame, p period) headerField {
	_, grand := aggregate(withRunning(frames, a.state, a.now), p, a.cfg.WeekStart)
	return headerField{label: "Summe", value: formatDuration(grand)}
}

// comparisonRow renders the neighbouring periods of p — the one before it and
// the larger one containing it — so a number has something to be read against.
// Zero reads as a dash: "0m" invites the question whether it means nothing was
// booked or nothing is known.
func (a *App) comparisonRow(p period) []headerField {
	cs := comparisonPeriods(p)
	if len(cs) == 0 {
		return nil
	}
	fields := make([]headerField, 0, len(cs))
	for _, c := range cs {
		d := sumInPeriod(a.frames, c.per, a.cfg.WeekStart)
		value := "–"
		if d > 0 {
			value = formatDuration(d)
		}
		fields = append(fields, headerField{label: c.label, value: value})
	}
	return fields
}

// headerFields returns the context rows of the header for the active mode. Up
// to four rows, which is what chromeHeight budgets the framed header from
// height 24 up; the comparison row is row two so that headerView can take it
// back out first when the terminal pays for only three. The last row is the
// timer in every mode — see fitHeaderRows.
//
// Rows that a mode has nothing to put in come back empty rather than blank:
// headerView drops them before the budget applies, so an invisible row never
// costs a visible one.
func (a *App) headerFields() [][]headerField {
	timer := a.timerField()
	switch a.mode {
	case modeList:
		n := 0
		projects := map[string]bool{}
		for _, r := range a.list.rows {
			if !r.isHeader {
				n++
				projects[r.frame.Project] = true
			}
		}
		filter := "—"
		if a.list.filtering {
			filter = a.list.filterInput.View()
		} else if a.list.filter != "" {
			filter = a.list.filter
		}
		return [][]headerField{
			{a.periodField("Zeitraum", a.list.per),
				a.sumFieldWithoutRunning(filterFrames(a.frames, a.list.filter), a.list.per)},
			a.comparisonRow(a.list.per),
			{{"Filter", filter}, {"", fmt.Sprintf("%d Frames · %d Projekte", n, len(projects))}},
			{{}, timer},
		}
	case modeReport:
		lines, _ := aggregate(withRunning(a.frames, a.state, a.now), a.report.per, a.cfg.WeekStart)
		return [][]headerField{
			{a.periodField("Report", a.report.per), a.sumFieldWithRunning(a.frames, a.report.per)},
			a.comparisonRow(a.report.per),
			{{}, {"", fmt.Sprintf("%d Projekte", len(lines))}},
			{{}, timer},
		}
	case modeOverview:
		// Fixed columns, so there is no period to shift and no neighbour to
		// compare against — the table already is the comparison. The sum is the
		// last column of the table, the one billing reads.
		cols := overviewColumns(a.now, a.cfg.WeekStart)
		rows, totals := buildOverview(withRunning(a.frames, a.state, a.now), cols, a.cfg.WeekStart)
		grand := time.Duration(0)
		if len(totals) > 0 {
			grand = totals[len(totals)-1]
		}
		return [][]headerField{
			{{"Übersicht", "Abrechnung"}, {"Summe gesamt", formatDuration(grand)}},
			{{}, {"", fmt.Sprintf("%d Projekte", len(rows))}},
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
		// The list scrolls itself to a height and lays its rows out against the
		// body width; the others are cut by fitBody.
		body = a.list.view(content, bodyWidth)
	}
	body = fitBody(body, content, bodyWidth)

	var parts []string
	if header := a.headerView(); header != "" {
		parts = append(parts, header)
	}
	if framed {
		parts = append(parts, panel(panelTitle(a.mode), body, a.width, panelBorder(a.mode)))
	} else {
		parts = append(parts, body)
	}
	parts = append(parts, a.footerView())
	return strings.Join(parts, "\n")
}

// headerView draws the context panel for the active mode. Together with
// footerView it occupies exactly chromeHeight(a.height) lines, which is what
// View's body arithmetic above is built on.
//
// Two things happen to the rows before renderHeader sees them, and only one of
// them is renderHeader's business. Empty rows go first: a mode that has no
// comparison to show (the period is "alle Frames", or the mode has no period at
// all) hands back an empty row, and letting it through would spend a line of the
// budget on nothing. Then, when the terminal pays for three rows instead of four,
// the comparison row gives way — it is the one field a user can work without,
// and it is the row this function knows the position of. What renderHeader does
// with the rest is its own affair: fitHeaderRows makes the count exact and keeps
// the timer last, so nothing here has to cut a tail and risk taking the timer
// with it.
func (a *App) headerView() string {
	rows := a.headerFields()
	compact := make([][]headerField, 0, len(rows))
	for _, r := range rows {
		if len(r) == 0 {
			continue
		}
		compact = append(compact, r)
	}
	// Only from three rows up: below that renderHeader collapses to a single
	// line built from the first value and the last one, and dropping a row there
	// changes nothing — while a budget of one is not a reason to throw away
	// three rows the collapsed line is going to read across anyway.
	if budget := headerRowBudget(a.height); budget >= 3 && len(compact) > budget {
		compact = slices.Delete(compact, 1, 2)
	}
	return renderHeader(a.width, a.height, a.version, compact)
}

// footerView draws the key hints, one line per group — but only as many groups
// as the terminal has lines for. When it has one, the groups are poured into it
// rather than the second one being dropped: the second group is where ? and q
// live, and losing them is losing the last place the keys are named.
//
// The lines are padded to the footer's share of the chrome, above the hints and
// not below them: a mode with a single group, or an error, would otherwise
// leave the frame one line short of the bottom of the terminal and the hints
// floating a row above it — the unfinished look fitBody was taught to avoid.
// Padding at all is what keeps the body panel from jumping a row when the mode
// changes.
func (a *App) footerView() string {
	groups := footerHints(a.mode)
	if n := footerLines(a.height); len(groups) > n {
		if n == 1 {
			groups = [][]string{mergeHints(groups)}
		} else {
			groups = groups[:n]
		}
	}
	footer := renderFooter(a.width, groups, a.errMsg)
	if n, have := footerLines(a.height), strings.Count(footer, "\n")+1; have < n {
		footer = strings.Repeat("\n", n-have) + footer
	}
	return footer
}

func helpView() string {
	// Twelve lines, one per key group: the help screen does not scroll, and at 20
	// terminal lines the panel around it fits exactly twelve — chromeHeight(20)
	// takes six and the panel border two. There is no slack left, so one more
	// line loses the tail to fitBody, and the tail is where the key that quits
	// sits. That is also why s/S and r/o share a line instead of taking two each.
	// TestHelpViewFitsTwentyLines derives the budget rather than repeating it.
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
