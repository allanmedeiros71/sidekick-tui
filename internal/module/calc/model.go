package calc

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

const (
	maxHistory   = 100
	shownHistory = 5
)

type entry struct {
	input  string
	result Result
}

// Model is the calculator's Bubble Tea model. It evaluates the input as you
// type; enter keeps the result in the history and clears the prompt, and
// up/down recall earlier inputs.
type Model struct {
	theme   theme.Theme
	input   textinput.Model
	result  Result
	err     error
	history []entry
	recall  int // index into history while browsing it with up/down
	now     func() time.Time
}

// New returns a calculator drawn with the given theme.
func New(t theme.Theme) *Model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "0xff + 1, 1.5GiB in MB, now + 3d, chmod 755, 10.0.0.0/24"
	in.PromptStyle = t.Selected
	in.TextStyle = t.Item
	in.PlaceholderStyle = t.Muted
	in.CharLimit = 256
	in.Width = 56
	in.Focus()
	return &Model{theme: t, input: in, now: time.Now}
}

// ID implements module.Module.
func (m *Model) ID() string { return "calc" }

// Title implements module.Module.
func (m *Model) Title() string { return "Calculator" }

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.Type {
		case tea.KeyEnter:
			m.commit()
			return m, nil
		case tea.KeyUp:
			m.browse(-1)
			return m, nil
		case tea.KeyDown:
			m.browse(1)
			return m, nil
		}
	}
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.recall = len(m.history)
		m.evaluate()
	}
	return m, cmd
}

func (m *Model) evaluate() {
	m.result, m.err = Eval(m.input.Value(), m.now())
}

func (m *Model) commit() {
	in := strings.TrimSpace(m.input.Value())
	if in == "" || m.err != nil {
		return
	}
	m.history = append(m.history, entry{input: in, result: m.result})
	if len(m.history) > maxHistory {
		m.history = m.history[len(m.history)-maxHistory:]
	}
	m.recall = len(m.history)
	m.input.SetValue("")
	m.result, m.err = Result{}, nil
}

// browse moves through earlier inputs; moving past the newest one clears
// the prompt.
func (m *Model) browse(delta int) {
	if len(m.history) == 0 {
		return
	}
	m.recall = max(0, min(len(m.history), m.recall+delta))
	if m.recall == len(m.history) {
		m.input.SetValue("")
	} else {
		m.input.SetValue(m.history[m.recall].input)
	}
	m.input.CursorEnd()
	m.evaluate()
}

// View implements tea.Model.
func (m *Model) View() string {
	t := m.theme
	var b strings.Builder
	b.WriteString(t.Title.Render("Calculator") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")

	result := m.result
	if m.input.Value() == "" && len(m.history) > 0 {
		result = m.history[len(m.history)-1].result
	}
	switch {
	case m.err != nil:
		b.WriteString(t.Error.Render(m.err.Error()) + "\n")
	case len(result.Lines) == 0:
		b.WriteString(t.Muted.Render("type an expression, a size, a date, a mode or a network") + "\n")
	default:
		width := 0
		for _, l := range result.Lines {
			width = max(width, len(l.Label))
		}
		for _, l := range result.Lines {
			b.WriteString(t.Muted.Render(fmt.Sprintf("%-*s", width, l.Label)) + "  " + t.Item.Render(l.Value) + "\n")
		}
	}

	if n := len(m.history); n > 0 {
		b.WriteString("\n" + t.Muted.Render("history") + "\n")
		for _, e := range m.history[max(0, n-shownHistory):] {
			b.WriteString(t.Item.Render(e.input) + t.Muted.Render(" = "+e.result.Summary()) + "\n")
		}
	}
	b.WriteString("\n" + t.Muted.Render("enter keep · ↑/↓ history · esc back"))
	return b.String()
}
