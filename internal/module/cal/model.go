package cal

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/clipboard"
	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

// copiedMsg reports the outcome of copying a date to the clipboard.
type copiedMsg struct {
	date string
	err  error
}

// Model is the calendar's Bubble Tea model. It opens on today; the arrow
// keys move the selected day, g jumps to a date and enter copies the
// selected one.
type Model struct {
	theme  theme.Theme
	styles styles
	cursor time.Time // civil date
	input  textinput.Model
	going  bool // the go-to prompt is open
	notice string
	isErr  bool
	now    func() time.Time
	copy   func(string) error
}

// New returns a calendar drawn with the given theme.
func New(t theme.Theme) *Model {
	in := textinput.New()
	in.Prompt = "go to> "
	in.Placeholder = "2026-12-25, 2026-12, today + 90d, +2w"
	in.PromptStyle = t.Selected
	in.TextStyle = t.Item
	in.PlaceholderStyle = t.Muted
	in.CharLimit = 64
	in.Width = 40
	m := &Model{
		theme: t,
		styles: styles{
			header:  t.Title,
			weekNum: t.Muted,
			day:     t.Item,
			weekend: t.Muted,
			today:   t.Title.Underline(true),
			cursor:  t.Cursor,
		},
		input: in,
		now:   time.Now,
		copy:  clipboard.Write,
	}
	m.cursor = m.today()
	return m
}

func (m *Model) today() time.Time { return civil(m.now()) }

// ID implements module.Module.
func (m *Model) ID() string { return "cal" }

// Title implements module.Module.
func (m *Model) Title() string { return "Calendar" }

// HandlesEsc implements module.EscHandler: esc closes the go-to prompt
// before it leaves the calendar.
func (m *Model) HandlesEsc() bool { return m.going }

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case copiedMsg:
		if msg.err != nil {
			m.setNotice("copy failed: "+msg.err.Error(), true)
		} else {
			m.setNotice("copied "+msg.date, false)
		}
		return m, nil
	case tea.KeyMsg:
		if m.going {
			return m.updatePrompt(msg)
		}
		return m.updateGrid(msg)
	}
	return m, nil
}

func (m *Model) updateGrid(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	switch msg.String() {
	case "left", "h":
		m.cursor = m.cursor.AddDate(0, 0, -1)
	case "right", "l":
		m.cursor = m.cursor.AddDate(0, 0, 1)
	case "up", "k":
		m.cursor = m.cursor.AddDate(0, 0, -7)
	case "down", "j":
		m.cursor = m.cursor.AddDate(0, 0, 7)
	case "pgup", "<", ",":
		m.cursor = addMonths(m.cursor, -1)
	case "pgdown", ">", ".":
		m.cursor = addMonths(m.cursor, 1)
	case "[":
		m.cursor = addMonths(m.cursor, -12)
	case "]":
		m.cursor = addMonths(m.cursor, 12)
	case "t", "home":
		m.cursor = m.today()
	case "g", "/":
		m.going = true
		m.input.SetValue("")
		return m, m.input.Focus()
	case "enter":
		date := m.cursor.Format("2006-01-02")
		copyFn := m.copy
		return m, func() tea.Msg { return copiedMsg{date: date, err: copyFn(date)} }
	}
	return m, nil
}

func (m *Model) updatePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.closePrompt()
		return m, nil
	case tea.KeyEnter:
		d, err := Parse(m.input.Value(), m.now(), m.cursor)
		if err != nil {
			m.setNotice(err.Error(), true)
			return m, nil
		}
		m.cursor = d
		m.closePrompt()
		return m, nil
	}
	m.notice = ""
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) closePrompt() {
	m.going = false
	m.notice = ""
	m.input.Blur()
}

func (m *Model) setNotice(s string, isErr bool) {
	m.notice, m.isErr = s, isErr
}

// View implements tea.Model.
func (m *Model) View() string {
	t := m.theme
	today := m.today()
	var b strings.Builder
	b.WriteString(t.Title.Render("Calendar") + "\n\n")
	b.WriteString(month(m.cursor, today, m.styles) + "\n\n")
	b.WriteString(t.Item.Render(describe(m.cursor, today)) + "\n")

	if m.going {
		b.WriteString("\n" + m.input.View() + "\n")
	}
	help := "←/→ day · ↑/↓ week · pgup/pgdn month · [ ] year · t today\ng go to · enter copy · esc back"
	if m.going {
		help = "enter go · esc cancel"
	}
	switch {
	case m.notice != "" && m.isErr:
		help = t.Error.Render(m.notice)
	case m.notice != "":
		help = t.Muted.Render(m.notice)
	default:
		help = t.Muted.Render(help)
	}
	b.WriteString("\n" + help)
	return b.String()
}
