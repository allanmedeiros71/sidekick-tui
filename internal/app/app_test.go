package app

import (
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
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.(Model).active != nil || m.(Model).notice == "" {
		t.Fatal("expected a coming-soon notice")
	}
}
