package inspect

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
	copied := new(string)
	m.copy = func(s string) error { *copied = s; return nil }
	return m, copied
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

// press sends a key and feeds the resulting message back, as Bubble Tea does.
func press(m *Model, k tea.KeyType) {
	_, cmd := m.Update(tea.KeyMsg{Type: k})
	if cmd != nil {
		m.Update(cmd())
	}
}

func TestModelInspectsWhileTyping(t *testing.T) {
	m, _ := newTestModel()
	typeText(m, "aGVsbG8=")
	if m.result.Kind != "base64" {
		t.Fatalf("kind = %q", m.result.Kind)
	}
	if v := m.View(); !strings.Contains(v, "hello") || !strings.Contains(v, "sha256") {
		t.Errorf("view should show the decoded text and the hashes:\n%s", v)
	}
}

func TestModelSelectAndCopy(t *testing.T) {
	m, copied := newTestModel()
	typeText(m, "hi")
	// encode: length, base64, ...
	press(m, tea.KeyDown)
	press(m, tea.KeyEnter)
	if *copied != "aGk=" {
		t.Fatalf("copied %q, want the base64 line", *copied)
	}
	if !strings.Contains(m.View(), "copied base64") {
		t.Error("view should confirm the copy")
	}
	press(m, tea.KeyUp)
	press(m, tea.KeyUp) // wraps around to the last line
	if l := m.lines[m.selected]; l.Label != "crc32" {
		t.Errorf("selected %q after wrapping", l.Label)
	}
	typeText(m, "!")
	if m.selected != 0 {
		t.Error("typing should reset the selection")
	}
}

func TestModelCopyError(t *testing.T) {
	m, _ := newTestModel()
	m.copy = func(string) error { return errors.New("no clipboard") }
	typeText(m, "x")
	press(m, tea.KeyEnter)
	if !strings.Contains(m.View(), "copy failed: no clipboard") {
		t.Error("view should report the copy error")
	}
}

func TestModelFitsWindow(t *testing.T) {
	m, _ := newTestModel()
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	typeText(m, `{"a":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15]}`)
	lines := strings.Split(m.View(), "\n")
	if len(lines) > 20-frameHeight {
		t.Errorf("view has %d lines, window fits %d", len(lines), 20-frameHeight)
	}
	for _, l := range lines {
		if w := len([]rune(stripANSI(l))); w > 60-frameWidth {
			t.Errorf("line %q is %d wide", l, w)
		}
	}
	// Scrolling down to the hashes keeps the selection visible.
	for range len(m.lines) - 1 {
		press(m, tea.KeyDown)
	}
	if !strings.Contains(m.View(), "> crc32") {
		t.Error("selected last line should be visible")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			esc = true
		case esc && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
			esc = false
		case !esc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 4); got != "abc…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("abc", 4); got != "abc" {
		t.Errorf("truncate = %q", got)
	}
}
