// Package app is the root Bubble Tea model: the popup frame and module menu.
package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
	"github.com/allanmedeiros71/sidekick-tui/internal/module"
	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

// Model is the top-level UI state.
type Model struct {
	theme  theme.Theme
	cursor int
	width  int
	height int
}

// New builds the root model from the loaded configuration.
func New(cfg config.Config) Model {
	return Model{theme: theme.Get(cfg.Theme)}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(module.Planned)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	t := m.theme
	var b strings.Builder
	b.WriteString(t.Title.Render("SideKick"))
	b.WriteString("\n\n")
	for i, mod := range module.Planned {
		prefix, style := "  ", t.Item
		if i == m.cursor {
			prefix, style = "> ", t.Selected
		}
		line := fmt.Sprintf("%s%-12s", prefix, mod.Title)
		b.WriteString(style.Render(line) + " " + t.Muted.Render(mod.Description) + "\n")
	}
	b.WriteString("\n" + t.Muted.Render("↑/↓ move · esc quit · modules coming soon"))

	box := t.Frame.Render(b.String())
	if m.width == 0 || m.height == 0 {
		return box
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
