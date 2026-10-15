package inspect

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

const (
	// frameWidth and frameHeight are what the popup frame (border and
	// padding) takes around the module.
	frameWidth  = 6
	frameHeight = 4
	// maxValueLines is how many lines of a multi-line value are shown.
	maxValueLines = 12
	maxLabelWidth = 12
)

// copiedMsg reports the outcome of copying a value to the clipboard.
type copiedMsg struct {
	label string
	err   error
}

// Model is the inspector's Bubble Tea model. It inspects the input as you
// type; up/down select a line and enter copies its value.
type Model struct {
	theme    theme.Theme
	input    textinput.Model
	result   Result
	lines    []Line // result.Lines(), cached for selection
	selected int
	notice   string
	width    int
	height   int
	now      func() time.Time
	copy     func(string) error
}

// New returns an inspector drawn with the given theme.
func New(t theme.Theme) *Model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "paste a JWT, JSON, UUID, URL, base64 or hex; or type uuid"
	in.PromptStyle = t.Selected
	in.TextStyle = t.Item
	in.PlaceholderStyle = t.Muted
	in.CharLimit = 1 << 16
	in.Width = 56
	in.Focus()
	return &Model{theme: t, input: in, now: time.Now, copy: copyToClipboard}
}

// ID implements module.Module.
func (m *Model) ID() string { return "inspect" }

// Title implements module.Module.
func (m *Model) Title() string { return "Inspector" }

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = max(20, msg.Width-frameWidth-len(m.input.Prompt)-1)
		return m, nil
	case copiedMsg:
		if msg.err != nil {
			m.notice = "copy failed: " + msg.err.Error()
		} else {
			m.notice = "copied " + msg.label
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp, tea.KeyShiftTab:
			m.move(-1)
			return m, nil
		case tea.KeyDown, tea.KeyTab:
			m.move(1)
			return m, nil
		case tea.KeyEnter:
			return m, m.copySelected()
		}
	}
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.evaluate()
	}
	return m, cmd
}

func (m *Model) evaluate() {
	m.result = Inspect(m.input.Value(), m.now())
	m.lines = m.result.Lines()
	m.selected = 0
	m.notice = ""
}

func (m *Model) move(delta int) {
	if len(m.lines) == 0 {
		return
	}
	m.selected = (m.selected + delta + len(m.lines)) % len(m.lines)
	m.notice = ""
}

func (m *Model) copySelected() tea.Cmd {
	if m.selected >= len(m.lines) {
		return nil
	}
	l := m.lines[m.selected]
	copyFn := m.copy
	return func() tea.Msg {
		return copiedMsg{label: l.Label, err: copyFn(l.Value)}
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	t := m.theme
	var b strings.Builder
	b.WriteString(t.Title.Render("Inspector") + "\n\n")
	b.WriteString(m.input.View() + "\n\n")

	rows, sel := m.rows()
	if len(rows) == 0 {
		rows = []string{t.Muted.Render("the input is decoded as you type")}
	}
	// Keep the selected line visible when the result is taller than the popup.
	if m.height > 0 {
		avail := max(3, m.height-frameHeight-6)
		if len(rows) > avail {
			start := max(0, min(sel-avail/2, len(rows)-avail))
			rows = rows[start : start+avail]
		}
	}
	b.WriteString(strings.Join(rows, "\n") + "\n")

	help := "↑/↓ select · enter copy · esc back"
	if m.notice != "" {
		help = m.notice
	}
	b.WriteString("\n" + t.Muted.Render(help))
	return b.String()
}

// rows renders the result one screen row per string, and returns the index
// of the row holding the selected line.
func (m *Model) rows() ([]string, int) {
	t := m.theme
	width := m.width
	if width == 0 {
		width = 80
	}
	width -= frameWidth

	labelWidth := 0
	for _, l := range m.lines {
		labelWidth = max(labelWidth, min(maxLabelWidth, len(l.Label)))
	}
	valueWidth := max(10, width-labelWidth-4)
	indent := strings.Repeat(" ", labelWidth+4)

	var rows []string
	sel, i := 0, 0
	for si, s := range m.result.Sections {
		if si > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, t.Title.Render(s.Title))
		for _, l := range s.Lines {
			prefix, label := "  ", t.Muted
			if i == m.selected {
				prefix, label = "> ", t.Selected
				sel = len(rows)
			}
			value := t.Item
			if l.Label == "error" {
				value = t.Error
			}
			vlines := strings.Split(l.Value, "\n")
			more := 0
			if len(vlines) > maxValueLines {
				more = len(vlines) - maxValueLines
				vlines = vlines[:maxValueLines]
			}
			for j, v := range vlines {
				v = truncate(v, valueWidth)
				if j == 0 {
					head := label.Render(prefix + fmt.Sprintf("%-*s", labelWidth, truncate(l.Label, labelWidth)))
					rows = append(rows, head+"  "+value.Render(v))
				} else {
					rows = append(rows, indent+value.Render(v))
				}
			}
			if more > 0 {
				rows = append(rows, indent+t.Muted.Render(fmt.Sprintf("… %d more lines, enter copies all", more)))
			}
			i++
		}
	}
	return rows, sel
}

// truncate shortens s to at most width cells, ending in "…" when cut.
func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	r = r[:min(len(r), width)]
	for len(r) > 0 && lipgloss.Width(string(r))+1 > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
