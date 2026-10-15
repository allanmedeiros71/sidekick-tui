package app

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
)

func TestEscQuits(t *testing.T) {
	_, cmd := New(config.Default()).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected tea.QuitMsg")
	}
}

func TestCursorStaysInBounds(t *testing.T) {
	var m tea.Model = New(config.Default())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if got := m.(Model).cursor; got != 0 {
		t.Fatalf("cursor = %d, want 0", got)
	}
}

func TestEnterOpensCalculatorAndEscReturns(t *testing.T) {
	var m tea.Model = New(config.Default())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.(Model).active == nil || m.(Model).active.ID() != "calc" {
		t.Fatal("enter on the first entry should open the calculator")
	}
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.(Model).active != nil {
		t.Fatal("esc should return to the menu")
	}
	if cmd != nil {
		t.Fatal("esc inside a module should not quit")
	}
}

func TestQTypesInsideModule(t *testing.T) {
	var m tea.Model = New(config.Default()).Open("calc")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("q inside a module should be typed, not quit")
		}
	}
	if m.(Model).active == nil {
		t.Fatal("module should still be open")
	}
}

func TestUnimplementedModuleShowsNotice(t *testing.T) {
	var m tea.Model = New(config.Default())
	for range 3 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.(Model).active != nil || m.(Model).notice == "" {
		t.Fatal("expected a coming-soon notice")
	}
}

func TestOpenedModuleLearnsWindowSize(t *testing.T) {
	var m tea.Model = New(config.Default())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 70, Height: 15})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	active := m.(Model).active
	if active == nil || active.ID() != "inspect" {
		t.Fatal("enter on the second entry should open the inspector")
	}
	for _, r := range "hello" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// The inspector's output is taller than 15 rows unless it knows the size.
	if n := strings.Count(m.(Model).active.View(), "\n") + 1; n > 15-4 {
		t.Fatalf("inspector view has %d rows in a 15-row window", n)
	}
}

// closer is a module that records being closed and can fail to.
type closer struct {
	closed int
	err    error
}

func (c *closer) Init() tea.Cmd                       { return nil }
func (c *closer) Update(tea.Msg) (tea.Model, tea.Cmd) { return c, nil }
func (c *closer) View() string                        { return "" }
func (c *closer) ID() string                          { return "closer" }
func (c *closer) Title() string                       { return "Closer" }
func (c *closer) Close() error                        { c.closed++; return c.err }

func TestLeavingModuleClosesIt(t *testing.T) {
	c := &closer{err: errors.New("disk full")}
	m := New(config.Default())
	m.active = c
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.closed != 1 || next.(Model).active == nil {
		t.Fatal("a module that fails to close should stay open")
	}
	c.err = nil
	next, _ = next.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.closed != 2 || next.(Model).active != nil {
		t.Fatal("esc should close the module and return to the menu")
	}
	m.active = c
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if c.closed != 3 || cmd == nil {
		t.Fatal("ctrl+c should close the module and quit")
	}
}
