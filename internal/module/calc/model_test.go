package calc

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

func newTestModel() *Model {
	m := New(theme.Default())
	m.now = func() time.Time { return refNow }
	return m
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func TestModelEvaluatesWhileTyping(t *testing.T) {
	m := newTestModel()
	typeText(m, "0xff+1")
	if got := get(m.result, "dec"); got != "256" {
		t.Fatalf("live result = %q, want 256", got)
	}
	if !strings.Contains(m.View(), "0x100") {
		t.Error("view should show the hex result")
	}
}

func TestModelHistory(t *testing.T) {
	m := newTestModel()
	typeText(m, "2*21")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.input.Value() != "" || len(m.history) != 1 {
		t.Fatalf("after enter: input %q, history %d", m.input.Value(), len(m.history))
	}
	if !strings.Contains(m.View(), "2*21 = 42") {
		t.Error("view should list the history entry")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.input.Value() != "2*21" {
		t.Errorf("up recalled %q", m.input.Value())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.input.Value() != "" {
		t.Errorf("down past the newest entry left %q", m.input.Value())
	}
}

func TestModelKeepsErrorsOutOfHistory(t *testing.T) {
	m := newTestModel()
	typeText(m, "1/0")
	if m.err == nil {
		t.Fatal("expected an error")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.history) != 0 {
		t.Error("errors should not be kept in history")
	}
	if !strings.Contains(m.View(), "division by zero") {
		t.Error("view should show the error")
	}
}
