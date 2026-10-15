package cal

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

func newTestModel() (*Model, *string) {
	m := New(theme.Default())
	m.now = func() time.Time { return refNow }
	m.cursor = m.today()
	copied := new(string)
	m.copy = func(s string) error { *copied = s; return nil }
	return m, copied
}

// key sends a key and, for enter, feeds the resulting message back, as
// Bubble Tea does. Other commands are the prompt's cursor blink, which waits.
func key(m *Model, k string) {
	var msg tea.KeyMsg
	switch k {
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "pgup":
		msg = tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		msg = tea.KeyMsg{Type: tea.KeyPgDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
	_, cmd := m.Update(msg)
	if cmd != nil && k == "enter" {
		if next := cmd(); next != nil {
			m.Update(next)
		}
	}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		key(m, string(r))
	}
}

func cursorIs(t *testing.T, m *Model, want string) {
	t.Helper()
	if got := m.cursor.Format("2006-01-02"); got != want {
		t.Errorf("cursor = %s, want %s", got, want)
	}
}

func TestModelNavigates(t *testing.T) {
	m, _ := newTestModel()
	cursorIs(t, m, "2026-10-15")
	key(m, "right")
	key(m, "down")
	cursorIs(t, m, "2026-10-23")
	key(m, "pgdown")
	cursorIs(t, m, "2026-11-23")
	key(m, "]")
	cursorIs(t, m, "2027-11-23")
	key(m, "pgup")
	key(m, "[")
	key(m, "up")
	key(m, "left")
	cursorIs(t, m, "2026-10-15")
	key(m, "pgdown")
	key(m, "t")
	cursorIs(t, m, "2026-10-15")
	if v := m.View(); !strings.Contains(v, "October 2026") || !strings.Contains(v, "week 42") {
		t.Errorf("view should show the month and the selected day:\n%s", v)
	}
}

func TestModelGoTo(t *testing.T) {
	m, _ := newTestModel()
	key(m, "g")
	if !m.HandlesEsc() {
		t.Fatal("the prompt should keep esc from leaving the calendar")
	}
	typeText(m, "today + 90d")
	key(m, "enter")
	cursorIs(t, m, "2027-01-13")
	if m.HandlesEsc() {
		t.Error("the prompt should close after a jump")
	}

	key(m, "g")
	typeText(m, "nonsense")
	key(m, "enter")
	if !m.going || !strings.Contains(m.View(), "is not a date") {
		t.Errorf("a bad date should keep the prompt open with the error:\n%s", m.View())
	}
	key(m, "esc")
	if m.going {
		t.Error("esc should close the prompt")
	}
	cursorIs(t, m, "2027-01-13")
}

func TestModelCopy(t *testing.T) {
	m, copied := newTestModel()
	key(m, "right")
	key(m, "enter")
	if *copied != "2026-10-16" {
		t.Fatalf("copied %q", *copied)
	}
	if !strings.Contains(m.View(), "copied 2026-10-16") {
		t.Error("view should confirm the copy")
	}
	m.copy = func(string) error { return errors.New("no clipboard") }
	key(m, "enter")
	if !strings.Contains(m.View(), "copy failed: no clipboard") {
		t.Error("view should report the copy error")
	}
}
