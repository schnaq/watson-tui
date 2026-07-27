package tui

import (
	"strings"

	keybind "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/schnaq/watson-tui/internal/watson"
)

// startModel is the prompt for starting a new timer.
type startModel struct {
	project textinput.Model
	tags    textinput.Model
	focus   int // 0 = project, 1 = tags
	errMsg  string
}

func newStartModel(frames []watson.Frame) startModel {
	p := textinput.New()
	p.Prompt = ""
	p.Placeholder = "Project"
	p.Width = 40
	p.ShowSuggestions = true
	p.SetSuggestions(projectNames(frames))
	p.KeyMap.AcceptSuggestion = keybind.NewBinding(keybind.WithKeys("right", "ctrl+e"))
	p.Focus()
	tg := textinput.New()
	tg.Prompt = ""
	tg.Placeholder = "tag1, tag2 (optional)"
	tg.Width = 40
	return startModel{project: p, tags: tg}
}

func (m startModel) view() string {
	var b strings.Builder
	cursors := [2]string{"  ", "  "}
	cursors[m.focus] = "> "
	b.WriteString(cursors[0] + "Project " + m.project.View() + "\n")
	b.WriteString(cursors[1] + "Tags    " + m.tags.View() + "\n")
	if m.errMsg != "" {
		b.WriteString("\n" + styleError.Render(m.errMsg))
	}
	return b.String()
}
