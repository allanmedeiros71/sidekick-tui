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
