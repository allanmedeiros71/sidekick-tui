// Package app is the root Bubble Tea model: the popup frame and module menu.
package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
	"github.com/allanmedeiros71/sidekick-tui/internal/module"
	"github.com/allanmedeiros71/sidekick-tui/internal/module/calc"
	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

// registry builds the implemented modules by ID. Menu entries without one
// are shown as "coming soon".
var registry = map[string]func(theme.Theme) module.Module{
	"calc": func(t theme.Theme) module.Module { return calc.New(t) },
}

// Implemented reports whether the module with the given ID can be opened.
func Implemented(id string) bool {
	_, ok := registry[id]
	return ok
}

// Model is the top-level UI state.
type Model struct {
	theme  theme.Theme
	cursor int
	width  int
	height int
	active module.Module // nil while the menu is shown
	notice string
	slide  slide
}

// New builds the root model from the loaded configuration.
func New(cfg config.Config) Model {
	return Model{theme: theme.Get(cfg.Theme), slide: newSlide(cfg.Animation)}
}

// Open returns m with the module id already open, as when sidekick is
// started with a module name. Unknown IDs leave the menu shown.
func (m Model) Open(id string) Model {
	for i, p := range module.Planned {
		if p.ID == id {
			m.cursor = i
		}
	}
	if build, ok := registry[id]; ok {
		m.active = build(m.theme)
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.slide.running() {
		cmds = append(cmds, slideTick())
	}
	if m.active != nil {
		cmds = append(cmds, m.active.Init())
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case slideTickMsg:
		m.slide.frame++
		if m.slide.running() {
			return m, slideTick()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.active != nil && msg.Type == tea.KeyEsc {
			m.active = nil
			return m, nil
		}
	}
	if m.active != nil {
		next, cmd := m.active.Update(msg)
		m.active = next.(module.Module)
		return m, cmd
	}
	if msg, ok := msg.(tea.KeyMsg); ok {
		return m.updateMenu(msg)
	}
	return m, nil
}

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	switch msg.String() {
	case "esc", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(module.Planned)-1 {
			m.cursor++
		}
	case "enter":
		p := module.Planned[m.cursor]
		build, ok := registry[p.ID]
		if !ok {
			m.notice = p.Title + " is coming soon"
			return m, nil
		}
		m.active = build(m.theme)
		return m, m.active.Init()
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var content string
	if m.active != nil {
		content = m.active.View()
	} else {
		content = m.menuView()
	}
	box := m.theme.Frame.Render(content)
	if m.width == 0 || m.height == 0 {
		if m.slide.running() {
			return "" // wait for the window size so the frame can slide in
		}
		return box
	}
	if m.slide.running() {
		return m.slide.place(box, m.width, m.height)
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) menuView() string {
	t := m.theme
	var b strings.Builder
	b.WriteString(t.Title.Render("SideKick"))
	b.WriteString("\n\n")
	for i, mod := range module.Planned {
		prefix, style := "  ", t.Item
		if i == m.cursor {
			prefix, style = "> ", t.Selected
		}
		desc := mod.Description
		if _, ok := registry[mod.ID]; !ok {
			desc += " (soon)"
		}
		line := fmt.Sprintf("%s%-12s", prefix, mod.Title)
		b.WriteString(style.Render(line) + " " + t.Muted.Render(desc) + "\n")
	}
	help := "↑/↓ move · enter open · esc quit"
	if m.notice != "" {
		help = m.notice
	}
	b.WriteString("\n" + t.Muted.Render(help))
	return b.String()
}
